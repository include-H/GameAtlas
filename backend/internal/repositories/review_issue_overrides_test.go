package repositories

import "testing"

func TestReviewIssueOverrideRepositoryListFiltersAndSortsResults(t *testing.T) {
	db := openRepositoryTagsTestDB(t)
	defer func() { _ = db.Close() }()

	repo := NewReviewIssueOverrideRepository(db)
	firstGameID := insertRepositoryGame(t, db, "override-a", "Override A", "public")
	secondGameID := insertRepositoryGame(t, db, "override-b", "Override B", "public")

	if _, err := repo.Upsert(firstGameID, "missing-summary", "ignored", stringPtr("alpha")); err != nil {
		t.Fatalf("first Upsert returned error: %v", err)
	}
	if _, err := repo.Upsert(firstGameID, "missing-cover", "ignored", stringPtr("beta")); err != nil {
		t.Fatalf("second Upsert returned error: %v", err)
	}
	if _, err := repo.Upsert(secondGameID, "missing-banner", "ignored", nil); err != nil {
		t.Fatalf("third Upsert returned error: %v", err)
	}

	all, err := repo.List(nil)
	if err != nil {
		t.Fatalf("List(nil) returned error: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("len(all) = %d, want 3", len(all))
	}
	if all[0].GameID != firstGameID || all[0].IssueKey != "missing-cover" {
		t.Fatalf("all[0] = %+v, want first game sorted by issue key", all[0])
	}
	if all[1].GameID != firstGameID || all[1].IssueKey != "missing-summary" {
		t.Fatalf("all[1] = %+v, want second first-game override", all[1])
	}
	if all[2].GameID != secondGameID || all[2].IssueKey != "missing-banner" {
		t.Fatalf("all[2] = %+v, want second game override", all[2])
	}

	filtered, err := repo.List([]int64{secondGameID})
	if err != nil {
		t.Fatalf("List(filtered) returned error: %v", err)
	}
	if len(filtered) != 1 || filtered[0].GameID != secondGameID {
		t.Fatalf("filtered = %+v, want only second game override", filtered)
	}
}

func TestReviewIssueOverrideRepositoryUpsertUpdatesExistingRow(t *testing.T) {
	db := openRepositoryTagsTestDB(t)
	defer func() { _ = db.Close() }()

	repo := NewReviewIssueOverrideRepository(db)
	gameID := insertRepositoryGame(t, db, "override-update", "Override Update", "public")

	created, err := repo.Upsert(gameID, "missing-cover", "ignored", stringPtr("first"))
	if err != nil {
		t.Fatalf("first Upsert returned error: %v", err)
	}

	updated, err := repo.Upsert(gameID, "missing-cover", "resolved", stringPtr("second"))
	if err != nil {
		t.Fatalf("second Upsert returned error: %v", err)
	}

	if updated.ID != created.ID {
		t.Fatalf("updated.ID = %d, want existing id %d", updated.ID, created.ID)
	}
	if updated.Status != "resolved" {
		t.Fatalf("updated.Status = %q, want resolved", updated.Status)
	}
	if updated.Reason == nil || *updated.Reason != "second" {
		t.Fatalf("updated.Reason = %v, want second", updated.Reason)
	}

	items, err := repo.List([]int64{gameID})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].ID != created.ID || items[0].Status != "resolved" {
		t.Fatalf("items[0] = %+v, want updated single row", items[0])
	}
}

func TestReviewIssueOverrideRepositoryListTreatsEmptySliceAsUnfiltered(t *testing.T) {
	db := openRepositoryTagsTestDB(t)
	defer func() { _ = db.Close() }()

	repo := NewReviewIssueOverrideRepository(db)
	firstGameID := insertRepositoryGame(t, db, "override-empty-a", "Override Empty A", "public")
	secondGameID := insertRepositoryGame(t, db, "override-empty-b", "Override Empty B", "public")

	if _, err := repo.Upsert(firstGameID, "missing-summary", "ignored", nil); err != nil {
		t.Fatalf("first Upsert returned error: %v", err)
	}
	if _, err := repo.Upsert(secondGameID, "missing-cover", "ignored", nil); err != nil {
		t.Fatalf("second Upsert returned error: %v", err)
	}

	items, err := repo.List([]int64{})
	if err != nil {
		t.Fatalf("List(empty slice) returned error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
}

func stringPtr(value string) *string {
	return &value
}
