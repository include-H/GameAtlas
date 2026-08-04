# Backend（GameAtlas 后端）

GameAtlas 的 Go 后端：提供游戏库 API、素材与文件管理、Wiki、认证、Steam / SteamGridDB 集成、设置管理与前端静态资源托管。

## 技术栈

- Go 1.22+ / Gin / SQLX
- SQLite（`mattn/go-sqlite3`，WAL 模式，单连接写入）
- 内嵌迁移（`go:embed *.sql`）与内嵌前端（`go:embed web/dist`）

## 快速开始

前置依赖：Go 1.22+、Node.js、npm、curl。

```bash
# 一键启动（根目录）：先构建前端、启动后端并等待健康检查，再启动 Vite
bash start-dev.sh
```

或分别启动：

```bash
cd backend
go run ./cmd/server          # 监听 :3000，自动执行迁移
```

```bash
cd frontend
npm run dev                  # :5173，/api、/assets、/data 代理到 :3000
```

后端健康检查：`GET http://127.0.0.1:3000/api/health`。

## 开发工作流

### 后端校验

```bash
cd backend
bash check.sh                # go test ./... + go vet ./...（自动设置 GODEBUG=goindex=0）
```

`check.sh` 中的 `GODEBUG=goindex=0` 用于规避部分发行版 Go 1.22.x 的 goindex 误报，不要手动移除。

### 前端校验（后端改动涉及联调时）

```bash
cd frontend
npm run test:run             # Vitest + jsdom
npm run lint                 # ESLint + 自定义文本按钮策略检查
npm run build                # vue-tsc --noEmit + vite build
```

### 合并前检查

- 后端：`bash check.sh` 通过；
- 前端：`npm run lint` 通过；
- 新增业务逻辑至少补 service 或 repository 级测试；影响接口行为的改动补 handler 回归测试；
- 新增迁移必须补迁移测试，覆盖"新库执行成功 + 重复执行幂等"；
- 不引入裸 `any`、不把单文件继续撑大、不新增无收益抽象。

### CI

- `ci.yml`：push / PR 到 `main` 时分别运行前端 `test:run` + `build` 与后端 `bash check.sh`；
- `release.yml`：推送 `v*` tag 时运行全部检查、构建发布包，并上传 GitHub Release + Docker Hub。

## 架构与分层

调用方向严格单向：

```text
handlers（协议层）→ services（业务层）→ repositories（存储层）
                             ↑
                       domain（领域对象）
```

职责边界（详细约定见 [docs/项目风格约定.md](../docs/项目风格约定.md)）：

- `handler`：参数解析、鉴权门禁、HTTP 状态码、响应封装；不写 SQL、不碰文件系统、不拼业务规则；
- `service`：业务校验、归一化、流程编排、跨 repository 聚合、文件与数据库协调；
- `repository`：只做存储相关的查询、更新、事务与行映射；
- `domain`：稳定的领域对象、输入输出结构、枚举与规则常量。

禁止跨层调用：例如 handler 直接 import repositories 包、service 向下透传表名 / 列名、repository 定义业务策略。

## 目录结构

```text
backend/
├── cmd/server/main.go       # 入口：加载配置 → 组装 App → 优雅关闭
├── internal/
│   ├── app/app.go           # 启动组装：DB、迁移、设置、备份、资产对账、路由
│   ├── config/              # 进程环境变量解析 + DB 设置映射（config.go / settings.go）
│   ├── db/                  # SQLite 连接与 PRAGMA、迁移执行器
│   ├── domain/              # 领域对象、输入输出结构、枚举、待处理项规则
│   ├── files/               # 素材存储、模糊搜索、ROM 根目录路径守卫
│   ├── http/
│   │   ├── handlers/        # 协议层
│   │   └── routes/router.go # 全部路由 + 静态资源 / 素材 / 数据目录服务
│   ├── repositories/        # SQL、事务、行映射
│   └── services/            # 业务逻辑；data/hitokoto_game_sentences.json
├── migrations/              # 000001_baseline.sql + embed.go（v1.1.0 起单基线）
├── web/                     # embed.go + dist/.gitkeep（发布构建时放入前端产物）
├── data/                    # 本地运行时数据（db.db、bg.jpg、gamelist/）
├── check.sh
└── go.mod / go.sum
```

## 启动流程

入口 `cmd/server/main.go`：

