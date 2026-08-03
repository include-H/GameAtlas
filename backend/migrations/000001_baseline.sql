-- GameAtlas baseline schema
-- Consolidated baseline schema, synchronized through migration 000004.

CREATE TABLE IF NOT EXISTS games (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    public_id TEXT NOT NULL DEFAULT '',
    title TEXT NOT NULL,
    title_alt TEXT,
    title_sort_key TEXT NOT NULL DEFAULT '',
    visibility TEXT NOT NULL DEFAULT 'public',
    summary TEXT,
    release_date TEXT,
    cover_image TEXT,
    banner_image TEXT,
    wiki_content TEXT,
    downloads INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    series_id INTEGER REFERENCES series(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS game_files (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    game_id INTEGER NOT NULL,
    file_path TEXT NOT NULL,
    label TEXT,
    notes TEXT,
    size_bytes INTEGER,
    sort_order INTEGER NOT NULL DEFAULT 0,
    source_created_at TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS game_assets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    game_id INTEGER NOT NULL,
    asset_uid TEXT,
    asset_type TEXT NOT NULL,
    path TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    position_x REAL,
    position_y REAL,
    width_pct REAL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS wiki_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    game_id INTEGER NOT NULL,
    content TEXT NOT NULL,
    change_summary TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS series (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    slug TEXT NOT NULL UNIQUE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS developers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    slug TEXT NOT NULL UNIQUE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS publishers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    slug TEXT NOT NULL UNIQUE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS game_developers (
    game_id INTEGER NOT NULL,
    developer_id INTEGER NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (game_id, developer_id),
    FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE,
    FOREIGN KEY (developer_id) REFERENCES developers(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS game_publishers (
    game_id INTEGER NOT NULL,
    publisher_id INTEGER NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (game_id, publisher_id),
    FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE,
    FOREIGN KEY (publisher_id) REFERENCES publishers(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS game_review_issue_overrides (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    game_id INTEGER NOT NULL,
    issue_key TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'ignored',
    reason TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE,
    UNIQUE (game_id, issue_key)
);

CREATE TABLE IF NOT EXISTS auth_login_attempts (
    source_key TEXT PRIMARY KEY,
    fail_count INTEGER NOT NULL DEFAULT 0,
    first_failed_unix INTEGER NOT NULL DEFAULT 0,
    last_failed_unix INTEGER NOT NULL DEFAULT 0,
    locked_until_unix INTEGER NOT NULL DEFAULT 0,
    expires_at_unix INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS asset_cleanup_tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    asset_path TEXT NOT NULL,
    source TEXT NOT NULL DEFAULT '',
    last_error TEXT,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (asset_path)
);

CREATE TABLE IF NOT EXISTS auth_sessions (
    token TEXT PRIMARY KEY,
    expires_at_unix INTEGER NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS favorite_games (
    game_id INTEGER PRIMARY KEY,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS app_settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes

CREATE INDEX IF NOT EXISTS idx_game_files_game_id ON game_files (game_id);
CREATE INDEX IF NOT EXISTS idx_game_assets_game_id ON game_assets (game_id);
CREATE INDEX IF NOT EXISTS idx_wiki_history_game_id ON wiki_history (game_id);
CREATE INDEX IF NOT EXISTS idx_game_review_issue_overrides_game_id ON game_review_issue_overrides (game_id);
CREATE INDEX IF NOT EXISTS idx_games_title ON games (title);
CREATE INDEX IF NOT EXISTS idx_games_title_sort_key ON games (title_sort_key, id);
CREATE INDEX IF NOT EXISTS idx_games_visibility ON games (visibility);
CREATE INDEX IF NOT EXISTS idx_games_updated_at ON games (updated_at);
CREATE INDEX IF NOT EXISTS idx_games_series_id ON games (series_id);
CREATE INDEX IF NOT EXISTS idx_games_release_date_id ON games (release_date DESC, id DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_games_public_id ON games (public_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_game_assets_game_type_path_unique ON game_assets (game_id, asset_type, path);
CREATE UNIQUE INDEX IF NOT EXISTS idx_game_assets_asset_uid_unique ON game_assets (asset_uid);
CREATE INDEX IF NOT EXISTS idx_game_assets_game_type_sort_id ON game_assets (game_id, asset_type, sort_order, id);
CREATE INDEX IF NOT EXISTS idx_auth_login_attempts_expires ON auth_login_attempts (expires_at_unix);
CREATE INDEX IF NOT EXISTS idx_asset_cleanup_tasks_updated_at ON asset_cleanup_tasks (updated_at);
CREATE INDEX IF NOT EXISTS idx_auth_sessions_expires ON auth_sessions (expires_at_unix);
CREATE INDEX IF NOT EXISTS idx_favorite_games_created_at ON favorite_games (created_at DESC);
