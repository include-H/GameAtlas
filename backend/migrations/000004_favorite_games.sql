CREATE TABLE IF NOT EXISTS favorite_games (
    game_id INTEGER PRIMARY KEY,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_favorite_games_created_at ON favorite_games (created_at DESC);
