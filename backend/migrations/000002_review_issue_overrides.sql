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

CREATE INDEX IF NOT EXISTS idx_game_review_issue_overrides_game_id
ON game_review_issue_overrides (game_id);
