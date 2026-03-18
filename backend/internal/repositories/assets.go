package repositories

import (
	"database/sql"
	"fmt"
	"strings"

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

func (r *AssetsRepository) UpdateGamePreviewVideoAssetUID(gameID int64, assetUID *string) error {
	result, err := r.db.Exec(`
		UPDATE games
		SET preview_video_asset_uid = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, assetUID, gameID)
	if err != nil {
		return fmt.Errorf("update preview video asset uid: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read preview video update rows: %w", err)
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *AssetsRepository) DeleteAssetByPath(gameID int64, assetType string, path string) (bool, error) {
	result, err := r.db.Exec(`
		DELETE FROM game_assets
		WHERE game_id = ? AND asset_type = ? AND path = ?
	`, gameID, assetType, path)
	if err != nil {
		return false, fmt.Errorf("delete %s asset: %w", assetType, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read deleted %s asset rows: %w", assetType, err)
	}
	return rows > 0, nil
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

func (r *AssetsRepository) GetScreenshotByID(gameID, assetID int64) (*domain.GameAsset, error) {
	var asset domain.GameAsset
	err := r.db.Get(&asset, `
		SELECT id, game_id, asset_uid, asset_type, path, sort_order, created_at
		FROM game_assets
		WHERE id = ? AND game_id = ? AND asset_type = 'screenshot'
	`, assetID, gameID)
	if err != nil {
		return nil, err
	}
	return &asset, nil
}

func (r *AssetsRepository) GetScreenshotByUID(gameID int64, assetUID string) (*domain.GameAsset, error) {
	var asset domain.GameAsset
	err := r.db.Get(&asset, `
		SELECT id, game_id, asset_uid, asset_type, path, sort_order, created_at
		FROM game_assets
		WHERE asset_uid = ? AND game_id = ? AND asset_type = 'screenshot'
	`, assetUID, gameID)
	if err != nil {
		return nil, err
	}
	return &asset, nil
}

func (r *AssetsRepository) GetAssetByUID(gameID int64, assetUID string, assetType string) (*domain.GameAsset, error) {
	var asset domain.GameAsset
	err := r.db.Get(&asset, `
		SELECT id, game_id, asset_uid, asset_type, path, sort_order, created_at
		FROM game_assets
		WHERE asset_uid = ? AND game_id = ? AND asset_type = ?
	`, assetUID, gameID, assetType)
	if err != nil {
		return nil, err
	}
	return &asset, nil
}

func (r *AssetsRepository) DeleteScreenshotByID(gameID, assetID int64) (*domain.GameAsset, error) {
	asset, err := r.GetScreenshotByID(gameID, assetID)
	if err != nil {
		return nil, err
	}

	result, err := r.db.Exec(`
		DELETE FROM game_assets
		WHERE id = ? AND game_id = ? AND asset_type = 'screenshot'
	`, assetID, gameID)
	if err != nil {
		return nil, fmt.Errorf("delete screenshot asset by id: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("read deleted screenshot rows by id: %w", err)
	}
	if rows == 0 {
		return nil, sql.ErrNoRows
	}

	return asset, nil
}

func (r *AssetsRepository) DeleteScreenshotByUID(gameID int64, assetUID string) (*domain.GameAsset, error) {
	asset, err := r.GetScreenshotByUID(gameID, assetUID)
	if err != nil {
		return nil, err
	}

	result, err := r.db.Exec(`
		DELETE FROM game_assets
		WHERE asset_uid = ? AND game_id = ? AND asset_type = 'screenshot'
	`, assetUID, gameID)
	if err != nil {
		return nil, fmt.Errorf("delete screenshot asset by uid: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("read deleted screenshot rows by uid: %w", err)
	}
	if rows == 0 {
		return nil, sql.ErrNoRows
	}

	return asset, nil
}

func (r *AssetsRepository) DeleteAssetByUID(gameID int64, assetUID string, assetType string) (*domain.GameAsset, error) {
	asset, err := r.GetAssetByUID(gameID, assetUID, assetType)
	if err != nil {
		return nil, err
	}

	result, err := r.db.Exec(`
		DELETE FROM game_assets
		WHERE asset_uid = ? AND game_id = ? AND asset_type = ?
	`, assetUID, gameID, assetType)
	if err != nil {
		return nil, fmt.Errorf("delete %s asset by uid: %w", assetType, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("read deleted %s asset rows by uid: %w", assetType, err)
	}
	if rows == 0 {
		return nil, sql.ErrNoRows
	}

	return asset, nil
}

func (r *AssetsRepository) UpdateScreenshotSortOrders(gameID int64, assetUIDs []string) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return fmt.Errorf("begin screenshot reorder tx: %w", err)
	}
	defer tx.Rollback()

	for index, assetUID := range assetUIDs {
		trimmed := strings.TrimSpace(assetUID)
		if trimmed == "" {
			return fmt.Errorf("empty screenshot asset uid")
		}

		result, err := tx.Exec(`
			UPDATE game_assets
			SET sort_order = ?
			WHERE game_id = ? AND asset_type = 'screenshot' AND asset_uid = ?
		`, index, gameID, trimmed)
		if err != nil {
			return fmt.Errorf("update screenshot sort order: %w", err)
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("read screenshot reorder rows: %w", err)
		}
		if rows == 0 {
			return sql.ErrNoRows
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit screenshot reorder tx: %w", err)
	}
	return nil
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
