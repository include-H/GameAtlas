# Backend

Go + Gin + SQLite 后端，提供游戏库 API、文件管理、Wiki、认证、Steam 集成及前端静态资源托管。

## 启动与校验

```bash
cd backend
cp .env.example .env
go run ./cmd/server         # 启动，监听 :3000

bash check.sh               # go test + go vet（自动设置 GODEBUG=goindex=0）
```

入口：`cmd/server/main.go`，启动流程：加载配置 → 打开 SQLite → 执行迁移 → 创建路由 → 启动 HTTP → 监听中断信号优雅关闭。

## 目录结构

```text
backend/
├── cmd/server/         # 入口
├── internal/
│   ├── domain/         # 领域对象、输入输出结构、枚举
│   ├── repositories/   # 数据访问、SQL、事务
│   ├── services/       # 业务逻辑、跨 repo 聚合
│   ├── http/handlers/  # 协议层：参数解析、状态码、响应格式
│   ├── http/routes/    # Gin 路由注册
│   ├── config/         # .env 加载
│   ├── db/             # SQLite 连接初始化
│   └── app/            # 启动组装
├── migrations/         # 编号正向迁移，启动时自动执行
├── web/dist/           # 内嵌前端资源（release 构建时复制）
└── check.sh            # 测试 + vet
```

层间调用方向：`handlers → services → repositories → domain`，不跨层。

## 配置

模板见 `backend/.env.example`，以下仅列关键项：

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `ADMIN_PASSWORD` | **必填**，为空则拒绝启动 | — |
| `DB_PATH` | SQLite 路径 | `data/db.db` |
| `ASSETS_DIR` | 素材目录 | `data/gamelist` |
| `PRIMARY_ROM_ROOT` | ROM 根目录，文件操作限制在此目录内 | `ROM` |
| `STATIC_DIR` | 磁盘前端目录，不存在时回退到内嵌 `web/dist` | `../frontend/dist` |
| `SMB_SHARE_ROOT` | 单共享 UNC 路径 | — |
| `SMB_PATH_MAPPINGS` | 多挂载点映射（分号分隔，见 `.env.example` 示例） | — |
| `VHD_DIFF_ROOT` | 客户端差分盘盘符 | `C:` |

路径配置支持绝对和相对路径，相对路径以 `.env` 所在目录为基准。

## 认证

- 通过 cookie `gameatlas_admin` 识别管理员身份
- `POST /api/auth/login` 校验密码后写入 HttpOnly cookie；HTTPS 时附加 `Secure`
- 登录失败按 `AUTH_MAX_FAILS` / `AUTH_COOLDOWN` / `AUTH_FAIL_WINDOW` 限流锁定
- 写操作要求管理员权限；文件下载沿用游戏可见性边界（公开可匿名，私有仅管理员）
- 下载统计 `POST /api/games/:id/files/:fileId/downloads` 按 `gameId+fileId+sourceKey` 做 10 分钟进程内去重（单进程近似，非跨实例限流）

## API

<details>
<summary>展开完整列表</summary>

**基础**
- `GET /api/health`

**认证**
- `POST /api/auth/login` · `POST /api/auth/logout` · `GET /api/auth/me`

**游戏**
- `GET /api/games` · `GET /api/games/timeline` · `GET /api/games/stats`
- `GET /api/games/:id` · `POST /api/games` · `DELETE /api/games/:id`

**文件**
- `GET /api/games/:id/files`
- `POST /api/games/:id/files/:fileId/downloads`
- `GET /api/games/:id/files/:fileId/download`
- `GET /api/games/:id/files/:fileId/launch-script`

**Wiki**
- `GET /api/games/:id/wiki` · `PUT /api/games/:id/wiki` · `GET /api/games/:id/wiki/history`

**元数据**
- `GET/POST /api/series` · `GET /api/series/:id`
- `GET/POST /api/developers` · `GET/POST /api/publishers`

**标签**
- `GET/POST /api/tag-groups` · `GET/POST /api/tags`

**待处理项**
- `GET /api/review-issue-overrides`
- `PUT/DELETE /api/games/:id/review-issues/:issueKey/ignore`

**素材**
- `POST /api/assets/cover` · `POST /api/assets/banner`
- `POST /api/assets/video` · `POST /api/assets/screenshot`
- `PUT /api/assets/screenshot/order` · `PUT /api/assets/video/order` · `PUT /api/assets/video/primary`
- `DELETE /api/assets`

**目录浏览**
- `GET /api/directory/default` · `GET /api/directory/list`

**Steam**
- `GET /api/steam/search` · `GET /api/steam/:appId/assets` · `GET /api/steam/proxy`

</details>

## 静态资源

- `/assets/*` — 面向 `ASSETS_DIR` 的游戏素材，私有游戏素材对匿名不可见
- `/data/*` — 面向 `ASSETS_DIR` 上级目录，仅允许图片/字体等白名单后缀
- 前端托管：磁盘 `STATIC_DIR` 优先，不存在则回退到内嵌 `web/dist`

## 迁移

迁移文件位于 `migrations/`，编号正向，启动时自动执行。**不要修改已有迁移文件**，新迁移递增编号即可。
