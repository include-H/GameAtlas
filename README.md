# GameAtlas

一个面向 NAS / 局域网场景的游戏库管理系统。

当前主线架构：

- 后端：Go + Gin + SQLite
- 前端：Vue 3 + TypeScript + Vite + Pinia + Arco Design Vue

项目目标是提供一个更稳定、易部署、适合家庭局域网或单机环境使用的游戏库服务，替代更早期的 Node/Vue 实现。

## 当前能力

- 游戏库浏览
  - 仪表盘、游戏列表、游戏详情
  - 搜索、分页、排序、系列 / 平台 / 标签筛选
  - 发售时间轴
- 媒体与素材
  - 封面、横幅、截图、预告视频上传与管理
  - 截图 / 视频排序
  - 主预告视频切换
- 文件管理
  - 为游戏登记多个文件条目
  - 校验文件必须位于允许的根目录内
  - 文件下载
  - 为 `.vhd` / `.vhdx` 生成 Windows 启动 BAT
- Wiki
  - Markdown 渲染
  - Wiki 编辑
  - 历史记录与恢复
  - 目录提取
- 元数据与标签
  - 系列、平台、开发商、发行商维护
  - 标签组与标签系统
  - 系列库与系列详情页
- 补全工作流
  - 待处理中心
  - 缺失封面 / 横幅 / 截图 / Wiki / 文件 / 基础信息识别
  - 支持忽略指定待处理项
- Steam 集成
  - Steam 搜索
  - 预览封面 / 横幅 / 截图 / 预告视频候选
  - 将远程素材落库到本地素材目录
- 前端体验
  - 匿名浏览，管理员增强编辑
  - 自定义共享背景图
  - 自定义全局字体
  - 本地 UI 偏好持久化

## 权限模型

- 默认浏览场景是匿名可访问。
- 管理员登录后可进行新增、编辑、删除、素材上传、Wiki 编辑、待处理项处理等操作。
- 登录态由后端 cookie 提供，前端通过 `/api/auth/me` 判断是否为管理员。
- 后端现在要求必须配置 `ADMIN_PASSWORD` 与非默认 `SESSION_SECRET`，缺失时会直接拒绝启动。
- 文件下载与 SMB 启动脚本仍支持“管理员或局域网访问”，这是当前局域网场景下保留的特定边界。

## 主要页面

- `/`
  仪表盘，包含轮播、统计、最近添加、下载最多。
- `/games`
  游戏库列表，支持搜索、筛选、排序、分页、收藏筛选。
- `/games/:id`
  游戏详情页，包含媒体轮播、下载版本、启动脚本、Wiki 展示。
- `/games/timeline`
  发售时间轴。
- `/games/pending`
  待处理中心，仅管理员可进入。
- `/series`
  系列库。
- `/series/:id`
  系列详情。
- `/wiki/:gameId/edit`
  Wiki 编辑页，仅管理员可进入。
- `/login`
  管理员登录页。

## 项目结构

```text
.
├── backend/            # Go 后端、数据库迁移、嵌入式前端资源
├── frontend/           # Vue 3 前端
├── Wiki/               # 仓库内维护的游戏 Wiki 与系列目录
├── build-release.sh    # 生产打包脚本
└── start-dev.sh        # 本地联调脚本
```

## 开发环境

建议准备：

- Go 1.22 或更高版本
- Node.js 与 npm
- `curl`

前端依赖使用 npm 管理，后端数据库为 SQLite。

## 本地开发

根目录直接执行：

```bash
bash start-dev.sh
```

脚本会自动：

- 检查 `go`、`npm`、`curl`
- 在前端依赖缺失时执行 `npm install`
- 预热 Go 依赖
- 启动后端
- 等待健康检查通过
- 启动前端开发服务器

默认地址：

- 前端开发：`http://127.0.0.1:5173`
- 后端接口：`http://127.0.0.1:3000`

如果当前目录没有 `backend/.env`，开发脚本会在检测到示例数据库和素材目录时，自动回退使用：

- `DB_PATH=data/app.db`
- `ASSETS_DIR=data/gamelist`

也可以分别手动启动：

```bash
cd backend
go run ./cmd/server
```

```bash
cd frontend
npm install
npm run dev
```

## 后端配置

后端默认读取：

- `backend/.env`

常用配置如下。

### 服务与路径

- `APP_ENV`
  运行环境，常见值为 `development` / `production`
- `HOST`
  服务监听地址，默认 `0.0.0.0`
- `PORT`
  服务端口，默认 `3000`
- `DB_PATH`
  SQLite 数据库路径，默认 `data/db.db`
- `STATIC_DIR`
  磁盘前端静态目录，开发环境默认 `../frontend/dist`
- `ASSETS_DIR`
  游戏素材目录，默认 `data/gamelist`

### 文件访问边界

- `PRIMARY_ROM_ROOT`
  唯一的 ROM 根目录；目录浏览、文件登记和下载都限制在这个目录及其子目录内

### 认证与登录限制

- `ADMIN_DISPLAY_NAME`
  前端显示用的管理员名称；后端运行时读取并通过接口返回给前端
- `ADMIN_PASSWORD`
  管理员密码
- `SESSION_SECRET`
  会话签名密钥；发布脚本会自动生成一个随机值
- `AUTH_MAX_FAILS`
  最大连续失败次数
- `AUTH_COOLDOWN`
  锁定冷却时间，例如 `10m`
- `AUTH_FAIL_WINDOW`
  失败统计窗口，例如 `30m`
- `AUTH_STATE_TTL`
  登录失败状态保留时长
