package db

import (
	"path/filepath"
	"testing"

	"github.com/jmoiron/sqlx"
)

func TestRunMigrationsOnFreshDatabaseAppliesAllEmbeddedFiles(t *testing.T) {
	db := openTestSQLite(t)
	defer func() {
		_ = db.Close()
	}()

	embedded, err := loadEmbeddedMigrations()
	if err != nil {
		t.Fatalf("loadEmbeddedMigrations returned error: %v", err)
	}

	if err := RunMigrations(db); err != nil {
		t.Fatalf("RunMigrations returned error: %v", err)
	}

	for _, item := range embedded {
		assertMigrationRecordedState(t, db, item.Name, true)
	}

	assertTableExists(t, db, "games")
	assertTableExists(t, db, "game_assets")
	assertTableExists(t, db, "tag_groups")
	assertColumnExists(t, db, "games", "series_id")
	assertIndexExists(t, db, "idx_games_release_date_id")
	assertIndexExists(t, db, "idx_games_public_id")
	assertIndexExists(t, db, "idx_game_assets_game_type_sort_id")
	assertIndexExists(t, db, "idx_games_series_id")

	var groupCount int
	if err := db.Get(&groupCount, "SELECT COUNT(*) FROM tag_groups"); err != nil {
		t.Fatalf("count tag_groups returned error: %v", err)
	}
	if groupCount < 4 {
		t.Fatalf("expected default tag groups to be seeded, got %d rows", groupCount)
	}
}

func TestRunMigrationsIsIdempotent(t *testing.T) {
	db := openTestSQLite(t)
	defer func() {
		_ = db.Close()
	}()

	embedded, err := loadEmbeddedMigrations()
	if err != nil {
		t.Fatalf("loadEmbeddedMigrations returned error: %v", err)
	}

	if err := RunMigrations(db); err != nil {
		t.Fatalf("RunMigrations first run returned error: %v", err)
	}
	if err := RunMigrations(db); err != nil {
		t.Fatalf("RunMigrations second run returned error: %v", err)
	}

	var total int
	if err := db.Get(&total, "SELECT COUNT(*) FROM schema_migrations"); err != nil {
		t.Fatalf("count schema_migrations returned error: %v", err)
	}
	if total != len(embedded) {
		t.Fatalf("schema_migrations count = %d, want %d", total, len(embedded))
	}

	for _, item := range embedded {
		var count int
		if err := db.Get(&count, "SELECT COUNT(*) FROM schema_migrations WHERE name = ?", item.Name); err != nil {
			t.Fatalf("count migration %q returned error: %v", item.Name, err)
		}
		if count != 1 {
			t.Fatalf("migration %q record count = %d, want 1", item.Name, count)
		}
	}
}

func TestApplyMigrationHandlesSemicolonInStringLiteral(t *testing.T) {
	db := openTestSQLite(t)
	defer func() {
		_ = db.Close()
	}()

	if err := ensureMigrationTable(db); err != nil {
		t.Fatalf("ensureMigrationTable returned error: %v", err)
	}

	const name = "999999_semicolon_in_string.sql"
	const migration = `
CREATE TABLE sample_notes (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	note TEXT NOT NULL
);
INSERT INTO sample_notes (note) VALUES ('hello;world');
`

	if err := applyMigration(db, name, migration); err != nil {
		t.Fatalf("applyMigration returned error: %v", err)
	}

	var note string
	if err := db.Get(&note, "SELECT note FROM sample_notes LIMIT 1"); err != nil {
		t.Fatalf("select inserted note returned error: %v", err)
	}
	if note != "hello;world" {
		t.Fatalf("inserted note = %q, want %q", note, "hello;world")
	}

	assertMigrationRecordedState(t, db, name, true)
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

func assertTableExists(t *testing.T, db *sqlx.DB, table string) {
	t.Helper()

	var exists int
	if err := db.Get(&exists, "SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = ? LIMIT 1", table); err != nil {
		t.Fatalf("check table %q returned error: %v", table, err)
	}
}

func assertIndexExists(t *testing.T, db *sqlx.DB, index string) {
	t.Helper()

	var exists int
	if err := db.Get(&exists, "SELECT 1 FROM sqlite_master WHERE type = 'index' AND name = ? LIMIT 1", index); err != nil {
		t.Fatalf("check index %q returned error: %v", index, err)
	}
}

func assertColumnExists(t *testing.T, db *sqlx.DB, table string, column string) {
	t.Helper()

	type tableInfoRow struct {
		CID          int     `db:"cid"`
		Name         string  `db:"name"`
		Type         string  `db:"type"`
		NotNull      int     `db:"notnull"`
		DefaultValue *string `db:"dflt_value"`
		PrimaryKey   int     `db:"pk"`
	}

	var rows []tableInfoRow
	if err := db.Select(&rows, "PRAGMA table_info("+table+")"); err != nil {
		t.Fatalf("check columns for table %q returned error: %v", table, err)
	}

	for _, row := range rows {
		if row.Name == column {
			return
		}
	}

	t.Fatalf("column %q does not exist on table %q", column, table)
}
