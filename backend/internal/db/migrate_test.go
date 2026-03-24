package db

import (
	"path/filepath"
	"testing"

	"github.com/jmoiron/sqlx"
)

func TestRunMigrationsOnFreshDatabaseUsesCurrentBaseline(t *testing.T) {
	db := openTestSQLite(t)
	defer func() {
		_ = db.Close()
	}()

	if err := RunMigrations(db); err != nil {
		t.Fatalf("RunMigrations returned error: %v", err)
	}

	assertMigrationRecordedState(t, db, "000001_initial_schema.sql", true)
	assertMigrationRecordedState(t, db, "000004_game_assets_uid.sql", false)
	assertMigrationRecordedState(t, db, "000008_tag_system.sql", false)
	assertMigrationRecordedState(t, db, "000011_game_files_source_created_at.sql", false)
	assertMigrationRecordedState(t, db, "000012_drop_games_views.sql", false)

	assertColumnExists(t, db, "games", "preview_video_asset_uid")
	assertColumnMissing(t, db, "games", "views")
	assertColumnExists(t, db, "game_assets", "asset_uid")
	assertColumnExists(t, db, "game_files", "source_created_at")
	assertTableExists(t, db, "game_review_issue_overrides")
	assertTableExists(t, db, "tag_groups")
	assertTableExists(t, db, "tags")
	assertTableExists(t, db, "game_tags")
	assertTableExists(t, db, "auth_login_attempts")
	assertIndexExists(t, db, "idx_games_release_date_id")
	assertTagGroupExists(t, db, "genre")
	assertTagGroupExists(t, db, "subgenre")
	assertTagGroupExists(t, db, "perspective")
	assertTagGroupExists(t, db, "theme")
}