1. `config.Load()`：解析进程环境变量，得到启动配置；
2. `app.New(cfg)`：
   - `db.OpenSQLite`：`foreign_keys=ON`、`journal_mode=WAL`、`busy_timeout=5000`、`synchronous=NORMAL`、`cache_size=-8000`，`SetMaxOpenConns(1)`；
   - `RunMigrations`：按文件名顺序执行内嵌迁移，`schema_migrations.name` 判重；
   - `settingsRepo.EnsureDefaults` → `ApplyRuntimeSettings`（DB 值覆盖 env 默认值）→ `NormalizeStoredRuntimePaths` → `cfg.Validate()`；
   - **启动备份**：`VACUUM main INTO` 生成 `startup` 备份 + 按保留份数清理；
   - 资产维护：重试未完成的删除任务（最多 100）、清理过期 `_staging`、重建缺失素材引用；后台 goroutine 做孤儿素材隔离；
   - `routes.New` → `http.Server{Addr, ReadHeaderTimeout}` → 启动定时备份 goroutine；
3. `signal.NotifyContext(SIGINT/SIGTERM)`：先取消备份 goroutine，再 `server.Shutdown` 优雅退出。

## 配置体系

区分两种来源：

- **启动引导项**：只有 `DB_PATH`，不写入 `app_settings`，仅通过进程环境变量覆盖；
- **运行时配置**：持久化在 SQLite `app_settings` 表，进程环境变量只在首次初始化时作为默认值来源；应用不读取、不写入、不删除 `.env` 文件。

### 引导项

| 变量 | 说明 | 默认值 |
| --- | --- | --- |
| `DB_PATH` | SQLite 数据库路径（相对于可执行文件所在目录） | `data/db.db` |

### 运行时配置（设置页可改）

| 变量 | 说明 | 默认值 |
| --- | --- | --- |
| `APP_ENV` | 运行环境，`production` 时 Gin 使用 ReleaseMode | `production` |
| `HOST` / `PORT` | 监听地址与端口 | `0.0.0.0` / `3000` |
| `STATIC_DIR` | 磁盘前端目录，不存在时回退内嵌 `web/dist` | `../frontend/dist` |
| `ASSETS_DIR` | 游戏素材目录 | `data/gamelist` |
| `PRIMARY_ROM_ROOT` | ROM 根目录，文件访问边界 | `/mnt` |
| `DB_BACKUP_ENABLED` | 启用数据库备份 | `true` |
| `DB_BACKUP_DIR` | 备份目录 | `data/backups` |
| `DB_BACKUP_INTERVAL` | 备份间隔 | `24h` |
| `DB_BACKUP_RETENTION_COUNT` | 保留份数，`0` 不自动清理 | `5` |
| `ADMIN_PASSWORD` / `ADMIN_DISPLAY_NAME` | 管理员密码与显示名 | `1234` / `Admin` |
| `AUTH_MAX_FAILS` / `AUTH_COOLDOWN` | 登录失败次数限制与冷却 | `5` / `10m` |
| `AUTH_FAIL_WINDOW` / `AUTH_STATE_TTL` | 失败计数窗口与锁定保留时间 | `30m` / `24h` |
| `AUTH_TRACK_BY` | 失败追踪方式：`ip` / `ip_ua` | `ip` |
| `WIKI_HISTORY_LIMIT` | Wiki 历史保留条数 | `100` |
| `SMB_PATH_MAPPINGS` | SMB 路径映射，分号分隔 `本地路径=UNC 路径` | — |
| `SMB_USERNAME` / `SMB_PASSWORD` | SMB 账号密码 | — |
| `VHD_DIFF_ROOT` | 客户端差分盘根路径（盘符） | `C:` |
| `PROXY` | 出站代理（http / https / socks5），空为直连 | — |
| `STEAMGRIDDB_API_KEY` | SteamGridDB API Key | — |
| `READ_HEADER_TIMEOUT` / `SHUTDOWN_TIMEOUT` | HTTP 读头超时 / 优雅关闭超时 | `5s` / `10s` |

配置改动经 `PUT /api/settings/config` 校验后写入 DB，返回"重启后生效"；`POST /api/settings/restart` 通过 `syscall.Exec` 重启进程。

## 数据库与迁移

- 迁移文件：`migrations/000001_*.sql` 起，按文件名（编号）顺序执行；
- 幂等：每个文件在一个事务内执行，并写入 `schema_migrations`（`name UNIQUE`），已记录则跳过；
- 规则：**只前向、不修改已有迁移**；新迁移使用递增编号与语义化命名；
- 启动失败策略：任何迁移失败都会中止启动，不允许"按表结构猜测"的隐式兼容分支；
- 测试要求：新增迁移必须覆盖新库执行成功 + 重复执行幂等。

现有迁移：

