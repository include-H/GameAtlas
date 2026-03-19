# Backend

Phase 2 Go backend foundation for the GameAtlas rebuild.

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
- `SMB_SHARE_ROOT`: SMB share root used when generating Windows launch BAT files, for example `\\192.168.1.4\Game`
  直接按正常 UNC 路径填写，不要写成代码里的转义形式 `\\\\192.168.1.4\\Game`
- `SMB_USERNAME` / `SMB_PASSWORD`: fixed SMB credentials written into generated BAT files
- `SMB_DRIVE_LETTER`: temporary Windows drive letter used by the BAT script, defaults to `Z:`
- `VHD_DIFF_ROOT`: local Windows directory where differencing VHDX files are created, defaults to `C:\Game\VHDCache`
- `PROXY`: one proxy value used by default for outbound requests
- `HTTP_PROXY` / `HTTPS_PROXY` / `STEAM_PROXY`: optional overrides if one module needs a different proxy
