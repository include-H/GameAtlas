# GameAtlas

GameAtlas 是一个面向 NAS / 局域网 / 家庭游戏库的网页游戏管理系统。它把零散在网络共享里的游戏整理成可浏览、可检索、可管理、可远程启动的媒体库，并通过 VHD 差分盘机制，让多台 Windows 设备共享同一份游戏文件、各自独立运行。

## 功能总览

| 能力 | 说明 |
| --- | --- |
| 网页游戏库 | 封面、横幅、截图、视频、Wiki、系列 / 开发商 / 发行商信息一站式管理 |
| 素材在线导入 | Steam 元数据与素材搜索，SteamGridDB 封面 / 横幅 / Logo 直搜，多选批量导入 |
| VHD 远程启动 | 服务端只读共享基础盘，Windows 客户端自动创建本地差分盘并挂载游玩 |
| 浏览器串流 | Moonlight Web 客户端（Go 代理 + WebCodecs），浏览器即开即玩 Sunshine 主机上的游戏，H.264 / HEVC / AV1 |
| 数据保护 | SQLite 在线备份（`VACUUM INTO`）、孤儿素材隔离、启动资产对账 |
| 权限体系 | 匿名可浏览公开游戏，管理员管理私有游戏与全部写操作 |

## 为什么要用 VHD 玩，而不是直接共享游戏文件夹

家庭 / NAS 场景里最常见的痛点是：游戏文件很大（动辄几十甚至上百 GB），而同一份游戏往往要在多台 Windows 电脑上玩。

如果直接把游戏文件夹放进 SMB 共享：

- **只读共享**：可以运行游戏，但存档、设置、补丁等一切写入都会失败；
- **读写共享**：多台机器同时写同一份文件，轻则互相覆盖存档，重则损坏游戏本体；
- **每台机器各复制一份**：硬盘空间成倍占用，更新一个版本要在所有机器上重复同步。

VHD 差分盘（differencing VHD）正好解决这个矛盾：

1. 把游戏完整装进一个基础 VHDX，这个文件只保存一份，放在 NAS 上通过 SMB **只读**共享；
2. 客户端首次启动时，脚本在本机磁盘上基于基础盘创建一个**差分盘**（diff disk）；
3. 游戏运行时的所有写入（存档、配置、日志）都落在本地差分盘里，基础盘永远不被改动；
4. 想换版本，管理员只替换服务器上的基础 VHDX 即可，客户端下次挂载自动使用新版本；
5. 想回到干净状态，删除本地差分盘重新生成即可，不影响其他机器。

所以 VHD 不是为了让文件"能运行"，而是为了在**只共享一份大文件**的前提下，让**每台客户端都拥有独立可写、随时可重置的私有副本**。

## VHD 远程启动

这是 GameAtlas 最核心的使用方式。

### 工作原理

1. 游戏文件登记为 `.vhd` / `.vhdx`，存放在服务端 `PRIMARY_ROM_ROOT` 目录内；
2. 在网页详情页点击"开始游玩"，后端生成一个 Windows 启动脚本（按游戏标题命名的 `.bat`，按标题下载）；
3. 客户端以管理员权限运行：脚本自动请求提权 → 复用已连接的 SMB 共享（已连接则跳过凭据注入）→ 创建本地差分盘（已存在则直接复用；**已挂载则跳过 attach 直接复用**）→ 通过 `diskpart` 挂载 VHD → 自动扫描盘符内的游戏主程序并启动；
4. 游玩结束后回到脚本窗口按任意键，脚本自动卸载 VHD、断开 SMB 共享并删除凭据。

脚本主体是 PowerShell（`.bat` 只是提权 + 解码内嵌 payload 的引导壳，完全免维护），提供动态菜单：**挂载并游玩（自动启动游戏，结束后自动卸载 + 清理凭据）**、**仅挂载**、**打开存档目录**（仅当游戏配置了存档路径时出现，便于手动备份）、**清理 SMB 凭据并断开共享**。

