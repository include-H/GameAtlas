# V2 API Contract

This document freezes the Phase 1 HTTP contract and DTO language for the rebuild.

## Conventions

Base path:
- `/api`

Response envelope:

```json
{
  "success": true,
  "data": {}
}
```

Paginated response:

```json
{
  "success": true,
  "data": [],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 1,
    "totalPages": 1
  }
}
```

Error response:

```json
{
  "success": false,
  "error": "message"
}
```

## DTOs

### `GameListItem`

```json
{
  "id": 1,
  "title": "Counter-Strike",
  "title_alt": "反恐精英",
  "summary": "Classic team shooter",
  "release_date": "2000-11-09",
  "engine": "GoldSrc",
  "cover_image": "/assets/games/1/cover.jpg",
  "banner_image": "/assets/games/1/banner.jpg",
  "needs_review": false,
  "views": 0,
  "downloads": 0,
  "created_at": "2026-03-16T00:00:00Z",
  "updated_at": "2026-03-16T00:00:00Z"
}
```

### `GameDetail`

```json
{
  "id": 1,
  "title": "Counter-Strike",
  "title_alt": "反恐精英",
  "summary": "Classic team shooter",
  "release_date": "2000-11-09",
  "engine": "GoldSrc",
  "cover_image": "/assets/games/1/cover.jpg",
  "banner_image": "/assets/games/1/banner.jpg",
  "wiki_content": "# Title",
  "wiki_content_html": "<h1>Title</h1>",
  "needs_review": false,
  "views": 0,
  "downloads": 0,
  "screenshots": [
    {
      "id": 10,
      "path": "/assets/games/1/screenshots/1.jpg",
      "sort_order": 0
    }
  ],
  "series": [],
  "platforms": [],
  "developers": [],
  "publishers": [],
  "files": [],
  "created_at": "2026-03-16T00:00:00Z",
  "updated_at": "2026-03-16T00:00:00Z"
}
```

### `GameWriteInput`

```json
{
  "title": "Counter-Strike",
  "title_alt": "反恐精英",
  "summary": "Classic team shooter",
  "release_date": "2000-11-09",
  "engine": "GoldSrc",
  "cover_image": "/assets/games/1/cover.jpg",
  "banner_image": "/assets/games/1/banner.jpg",
  "needs_review": false,
  "series_ids": [1],
  "platform_ids": [1],
  "developer_ids": [1],
  "publisher_ids": [1]
}
```

Rules:
- no `platform` string shortcut
- no `file_path` or `file_paths` on game create/update payloads
- files are managed through `game_files` endpoints only

### `GameFile`

```json
{
  "id": 1,
  "game_id": 1,
  "file_path": "/nas/games/Counter-Strike/CS完整版.vhd",
  "label": "完整版",
  "notes": "原始收藏版本",
  "size_bytes": 123456789,
  "sort_order": 0,
  "created_at": "2026-03-16T00:00:00Z",
  "updated_at": "2026-03-16T00:00:00Z"
}
```

### `GameFileWriteInput`

```json
{
  "file_path": "/nas/games/Counter-Strike/CS完整版.vhd",
  "label": "完整版",
  "notes": "原始收藏版本",
  "sort_order": 0
}
```

### `WikiPayload`

```json
{
  "content": "# Counter-Strike",
  "change_summary": "完善简介"
}
```

### `AssetUploadResponse`

```json
{
  "path": "/assets/games/1/cover.jpg"
}
```

### `MetadataItem`

```json
{
  "id": 1,
  "name": "Valve",
  "slug": "valve",
  "sort_order": 0,
  "created_at": "2026-03-16T00:00:00Z"
}
```

### `DirectoryListResponse`

```json
{
  "current_path": "/nas/games",
  "parent_path": null,
  "items": [
    {
      "name": "Counter-Strike",
      "path": "/nas/games/Counter-Strike",
      "is_directory": true,
      "size_bytes": null
    }
  ]
}
```

## Routes

### System

- `GET /api/health`

### Games

- `GET /api/games`
- `GET /api/games/:id`
- `POST /api/games`
- `PUT /api/games/:id`
- `DELETE /api/games/:id`

List query params:
- `page`
- `limit`
- `search`
- `series`
- `platform`
- `needs_review`
- `sort`
- `order`

Allowed `sort` values:
- `title`
- `created_at`
- `updated_at`
- `views`
- `downloads`

Allowed `order` values:
- `asc`
- `desc`

### Game Files

- `GET /api/games/:id/files`
- `POST /api/games/:id/files`
- `PUT /api/games/:id/files/:fileId`
- `DELETE /api/games/:id/files/:fileId`

### Downloads

- `GET /api/games/:id/files/:fileId/download`

Behavior:
- stream directly from the registered `file_path`
- support HTTP range requests
- set `Content-Disposition` with the real filename
- return `404` if the registered file does not exist

### Wiki

- `GET /api/games/:id/wiki`
- `PUT /api/games/:id/wiki`
- `GET /api/games/:id/wiki/history`

### Assets

- `POST /api/assets/cover`
- `POST /api/assets/banner`
- `POST /api/assets/screenshot`
- `DELETE /api/assets`

Expected upload metadata:
- target `game_id`
- uploaded file or validated remote image URL in later phases

### Metadata

- `GET /api/series`
- `POST /api/series`
- `GET /api/platforms`
- `POST /api/platforms`
- `GET /api/developers`
- `POST /api/developers`
- `GET /api/publishers`
- `POST /api/publishers`

### Directory Assist

- `GET /api/directory/default`
- `GET /api/directory/list`

### Steam Assist

- `GET /api/steam/search`
- `GET /api/steam/:appId/assets`
- `POST /api/steam/:appId/apply-assets`

## Explicitly Removed Endpoints

These old routes are not part of V2:
- `GET /api/games/search`
- `GET /api/games/stats`
- `GET /api/games/recent`
- `GET /api/games/most-played`
- `GET /api/games/favorites`
- `GET /api/games/platforms`
- `POST /api/games/platforms`
- `POST /api/games/batch`
- `POST /api/games/:id/favorite`
- `POST /api/games/:id/play`
- `GET /api/games/:id/versions`
- `GET /api/games/pending-review`
- `POST /api/games/:id/needs-review`
- `GET /api/games/:id/download`
- `GET /api/games/:id/download-info`
- `POST /api/upload/download-from-url`
- `DELETE /api/upload/file`
- all websocket routes
- all auth and register/login routes

