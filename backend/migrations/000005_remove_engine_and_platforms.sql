PRAGMA foreign_keys = OFF;

DROP TABLE IF EXISTS game_platforms;
DROP TABLE IF EXISTS platforms;

CREATE TABLE games_new (
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

INSERT INTO games_new (
    id, public_id, title, title_alt, title_sort_key, visibility, summary, release_date,
    cover_image, banner_image, wiki_content, downloads, created_at, updated_at, series_id
)
SELECT
    id, public_id, title, title_alt, title_sort_key, visibility, summary, release_date,
    cover_image, banner_image, wiki_content, downloads, created_at, updated_at, series_id
FROM games;

DROP TABLE games;
ALTER TABLE games_new RENAME TO games;

CREATE INDEX IF NOT EXISTS idx_games_title ON games (title);
CREATE UNIQUE INDEX IF NOT EXISTS idx_games_public_id ON games (public_id);
CREATE INDEX IF NOT EXISTS idx_games_title_sort_key ON games (title_sort_key, id);
CREATE INDEX IF NOT EXISTS idx_games_visibility ON games (visibility);
CREATE INDEX IF NOT EXISTS idx_games_release_date_id ON games (release_date DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_games_updated_at ON games (updated_at);
CREATE INDEX IF NOT EXISTS idx_games_series_id ON games (series_id);

PRAGMA foreign_keys = ON;
