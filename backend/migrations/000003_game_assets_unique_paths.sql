DELETE FROM game_assets
WHERE id IN (
    SELECT duplicate.id
    FROM game_assets AS duplicate
    JOIN (
        SELECT game_id, asset_type, path, MIN(id) AS keep_id
        FROM game_assets
        GROUP BY game_id, asset_type, path
        HAVING COUNT(*) > 1
    ) AS grouped
      ON grouped.game_id = duplicate.game_id
     AND grouped.asset_type = duplicate.asset_type
     AND grouped.path = duplicate.path
    WHERE duplicate.id <> grouped.keep_id
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_game_assets_game_type_path_unique
ON game_assets (game_id, asset_type, path);
