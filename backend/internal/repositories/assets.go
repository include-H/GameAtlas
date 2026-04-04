package repositories

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/hao/game/internal/domain"
)

type AssetsRepository struct {
	db *sqlx.DB
}

func NewAssetsRepository(db *sqlx.DB) *AssetsRepository {
	return &AssetsRepository{db: db}
}

func (r *AssetsRepository) addAsset(gameID int64, assetUID string, assetType string, path string, sortOrder int) (*domain.GameAsset, error) {
	var asset domain.GameAsset
	err := r.db.Get(&asset, `
		INSERT INTO game_assets (game_id, asset_uid, asset_type, path, sort_order)
		VALUES (?, ?, ?, ?, ?)
		RETURNING id, game_id, asset_uid, asset_type, path, sort_order, created_at
	`, gameID, assetUID, assetType, path, sortOrder)
	if err != nil {
		return nil, fmt.Errorf("insert %s asset: %w", assetType, err)
	}
	return &asset, nil
}

func (r *AssetsRepository) AddScreenshot(gameID int64, assetUID string, path string, sortOrder int) (*domain.GameAsset, error) {
	return r.addAsset(gameID, assetUID, "screenshot", path, sortOrder)
}

func (r *AssetsRepository) AddVideo(gameID int64, assetUID string, path string, sortOrder int) (*domain.GameAsset, error) {
	return r.addAsset(gameID, assetUID, "video", path, sortOrder)
}

func (r *AssetsRepository) UpdateGameImage(gameID int64, column string, path *string) error {
	query := fmt.Sprintf("UPDATE games SET %s = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", column)
	result, err := r.db.Exec(query, path, gameID)
	if err != nil {
		return fmt.Errorf("update game image column %s: %w", column, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read updated image rows: %w", err)
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}
