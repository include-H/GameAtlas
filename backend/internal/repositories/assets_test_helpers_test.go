package repositories

import (
	"testing"

	"github.com/jmoiron/sqlx"
)

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