- `000001_baseline.sql` — 单基线（games、game_files、game_assets、wiki_history、series、developers、publishers、关联表、review overrides、auth 表、favorite_games、app_settings、`games.logo_visible`、`public_id` 小写化）

> v1.1.0 起迁移改为单基线：`000001_baseline.sql` 直接表达当前最终 schema（不再回放历史增量语句，废弃字段直接不创建）；原 `000002`（logo_visible）、`000004`（app_settings）已并入基线，`000003` 属历史数据修复、不进入基线。升级窗口：全新数据库或已完整执行旧迁移的数据库；回滚：恢复数据库备份并运行旧版本二进制。下一个增量迁移从 `000002_xxx.sql` 开始编号。

## 核心实现速览

### 认证与会话

- Cookie `gameatlas_admin`，HttpOnly，HTTPS 时附加 `Secure`；会话 TTL 30 天，进程内验证缓存 30 秒；
- 登录失败按 HMAC 指纹（`ip` 或 `ip_ua`）计数并限流锁定；
- 写端点统一 `requireAdmin`；文件下载遵循游戏可见性（公开匿名可下，私有仅管理员）；
- 下载统计按 `gameId+fileId+sourceKey` 做 10 分钟进程内去重，只防重复点击误计，不承诺跨实例语义。

### 游戏与聚合更新

- `POST /api/games` 只创建基础行；完整编辑走 `PUT /api/games/:publicId/aggregate` 全量替换语义（缺省关系视为清空，不支持稀疏 patch）；
- 聚合更新同时处理文件、素材顺序、Logo 位置、系列 / 开发商 / 发行商，删除失败时返回 `asset_delete_paths` warnings 并登记重试任务；
- 列表支持排序白名单（标题 / 创建 / 更新 / 发售日 / 下载量 / 随机 / 待处理数）与方向白名单。

### 待处理项

- 四类问题：缺图片、缺 Wiki、缺文件、基础信息不完整，细分到封面 / 横幅 / 截图 / Logo / 简介等明细；
- 严重度规则：缺文件即 severe，缺 3 项以上或图片 + Wiki 同时缺也视为 severe；
- 忽略状态存 `game_review_issue_overrides`，可填写原因。

### 素材生命周期

- 上传先落 `_staging`，编辑提交后移动到正式路径并登记到 `game_assets`；
- 启动时：重试失败删除任务（≤100）、清理超过 1 小时的 staging、重建缺失素材引用（封面 / 横幅 / `game_assets` 行）；
- 孤儿素材：后台扫描未登记文件，移动到 `data/orphaned-assets/` 隔离，保留 7 天，无恢复流程（仅作排查缓冲）；
- 素材删除失败会登记 `asset_cleanup_tasks`，下次启动重试。

### 文件下载与路径边界

- 所有游戏文件访问经过 `files.Guard`：以 `PRIMARY_ROM_ROOT` 为根，`EvalSymlinks + filepath.Rel` 校验，杜绝路径穿越；
- 下载端点返回文件流（带大小 / 修改时间），私有游戏对匿名 404；
- 只有 `.vhd` / `.vhdx` 提供启动脚本，其余扩展名走普通下载。

### Windows 启动脚本（VHD 远程启动）

`services/windows_launch.go` 生成 BAT：

1. 校验 SMB 配置、游戏可见性、文件路径与扩展名；
2. 根据 `SMB_PATH_MAPPINGS` 把本地路径映射为 UNC 路径（最长前缀优先，`EvalSymlinks` 后比对）；
3. 渲染脚本：提权 → `cmdkey` + `net use` 连接共享 → 差分盘不存在则 `create vdisk ... parent=...`，已存在则直接 `attach` → `diskpart` 挂载；
4. 差分盘路径为 `<VHD_DIFF_ROOT>\<基础盘文件名>`，盘符由 `VHD_DIFF_ROOT` 规范化（非盘符输入回退 `C:`）；
5. 脚本同时提供删除 SMB 凭据 / 断开共享的选项。

脚本携带 SMB 凭据，是面向家庭可信环境的显式设计约束，不是多用户安全默认值。

### Steam / SteamGridDB

- Steam：`/api/steam/search`、`/api/steam/:appId/assets`（元数据 + 素材候选），图片经 `/api/steam/proxy` 代理下载，避免浏览器直连外部；
- SteamGridDB：按 appId / gameId 获取 grids / heroes / logos / icons，需要 API Key；
- Steam 与 SteamGridDB 出站请求统一走 `PROXY` 配置（空则直连）；
- 图片代理有 host 白名单（steam / steamstatic / steamgriddb），素材直下载路径有内网 IP 拦截与 DNS 重绑定防护。

