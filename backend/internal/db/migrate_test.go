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
	assertIndexExists(t, db, "idx_games_release_date_id")

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
