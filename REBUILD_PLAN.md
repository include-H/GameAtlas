# NAS Game Library Manager Rebuild Plan

## 1. Project Goal

Build a web-based management system for a NAS-hosted game library.

Core goals:
- Manage game entries in a clean admin interface
- Attach one or more downloadable files to each game
- Provide a game detail page with wiki/introduction content
- Download game files through HTTP instead of SMB browsing
- Manage cover, banner, and screenshot assets
- Use Steam only as an asset-assist source for images and metadata lookup
- Support stable production deployment on NAS through Docker or a single backend binary

Explicit non-goals:
- No multi-user system
- No JWT/auth/roles
- No favorites
- No community/social features
- No batch import center
- No automatic library ingestion as a primary workflow

The intended workflow is:
1. Create a game entry with a title
2. Edit the game later
3. Add one or more real file paths from the NAS
4. Maintain wiki text and media assets
5. Download files directly from the web UI

## 2. Product Definition

This system is not a storefront or a community wiki.

It is a:
- NAS game library web manager
- downloadable file index
- wiki-enhanced game archive browser

The key domain model is:
- `Game`: the entry shown in the UI
- `GameFile`: an actual downloadable file for that game

Examples:
- Game: `Counter-Strike`
- Files:
  - `CS完整版.vhd`
  - `Esai Cs1.6 Ver3248 纯净版.vhd`
  - `Esai Cs1.6 Ver3248 各地图包.vhd`

So the old concept of "version" should be replaced with "file entry" or `GameFile`.

## 3. Recommended Technical Direction

### Backend

Recommended: Go

Reason:
- Better fit for NAS deployment
- Easier single-binary delivery
- Better long-running operational simplicity than the current Node backend
- Very suitable for file streaming, HTTP range requests, and filesystem-heavy workloads
- Easy Docker packaging

Suggested backend stack:
- Language: Go
- Router: `gin`
- Database: SQLite
- SQL layer: `sqlx` with handwritten SQL
- Markdown rendering: `goldmark`
- Config: environment variables + simple config struct

Avoid:
- Over-complicated ORM-heavy design
- Rebuilding a user/auth system
- Premature microservice decomposition

### Frontend

Keep:
- Vue 3
- TypeScript
- Pinia

But redesign the structure to match the new domain model and drop compatibility layers.

### Deployment

Support both:
- Docker deployment
- Standalone backend binary deployment

Preferred production direction:
- Backend in Go
- Frontend built into static files
- Either:
  - served by the Go app directly
  - or served separately behind nginx/Caddy

## 4. System Modules

### Backend Modules

1. `games`
- CRUD for game entries

2. `game_files`
- Manage downloadable files for each game

3. `wiki`
- Store and render markdown content
- Track edit history

4. `assets`
- Upload/delete cover, banner, screenshots

5. `metadata`
- Series
- Platforms
- Developers
- Publishers

6. `directory`
- Browse allowed NAS directories safely
- Assist with selecting file paths

7. `steam`
- Search Steam
- Fetch image candidates
- Apply selected assets to a game

8. `download`
- Stream files
- Support HTTP range requests

### Frontend Modules

1. `dashboard`
2. `games-list`
3. `game-detail`
4. `game-edit`
5. `metadata-settings`
6. `shared-ui`

## 5. Core Data Model

### 5.1 games

Fields:
- `id`
- `title`
- `title_alt`
- `summary`
- `release_date`
- `engine`
- `cover_image`
- `banner_image`
- `wiki_content`
- `wiki_content_html`
- `needs_review`
- `created_at`
- `updated_at`

Notes:
- `summary` is a short intro, not a dump field for file metadata
- `wiki_content` is markdown
- `wiki_content_html` is cached rendered HTML

### 5.2 game_files

Fields:
- `id`
- `game_id`
- `file_path`
- `label`
- `notes`
- `size_bytes`
- `sort_order`
- `created_at`
- `updated_at`

