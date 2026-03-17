# V2 Migration Checklist

This checklist captures what must change from the legacy Node/Vue codebase before implementation continues.

## Domain Renames

- [ ] rename all "version" language to `GameFile`
- [ ] rename old storage/table names such as `game_file_paths` to `game_files`
- [ ] remove DTO fields that imply one default file on `games`

## Table-Level Migration Targets

- [ ] create new `games` table with only the V2 fields
- [ ] create new `game_files` table
- [ ] create new `game_assets` table for screenshots
- [ ] create `wiki_history`
- [ ] create `series`, `platforms`, `developers`, `publishers`
- [ ] create `game_series`, `game_platforms`, `game_developers`, `game_publishers`
- [ ] enforce foreign keys and duplicate-prevention indexes

## Remove From Schema

- [ ] drop `users`
- [ ] drop auth-related tables and tokens
- [ ] drop `user_favorites`
- [ ] drop any batch import specific tables or columns
- [ ] drop compatibility-era fields that are not used by V2
- [ ] drop user linkage columns like `created_by`, `wiki_updated_by`
- [ ] drop `wiki_updated_at` if it only exists for old UI bookkeeping

## Reshape Existing Game Data

- [ ] migrate `file_path` into one `game_files` row when a legacy game has a single file
- [ ] migrate `file_paths` arrays into multiple `game_files` rows
- [ ] move screenshot arrays into `game_assets`
- [ ] preserve `views` and `downloads` counters if present
- [ ] preserve `wiki_content` and regenerate `wiki_content_html`
- [ ] preserve metadata relations for series, platforms, developers, and publishers

## API Contract Cleanup

- [ ] remove auth guards and role assumptions from handlers
- [ ] remove favorites endpoints and payloads
- [ ] remove batch import endpoints
- [ ] replace legacy download endpoints with `GET /api/games/:id/files/:fileId/download`
- [ ] replace legacy versions endpoint with `/api/games/:id/files`
- [ ] keep `GET /api/health` as the minimal system endpoint

## Frontend Cleanup Targets

- [ ] remove login, register, and permission flows
- [ ] remove auth store and related route guards
- [ ] remove compatibility service wrappers that invent version semantics
- [ ] refactor detail/edit pages to consume `files` instead of `versions`
- [ ] refactor uploads UI so screenshots are separate assets, not a field on `games`

## Security Carryover

- [ ] validate `game_files.file_path` against allowed roots
- [ ] resolve symlinks before directory listing or download
- [ ] reject traversal attempts
- [ ] permit downloads only for files registered in `game_files`
- [ ] validate uploaded or remote asset content type in later phases

## Ready For Phase 2 When

- [ ] Go module path is chosen
- [ ] backend folder layout follows the rebuild plan
- [ ] first SQL migration is derived from `docs/phase1/v2-schema.md`
- [ ] HTTP handlers use the route list from `docs/phase1/api-contract.md`
