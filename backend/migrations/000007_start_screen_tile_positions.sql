ALTER TABLE start_screen_tiles ADD COLUMN column_index INTEGER NOT NULL DEFAULT 0;
ALTER TABLE start_screen_tiles ADD COLUMN grid_row INTEGER NOT NULL DEFAULT 0;
ALTER TABLE start_screen_tiles ADD COLUMN grid_col INTEGER NOT NULL DEFAULT 0;

-- 旧数据没有显式坐标：按原有 sort_order 近似铺回 2x6 列，避免升级后全部叠在原点。
UPDATE start_screen_tiles
SET
    column_index = sort_order / 12,
    grid_row = (sort_order / 2) % 6,
    grid_col = sort_order % 2;
