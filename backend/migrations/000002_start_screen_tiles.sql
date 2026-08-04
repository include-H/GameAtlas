CREATE TABLE IF NOT EXISTS start_screen_tiles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    game_id INTEGER NOT NULL UNIQUE,
    tile_size TEXT NOT NULL DEFAULT 'small' CHECK (tile_size IN ('small', 'wide', 'large')),
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_start_screen_tiles_sort_order ON start_screen_tiles (sort_order, id);