Notes:
- `file_path` points to the real file on the NAS
- `label` is display text like `纯净版`, `完整版`, `地图包`
- `notes` is optional descriptive text
- do not overload `summary` with file notes

### 5.3 series

Fields:
- `id`
- `name`
- `slug`
- `description`
- `parent_series_id`
- `sort_order`
- `created_at`

### 5.4 platforms

Fields:
- `id`
- `name`
- `slug`
- `sort_order`
- `created_at`

### 5.5 developers

Fields:
- `id`
- `name`
- `slug`
- `sort_order`
- `created_at`

### 5.6 publishers

Fields:
- `id`
- `name`
- `slug`
- `sort_order`
- `created_at`

### 5.7 relation tables

- `game_series`
- `game_platforms`
- `game_developers`
- `game_publishers`

Each should contain:
- foreign keys
- `sort_order` where useful

### 5.8 wiki_history

Fields:
- `id`
- `game_id`
- `content`
- `change_summary`
- `created_at`

No user linkage is required in the rebuilt system.

## 6. Database Design Principles

- SQLite remains acceptable
- Keep schema simple and explicit
- Use foreign keys
- Do not keep abandoned user/auth tables in the new system
- Avoid compatibility columns unless they are still used by current UI
- Store image paths as relative app paths where possible
- Cache file size in `game_files`, but allow refresh if needed

## 7. API Design

### 7.1 Games

- `GET /api/games`
- `GET /api/games/:id`
- `POST /api/games`
- `PUT /api/games/:id`
- `DELETE /api/games/:id`

Query params for list:
- `page`
- `limit`
- `search`
- `series`
- `platform`
- `needs_review`
- `sort`
- `order`

Allowed sort fields:
- `title`
- `created_at`
- `updated_at`
- `views`
- `downloads`

### 7.2 Game Files

- `GET /api/games/:id/files`
- `POST /api/games/:id/files`
- `PUT /api/games/:id/files/:fileId`
- `DELETE /api/games/:id/files/:fileId`

This is the replacement for the old "versions" API.

### 7.3 Downloads

- `GET /api/games/:id/files/:fileId/download`

Requirements:
- Stream directly from NAS file path
- Support range requests
- Return correct filename
- Validate file exists
- Return clean errors

### 7.4 Wiki

- `GET /api/games/:id/wiki`
- `PUT /api/games/:id/wiki`
- `GET /api/games/:id/wiki/history`

### 7.5 Assets

- `POST /api/assets/cover`
- `POST /api/assets/banner`
- `POST /api/assets/screenshot`
- `DELETE /api/assets`

### 7.6 Metadata

- `GET /api/series`
- `POST /api/series`
- `GET /api/developers`
- `POST /api/developers`
- `GET /api/publishers`
- `POST /api/publishers`
- `GET /api/platforms`
- `POST /api/platforms`

### 7.7 Directory Assist

- `GET /api/directory/default`
- `GET /api/directory/list`

Must be sandboxed to allowed roots only.

### 7.8 Steam Assist

- `GET /api/steam/search`
- `GET /api/steam/:appId/assets`
- `POST /api/steam/:appId/apply-assets`

Steam is an assist tool only, not a source of truth for game creation.

## 8. Backend Architecture

Suggested structure:

```text
backend/
├── cmd/server/
├── internal/
│   ├── app/
│   ├── config/
│   ├── db/
│   ├── domain/
│   ├── repositories/
│   ├── services/
│   ├── http/
│   │   ├── handlers/
│   │   ├── middleware/
│   │   └── routes/
│   ├── steam/
│   ├── markdown/
│   └── files/
├── web/
└── migrations/
```

Layer rules:
- handlers: request/response only
- services: business logic
- repositories: database queries only
- files helpers: filesystem and download helpers only

## 9. Frontend Architecture

Suggested structure:

