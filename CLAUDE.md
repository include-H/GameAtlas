# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

GameAtlas — game management system for NAS/local network game libraries. Go + Gin + SQLite backend, Vue 3 + TypeScript + Vite + Pinia + Arco Design Vue frontend. Monorepo: `backend/` and `frontend/` are independent packages.

The primary use case is managing game metadata, assets, and wiki pages, with VHD/VHDX remote launch support for Windows clients over SMB.

## Quick Start

```bash
# One-command dev start (backend → health check → frontend)
bash start-dev.sh

# Or manually:
cd backend && go run ./cmd/server        # → :3000
cd frontend && npm run dev                # → :5173
```

Prerequisites: Go 1.22+, Node.js, npm, curl.

## Dev Commands

| Task | Command | Working Dir |
|------|---------|-------------|
| Backend start | `go run ./cmd/server` | `backend/` |
| Backend tests+vet | `bash check.sh` | `backend/` |
| Backend tests only | `go test ./...` | `backend/` |
| Run single backend test | `go test ./internal/services -run TestName` | `backend/` |
| Frontend dev server | `npm run dev` | `frontend/` |
| Frontend build | `npm run build` (vue-tsc --noEmit && vite build) | `frontend/` |
| Frontend tests | `npm run test:run` | `frontend/` |
| Frontend lint | `npm run lint` (ESLint + custom button policy check) | `frontend/` |
| Release build | `bash build-release.sh [version]` | root |

## Architecture

### Backend (`backend/`)

Entry point: `backend/cmd/server/main.go`

Layer structure (`backend/internal/`):
- `domain/` — domain objects, input/output structs, enums
- `repositories/` — data access, SQL queries, transactions
- `services/` — business logic, flow orchestration, cross-repo aggregation
- `http/handlers/` — protocol layer: param parsing, status codes, response formatting
- `http/routes/` — Gin route registration
- `config/` — startup defaults and DB-backed setting mapping
- `db/` — SQLite connection initialization
- `app/` — app bootstrap wiring

**Strict layer boundaries**: handlers → services → repositories → domain. No cross-layer calls. Handlers do protocol only — no SQL, no filesystem, no business logic.

Database: SQLite with numbered forward-only migrations in `backend/migrations/`. Auto-applied at startup, deduplicated by `schema_migrations.name`. **Never modify existing migration files.** New migrations: increment number, semantic name, forward-only SQL only.

Config: runtime settings live in SQLite `app_settings`. `DB_PATH` remains a bootstrap setting and defaults to `data/db.db`; it can be overridden via process environment. A legacy `.env` is imported once and automatically deleted after settings are persisted.

Frontend serving: Backend checks `STATIC_DIR` (default `../frontend/dist`) first, falls back to embedded `web/dist`.

### Frontend (`frontend/`)

Entry point: `frontend/src/main.ts`

Key structure (`frontend/src/`):
- `views/` — page components (PascalCase naming)
- `components/` — shared components
- `composables/` and `hooks/` — reusable composition functions (`useXxx` naming)
- `services/` — API service layer (split by resource)
- `stores/` — Pinia stores
- `router/` — Vue Router config
- `assets/` — styles and assets (global glass effects in `assets/style.css`)

Vite config (`vite.config.ts`):
- `envDir` points to `../backend/` for optional build-time values; runtime configuration comes from the backend API
- Dev proxy: `/api`, `/assets`, `/data` → `http://127.0.0.1:3000`
- Arco Design components auto-imported via `unplugin-vue-components`

## CI

- **CI** (`.github/workflows/ci.yml`): frontend `npm run test:run` + `npm run build`, backend `bash check.sh`. Runs on push/PR to `main`.
- **Release** (`.github/workflows/release.yml`): triggered by pushing `v*` tags. Builds release, uploads linux-amd64 tar.gz/zip + checksums to GitHub Release.

## Style Conventions (High-Priority)

### Frontend
- **NO bare `any`** — use `unknown` then narrow to precise type. ESLint enforces `@typescript-eslint/no-explicit-any: error`.
- Component naming: PascalCase (`.vue` files). Composables: `useXxx`.
- Text action buttons: use `app-text-action-btn` class. Business classes for layout only.
- Glass containers: use `app-glass-surface` / `app-glass-surface--interactive` from `src/assets/style.css`. No inline glass effect combinations.
- Custom lint policy: `npm run lint:policy` runs `scripts/check-text-action-buttons.mjs`.
- Components >800 lines default to split review.

### Backend
- Layer discipline: handlers → services → repositories → domain. No cross-layer calls.
- Minimal dependencies: prefer stdlib + Gin + SQLX + SQLite. No wrapper/adapter/no-op abstractions.
- Migrations: forward-only, numbered, never edit existing ones.
- Error handling: repo returns raw errors, service normalizes (e.g., `ErrNotFound`), handler maps to HTTP.

### Pre-commit Checks
- Frontend: `npm run lint` must pass
- Backend: `cd backend && bash check.sh` must pass
- `check.sh` sets `GODEBUG=goindex=0` to work around Go 1.22.x goindex bug on some distros

## Key Environment Variables

| Variable | Purpose | Required |
|----------|---------|----------|
| `ADMIN_PASSWORD` | Admin password for auth | No (default: `1234`, stored in DB settings) |
| `DB_PATH` | SQLite bootstrap DB path | No (default: `data/db.db`) |
| `STATIC_DIR` | Disk frontend dir (fallback to embedded) | No |
| `ASSETS_DIR` | Game asset directory | No (default: `data/gamelist`) |
| `PRIMARY_ROM_ROOT` | ROM root for file access boundaries | No (default: `ROM`) |
| `STEAMGRIDDB_API_KEY` | SteamGridDB API key for online cover/banner/logo search | No |

## Testing

- **Frontend**: Vitest + jsdom. Tests in `src/**/*.test.ts`. Run: `npm run test:run`
- **Backend**: Go std testing. Run: `go test ./...` (or `bash check.sh` for tests + vet)
- Migration changes require tests covering "new DB success + idempotent re-run"

## Important Gotchas

1. **`backend/web/dist/` is embedded into the Go binary** — the release build copies frontend dist here before building. Never commit built artifacts.
2. **Runtime settings are DB-backed** — update configurable values through the settings page or `app_settings`; keep `DB_PATH` as the only bootstrap override when needed.
3. **Legacy `.env` is one-time migration input** — after a successful DB import it is deleted; do not add new env templates or write settings back to env files.
4. **`docs/项目风格约定.md`** is the authoritative style guide. Read it before making significant changes.
5. **`release/` directory is gitignored** — build artifacts are not committed.
