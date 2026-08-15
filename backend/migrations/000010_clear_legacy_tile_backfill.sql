-- 000009 早前版本会把旧三档裁剪图路径回填进 image_path；这些 /assets/start-screen/
-- 路径不是游戏素材原图，新校验（必须是 /assets/{publicId}/）会拒绝保存。
-- 已应用的库在此清空回填值；新库 000009 已不回填，本文件的前半部分为空操作。
UPDATE start_screen_tiles SET image_path = NULL WHERE image_path LIKE '/assets/start-screen/%';

-- 旧三档裁剪图字段一并清空：磁贴已不再使用派生裁剪图，清空后旧文件失去
-- 引用，由启动时的孤儿资产清扫移入隔离区（7 天后删除）。
UPDATE start_screen_tiles
SET image_small_path = NULL, image_wide_path = NULL, image_large_path = NULL;
