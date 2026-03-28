# Backend

GameAtlas 当前主线后端，提供游戏库 API、文件与素材管理、Wiki、标签、认证、Steam 集成，以及前端静态资源托管。

## 技术栈

- Go
- Gin
- SQLite
- `sqlx`

当前 `go.mod` 声明版本为 Go 1.22。

## 当前能力

- 配置加载
  - 自动读取 `backend/.env`
  - 支持路径、认证、代理、VHD 启动脚本等配置
- 数据层
  - SQLite 连接初始化
  - 自动执行数据库迁移
- 游戏库
  - 列表、详情、创建、更新、删除
  - 搜索、分页、排序、系列 / 平台 / 标签筛选
  - 统计数据
  - 发售时间轴
  - 公有 / 私有可见性控制
- 文件管理
  - 游戏文件条目管理
  - 路径必须落在允许根目录内
  - 下载文件
  - 为 `.vhd` / `.vhdx` 生成 Windows 启动 BAT
- 素材管理
  - 封面、横幅、截图、视频上传
  - 删除、排序、主视频切换
  - 将远程素材下载并保存到本地素材目录
- Wiki
  - 读取、更新
  - Markdown 渲染
  - 历史记录
  - 自动裁剪历史条目
- 元数据
  - `series`
  - `platforms`
  - `developers`
  - `publishers`
- 标签系统
  - 标签组
  - 标签
  - 游戏标签绑定与过滤
- 待处理项忽略
  - 缺封面、缺横幅、缺截图、缺 Wiki、缺文件、缺摘要等问题的忽略覆盖
- 认证
  - 管理员登录 / 退出 / 当前身份
  - 登录失败次数统计与冷却锁定
- Steam 集成
  - 搜索
  - 素材预览
  - 素材落库
- 静态资源托管
  - 磁盘 `frontend/dist` 优先
  - 内嵌 `web/dist` 兜底
  - `/assets/*` 与 `/data/*` 只读公开

## 快速启动

1. 安装 Go 1.22 或更新版本。
2. 复制 `backend/.env.example` 为 `backend/.env`。
3. 按你的本地环境修改数据库、素材目录、ROM 根目录等配置。
4. 运行：

```bash
cd backend
go run ./cmd/server
```

默认情况下：

- API 位于 `/api/*`
- 健康检查为 `/api/health`
- 如果磁盘 `frontend/dist` 存在，后端会优先托管它
- 如果磁盘静态资源不存在，后端会回退到嵌入式 `web/dist`

## 本地校验

推荐在 `backend/` 下执行：

```bash
bash check.sh
```

这个脚本会运行：

- `go test ./...`
- `go vet ./...`

补充说明：

- 当前部分 Linux 发行版自带的 Go 1.22.x 存在 `goindex` 误判标准库子包的问题，可能把 `net/http/httptest` 之类的标准库包错误报成 “not in std”。
- `check.sh` 会自动追加 `GODEBUG=goindex=0`，用来绕过这类环境问题，不影响仓库代码行为。

## 入口与启动流程

服务入口：

- [cmd/server/main.go](/home/Hao/Game/Game/backend/cmd/server/main.go)

启动时会依次完成：

- 加载配置
- 打开 SQLite
- 执行 migrations
- 创建 Gin 路由
- 启动 HTTP 服务
- 监听中断信号并优雅关闭

## 配置说明

可参考：

- [backend/.env.example](/home/Hao/Game/Game/backend/.env.example)

### 服务与路径

- `APP_ENV`
  运行环境，默认 `development`
- `HOST`
  监听地址，默认 `0.0.0.0`
- `PORT`
  服务端口，默认 `3000`
- `DB_PATH`
  SQLite 数据库路径
- `STATIC_DIR`
  磁盘前端静态目录，默认 `../frontend/dist`
- `ASSETS_DIR`
  游戏素材目录，默认 `data/gamelist`

### 文件访问边界

- `PRIMARY_ROM_ROOT`
  唯一的 ROM 根目录；目录浏览、文件登记和下载都限制在这个目录及其子目录内

### 管理员认证

- `ADMIN_DISPLAY_NAME`
  前端显示用的管理员名称；后端运行时读取并通过接口返回给前端
- `ADMIN_PASSWORD`
  管理员密码
- `SESSION_SECRET`
  cookie 会话签名密钥；发布脚本会自动生成一个随机值
- `AUTH_MAX_FAILS`
  最大连续失败次数
- `AUTH_COOLDOWN`
  达到上限后的冷却时间，例如 `10m`
- `AUTH_FAIL_WINDOW`
  失败统计窗口，例如 `30m`
- `AUTH_STATE_TTL`
  登录失败状态保留时间，例如 `24h`
- `AUTH_TRACK_BY`
  登录限流维度，支持 `ip` 与 `ip_ua`

注意：

- `ADMIN_PASSWORD` 不能为空。
- `SESSION_SECRET` 不能为空，也不能保留默认值 `change-me`。
- 任一条件不满足时，后端会直接拒绝启动。

