# Backend

Go + Gin + SQLite 后端，提供游戏库 API、文件管理、Wiki、认证、Steam/SteamGridDB 集成、设置管理及前端静态资源托管。

## 启动与校验

```bash
cd backend
go run ./cmd/server         # 启动，监听 :3000

bash check.sh               # go test + go vet（自动设置 GODEBUG=goindex=0）
```

入口：`cmd/server/main.go`，启动流程：

1. `config.Load()` — 解析进程环境变量
2. `app.New(cfg)`:
   - `db.OpenSQLite`（PRAGMA: `foreign_keys=ON`, `journal_mode=WAL`, `busy_timeout=5000`, `synchronous=NORMAL`, `cache_size=-8000`；`SetMaxOpenConns(1)`）
   - `RunMigrations`（嵌入迁移，`schema_migrations` 去重）
   - `settingsRepo.EnsureDefaults` → `ApplyRuntimeSettings`（DB 值覆盖 env 默认值）→ `cfg.Validate()`
   - **启动备份**：`VACUUM INTO` 创建 `startup` 备份 + 保留期清理
   - 资产维护：清理待重试任务（最多 100）、清理 `_staging`、重建缺失素材引用；**后台 goroutine** 做孤儿素材隔离
   - `routes.New` → `http.Server{Addr: host:port, ReadHeaderTimeout}` → `StartPeriodic`（24h 周期备份）
3. `signal.NotifyContext(SIGINT/SIGTERM)` → 优雅关闭（先 cancel 备份 goroutine，再 `server.Shutdown`）

## 目录结构

```text
backend/
├── cmd/server/main.go       # 入口
├── internal/
│   ├── app/app.go           # 启动组装
│   ├── config/              # config.go + settings.go（DB 配置映射）+ config_test.go
│   ├── db/                  # sqlite.go（连接+PRAGMA）+ migrate.go（迁移）
│   ├── domain/              # 领域对象与输入输出结构
│   ├── files/               # assets.go / fuzzy.go / path_guard.go（ROM 根目录边界守卫）
│   ├── http/
│   │   ├── handlers/        # 协议层：参数解析、状态码、响应格式
│   │   └── routes/router.go # 全部路由 + 静态资源/素材/数据目录服务
│   ├── repositories/        # 数据访问、SQL、事务
│   └── services/            # 业务逻辑、跨 repo 聚合 + services/data/hitokoto_game_sentences.json
├── migrations/              # 000001..000004 + embed.go
├── web/                     # embed.go（go:embed all:dist）+ dist/.gitkeep
├── data/                    # 本地运行时数据（db.db、bg.jpg、gamelist/）
├── check.sh
├── go.mod / go.sum
└── README.md
```

层间调用方向：`handlers → services → repositories → domain`，不跨层。

## 配置

运行配置存储在 SQLite 的 `app_settings` 表中，设置页读写该表。以下仅列关键项：

### 引导项（不存入 DB）

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `DB_PATH` | SQLite 启动引导路径 | `data/db.db` |

### 运行时配置（存入 DB，设置页可改）

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `APP_ENV` | 运行环境 | `production` |
| `HOST` | 监听地址 | `0.0.0.0` |
| `PORT` | 监听端口 | `3000` |
| `STATIC_DIR` | 磁盘前端目录，不存在时回退到内嵌 `web/dist` | `../frontend/dist` |
| `ASSETS_DIR` | 素材目录 | `data/gamelist` |
| `PRIMARY_ROM_ROOT` | ROM 根目录，文件操作限制在此目录内 | `/mnt` |
| `DB_BACKUP_ENABLED` | 启用数据库备份 | `true` |
| `DB_BACKUP_DIR` | SQLite 在线备份目录 | `data/backups` |
| `DB_BACKUP_INTERVAL` | 备份间隔 | `24h` |
| `DB_BACKUP_RETENTION_COUNT` | 备份保留份数，`0` 表示不自动清理 | `5` |
| `ADMIN_PASSWORD` | 管理员密码 | `1234` |
| `ADMIN_DISPLAY_NAME` | 管理员显示名称 | `Admin` |
| `AUTH_MAX_FAILS` | 登录失败最大次数 | `5` |
| `AUTH_COOLDOWN` | 锁定冷却时间 | `10m` |
| `AUTH_FAIL_WINDOW` | 失败计数窗口 | `30m` |
| `AUTH_STATE_TTL` | 锁定状态保留时间 | `24h` |
| `AUTH_TRACK_BY` | 指纹来源：`ip` / `ip_ua` | `ip` |
| `WIKI_HISTORY_LIMIT` | Wiki 历史保留条数 | `100` |
| `SMB_PATH_MAPPINGS` | SMB 路径映射（分号分隔） | — |
| `SMB_USERNAME` | SMB 用户名 | — |
| `SMB_PASSWORD` | SMB 密码 | — |
| `VHD_DIFF_ROOT` | 客户端差分盘盘符 | `C:` |
| `PROXY` | HTTP/HTTPS/SOCKS5 代理，空=直连 | — |
| `STEAMGRIDDB_API_KEY` | SteamGridDB API Key | — |
| `READ_HEADER_TIMEOUT` | HTTP 读头超时 | `5s` |
| `SHUTDOWN_TIMEOUT` | 优雅关闭超时 | `10s` |

旧版部署中的 `.env` 不再被自动读取、导入或删除；运行时配置请通过设置页录入。

## 认证

