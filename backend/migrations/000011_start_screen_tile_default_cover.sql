-- 未选图的磁贴默认使用游戏首张截图作为磁贴原图（截图是游戏自有素材、高分辨率，
-- 符合"只用原图"约束）；没有截图时退而求其次用封面；两者都没有则保持空图，
-- 显示首字母色块。
UPDATE start_screen_tiles
SET image_path = (
  SELECT ga.path
  FROM game_assets ga
  WHERE ga.game_id = start_screen_tiles.game_id
    AND ga.asset_type = 'screenshot'
    AND COALESCE(TRIM(ga.path), '') != ''
  ORDER BY ga.sort_order ASC, ga.id ASC
  LIMIT 1
)
WHERE image_path IS NULL
  AND EXISTS (
    SELECT 1 FROM game_assets ga
    WHERE ga.game_id = start_screen_tiles.game_id
      AND ga.asset_type = 'screenshot'
      AND COALESCE(TRIM(ga.path), '') != ''
  );

UPDATE start_screen_tiles
SET image_path = (SELECT g.cover_image FROM games g WHERE g.id = start_screen_tiles.game_id)
WHERE image_path IS NULL
  AND EXISTS (
    SELECT 1 FROM games g
    WHERE g.id = start_screen_tiles.game_id
      AND COALESCE(TRIM(g.cover_image), '') != ''
  );
