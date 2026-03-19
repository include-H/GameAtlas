PRAGMA foreign_keys = OFF;

CREATE TABLE series_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    slug TEXT NOT NULL UNIQUE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO series_new (id, name, slug, sort_order, created_at)
SELECT id, name, slug, sort_order, created_at
FROM series;

DROP TABLE series;

ALTER TABLE series_new RENAME TO series;

PRAGMA foreign_keys = ON;