### Wiki

- `WIKI_HISTORY_LIMIT`
  每个游戏最多保留多少条 Wiki 历史记录，`0` 表示不自动裁剪

### 代理

- `PROXY`
  默认出站代理

### VHD / VHDX 启动脚本

- `SMB_SHARE_ROOT`
  生成 BAT 时使用的 UNC 根路径，例如 `\\192.168.1.4\Game`
- `SMB_PATH_MAPPINGS`
  多挂载点场景下使用的“本地路径=UNC 根路径”映射，多个条目用 `;` 分隔，例如 `/mnt/Mount/Game/Game=\\\\192.168.1.4\\Game1;/mnt/Mount/Game/Gal=\\\\192.168.1.4\\Gal`
- `SMB_USERNAME`
  固定 SMB 用户名
- `SMB_PASSWORD`
  固定 SMB 密码
- `VHD_DIFF_ROOT`
  差分 VHDX 所在盘符根，例如 `C:` 或 `D:`

## 认证与访问控制

- 管理员身份通过 cookie `gameatlas_admin` 判断。
- `/api/auth/login` 会校验管理员密码，并在成功后写入 HttpOnly cookie。
- 登录失败会按 `AUTH_*` 配置记录次数，并在达到阈值后短时锁定。
- 大多数写操作要求管理员权限。
- 文件下载和启动脚本下载额外支持“管理员或本地子网访问”。
- 下载统计通过显式接口 `POST /api/games/:id/files/:fileId/downloads` 完成，并在进程内按 `gameId + fileId + sourceKey` 做 10 分钟时间窗去重。
- 对于私有游戏，未登录用户无法通过 `/assets/*` 直接访问其素材。

## API 概览

### 基础

- `GET /api/health`
  健康检查

### 认证

- `POST /api/auth/login`
- `POST /api/auth/logout`
- `GET /api/auth/me`

### 游戏

- `GET /api/games`
- `GET /api/games/timeline`
- `GET /api/games/stats`
- `GET /api/games/:id`
- `POST /api/games`
- `PUT /api/games/:id`
- `DELETE /api/games/:id`

### 游戏文件

- `GET /api/games/:id/files`
- `POST /api/games/:id/files`
- `PUT /api/games/:id/files/:fileId`
- `DELETE /api/games/:id/files/:fileId`
- `POST /api/games/:id/files/:fileId/downloads`
- `GET /api/games/:id/files/:fileId/download`
- `GET /api/games/:id/files/:fileId/launch-script`

### Wiki

- `GET /api/games/:id/wiki`
- `PUT /api/games/:id/wiki`
- `GET /api/games/:id/wiki/history`

### 元数据

- `GET /api/series`
- `GET /api/series/:id`
- `POST /api/series`
- `GET /api/platforms`
- `POST /api/platforms`
- `GET /api/developers`
- `POST /api/developers`
- `GET /api/publishers`
- `POST /api/publishers`

### 标签

- `GET /api/tag-groups`
- `POST /api/tag-groups`
- `GET /api/tags`
- `POST /api/tags`

### 待处理项忽略

- `GET /api/review-issue-overrides`
- `PUT /api/games/:id/review-issues/:issueKey/ignore`
- `DELETE /api/games/:id/review-issues/:issueKey/ignore`

### 素材

- `POST /api/assets/cover`
- `POST /api/assets/banner`
- `POST /api/assets/video`
- `POST /api/assets/screenshot`
- `PUT /api/assets/screenshot/order`
- `PUT /api/assets/video/order`
- `PUT /api/assets/video/primary`
- `DELETE /api/assets`

### 目录浏览

- `GET /api/directory/default`
- `GET /api/directory/list`

### Steam

- `GET /api/steam/search`
- `GET /api/steam/:appId/assets`
- `POST /api/steam/:appId/apply-assets`
- `GET /api/steam/proxy`

## 素材与静态资源

### `/assets/*`

- 面向游戏素材目录 `ASSETS_DIR`
- 实际访问前会校验目标游戏是否存在
- 私有游戏素材对匿名请求不可见

### `/data/*`

- 面向 `ASSETS_DIR` 的上级数据目录
- 只允许图片和字体等白名单后缀
- 适合暴露自定义背景图、字体等公共资源

### 前端托管

- 若 `STATIC_DIR/index.html` 存在，优先从磁盘托管
- 否则回退到嵌入式前端资源
- 生产打包时会将前端构建产物复制到 `backend/web/dist` 并嵌入可执行文件

## 数据库迁移

- 迁移文件位于 [migrations](/home/Hao/Game/Game/backend/migrations)
- 服务启动时会自动执行未应用的迁移
- 当前迁移已覆盖初始表结构、待处理项忽略、素材 UID、视频主资源、可见性、标签系统、登录失败记录等能力

## 说明

- 当前 `backend/README.md` 以实际 Go 后端主线为准，不再描述更早期“基础骨架”阶段。
- 如需整项目的运行与发布说明，请同时查看根 [README.md](/home/Hao/Game/Game/README.md)。
