CREATE INDEX IF NOT EXISTS idx_games_release_date_id
ON games (release_date DESC, id DESC);
