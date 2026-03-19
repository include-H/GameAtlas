CREATE TABLE IF NOT EXISTS tag_groups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT,
    sort_order INTEGER NOT NULL DEFAULT 0,
    allow_multiple INTEGER NOT NULL DEFAULT 1,
    is_filterable INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    group_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    parent_id INTEGER,
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(group_id, name),
    UNIQUE(group_id, slug),
    FOREIGN KEY (group_id) REFERENCES tag_groups(id) ON DELETE CASCADE,
    FOREIGN KEY (parent_id) REFERENCES tags(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS game_tags (
    game_id INTEGER NOT NULL,
    tag_id INTEGER NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (game_id, tag_id),
    FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_tags_group_id ON tags (group_id);
CREATE INDEX IF NOT EXISTS idx_tags_parent_id ON tags (parent_id);
CREATE INDEX IF NOT EXISTS idx_tags_is_active ON tags (is_active);
CREATE INDEX IF NOT EXISTS idx_game_tags_tag_id ON game_tags (tag_id);

INSERT INTO tag_groups (key, name, description, sort_order, allow_multiple, is_filterable)
SELECT 'genre', '题材', '游戏的主要题材分类', 10, 1, 1
WHERE NOT EXISTS (SELECT 1 FROM tag_groups WHERE key = 'genre');

INSERT INTO tag_groups (key, name, description, sort_order, allow_multiple, is_filterable)
SELECT 'subgenre', '子类型', '更细分的玩法或内容类型', 20, 1, 1
WHERE NOT EXISTS (SELECT 1 FROM tag_groups WHERE key = 'subgenre');

INSERT INTO tag_groups (key, name, description, sort_order, allow_multiple, is_filterable)
SELECT 'perspective', '视角', '游戏视角或镜头表现方式', 30, 1, 1
WHERE NOT EXISTS (SELECT 1 FROM tag_groups WHERE key = 'perspective');

INSERT INTO tag_groups (key, name, description, sort_order, allow_multiple, is_filterable)
SELECT 'theme', '内容属性', '内容题材或受众导向标签', 40, 1, 1
WHERE NOT EXISTS (SELECT 1 FROM tag_groups WHERE key = 'theme');
