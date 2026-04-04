package repositories

import "testing"

func TestGameFilesRepositoryListByGameIDOrdersBySortOrderThenID(t *testing.T) {
	db := openRepositoryTagsTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertRepositoryGame(t, db, "repo-files-order", "Repo Files Order", "public")
	firstID := insertRepositoryGameFile(t, db, gameID, "/roms/second.rom")
	if _, err := db.Exec(`UPDATE game_files SET sort_order = 2 WHERE id = ?`, firstID); err != nil {
		t.Fatalf("update first sort_order: %v", err)
	}
	secondID := insertRepositoryGameFile(t, db, gameID, "/roms/first-a.rom")
	if _, err := db.Exec(`UPDATE game_files SET sort_order = 0 WHERE id = ?`, secondID); err != nil {
		t.Fatalf("update second sort_order: %v", err)
	}
	thirdID := insertRepositoryGameFile(t, db, gameID, "/roms/first-b.rom")
	if _, err := db.Exec(`UPDATE game_files SET sort_order = 0 WHERE id = ?`, thirdID); err != nil {
		t.Fatalf("update third sort_order: %v", err)
	}

	files, err := NewGameFilesRepository(db).ListByGameID(gameID)
	if err != nil {
		t.Fatalf("ListByGameID returned error: %v", err)
	}
	if len(files) != 3 {
		t.Fatalf("len(files) = %d, want 3", len(files))
	}
	if files[0].ID != secondID || files[1].ID != thirdID || files[2].ID != firstID {
		t.Fatalf("files order = %+v, want sort_order asc then id asc", files)
	}
}
