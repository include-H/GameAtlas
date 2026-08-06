# GameAtlas - Agent Instructions

## Project Overview

Game management system for NAS/local network game libraries. Go + Gin + SQLite backend, Vue 3 + TypeScript + Vite + Pinia + Arco Design Vue frontend. Monorepo: `backend/` and `frontend/` are independent packages sharing a root.

## Quick Start

```bash
# One-command dev start (starts backend, waits for health, then frontend)
bash start-dev.sh

# Or manually:
cd backend && go run ./cmd/server        # → :3000
cd frontend && npm run dev                # → :5173
```

Prerequisites: Go 1.25+, Node.js, npm, curl.

## Toolchain Environment

Node.js and npm are managed by nvm. Non-interactive agent shells may not load nvm automatically, so initialize it before any frontend command:

```bash
export NVM_DIR="${NVM_DIR:-$HOME/.nvm}"
[ -s "$NVM_DIR/nvm.sh" ] && . "$NVM_DIR/nvm.sh"
```

## Dev Commands

| Task | Command | Working Dir |
|------|---------|-------------|
| Backend start | `go run ./cmd/server` | `backend/` |
| Backend tests+vet | `bash check.sh` | `backend/` |
| Backend tests only | `go test ./...` | `backend/` |
| Frontend dev server | `npm run dev` | `frontend/` |
| Frontend build | `npm run build` (runs `vue-tsc --noEmit && vite build`) | `frontend/` |
| Frontend tests | `npm run test:run` | `frontend/` |
| Frontend lint | `npm run lint` (ESLint + custom button policy check) | `frontend/` |

## CI (GitHub Actions)

- **CI** (`.github/workflows/ci.yml`): frontend `npm run test:run` + `npm run build`, backend `bash check.sh`. Runs on push/PR to `main`.
- **Release** (`.github/workflows/release.yml`): Triggered by pushing `v*` tags. Runs tests, builds release, uploads `linux-amd64` tar.gz/zip + checksums to GitHub Release.

## Release Build

```bash
# Build release package (auto timestamp version)
bash build-release.sh

# Or with explicit version
bash build-release.sh v1.0.0
```

Outputs to `release/game-release-<version>/`. The script:
1. Builds frontend (`npm run build`)
2. Copies frontend dist to `backend/web/dist/` for embedding
3. Builds Go binary with `-trimpath -ldflags="-s -w"`
4. Cleans up embedded web dir

**Do NOT commit `backend/web/dist/` contents** — they're gitignored. CI/release workflow creates a `.gitkeep` placeholder.

## Backend Architecture

**Entry point**: `backend/cmd/server/main.go`

**Layer structure** (`backend/internal/`):
- `domain/` — domain objects, input/output structs, enums
- `repositories/` — data access, SQL queries, transactions
- `services/` — business logic, flow orchestration, cross-repo aggregation
- `http/handlers/` — protocol layer: param parsing, status codes, response formatting
- `http/routes/` — Gin route registration
- `config/` — startup defaults and DB-backed setting mapping
- `db/` — SQLite connection initialization
- `app/` — app bootstrap wiring

**Strict layer boundaries** (from `docs/项目风格约定.md`):
- Handlers: protocol only. NO SQL, NO filesystem, NO business logic.
- Services: business rules, validation, orchestration.
- Repositories: storage-only (queries, updates, transactions).
- Domain: stable objects, input/output structs, enums.

**Database**: SQLite. Migrations in `backend/migrations/` (numbered `.sql` files, embedded via `//go:embed *.sql`). Auto-applied at startup, deduplicated by `schema_migrations.name`. New migrations: increment number, semantic name, forward-only SQL. **Never modify existing migration files.**

**Config**: Runtime settings live in SQLite `app_settings`. `DB_PATH` remains a bootstrap setting and defaults to `data/db.db`; it can be overridden via process environment. Process environment variables can provide initial runtime values; `.env` files are not read or written by the application.

**Frontend serving**: Backend checks `STATIC_DIR` (default `../frontend/dist`) first, falls back to embedded `web/dist`.

## Frontend Architecture

**Entry point**: `frontend/src/main.ts`

**Key structure** (`frontend/src/`):
- `views/` — page components (PascalCase naming)
- `components/` — shared components
- `composables/` and `hooks/` — reusable composition functions (`useXxx` naming)
- `services/` — API service layer (split by resource, NOT monolithic)
- `stores/` — Pinia stores
- `router/` — Vue Router config
- `assets/` — styles and assets (global glass effects in `assets/style.css`)

**Vite config** (`vite.config.ts`):
- `envDir` points to `../backend/` for optional build-time values; runtime configuration comes from the backend API
- Dev proxy: `/api`, `/assets`, `/data` → `http://127.0.0.1:3000`
- Arco Design components auto-imported via `unplugin-vue-components`
- Build output: `dist/ui/` for assets

## Style Conventions (High-Priority Rules)