存档目录支持 Windows 变量模板（`%USERPROFILE%`、`%APPDATA%`、`%LOCALAPPDATA%`、`%GAME_DRIVE%` 游戏盘符等），在游戏编辑页配置，例如 `%USERPROFILE%\Documents\My Games\GT4\SaveGame`。存档打开逻辑按模板自动分流：模板引用 `%GAME_DRIVE%`（存档在 VHD 内）时先挂载 VHD 再打开并随后卸载清理；模板为本地路径时直接打开文件夹，不触碰 SMB 与 VHD。

### 服务端条件

- 游戏文件位于 `PRIMARY_ROM_ROOT` 内，扩展名为 `.vhd` / `.vhdx`；
- 目录通过 SMB 共享暴露给客户端；
- 在设置页正确配置 `SMB_PATH_MAPPINGS`、`SMB_USERNAME`、`SMB_PASSWORD`、`VHD_DIFF_ROOT`。

### 客户端条件

- 能访问服务端 SMB 共享；
- 支持 `cmdkey`、`net use`、`diskpart`、PowerShell 5.1+（Windows 自带）；
- 脚本需要管理员权限运行（自动请求提权）。

### 从零到可玩

1. 在 Windows 上用 PowerShell（管理员）创建基础 VHDX 并灌入游戏：

```powershell
$vhdx = "D:\GameBase\PS2_GT4.vhdx"

# 创建动态扩展的 VHDX（大小按需增长）
New-VHD -Path $vhdx -SizeBytes 120GB -Dynamic
Mount-VHD -Path $vhdx

# 初始化磁盘、创建分区并格式化为 NTFS
$diskNumber = (Get-VHD -Path $vhdx).DiskNumber
Initialize-Disk -Number $diskNumber -PartitionStyle GPT
$partition = New-Partition -DiskNumber $diskNumber -UseMaximumSize -AssignDriveLetter
Format-Volume -Partition $partition -FileSystem NTFS -NewFileSystemLabel "GT4_BASE"

# 把游戏文件复制到新盘符
$drive = "$($partition.DriveLetter):"
Copy-Item -Path "D:\Downloads\GT4" -Destination $drive -Recurse

# 卸载 VHD，然后放到 NAS 上
Dismount-VHD -Path $vhdx
```

2. 将 VHDX 放到服务端 `PRIMARY_ROM_ROOT` 对应的目录；
3. 后台新增游戏，在"文件版本"里添加这个 `.vhd` / `.vhdx` 路径；
4. 在设置页配置 SMB 映射（例如 `/mnt/Game=\\192.168.1.4\Game`）与账号密码；
5. 打开游戏详情页，点击"开始游玩"下载 BAT，在 Windows 客户端上管理员运行。

### 常见问题

- `SMB_PATH_MAPPINGS` 配错 → 客户端连不上共享；
- `PRIMARY_ROM_ROOT` 与 SMB 实际暴露的目录不一致 → 服务端找不到文件或路径被拒绝；
- 文件不在 `PRIMARY_ROM_ROOT` 内，或不是 `.vhd` / `.vhdx` → 详情页不提供"开始游玩"；
- 客户端没有管理员权限 → `diskpart` 挂载失败；
- 差分盘路径为 `<VHD_DIFF_ROOT>\<基础盘文件名>`（例如 `C:\PS2_GT4.vhdx`），同名基础盘会产生冲突，建议基础盘文件名唯一。

> 安全提示：BAT 脚本中会携带 SMB 账号密码，因此本功能面向家庭 / 内网等可信环境。建议 SMB 共享使用**只读账号**；游玩结束后脚本会自动删除已保存的凭据，若选择了"仅挂载"，请在公共机器上使用清理选项。

## 浏览器串流（Moonlight Web）

无需安装任何客户端，浏览器直接玩 Sunshine / GameStream 主机上的游戏——NAS 局域网内任何带 Chromium 系浏览器的设备（笔记本 / 平板 / 电视）即开即玩。

### 架构

