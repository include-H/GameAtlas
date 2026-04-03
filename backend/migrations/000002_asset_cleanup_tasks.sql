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

CREATE INDEX IF NOT EXISTS idx_asset_cleanup_tasks_updated_at ON asset_cleanup_tasks (updated_at);
