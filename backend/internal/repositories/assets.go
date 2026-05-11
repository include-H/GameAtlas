package repositories

import (
	"database/sql"
	"errors"
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

func (r *AssetsRepository) AddCover(gameID int64, assetUID string, path string, sortOrder int) (*domain.GameAsset, error) {
	return r.addAsset(gameID, assetUID, "cover", path, sortOrder)
}

func (r *AssetsRepository) AddLogo(gameID int64, assetUID string, path string, sortOrder int) (*domain.GameAsset, error) {
	return r.addAsset(gameID, assetUID, "logo", path, sortOrder)
}

func (r *AssetsRepository) AddBanner(gameID int64, assetUID string, path string, sortOrder int) (*domain.GameAsset, error) {
	return r.addAsset(gameID, assetUID, "banner", path, sortOrder)
}

func (r *AssetsRepository) UpdateLogoPosition(gameID int64, assetUID string, posX, posY, widthPct *float64) error {
	_, err := r.db.Exec(`
		UPDATE game_assets
		SET position_x = ?, position_y = ?, width_pct = ?
		WHERE game_id = ? AND asset_type = 'logo' AND asset_uid = ?
	`, posX, posY, widthPct, gameID, assetUID)
	if err != nil {
		return fmt.Errorf("update logo position: %w", err)
	}
	return nil
}

func (r *AssetsRepository) DeleteByUID(gameID int64, assetType string, assetUID string) (string, error) {
	var path string
	err := r.db.Get(&path, `
		DELETE FROM game_assets
		WHERE game_id = ? AND asset_type = ? AND asset_uid = ?
		RETURNING path
	`, gameID, assetType, assetUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("delete %s by uid: %w", assetType, err)
	}
	return path, nil
}