浏览器无法发裸 UDP，GameStream 协议需要本地代理桥接：

```
浏览器串流页 (:47999, HTTPS 自签)         NAS Go 代理（并入 server 进程）          Sunshine 主机
  Vue 3 + Arco UI（云串流页，独立文档）  ◄── WebSocket 多路复用 ──►  UDP/TCP 转发 ──►  ATRI 等
  WebCodecs 硬解 + Pointer/Keyboard Lock  /api/pair /api/applist /api/launch（mTLS 持证）
```

- **串流端口**：默认 `:47999`（`STREAM_PORT` 可配），自签 TLS；主站 `:3000` 保持 HTTP 不受影响
- **配对**：5 步 NvHTTP 握手由 Go 代理代跑；配对身份（客户端证书）与主机证书缓存在 `data/streaming/`，**多浏览器共享同一设备身份**（Sunshine 里只显示一个 GameAtlas 设备）
- **解码**：wasm 跑 moonlight-common-c 协议栈（RTSP/FEC/分帧），WebCodecs 硬件解码（H.264 默认 / HEVC / AV1 按设备能力可选），1080p60 起步
- **主机与设置持久化**：主机列表、串流设置存 `data/streaming/hosts.json` / `stream-settings.json`，换设备不丢
- **wasm 来源**：`frontend/public/wasm/` 为 moonlight-common-c（GPL-2.0-or-later）官方预编译产物，构建脚本 `wasm/build.sh`（需 Emscripten）保留

### 使用

1. 浏览器打开 `https://<NAS>:47999`（首次需信任自签证书，点"高级 → 继续"）
2. 添加 Sunshine 主机 IP → 配对（Sunshine Web UI 输入 4 位 PIN）
3. 选应用 → 串流。默认窗口模式（鼠标 1:1），全屏按钮可选

### 快捷键与操作

| 按键 | 行为 |
|---|---|
| 点击画面 | 获取鼠标（Pointer Lock） |
| Esc 短按 / 长按 | 游戏内 Esc（需全屏）/ 释放鼠标 |
| Ctrl+Alt+Shift+Q | 退出串流 |

> 已知限制：远端主机建议 100% 显示缩放（否则鼠标映射比例错乱）；HEVC 解码需浏览器端插件（Windows Edge 需 HEVC 视频扩展）；窗口模式下 Esc 由浏览器消费（游戏内 Esc 需全屏）。

## 前端功能

### 首页（Dashboard）

- 顶部数据统计：游戏总数、总下载数、收藏数、待处理数；
- 游戏横排推荐：最近添加、下载最多、我的收藏、最近更新；
- 待处理概览与快捷入口（添加游戏、进入待处理工作台、游戏店、游戏库）。

### 游戏库

- 卡片网格 / 列表两种视图；
- 搜索标题，按标题、发售日期、下载量、创建 / 更新时间、随机等方式排序；
- 支持仅收藏、仅私有、按系列 / 待处理问题筛选；
- 分页、排序、筛选状态同步到 URL，可分享、可刷新。

### 发售时间线

按发售日期排列全部游戏的时间线视图，适合按年代浏览库容。

### 游戏店

沉浸式"电玩店"浏览场景：货架陈列、海报墙、CRT 电视播放预览视频，提供新到货 / 畅销榜等随机发现入口，适合"不知道玩什么"时随便逛逛。

### 开始屏幕（Win8 磁贴）

顶栏"开始"按钮打开全屏磁贴墙，默认铺开收藏游戏；点击磁贴直接开始游戏（单个 VHD 版本直接启动，多版本弹窗选择，无可启动版本则进详情页）；支持编辑模式：拖动排序、切换小/宽/大三种磁贴尺寸、增删磁贴、为每列命名，并可用游戏 banner 裁剪出各尺寸磁贴图；自定义布局跨设备保存在服务端。

### 游戏详情

- 截图轮播与视频预览；
- 简介、Wiki 正文、系列 / 开发商 / 发行商信息；
- 文件版本列表：每个版本可下载，`.vhd` / `.vhdx` 版本提供"开始游玩"；
- 收藏、编辑、公开 / 私有可见性控制。