```text
frontend/src/
├── pages/
│   ├── DashboardPage.vue
│   ├── GamesListPage.vue
│   ├── GameDetailPage.vue
│   ├── GameEditPage.vue
│   └── MetadataPage.vue
├── components/
│   ├── games/
│   ├── files/
│   ├── wiki/
│   ├── assets/
│   └── shared/
├── stores/
│   ├── gamesList.ts
│   ├── gameDetail.ts
│   ├── metadata.ts
│   └── ui.ts
├── services/
│   ├── api.ts
│   ├── games.ts
│   ├── files.ts
│   ├── wiki.ts
│   ├── metadata.ts
│   ├── assets.ts
│   └── steam.ts
└── router/
```

Rules:
- list state lives in list store
- detail state lives in detail store
- edit form state should mostly be local to the page
- no fake auth/permission layer
- no compatibility service wrappers that invent fields

## 10. UI Pages

### 10.1 Dashboard

Purpose:
- quick overview
- recent additions
- pending review games
- recent updates

### 10.2 Games List

Features:
- search
- filters
- sort
- pagination
- link to edit/details

### 10.3 Game Detail

Features:
- title and summary
- cover/banner/screenshots
- wiki content
- list of downloadable files
- direct download buttons

### 10.4 Game Edit

Features:
- edit base info
- edit metadata relations
- manage file list
- pick NAS paths through directory browser
- manage images
- edit wiki or jump to wiki editor

### 10.5 Metadata Settings

Features:
- create and manage series/platforms/developers/publishers

## 11. Files Instead of Versions

This must be enforced consistently.

Old concept:
- `GameVersion`
- version-specific compatibility logic

New concept:
- `GameFile`

Why:
- the actual domain is downloadable NAS files
- not all files are semantic software releases
- file labels are user-facing and flexible

Examples:
- `完整版`
- `纯净版`
- `地图包`
- `中文免安装版`

## 12. Security Requirements

### 12.1 Path safety

Must:
- validate selected paths against configured roots
- resolve symlinks safely
- reject traversal attempts
- only allow file download from registered game file entries

### 12.2 Remote image fetching

Must:
- allow only `http` and `https`
- block localhost and private network targets
- validate image content type if downloading remote assets

### 12.3 Upload safety

Must:
- validate file types
- validate target game ID
- store under controlled directories

## 13. Deployment Design

### 13.1 Docker mode

Recommended for NAS production.

Need:
- app container
- mounted SQLite file
- mounted uploads/assets directory
- mounted NAS games root read-only or read-write depending on need

Suggested mounts:
- `/app/data`
- `/app/assets`
- `/mnt/games`

### 13.2 Single binary mode

Recommended if backend is rewritten in Go.

Need:
- one backend binary
- one config file or env file
- one SQLite DB file
- one assets directory
- one configured NAS games root

### 13.3 Reverse proxy

Optional but recommended:
- Caddy
- nginx

Useful for:
- HTTPS
- compression
- large download proxy behavior

## 14. Config Requirements

Need configuration items:
- `PORT`
- `HOST`
- `DB_PATH`
- `ASSETS_PATH`
- `GAMES_ROOT`
- `ALLOWED_BROWSE_ROOTS`
- `STEAM_HTTP_PROXY` optional
- `PUBLIC_BASE_URL`

Do not require:
- JWT secret
- auth config
- role config

## 15. Features to Remove From Current System

These should not exist in the rebuilt system:
- login page
- register page
- auth store
- auth routes
- JWT types
- permission directives
- permission hooks
- favorites endpoints
- favorites UI
- `user_favorites`
- all fake role-based guards
- "mock task" download compatibility layer
- batch import flow that writes version metadata into summary

## 16. Migration Targets From Current System

Keep conceptually:
- games
- wiki
- metadata dictionaries
- screenshots/cover/banner handling
- directory browsing
- Steam image assist
- download streaming

Replace:
- versions -> files
- auth/favorites -> remove
- batch import -> remove
- compatibility fields -> remove

## 17. Implementation Plan

### Phase 1: Define V2 Contract

Goal:
- finalize domain language and API contract before coding

