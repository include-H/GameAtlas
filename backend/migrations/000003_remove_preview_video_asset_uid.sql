DROP INDEX IF EXISTS idx_games_preview_video_asset_uid;

ALTER TABLE games DROP COLUMN preview_video_asset_uid;
