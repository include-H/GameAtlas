# Backend

Phase 2 Go backend foundation for the NAS Game Library rebuild.

## Current scope

- config loading from environment variables
- automatic `.env` loading from `backend/.env`
- SQLite connection bootstrap
- SQL migration runner
- Gin router with `/api/health`
- optional static frontend serving

## Quick start

1. Install Go 1.23 or newer.
2. Copy `backend/.env.example` to `backend/.env` and fill in your local paths.
3. Run `cd backend && go run ./cmd/server`.

The server serves API routes at `/api/*`. If `frontend/dist` exists, it is also served by the backend.

## Recommended `.env` values

- `DB_PATH`: SQLite database file path
- `ALLOWED_LIBRARY_ROOTS`: comma-separated NAS roots allowed for browsing and file selection
- `PRIMARY_ROM_ROOT`: preferred ROM root for future directory picker defaults
- `PROXY`: one proxy value used by default for outbound requests
- `HTTP_PROXY` / `HTTPS_PROXY` / `STEAM_PROXY`: optional overrides if one module needs a different proxy
