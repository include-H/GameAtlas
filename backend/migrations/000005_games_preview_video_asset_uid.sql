ALTER TABLE games ADD COLUMN preview_video_asset_uid TEXT;

UPDATE games
SET preview_video_asset_uid = (
    SELECT ga.asset_uid
    FROM game_assets ga
    WHERE ga.game_id = games.id
      AND ga.asset_type = 'video'
    ORDER BY ga.sort_order ASC, ga.id ASC
    LIMIT 1
)
WHERE COALESCE(preview_video_asset_uid, '') = '';

CREATE INDEX IF NOT EXISTS idx_games_preview_video_asset_uid
ON games (preview_video_asset_uid);
