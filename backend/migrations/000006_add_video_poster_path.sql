-- 预告片封面帧：上传视频时由前端抽取一帧，路径存在 video 资产行上。
ALTER TABLE game_assets ADD COLUMN poster_path TEXT;
