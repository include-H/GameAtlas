-- 开始屏幕磁贴改为单图 + 焦点：直接用游戏素材原图（object-fit/object-position 渲染），
-- 不再生成三档裁剪小图。旧裁剪图不继承（小图拉伸糊且非本游戏素材路径），
-- 由 000010 对已应用的库清空回填值；存量磁贴回到"未选图"状态，用新选择器重选。
ALTER TABLE start_screen_tiles ADD COLUMN image_path TEXT;
ALTER TABLE start_screen_tiles ADD COLUMN focus_x INTEGER NOT NULL DEFAULT 50;
ALTER TABLE start_screen_tiles ADD COLUMN focus_y INTEGER NOT NULL DEFAULT 50;