### Frontend
- **NO bare `any`** — use `unknown` → precise type. ESLint enforces `@typescript-eslint/no-explicit-any: error`.
- **Component naming**: PascalCase (`.vue` files). Composables: `useXxx`.
- **Text action buttons**: use `app-text-action-btn` class. Business classes for layout only.
- **Glass containers**: use `app-glass-surface` / `app-glass-surface--interactive` from `src/assets/style.css`. NO inline glass effect combinations.
- **Custom lint policy**: `npm run lint:policy` runs `scripts/check-text-action-buttons.mjs` to enforce button conventions.
- Components >800 lines default to split review.

### Backend
- **Layer discipline**: handlers → services → repositories → domain. No cross-layer calls.
- **Minimal dependencies**: prefer stdlib + Gin + SQLX + SQLite. No wrapper/adapter/no-op abstractions.
- **Migrations**: forward-only, numbered, never edit existing ones. Run `bash check.sh` to verify.
- **Error handling**: repo returns raw errors, service normalizes (e.g., `ErrNotFound`), handler maps to HTTP.

### Git Commit Messages
- **Conventional Commits**: `type(scope): 中文描述`，如 `fix(ui): ...`、`feat(media): ...`；type 使用 fix/feat/docs/style/perf/refactor/chore/ci/release。
- **标题一句概括**：讲清"改了什么、解决了什么"，不堆砌、不省略关键信息。
- **正文四段式**（重要修复/复杂变更必须写，简单变更可省略）：
  - `背景`：问题从哪来、涉及的历史提交
  - `根因`：真正的成因，含代码/机制层面的细节
  - `修复`：具体改动、影响范围、为什么这样修
  - `验证`：跑过的测试、构建与手工验证结果
- **参考范例**：`30baa74`（fix(ui): 修复全站弹窗内下拉框被自身遮罩遮挡不可见）。
- **发布**：主线上修复/功能按语义化版本打 annotated tag（如 `v1.1.1`），推送 GitHub 触发 Release 工作流，并同步 Gitea 镜像。

### Pre-commit Checks
- Frontend: `npm run lint` must pass
- Backend: `cd backend && bash check.sh` must pass
- `check.sh` sets `GODEBUG=goindex=0` to work around the Go 1.22.x goindex bug on some distros (retained harmlessly on newer Go)

## Key Environment Variables

| Variable | Purpose | Required |
|----------|---------|----------|
| `ADMIN_PASSWORD` | Admin password for auth | No (default: `1234`, stored in DB settings) |
| `DB_PATH` | SQLite bootstrap DB path | No (default: `data/db.db`) |
| `STATIC_DIR` | Disk frontend dir (fallback to embedded) | No |
| `ASSETS_DIR` | Game asset directory | No (default: `data/gamelist`) |
| `PRIMARY_ROM_ROOT` | ROM root for file access boundaries | No (default: `ROM`) |
| `VHD_DIFF_ROOT` | Client diff disk root (e.g., `C:`) | No |

## Testing

- **Frontend**: Vitest + jsdom. Tests in `src/**/*.test.ts`. Run: `npm run test:run`
- **Backend**: Go std testing. Run: `go test ./...` (or `bash check.sh` for tests + vet)
- Migration changes require tests covering "new DB success + idempotent re-run"

## Important Gotchas

1. **`backend/web/dist/` is embedded into the Go binary** — the release build copies frontend dist here before building. CI creates a `.gitkeep` placeholder. Never commit built artifacts.
2. **Runtime settings are DB-backed** — update configurable values through the settings page or `app_settings`; keep `DB_PATH` as the only bootstrap override when needed.
3. **`backend/check.sh`Appends `GODEBUG=goindex=0`** — required on some Linux distros with Go 1.22.x to avoid false "not in std" errors; retained harmlessly on Go 1.25+.
4. **Custom frontend lint step** — `npm run lint` includes a custom button policy check (`lint:policy`) beyond standard ESLint.
5. **`docs/项目风格约定.md`** — the authoritative style guide. Read it before making significant changes.
6. **`release/` directory is gitignored** — build artifacts are not committed.
7. **Vue Router `route.query` reference trap** — `watch(() => route.query)` fires on EVERY navigation because Vue Router rebuilds the query object on each route change, even when contents are identical. Never use the watcher alone to trigger a reload; compare CONTENT (e.g. a `JSON.stringify` snapshot) before refetching. Reference: the `lastLoadedQueryKey` guard in `frontend/src/composables/useGamesView.ts`.
8. **Never refetch page 1 on keep-alive restore** — when a kept-alive list page reactivates (`onActivated`), a full reload with `append:false` replaces the deep-loaded store and destroys the scroll position (canvas height shrinks → `scrollTo` clamps to 0 → white screen / flicker / fallback-to-top, all symptoms of the same root cause). Skip the reload when the route query content is unchanged. Full saga: `docs/2026.8.6_游戏库虚拟滚动恢复陷阱.md`.
9. **Shared scroll container + virtual list restoration** — `.content` keeps `scrollTop` across routes. For SPA same-document navigations, disable the browser's native scroll restoration (`history.scrollRestoration = 'manual'` in `frontend/src/main.ts`), restore scroll after a double `requestAnimationFrame` (layout fully settled), and always re-align the virtual list to the container's ACTUAL `scrollTop` afterwards — worst case degrades to top, never a blank viewport.