- 通过 cookie `gameatlas_admin` 识别管理员身份
- `POST /api/auth/login` 校验密码后写入 HttpOnly cookie；HTTPS 时附加 `Secure`
- 登录失败按 HMAC 源指纹（ip 或 ip_ua）限流锁定
- 会话 TTL 30 天，验证缓存 30 秒
- 写操作要求管理员权限；文件下载沿用游戏可见性边界（公开可匿名，私有仅管理员）
- 下载统计 `POST /api/games/:publicId/files/:fileId/downloads` 按 `gameId+fileId+sourceKey` 做 10 分钟进程内去重

## API

<details>
<summary>展开完整列表</summary>

**基础**
- `GET /api/health`

**认证**
- `POST /api/auth/login` · `POST /api/auth/logout` · `GET /api/auth/me`

**一言**
- `GET /api/hitokoto`（参数：`c`、`min_length`、`max_length`、`encode=text`）

**待处理项**
- `GET /api/pending-issues`

**游戏**
- `GET /api/games` · `GET /api/games/timeline` · `GET /api/games/stats`
- `GET /api/games/:publicId` · `POST /api/games` · `DELETE /api/games/:publicId`
- `PUT /api/games/:publicId/favorite` · `DELETE /api/games/:publicId/favorite`
- `PUT /api/games/:publicId/aggregate`（聚合更新，返回 `asset_delete_paths` warnings）
- `POST /api/games/refresh-sizes`（管理员，刷新文件大小）

**文件**
- `GET /api/games/:publicId/files`
- `POST /api/games/:publicId/files/:fileId/downloads`
- `GET /api/games/:publicId/files/:fileId/download`
- `GET /api/games/:publicId/files/:fileId/launch-script`

**Wiki**
- `GET /api/games/:publicId/wiki` · `PUT /api/games/:publicId/wiki` · `GET /api/games/:publicId/wiki/history`

**元数据**
- `GET/POST /api/series` · `GET /api/series/:id`
- `GET/POST /api/developers` · `GET/POST /api/publishers`

**待处理项忽略**
- `PUT/DELETE /api/games/:publicId/review-issues/:issueKey/ignore`

**素材**
- `POST /api/assets/cover` · `POST /api/assets/banner`
- `POST /api/assets/video` · `POST /api/assets/screenshot`
- `POST /api/assets/logo`

**目录浏览**
- `GET /api/directory/default` · `GET /api/directory/list` · `GET /api/directory/search`

**设置**
- `GET /api/settings/config` · `PUT /api/settings/config`（需管理员）
- `POST /api/settings/bg` · `DELETE /api/settings/bg`（需管理员）
- `POST /api/settings/restart`（需管理员，syscall.Exec 重启进程）

**Steam**
- `GET /api/steam/search` · `GET /api/steam/:appId/assets` · `GET /api/steam/proxy`

**SteamGridDB**（需 API Key）
- `GET /api/steamgriddb/available` · `GET /api/steamgriddb/search`
- `GET /api/steamgriddb/:appId/grids` · `GET /api/steamgriddb/:appId/heroes`
- `GET /api/steamgriddb/:appId/logos` · `GET /api/steamgriddb/:appId/icons`
- `GET /api/steamgriddb/game/:gameId/grids` · `GET /api/steamgriddb/game/:gameId/heroes`
- `GET /api/steamgriddb/game/:gameId/logos`

</details>

## 静态资源

- `/assets/*` — 面向 `ASSETS_DIR` 的游戏素材，私有游戏素材对匿名不可见
- `/data/*` — 面向 `ASSETS_DIR` 上级目录，仅允许图片/字体等白名单后缀
- 前端托管：磁盘 `STATIC_DIR` 优先，不存在则回退到内嵌 `web/dist`

## 迁移

迁移文件位于 `migrations/`，编号正向，启动时自动执行。**不要修改已有迁移文件**，新迁移递增编号即可。

现有迁移：
- `000001_baseline.sql` — 基础表结构（games、game_files、game_assets、wiki_history、series、developers、publishers、关联表、review overrides、auth 表、favorite_games、app_settings）
- `000002_logo_visible.sql` — games.logo_visible 字段
- `000003_normalize_public_id.sql` — public_id 小写化
- `000004_app_settings.sql` — 幂等创建 app_settings 表

迁移机制：`go:embed *.sql` → 按文件名字典序执行，每文件在事务内执行并写入 `schema_migrations`（UNIQUE name），已记录则跳过。

## 测试

```bash
GODEBUG=goindex=0 go test -cover ./...
```

测试覆盖率（2026-08 实测）：

| 包 | 覆盖率 |
|----|--------|
| internal/config | 62.5% |
| internal/db | 69.4% |
| internal/domain | 60.4% |
| internal/files | 24.1% |
| internal/http/handlers | 59.8% |
| internal/http/routes | 14.2% |
| internal/repositories | 45.9% |
| internal/services | 62.5% |

`cmd/server`、`internal/app`、`migrations`、`web` 无测试。

## 构建

```bash
# 在仓库根目录执行
bash build-release.sh              # 输出到 release/game-release-<version>/
bash build-release.sh v1.0.0       # 自定义版本名
```

构建流程：
1. `npm run build` 前端
2. 复制 dist 到 `backend/web/dist/`（先清空+`.gitkeep`）
3. `go build -trimpath -ldflags="-s -w"` 输出 `game-server`
4. 建 `data/gamelist`、`ROM`、可选复制 `data/bg.jpg`
5. 复制根 README 与 backend README
6. 生成 `start.sh`
7. 再次清理内嵌目录

版本默认时间戳或 `v*` 参数。
