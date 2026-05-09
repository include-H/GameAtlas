-- +migrate Up
DROP TABLE IF EXISTS game_tags;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS tag_groups;

-- +migrate Down
-- Tag system removed; no down migration.
