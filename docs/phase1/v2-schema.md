# V2 Schema Contract

This document freezes the Phase 1 backend domain model for the rebuild.

## Naming

Retained domain names:
- `Game`
- `GameFile`

Removed naming:
- `GameVersion`
- `game_file_paths`
- any "favorite", "auth", or "batch import" domain language

## Table: `games`

Purpose:
- stores the main game entry shown in lists, detail pages, and editing flows

Columns:

| Column | Type | Null | Notes |
| --- | --- | --- | --- |
| `id` | integer | no | primary key |
| `title` | text | no | main display title |
| `title_alt` | text | yes | alternate or Chinese title |
| `summary` | text | yes | short intro text |
| `release_date` | text | yes | ISO date string preferred |
| `engine` | text | yes | free text |
| `cover_image` | text | yes | relative asset path |
| `banner_image` | text | yes | relative asset path |
| `wiki_content` | text | yes | markdown source |
| `wiki_content_html` | text | yes | cached rendered HTML |
| `needs_review` | integer | no | boolean flag, default `0` |
| `views` | integer | no | default `0`, used for sorting/stats |
| `downloads` | integer | no | default `0`, used for sorting/stats |
| `created_at` | text | no | timestamp |
| `updated_at` | text | no | timestamp |

Decisions:
- `summary` only stores a short overview, never file/version notes.
- `wiki_content` remains on `games` for the main editable body.
- `wiki_content_html` is cached output and may be regenerated.
- `views` and `downloads` stay because they are in the agreed sort contract.
- No user linkage fields are kept on this table.
- Screenshots are not stored inline on `games`; they belong in `game_assets`.

## Table: `game_files`

Purpose:
- stores actual NAS-backed downloadable files for one game

Columns:

| Column | Type | Null | Notes |
| --- | --- | --- | --- |
| `id` | integer | no | primary key |
| `game_id` | integer | no | foreign key to `games.id` |
| `file_path` | text | no | absolute NAS file path after validation |
| `label` | text | yes | display text such as `完整版` |
| `notes` | text | yes | optional descriptive text |
| `size_bytes` | integer | yes | cached file size |
| `sort_order` | integer | no | default `0` |
| `created_at` | text | no | timestamp |
| `updated_at` | text | no | timestamp |

Decisions:
- This table fully replaces the old "versions" concept.
- Each row must map to one real file.
- `size_bytes` may be refreshed by a service later if the file changes.

## Table: `game_assets`

Purpose:
- stores screenshot references separately from cover and banner

Columns:

| Column | Type | Null | Notes |
| --- | --- | --- | --- |
| `id` | integer | no | primary key |
| `game_id` | integer | no | foreign key to `games.id` |
| `asset_type` | text | no | `screenshot` initially |
| `path` | text | no | relative asset path |
| `sort_order` | integer | no | default `0` |
| `created_at` | text | no | timestamp |

Decisions:
- `cover_image` and `banner_image` stay directly on `games`.
- Screenshot lists move out of the old JSON-style array field into rows.
- The table is intentionally generic enough for future asset expansion.

## Table: `wiki_history`

Purpose:
- stores wiki edit snapshots for history viewing and rollback support later

Columns:

| Column | Type | Null | Notes |
| --- | --- | --- | --- |
| `id` | integer | no | primary key |
| `game_id` | integer | no | foreign key to `games.id` |
| `content` | text | no | markdown snapshot |
| `change_summary` | text | yes | optional edit note |
| `created_at` | text | no | timestamp |

Decisions:
- no `created_by` or user linkage is retained in V2

## Metadata Tables

### `series`

| Column | Type | Null | Notes |
| --- | --- | --- | --- |
| `id` | integer | no | primary key |
| `name` | text | no | unique display name |
| `slug` | text | no | unique slug |
| `description` | text | yes | optional |
| `parent_series_id` | integer | yes | self reference |
| `sort_order` | integer | no | default `0` |
| `created_at` | text | no | timestamp |

### `platforms`

| Column | Type | Null | Notes |
| --- | --- | --- | --- |
| `id` | integer | no | primary key |
| `name` | text | no | unique display name |
| `slug` | text | no | unique slug |
| `sort_order` | integer | no | default `0` |
| `created_at` | text | no | timestamp |

### `developers`

| Column | Type | Null | Notes |
| --- | --- | --- | --- |
| `id` | integer | no | primary key |
| `name` | text | no | unique display name |
| `slug` | text | no | unique slug |
| `sort_order` | integer | no | default `0` |
| `created_at` | text | no | timestamp |

### `publishers`

| Column | Type | Null | Notes |
| --- | --- | --- | --- |
| `id` | integer | no | primary key |
| `name` | text | no | unique display name |
| `slug` | text | no | unique slug |
| `sort_order` | integer | no | default `0` |
| `created_at` | text | no | timestamp |

## Relation Tables

Required join tables:
- `game_series`
- `game_platforms`
- `game_developers`
- `game_publishers`

Shared shape:

| Column | Type | Null | Notes |
| --- | --- | --- | --- |
| `game_id` | integer | no | foreign key to `games.id` |
| relation foreign key | integer | no | e.g. `series_id` |
| `sort_order` | integer | no | default `0` |

Decisions:
- composite uniqueness should prevent duplicate relations
- foreign keys should be enforced

## Removed From V2

These concepts are frozen as removed and should not reappear in the rebuild:
- `users`
- `roles`
- `jwt`
- `user_favorites`
- `created_by`
- `wiki_updated_by`
- `wiki_updated_at`
- `favorite` query behavior
- batch import tables or DTOs
- version-specific compatibility fields
- websocket/event stream requirements

