CREATE INDEX IF NOT EXISTS idx_game_assets_game_type_sort_id
ON game_assets (game_id, asset_type, sort_order, id);
