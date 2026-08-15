-- 宽磁贴（2x4）活磁贴：轮播序列 = image_path（首帧）+ flip_images（追加帧，JSON 数组，
-- 上限 3 张，共 4 帧）。方形磁贴不使用该字段；NULL 表示不启用轮播。
ALTER TABLE start_screen_tiles ADD COLUMN flip_images TEXT;
