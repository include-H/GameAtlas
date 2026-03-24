CREATE TABLE IF NOT EXISTS games (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    title_alt TEXT,
    visibility TEXT NOT NULL DEFAULT 'public',
    summary TEXT,
    release_date TEXT,
    engine TEXT,
    preview_video_asset_uid TEXT,
    cover_image TEXT,
    banner_image TEXT,
    wiki_content TEXT,
    wiki_content_html TEXT,
    needs_review INTEGER NOT NULL DEFAULT 0,
    downloads INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
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

CREATE TABLE IF NOT EXISTS platforms (
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

CREATE TABLE IF NOT EXISTS game_series (
    game_id INTEGER NOT NULL,
    series_id INTEGER NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (game_id, series_id),
    FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE,
    FOREIGN KEY (series_id) REFERENCES series(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS game_platforms (
    game_id INTEGER NOT NULL,
    platform_id INTEGER NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (game_id, platform_id),
    FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE,
    FOREIGN KEY (platform_id) REFERENCES platforms(id) ON DELETE CASCADE
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

CREATE TABLE IF NOT EXISTS tag_groups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT,
    sort_order INTEGER NOT NULL DEFAULT 0,
    allow_multiple INTEGER NOT NULL DEFAULT 1,
    is_filterable INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    group_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    parent_id INTEGER,
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(group_id, name),
    UNIQUE(group_id, slug),
    FOREIGN KEY (group_id) REFERENCES tag_groups(id) ON DELETE CASCADE,
    FOREIGN KEY (parent_id) REFERENCES tags(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS game_tags (
    game_id INTEGER NOT NULL,
    tag_id INTEGER NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (game_id, tag_id),
    FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS auth_login_attempts (
    source_key TEXT PRIMARY KEY,
    fail_count INTEGER NOT NULL DEFAULT 0,
    first_failed_unix INTEGER NOT NULL DEFAULT 0,
    last_failed_unix INTEGER NOT NULL DEFAULT 0,
    locked_until_unix INTEGER NOT NULL DEFAULT 0,
    expires_at_unix INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_game_files_game_id ON game_files (game_id);
CREATE INDEX IF NOT EXISTS idx_game_assets_game_id ON game_assets (game_id);
CREATE INDEX IF NOT EXISTS idx_wiki_history_game_id ON wiki_history (game_id);
CREATE INDEX IF NOT EXISTS idx_game_review_issue_overrides_game_id ON game_review_issue_overrides (game_id);
CREATE INDEX IF NOT EXISTS idx_games_title ON games (title);
CREATE INDEX IF NOT EXISTS idx_games_visibility ON games (visibility);
CREATE INDEX IF NOT EXISTS idx_games_updated_at ON games (updated_at);
CREATE INDEX IF NOT EXISTS idx_games_preview_video_asset_uid ON games (preview_video_asset_uid);
CREATE INDEX IF NOT EXISTS idx_games_release_date_id ON games (release_date DESC, id DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_game_assets_game_type_path_unique ON game_assets (game_id, asset_type, path);
CREATE UNIQUE INDEX IF NOT EXISTS idx_game_assets_asset_uid_unique ON game_assets (asset_uid);
CREATE INDEX IF NOT EXISTS idx_tags_group_id ON tags (group_id);
CREATE INDEX IF NOT EXISTS idx_tags_parent_id ON tags (parent_id);
CREATE INDEX IF NOT EXISTS idx_tags_is_active ON tags (is_active);
CREATE INDEX IF NOT EXISTS idx_game_tags_tag_id ON game_tags (tag_id);
CREATE INDEX IF NOT EXISTS idx_auth_login_attempts_expires ON auth_login_attempts (expires_at_unix);

INSERT INTO tag_groups (key, name, description, sort_order, allow_multiple, is_filterable)
SELECT 'genre', '题材', '游戏的主要题材分类', 10, 1, 1
WHERE NOT EXISTS (SELECT 1 FROM tag_groups WHERE key = 'genre');

INSERT INTO tag_groups (key, name, description, sort_order, allow_multiple, is_filterable)
SELECT 'subgenre', '子类型', '更细分的玩法或内容类型', 20, 1, 1
WHERE NOT EXISTS (SELECT 1 FROM tag_groups WHERE key = 'subgenre');

INSERT INTO tag_groups (key, name, description, sort_order, allow_multiple, is_filterable)
SELECT 'perspective', '视角', '游戏视角或镜头表现方式', 30, 1, 1
WHERE NOT EXISTS (SELECT 1 FROM tag_groups WHERE key = 'perspective');

INSERT INTO tag_groups (key, name, description, sort_order, allow_multiple, is_filterable)
SELECT 'theme', '内容属性', '内容题材或受众导向标签', 40, 1, 1
WHERE NOT EXISTS (SELECT 1 FROM tag_groups WHERE key = 'theme');
