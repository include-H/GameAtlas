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

func TestTagsRepositoryGroupTagIDsGroupsDistinctSortedIDs(t *testing.T) {
	db := openRepositoryTagsTestDB(t)
	defer func() { _ = db.Close() }()

	repo := NewTagsRepository(db)
	genreID := insertRepositoryTagGroup(t, db, "custom-genre", "Genre", true, true)
	themeID := insertRepositoryTagGroup(t, db, "custom-theme", "Theme", true, true)
	actionID := insertRepositoryTag(t, db, genreID, "Action", "action", true)
	rpgID := insertRepositoryTag(t, db, genreID, "RPG", "rpg", true)
	scifiID := insertRepositoryTag(t, db, themeID, "Sci-Fi", "scifi", true)

	grouped, err := repo.GroupTagIDs([]int64{scifiID, rpgID, actionID, rpgID})
	if err != nil {
		t.Fatalf("GroupTagIDs returned error: %v", err)
	}

	if len(grouped) != 2 {
		t.Fatalf("len(grouped) = %d, want 2", len(grouped))
	}
	if got := grouped[genreID]; len(got) != 2 || got[0] != actionID || got[1] != rpgID {
		t.Fatalf("grouped[%d] = %#v, want [%d %d]", genreID, got, actionID, rpgID)
	}
	if got := grouped[themeID]; len(got) != 1 || got[0] != scifiID {
		t.Fatalf("grouped[%d] = %#v, want [%d]", themeID, got, scifiID)
	}
}

func TestTagsRepositoryGroupTagIDsRejectsInactiveSelections(t *testing.T) {
	db := openRepositoryTagsTestDB(t)
	defer func() { _ = db.Close() }()

	repo := NewTagsRepository(db)
	groupID := insertRepositoryTagGroup(t, db, "custom-theme", "Theme", true, true)
	activeID := insertRepositoryTag(t, db, groupID, "Active", "active", true)
	inactiveID := insertRepositoryTag(t, db, groupID, "Inactive", "inactive", false)

	_, err := repo.GroupTagIDs([]int64{activeID, inactiveID})
	if err == nil || !strings.Contains(err.Error(), "missing tag ids") {
		t.Fatalf("GroupTagIDs error = %v, want missing tag ids", err)
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
