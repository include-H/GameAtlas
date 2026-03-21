ALTER TABLE games ADD COLUMN visibility TEXT NOT NULL DEFAULT 'public';

CREATE INDEX IF NOT EXISTS idx_games_visibility ON games (visibility);