### Wiki 系统

- Markdown 编辑与渲染，支持目录（TOC）与 `:::epigraph` 引用块；
- 每个游戏保留历史版本与变更摘要，可回溯。

### Wiki 批量同步

本地 `Game_Wiki` 仓库（Markdown 源文件）与生产库之间可以用脚本批量同步 Wiki 正文，适合在仓库里统一维护、再推到服务器：

```bash
python3 scripts/sync_wiki_to_prod.py            # 增量同步（内容无变化自动跳过）
python3 scripts/sync_wiki_to_prod.py --force    # 强制覆盖全部
python3 scripts/sync_wiki_to_prod.py --dry-run  # 只打印计划，不写入
python3 scripts/sync_wiki_to_prod.py --game "半条命"  # 只同步标题匹配的游戏
python3 scripts/sync_wiki_to_prod.py --list     # 列出全部游戏与本地文件映射
python3 scripts/sync_wiki_to_prod.py --unmatched  # 显示无法匹配的游戏
```

- 生产同步必须显式提供 `GA_URL` / `GA_PASSWORD`，避免脚本误连生产库或使用仓库内默认凭据；
- 通过 `GET /api/games` 分页拉取生产库全部游戏（含无发售日期的条目），按标题自动匹配 + 手工映射表（`MANUAL_MAP`）解析到本地 `Game_Wiki` 文件，重复标题按 `public_id` 消歧；
- 写入走管理员会话 + `PUT /api/games/:publicId/wiki`，每次写入附带"同步本地重构后的 Wiki"变更摘要，可在历史版本中回溯。

### 系列库 / 发行商库

按系列和发行商聚合游戏，提供分组浏览与详情页。

### 待处理工作台（管理员）

自动检查游戏完整性：缺封面 / 横幅 / 截图 / Logo、缺 Wiki、缺文件、缺开发商 / 发行商 / 简介；支持严重度标记、按问题筛选、批量忽略并填写原因。

### 素材导入

- Steam 搜索：一键带入标题、别名、简介、发售日期、开发商、发行商；
- 封面 / 横幅 / 截图：可从 Steam 或 SteamGridDB 多选导入；
- Logo：SteamGridDB 直搜，多选导入；
- 文件浏览器：模糊搜索服务端目录，直接选择游戏文件路径。

### 设置页（管理员）

- 认证配置（管理员密码、登录失败次数、冷却时间等）；
- 路径与备份策略（素材目录、ROM 根目录、备份间隔与保留份数）；
- SMB / VHD 配置（路径映射、账号密码、差分盘根路径）；
- 网络配置（出站代理、SteamGridDB API Key）；
- 自定义全局背景图上传 / 删除；
- 数据维护（扫描游戏文件并刷新大小）、重启服务端。

## 界面预览

| 首页 | 时间线 | 系列库 |
| --- | --- | --- |
| ![首页](Readme/首页.jpg) | ![时间线](Readme/时间线.jpg) | ![系列库](Readme/系列库.jpg) |

| 游戏编辑页 | 详情页 |
| --- | --- |
| ![游戏编辑页](Readme/游戏编辑页.jpg) | ![详情页](Readme/详情页.jpg) |

## 技术栈

- 后端：Go + Gin + SQLite（SQLX）
- 前端：Vue 3 + TypeScript + Vite + Pinia + Arco Design Vue

## 本地开发

需要 Go 1.25+、Node.js、npm、curl。

```bash
bash start-dev.sh          # 一键启动：后端 :3000，前端 :5173
```

或手动启动：

```bash
cd backend && go run ./cmd/server
cd frontend && npm install && npm run dev
```

## 部署

### 方式一：Docker（推荐）

```bash
docker run -d \
  --name gameatlas \
  -p 3000:3000 \
  -v /mnt/Docker/GameAtlas/data:/app/data \
  -v /mnt:/mnt:ro \
  -e ADMIN_PASSWORD=yourpassword \
  hao0114/gameatlas:latest
```