Tasks:
1. confirm final naming:
   - `Game`
   - `GameFile`
2. confirm retained fields on `games`
3. confirm `game_files` schema
4. confirm routes and DTOs
5. freeze what is removed from the old system

Deliverables:
- finalized schema doc
- finalized route list
- migration checklist

### Phase 2: Backend Foundation

Goal:
- create a clean Go backend skeleton

Tasks:
1. initialize Go module
2. create config loader
3. create SQLite connection layer
4. create migration setup
5. create basic router and health endpoint
6. add static frontend serving strategy

Deliverables:
- bootable backend service
- empty DB migration pipeline

### Phase 3: Games and Files

Goal:
- implement the main domain first

Tasks:
1. implement `games` repository
2. implement `game_files` repository
3. implement games CRUD service
4. implement file CRUD service
5. implement file download endpoint with range support
6. implement list/detail DTOs

Deliverables:
- working list/detail/edit/delete/download flow

### Phase 4: Wiki

Goal:
- support the core introduction content

Tasks:
1. implement markdown rendering
2. implement wiki save/update
3. implement wiki history
4. expose wiki endpoints

Deliverables:
- editable wiki with history

### Phase 5: Metadata

Goal:
- support structured categorization

Tasks:
1. implement series/platforms/developers/publishers tables and repositories
2. implement relation updates for games
3. implement metadata endpoints

Deliverables:
- game edit form can manage metadata relations

### Phase 6: Assets and Steam Assist

Goal:
- support image maintenance

Tasks:
1. implement local upload endpoints
2. implement asset delete endpoint
3. implement Steam search endpoint
4. implement Steam asset preview/apply flow

Deliverables:
- covers, banners, screenshots manageable from UI

### Phase 7: Directory Assist

Goal:
- support selecting NAS file paths safely

Tasks:
1. implement safe browse-root config
2. implement list/default directory endpoints
3. implement file-only selection logic in UI

Deliverables:
- editor can pick NAS files without typing long paths manually

### Phase 8: Frontend Rebuild

Goal:
- build a clean UI on top of the new contract

Tasks:
1. set up new service modules
2. rebuild list page
3. rebuild detail page
4. rebuild edit page
5. rebuild metadata page
6. remove auth/favorites UI

Deliverables:
- complete usable V2 frontend

### Phase 9: Deployment

Goal:
- make it production-ready for NAS use

Tasks:
1. write Dockerfile
2. write docker-compose example
3. define volume mounts
4. define backup points for DB/assets
5. optionally build static Go binary release

Deliverables:
- Docker deployment
- optional single-binary deployment doc

### Phase 10: Data Migration and Cutover

Goal:
- move from old system to new system safely

Tasks:
1. export current useful data
2. map old games into new schema
3. map old file paths into `game_files`
4. verify wiki content migration
5. verify asset path migration
6. smoke test download flow

Deliverables:
- migrated usable dataset
- cutover checklist

## 18. Development Rules For New Chat

In the next development chat:
- do not continue patching the old backend architecture
- build against this design unless requirements change
- prefer Go backend implementation
- remove features instead of preserving fake compatibility
- treat file download as a first-class feature
- treat NAS path safety as high priority
- keep the system single-admin and auth-free unless explicitly reintroduced

## 19. Open Questions To Revisit Before Full Build

These should be re-confirmed before coding starts:
- Should one game allow multiple files with identical labels?
- Should the system compute file sizes on save, on demand, or both?
- Should screenshots be stored as ordered list only, or also have captions?
- Should the wiki editor stay inline, or become a separate dedicated edit page?
- Should the backend serve frontend static files directly in production?
- Is read-only access to the NAS enough, or does the app need write/delete operations on game files?

## 20. Final Recommendation

Proceed with:
- Go backend rewrite
- Vue frontend cleanup/rebuild
- no auth system
- no favorites
- no import center
- files as first-class downloadable entities
- Docker-first deployment, with optional single-binary release