func TestRunMigrationsUpgradesLegacyDatabase(t *testing.T) {
	db := openTestSQLite(t)
	defer func() {
		_ = db.Close()
	}()

	createLegacySchema(t, db)
	insertLegacyMigrationRecord(t, db, "000001_initial_schema.sql")

	mustExec(t, db, `
		INSERT INTO games (
			id, title, title_alt, visibility, summary, release_date, engine, cover_image, banner_image,
			wiki_content, wiki_content_html, needs_review, views, downloads, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, 1, "Legacy Game", nil, "public", nil, "2024-01-01", nil, nil, nil, nil, nil, 0, 0, 0)
	mustExec(t, db, `
		INSERT INTO game_assets (id, game_id, asset_type, path, sort_order, created_at)
		VALUES
			(10, 1, 'screenshot', 'shots/a.png', 0, CURRENT_TIMESTAMP),
			(11, 1, 'screenshot', 'shots/a.png', 1, CURRENT_TIMESTAMP),
			(12, 1, 'video', 'videos/trailer.mp4', 0, CURRENT_TIMESTAMP)
	`)

	if err := RunMigrations(db); err != nil {
		t.Fatalf("RunMigrations returned error: %v", err)
	}

	assertMigrationRecordedState(t, db, "000003_game_assets_unique_paths.sql", true)
	assertMigrationRecordedState(t, db, "000004_game_assets_uid.sql", true)
	assertMigrationRecordedState(t, db, "000005_games_preview_video_asset_uid.sql", true)
	assertMigrationRecordedState(t, db, "000008_tag_system.sql", true)
	assertMigrationRecordedState(t, db, "000010_auth_login_attempts.sql", true)
	assertMigrationRecordedState(t, db, "000011_game_files_source_created_at.sql", true)
	assertMigrationRecordedState(t, db, "000012_drop_games_views.sql", true)

	assertColumnExists(t, db, "game_assets", "asset_uid")
	assertColumnExists(t, db, "games", "preview_video_asset_uid")
	assertColumnMissing(t, db, "games", "views")
	assertColumnExists(t, db, "game_files", "source_created_at")
	assertTableExists(t, db, "game_review_issue_overrides")
	assertTableExists(t, db, "tag_groups")
	assertTableExists(t, db, "auth_login_attempts")
	assertIndexExists(t, db, "idx_game_assets_game_type_path_unique")

	var assetCount int
	if err := db.Get(&assetCount, "SELECT COUNT(*) FROM game_assets WHERE game_id = ? AND asset_type = 'screenshot' AND path = 'shots/a.png'", 1); err != nil {
		t.Fatalf("count deduped screenshot assets: %v", err)
	}
	if assetCount != 1 {
		t.Fatalf("expected duplicate screenshot assets to be collapsed to 1 row, got %d", assetCount)
	}

	var previewVideoAssetUID string
	if err := db.Get(&previewVideoAssetUID, "SELECT preview_video_asset_uid FROM games WHERE id = ?", 1); err != nil {
		t.Fatalf("load preview video asset uid: %v", err)
	}
	if previewVideoAssetUID == "" {
		t.Fatalf("expected preview_video_asset_uid to be backfilled")
	}

	var videoAssetUID string
	if err := db.Get(&videoAssetUID, "SELECT asset_uid FROM game_assets WHERE id = ?", 12); err != nil {
		t.Fatalf("load video asset uid: %v", err)
	}
	if videoAssetUID == "" {
		t.Fatalf("expected video asset uid to be backfilled")
	}
	if previewVideoAssetUID != videoAssetUID {
		t.Fatalf("preview video asset uid = %q, want %q", previewVideoAssetUID, videoAssetUID)
	}
}

func TestRunMigrationsRecognizesLegacyMigrationNames(t *testing.T) {
	db := openTestSQLite(t)
	defer func() {
		_ = db.Close()
	}()

	if err := RunMigrations(db); err != nil {
		t.Fatalf("RunMigrations returned error: %v", err)
	}

	mustExec(t, db, "DELETE FROM schema_migrations WHERE name = ?", "000002_game_review_issue_overrides.sql")
	mustExec(t, db, "INSERT INTO schema_migrations (name) VALUES (?)", "000002_review_issue_overrides.sql")

	if err := RunMigrations(db); err != nil {
		t.Fatalf("RunMigrations rerun returned error: %v", err)
	}

	assertMigrationRecordedState(t, db, "000002_review_issue_overrides.sql", true)
	assertMigrationRecordedState(t, db, "000002_game_review_issue_overrides.sql", false)
}

func createLegacySchema(t *testing.T, db *sqlx.DB) {
	t.Helper()

	statements := splitMigrationStatements(legacyInitialSchema)
	for _, stmt := range statements {
		mustExec(t, db, stmt)
	}
}

func insertLegacyMigrationRecord(t *testing.T, db *sqlx.DB, name string) {
	t.Helper()

	if err := ensureMigrationTable(db); err != nil {
		t.Fatalf("ensureMigrationTable returned error: %v", err)
	}
	mustExec(t, db, "INSERT INTO schema_migrations (name) VALUES (?)", name)
}

func openTestSQLite(t *testing.T) *sqlx.DB {
	t.Helper()

	path := filepath.Join(t.TempDir(), "app.db")
	db, err := OpenSQLite(path)
	if err != nil {
		t.Fatalf("OpenSQLite returned error: %v", err)
	}

	return db
}

func mustExec(t *testing.T, db *sqlx.DB, query string, args ...any) {
	t.Helper()

	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("exec %q returned error: %v", query, err)
	}
}

func assertMigrationRecordedState(t *testing.T, db *sqlx.DB, name string, want bool) {
	t.Helper()

	got, err := hasMigration(db, name)
	if err != nil {
		t.Fatalf("hasMigration(%q) returned error: %v", name, err)
	}
	if got != want {
		t.Fatalf("hasMigration(%q) = %v, want %v", name, got, want)
	}
}

func assertColumnExists(t *testing.T, db *sqlx.DB, table string, column string) {
	t.Helper()

	exists, err := hasTableColumn(db, table, column)
	if err != nil {
		t.Fatalf("hasTableColumn(%q, %q) returned error: %v", table, column, err)
	}
	if !exists {
		t.Fatalf("expected column %s.%s to exist", table, column)
	}
}

func assertColumnMissing(t *testing.T, db *sqlx.DB, table string, column string) {
	t.Helper()

	exists, err := hasTableColumn(db, table, column)
	if err != nil {
		t.Fatalf("hasTableColumn(%q, %q) returned error: %v", table, column, err)
	}
	if exists {
		t.Fatalf("expected column %s.%s to be removed", table, column)
	}
}

func assertTableExists(t *testing.T, db *sqlx.DB, table string) {
	t.Helper()

	exists, err := hasTable(db, table)
	if err != nil {
		t.Fatalf("hasTable(%q) returned error: %v", table, err)
	}
	if !exists {
		t.Fatalf("expected table %s to exist", table)
	}
}

func assertIndexExists(t *testing.T, db *sqlx.DB, index string) {
	t.Helper()

	exists, err := hasIndex(db, index)
	if err != nil {
		t.Fatalf("hasIndex(%q) returned error: %v", index, err)
	}
	if !exists {
		t.Fatalf("expected index %s to exist", index)
	}
}

func assertTagGroupExists(t *testing.T, db *sqlx.DB, key string) {
	t.Helper()

	exists, err := hasTagGroupKey(db, key)
	if err != nil {
		t.Fatalf("hasTagGroupKey(%q) returned error: %v", key, err)
	}
	if !exists {
		t.Fatalf("expected tag group %s to exist", key)
	}
}

const legacyInitialSchema = `
CREATE TABLE IF NOT EXISTS games (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	title TEXT NOT NULL,
	title_alt TEXT,
	visibility TEXT NOT NULL DEFAULT 'public',
	summary TEXT,
	release_date TEXT,
	engine TEXT,
	cover_image TEXT,
	banner_image TEXT,
	wiki_content TEXT,
	wiki_content_html TEXT,
	needs_review INTEGER NOT NULL DEFAULT 0,
	views INTEGER NOT NULL DEFAULT 0,
	downloads INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS game_files (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	game_id INTEGER NOT NULL,
	file_path TEXT NOT NULL,
	label TEXT,
	notes TEXT,
	size_bytes INTEGER,
	sort_order INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS game_assets (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	game_id INTEGER NOT NULL,
	asset_type TEXT NOT NULL,
	path TEXT NOT NULL,
	sort_order INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS wiki_history (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	game_id INTEGER NOT NULL,
	content TEXT NOT NULL,
	change_summary TEXT,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS series (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL UNIQUE,
	slug TEXT NOT NULL UNIQUE,
	sort_order INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS platforms (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL UNIQUE,
	slug TEXT NOT NULL UNIQUE,
	sort_order INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS developers (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL UNIQUE,
	slug TEXT NOT NULL UNIQUE,
	sort_order INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS publishers (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL UNIQUE,
	slug TEXT NOT NULL UNIQUE,
	sort_order INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS game_series (
	game_id INTEGER NOT NULL,
	series_id INTEGER NOT NULL,
	sort_order INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (game_id, series_id),
	FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE,
	FOREIGN KEY (series_id) REFERENCES series(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS game_platforms (
	game_id INTEGER NOT NULL,
	platform_id INTEGER NOT NULL,
	sort_order INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (game_id, platform_id),
	FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE,
	FOREIGN KEY (platform_id) REFERENCES platforms(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS game_developers (
	game_id INTEGER NOT NULL,
	developer_id INTEGER NOT NULL,
	sort_order INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (game_id, developer_id),
	FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE,
	FOREIGN KEY (developer_id) REFERENCES developers(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS game_publishers (
	game_id INTEGER NOT NULL,
	publisher_id INTEGER NOT NULL,
	sort_order INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (game_id, publisher_id),
	FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE,
	FOREIGN KEY (publisher_id) REFERENCES publishers(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_game_files_game_id ON game_files (game_id);
CREATE INDEX IF NOT EXISTS idx_game_assets_game_id ON game_assets (game_id);
CREATE INDEX IF NOT EXISTS idx_wiki_history_game_id ON wiki_history (game_id);
CREATE INDEX IF NOT EXISTS idx_games_title ON games (title);
CREATE INDEX IF NOT EXISTS idx_games_visibility ON games (visibility);
CREATE INDEX IF NOT EXISTS idx_games_updated_at ON games (updated_at);
`