### 数据库备份

- `DatabaseBackupService` 直接持有 `*sqlx.DB`，执行 SQLite 原生 `VACUUM main INTO`（有意保留的基础设施边界）；
- 启动即备份一次（`startup`），随后按 `DB_BACKUP_INTERVAL` 定时备份（`scheduled`），按保留份数清理；
- WAL 模式下不直接复制 `db.db` 文件。

## API 一览

<details>
<summary>展开完整列表</summary>

**基础 / 认证**
- `GET /api/health`
- `POST /api/auth/login` · `POST /api/auth/logout` · `GET /api/auth/me`

**一言**
- `GET /api/hitokoto`（参数：`c`、`min_length`、`max_length`、`encode=text`）

**待处理项**
- `GET /api/pending-issues`

**游戏**
- `GET /api/games` · `GET /api/games/timeline` · `GET /api/games/stats` · `GET /api/games/preview-videos`
- `GET /api/games/:publicId` · `POST /api/games` · `DELETE /api/games/:publicId`
- `PUT /api/games/:publicId/aggregate` · `PUT/DELETE /api/games/:publicId/favorite`
- `POST /api/games/refresh-sizes`（管理员）

**文件与启动**
- `GET /api/games/:publicId/files`
- `POST /api/games/:publicId/files/:fileId/downloads`
- `GET /api/games/:publicId/files/:fileId/download`
- `GET /api/games/:publicId/files/:fileId/launch-script`

**Wiki**
- `GET /api/games/:publicId/wiki` · `PUT /api/games/:publicId/wiki` · `GET /api/games/:publicId/wiki/history`

**元数据**
- `GET/POST /api/series` · `GET /api/series/:id`
- `GET/POST /api/developers`
- `GET/POST /api/publishers` · `GET /api/publishers/:id`

**待处理项忽略**
- `PUT/DELETE /api/games/:publicId/review-issues/:issueKey/ignore`

**素材**
- `POST /api/assets/cover` · `banner` · `video` · `screenshot` · `logo`

**目录浏览**
- `GET /api/directory/default` · `GET /api/directory/list` · `GET /api/directory/search`

**设置**
- `GET/PUT /api/settings/config` · `POST/DELETE /api/settings/bg` · `POST /api/settings/restart`（均需管理员）

**开始屏幕磁贴**
- `GET /api/start-screen/tiles` · `PUT /api/start-screen/tiles`（保存需管理员）

**Steam / SteamGridDB**
- `GET /api/steam/search` · `GET /api/steam/:appId/assets` · `GET /api/steam/proxy`
- `GET /api/steamgriddb/available` · `GET /api/steamgriddb/search`
- `GET /api/steamgriddb/:appId/grids` · `heroes` · `logos` · `icons`
- `GET /api/steamgriddb/game/:gameId/grids` · `heroes` · `logos`

</details>

## 测试

```bash
cd backend
bash check.sh                    # go test ./... + go vet ./...
go test -cover ./...             # 查看覆盖率
```

测试组织约定：

- Go 标准测试，测试文件与源码同包；
- 重点覆盖：删除与清理、鉴权与访问控制、文件边界与下载、聚合更新与事务一致性、迁移幂等；
- 测试优先断言行为边界（状态码、错误语义、数据结果），不绑定 SQL 实现细节；
- `cmd/server`、`internal/app`、`migrations`、`web` 目前无测试，改动这些包时优先补启动 / 迁移级测试。

## 构建与发布

```bash
# 仓库根目录
bash build-release.sh              # 默认版本：时间戳
bash build-release.sh v1.0.0       # 自定义版本
```

流程：

1. `npm run build` 构建前端；
2. 复制 `frontend/dist` 到 `backend/web/dist/`（先清空并保留 `.gitkeep`）；
3. `go build -trimpath -ldflags="-s -w" -o release/game-release-<version>/game-server ./cmd/server`；
4. 创建 `data/gamelist`、`ROM`，可选复制 `data/bg.jpg`，复制 README 并生成 `start.sh`；
5. 清理内嵌 web 目录。

`backend/web/dist/` 内容被 gitignore，**不要提交构建产物**；CI 只保留 `.gitkeep` 占位。

Docker 镜像（`Dockerfile`）在构建阶段把前端产物复制到 `web/dist` 后编译，运行镜像基于 Alpine，挂载 `/app/data` 持久化。

## 文档导航

- 功能与使用说明：[README.md](../README.md)
- 前后端开发规范：[docs/项目风格约定.md](../docs/项目风格约定.md)
- 审计基线与收口原则：[docs/项目审查.md](../docs/项目审查.md)
