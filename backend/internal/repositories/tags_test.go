package repositories

import (
	"path/filepath"
	"strings"
	"testing"

	dbpkg "github.com/hao/game/internal/db"
	"github.com/jmoiron/sqlx"
)

func TestTagsRepositoryValidateTagSelectionRejectsMultipleTagsInSingleSelectGroup(t *testing.T) {
	db := openRepositoryTagsTestDB(t)
	defer func() { _ = db.Close() }()

	repo := NewTagsRepository(db)
	groupID := insertRepositoryTagGroup(t, db, "rating", "Rating", false, true)
	easyID := insertRepositoryTag(t, db, groupID, "Easy", "easy", true)
	hardID := insertRepositoryTag(t, db, groupID, "Hard", "hard", true)

	_, err := repo.ValidateTagSelection([]int64{easyID, hardID})
	if err == nil || !strings.Contains(err.Error(), "multiple tags selected in single-select group") {
		t.Fatalf("ValidateTagSelection error = %v, want single-select conflict", err)
	}
}

func openRepositoryTagsTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := dbpkg.OpenSQLite(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("OpenSQLite returned error: %v", err)
	}
	if err := dbpkg.RunMigrations(db); err != nil {
		_ = db.Close()
		t.Fatalf("RunMigrations returned error: %v", err)
	}

	return db
}

func insertRepositoryTagGroup(t *testing.T, db *sqlx.DB, key string, name string, allowMultiple bool, isFilterable bool) int64 {
	t.Helper()

	result, err := db.Exec(`
		INSERT INTO tag_groups (key, name, allow_multiple, is_filterable)
		VALUES (?, ?, ?, ?)
	`, key, name, boolToInt(allowMultiple), boolToInt(isFilterable))
	if err != nil {
		t.Fatalf("insert tag group: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId returned error: %v", err)
	}

	return id
}

func insertRepositoryTag(t *testing.T, db *sqlx.DB, groupID int64, name string, slug string, isActive bool) int64 {
	t.Helper()

	result, err := db.Exec(`
		INSERT INTO tags (group_id, name, slug, is_active)
		VALUES (?, ?, ?, ?)
	`, groupID, name, slug, boolToInt(isActive))
	if err != nil {
		t.Fatalf("insert tag: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId returned error: %v", err)
	}

	return id
}
