ALTER TABLE games ADD COLUMN title_sort_key TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_games_title_sort_key ON games (title_sort_key, id);
