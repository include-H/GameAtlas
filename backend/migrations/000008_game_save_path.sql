-- 游戏存档目录模板（Windows BAT 内嵌变量形式，如 %USERPROFILE%\Documents\My Games\GT4\SaveGame）。
-- 空串表示未配置：启动脚本不提供"打开存档目录"选项。
ALTER TABLE games ADD COLUMN save_path_template TEXT NOT NULL DEFAULT '';
