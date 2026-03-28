package repositories

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/jmoiron/sqlx"
)

func TestAssetsRepositoryUpdateVideoSortOrdersTrimsUIDsAndCommits(t *testing.T) {
	db := openRepositoryTagsTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertRepositoryAssetGame(t, db, "asset-reorder", "Asset Reorder")
	insertRepositoryAsset(t, db, gameID, "video-a", "video", "/assets/asset-reorder/video-a.mp4", 5)
	insertRepositoryAsset(t, db, gameID, "video-b", "video", "/assets/asset-reorder/video-b.mp4", 6)

	repo := NewAssetsRepository(db)
	if err := repo.UpdateVideoSortOrders(gameID, []string{" video-b ", " video-a "}); err != nil {
		t.Fatalf("UpdateVideoSortOrders returned error: %v", err)
	}

	if got := repositoryAssetSortOrder(t, db, "video-b"); got != 0 {
		t.Fatalf("video-b sort_order = %d, want 0", got)
	}
	if got := repositoryAssetSortOrder(t, db, "video-a"); got != 1 {
		t.Fatalf("video-a sort_order = %d, want 1", got)
	}
}

func TestAssetsRepositoryUpdateVideoSortOrdersRollsBackOnMissingAsset(t *testing.T) {
	db := openRepositoryTagsTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertRepositoryAssetGame(t, db, "asset-rollback", "Asset Rollback")
	insertRepositoryAsset(t, db, gameID, "video-a", "video", "/assets/asset-rollback/video-a.mp4", 5)
	insertRepositoryAsset(t, db, gameID, "video-b", "video", "/assets/asset-rollback/video-b.mp4", 6)

	repo := NewAssetsRepository(db)
	err := repo.UpdateVideoSortOrders(gameID, []string{"video-a", "missing"})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("UpdateVideoSortOrders error = %v, want sql.ErrNoRows", err)
	}

	if got := repositoryAssetSortOrder(t, db, "video-a"); got != 5 {
		t.Fatalf("video-a sort_order = %d, want rollback to 5", got)
	}
	if got := repositoryAssetSortOrder(t, db, "video-b"); got != 6 {
		t.Fatalf("video-b sort_order = %d, want rollback to 6", got)
	}
}

func insertRepositoryAssetGame(t *testing.T, db *sqlx.DB, publicID string, title string) int64 {
	t.Helper()

	result, err := db.Exec(`
		INSERT INTO games (public_id, title, visibility)
		VALUES (?, ?, 'public')
	`, publicID, title)
	if err != nil {
		t.Fatalf("insert asset test game: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId returned error: %v", err)
	}

	return id
}

func insertRepositoryAsset(t *testing.T, db *sqlx.DB, gameID int64, assetUID string, assetType string, path string, sortOrder int) int64 {
	t.Helper()

	result, err := db.Exec(`
		INSERT INTO game_assets (game_id, asset_uid, asset_type, path, sort_order)
		VALUES (?, ?, ?, ?, ?)
	`, gameID, assetUID, assetType, path, sortOrder)
	if err != nil {
		t.Fatalf("insert asset: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId returned error: %v", err)
	}

	return id
}

func repositoryAssetSortOrder(t *testing.T, db *sqlx.DB, assetUID string) int {
	t.Helper()

	var sortOrder int
	if err := db.Get(&sortOrder, `
		SELECT sort_order
		FROM game_assets
		WHERE asset_uid = ?
	`, assetUID); err != nil {
		t.Fatalf("load asset sort_order: %v", err)
	}

	return sortOrder
}
