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

func (r *AssetsRepository) AddScreenshot(gameID int64, path string, sortOrder int) (*domain.GameAsset, error) {
	var asset domain.GameAsset
	err := r.db.Get(&asset, `
		INSERT INTO game_assets (game_id, asset_type, path, sort_order)
		VALUES (?, 'screenshot', ?, ?)
		RETURNING id, game_id, asset_type, path, sort_order, created_at
	`, gameID, path, sortOrder)
	if err != nil {
		return nil, fmt.Errorf("insert screenshot asset: %w", err)
	}
	return &asset, nil
}

func (r *AssetsRepository) DeleteScreenshot(gameID int64, path string) (bool, error) {
	result, err := r.db.Exec(`
		DELETE FROM game_assets
		WHERE game_id = ? AND asset_type = 'screenshot' AND path = ?
	`, gameID, path)
	if err != nil {
		return false, fmt.Errorf("delete screenshot asset: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read deleted screenshot rows: %w", err)
	}
	return rows > 0, nil
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
