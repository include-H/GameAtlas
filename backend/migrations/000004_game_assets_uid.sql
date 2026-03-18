ALTER TABLE game_assets ADD COLUMN asset_uid TEXT;

UPDATE game_assets
SET asset_uid = printf('%d-%06x', game_id, id)
WHERE asset_uid IS NULL OR TRIM(asset_uid) = '';

CREATE UNIQUE INDEX IF NOT EXISTS idx_game_assets_asset_uid_unique
ON game_assets (asset_uid);