- `AUTH_TRACK_BY`
  登录限流维度，当前常用 `ip` 或 `ip_ua`

### Wiki

- `WIKI_HISTORY_LIMIT`
  每个游戏保留的 Wiki 历史条目数，`0` 表示不自动清理

### 代理

- `PROXY`
  默认出站代理

### VHD / VHDX 启动脚本

- `SMB_SHARE_ROOT`
  生成 BAT 时使用的 SMB 根路径，例如 `\\192.168.1.4\Game`
- `SMB_USERNAME`
  固定 SMB 用户名
- `SMB_PASSWORD`
  固定 SMB 密码
- `VHD_DIFF_ROOT`
  差分 VHDX 挂载根盘符，例如 `C:`

## 数据与素材目录约定

默认路径示例：

- 数据库：`backend/data/db.db`
- 游戏素材：`backend/data/gamelist`
- ROM 根目录：`backend/ROM`

常见素材结构：

```text
backend/data/gamelist/1/cover.jpg
backend/data/gamelist/1/banner.jpg
backend/data/gamelist/1/1.jpg
backend/data/gamelist/1/2.jpg
backend/data/gamelist/1/video-1.mp4
```

后端会公开只读素材地址：

- `/assets/*`
  游戏封面、横幅、截图、视频等
- `/data/*`
  自定义背景图、字体等公共静态资源

## 自定义背景和字体

前端会自动尝试读取以下固定文件名：

- `backend/data/bg.jpg`
  作为共享背景图；若不存在，则回退到系统挑选的游戏素材背景

全局默认字体已经随前端构建一起打包，当前仓库默认使用：

- `frontend/src/assets/fonts/LXGWWenKaiGBScreen.woff2`

如果要替换全局字体，更新前端资源并重新构建发布包即可。

## Wiki 说明

Wiki 内容使用 Markdown。

后端负责保存内容、生成 HTML 和维护历史记录；前端额外提供目录提取与部分展示增强样式。

仓库内另外维护了一套人工整理的系列 Wiki 文档目录，位于：

- `Wiki/README.md`
  游戏 Wiki 总目录
- `Wiki/Assassins Creed/README.md`
  《刺客信条》系列目录
- `Wiki/Call Of Duty/README.md`
  《使命召唤》系列目录
- `Wiki/Half-Life/README.md`
  《半条命》系列目录
- `Wiki/Counter-Strike/README.md`
  《反恐精英》系列目录
- `Wiki/Batman/README.md`
  蝙蝠侠游戏目录

这套目录主要用于沉淀“中文整理型游戏 Wiki”条目，和应用内单游戏 Wiki 能力并不冲突：

- 应用内 Wiki 更偏向每个游戏的数据内容展示与编辑
- 仓库内 `Wiki/` 更偏向系列索引、写作规范与长期整理沉淀

当前前端还支持题记块语法 `:::epigraph`，适合在条目开头放引言、题记或引用。示例：

```md
:::epigraph
如果星辰日月都位于人类贪婪之手可触距离之内。
The sun, the moon and the stars would have disappeared long ago...
那么他们都将不复存在。
had they happened to be in the reach of predatory human hands.
-- Havelock Ellis
:::
```

## 生产打包

执行：

```bash
bash build-release.sh
```

或指定版本名：

```bash
bash build-release.sh v1.0.0
```

脚本会：

- 构建前端
- 将前端构建结果复制到后端嵌入目录
- 编译 `game-server`
- 生成独立发布目录
- 写入运行用 `.env`
- 自动生成随机 `SESSION_SECRET`

发布目录类似：

```text
release/game-release-<version>/
├── game-server
├── start.sh
├── .env
├── data/
│   ├── gamelist/
│   └── bg.jpg
└── ROM/
```

说明：

- 发布包会内嵌前端静态资源。
- 生产运行时不需要额外携带 `frontend/dist`。
- 首次运行后数据库会在 `data/db.db` 处自动创建。

## 部署建议

进入发布目录后可直接执行：

```bash
./start.sh
```

也可以将 `game-server` 配成 `systemd` 服务。

即使部署在纯内网环境，当前后端也要求配置 `ADMIN_PASSWORD`，并使用安全的 `SESSION_SECRET`。发布脚本会自动写入随机 `SESSION_SECRET`；如果存在跨网段访问、反向代理暴露或公网入口，仍建议补充 HTTPS、外层访问控制，以及对“局域网免登录下载/启动”边界做网络隔离。

## 后端接口概览

当前主要接口包括：

- `/api/auth/*`
  登录、退出、当前身份
- `/api/games/*`
  游戏列表、详情、统计、时间轴、创建、更新、删除
- `/api/games/:id/files/*`
  游戏文件条目与下载
- `/api/games/:id/wiki*`
  Wiki 与历史记录
- `/api/series`、`/api/platforms`、`/api/developers`、`/api/publishers`
  元数据接口
- `/api/tag-groups`、`/api/tags`
  标签系统
- `/api/assets/*`
  素材上传、删除、排序、主视频切换
- `/api/directory/*`
  目录浏览
- `/api/steam/*`
  Steam 搜索、素材预览、素材落库
- `/api/games/:id/review-issues/*`
  待处理项忽略

## 当前说明

- 当前仓库以 `backend/` 和 `frontend/` 为准。
- 根 README 以当前 Go 后端 + Vue 3 前端实现为主，不再描述更早期旧版本结构。
- 如需生成可部署版本，优先使用 [build-release.sh](/home/Hao/Game/build-release.sh)。