或使用仓库内的 `docker-compose.yml`。

首次启动会自动创建数据库和默认配置；之后运行配置由数据库管理。

### 方式二：二进制发布包

```bash
bash build-release.sh              # 输出到 release/game-release-<version>/
bash build-release.sh v1.0.0       # 自定义版本名
```

发布目录结构：

```text
release/game-release-<version>/
├── game-server
├── data/                 # 首次启动自动创建
│   ├── db.db             # SQLite 数据库与运行配置
│   ├── backups/          # 在线数据库备份
│   ├── orphaned-assets/  # 被隔离的未登记素材
│   └── gamelist/         # 素材目录
└── ROM/                  # 游戏文件目录
```

进入发布目录运行 `./game-server` 或 `./start.sh` 即可。

### GitHub Release 自动发版

推送 `v*` 格式的 tag 会自动构建并上传 GitHub Release 与 Docker Hub：

```bash
git tag v1.0.0 && git push origin v1.0.0
```

产物：
- GitHub Release：`linux-amd64` 的 `tar.gz`、`zip` 与 `sha256` 校验文件；
- Docker Hub：`hao0114/gameatlas:v1.0.0` 与 `hao0114/gameatlas:latest`。

## 配置

运行配置存储在 SQLite 的 `app_settings` 表中，首次启动写入默认值（默认管理员密码 `1234`）。登录后可在设置页修改密码、素材目录、ROM 根目录、SMB 映射、备份策略、SteamGridDB API Key 等；大部分配置保存后需重启服务生效。

> 默认密码 `1234` 仅面向家庭 / 内网可信部署保留（保证未显式配置时也能开箱即用）。暴露到公网前必须先在设置页或部署环境变量（`ADMIN_PASSWORD`）中改掉默认密码。

### 串流相关配置

| 环境变量 | 默认 | 说明 |
|---|---|---|
| `STREAM_ENABLED` | `true` | 浏览器串流总开关 |
| `STREAM_HOST` / `STREAM_PORT` | `0.0.0.0` / `47999` | 串流端口监听地址与端口 |
| `STREAM_DATA_DIR` | `data/streaming` | 配对身份 / 主机证书 / 主机与设置 JSON 存放目录 |
| `STREAM_WWW_ROOT` | `../frontend/dist/streaming-www` | 串流页前端产物目录（vite 构建自动组装） |

旧版部署中的 `.env` 不再被自动读取、导入或删除；运行时配置请通过设置页录入。

`DB_PATH` 是唯一的启动引导项，默认 `data/db.db`；如需更改数据库文件位置：

```bash
DB_PATH=/path/to/app.db ./game-server
```

## 数据保护

- 启动时用 SQLite `VACUUM INTO` 创建一致性备份，之后按配置间隔定时备份（默认 24 小时），并按保留份数清理（默认保留 5 份）；
- 启动扫描到数据库未登记的封面、截图、视频等素材时，统一移动到 `data/orphaned-assets/` 隔离，固定保留 7 天，便于在 NAS 文件管理器中排查；
- 不会直接复制 WAL 模式下的单个 `db.db` 文件作为备份；手工备份请使用应用生成的备份文件或 SQLite 在线备份命令。

## 权限

- 匿名可浏览公开游戏；
- 管理员登录后可执行写操作（增删改、上传素材、编辑 Wiki、待处理工作台、设置页）；
- 私有游戏仅管理员可见，其素材对匿名访问同样隐藏；
- 收藏是全局状态（`favorite_games` 无用户维度），匿名访客也可切换收藏；
- 暴露到公网时建议加反向代理 + HTTPS + 外层访问控制。

## 更多文档

- 后端开发指南：[backend/README.md](backend/README.md)
- 开发规范：[docs/项目风格约定.md](docs/项目风格约定.md)
- 历史审计基线：[docs/项目审查.md](docs/项目审查.md)
