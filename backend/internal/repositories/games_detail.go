package repositories

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/hao/game/internal/domain"
)

func (r *GamesRepository) GetByID(id int64) (*domain.Game, error) {
	const query = `
		SELECT
			id,
			public_id,
			title,
			title_alt,
			visibility,
			summary,
			release_date,
			cover_image,
			banner_image,
			wiki_content,
			downloads,
			NULL AS primary_screenshot,
			logo_visible,
			CASE WHEN EXISTS (SELECT 1 FROM favorite_games fg WHERE fg.game_id = games.id) THEN 1 ELSE 0 END AS is_favorite,
			created_at,
			updated_at
		FROM games
		WHERE id = ?`

	var game domain.Game
	if err := r.db.Get(&game, query, id); err != nil {
		return nil, err
	}

	return &game, nil
}

func (r *GamesRepository) GetByPublicID(publicID string) (*domain.Game, error) {
	const query = `
		SELECT
			id,
			public_id,
			title,
			title_alt,
			visibility,
			summary,
			release_date,
			cover_image,
			banner_image,
			wiki_content,
			downloads,
			NULL AS primary_screenshot,
			logo_visible,
			CASE WHEN EXISTS (SELECT 1 FROM favorite_games fg WHERE fg.game_id = games.id) THEN 1 ELSE 0 END AS is_favorite,
			created_at,
			updated_at
		FROM games
		WHERE public_id = ?`

	var game domain.Game
	if err := r.db.Get(&game, query, strings.ToLower(strings.TrimSpace(publicID))); err != nil {
		return nil, err
	}

	return &game, nil
}

func (r *GamesRepository) ResolveIDByPublicID(publicID string) (int64, error) {
	trimmed := strings.TrimSpace(publicID)
	if trimmed == "" {
		return 0, sql.ErrNoRows
	}

	var id int64
	if err := r.db.Get(&id, "SELECT id FROM games WHERE public_id = ? LIMIT 1", strings.ToLower(trimmed)); err != nil {
		return 0, err
	}
	return id, nil
}

func (r *GamesRepository) IncrementDownloads(id int64) error {
	_, err := r.db.Exec(`
		UPDATE games
		SET downloads = downloads + 1
		WHERE id = ?
	`, id)
	if err != nil {
		return fmt.Errorf("increment game downloads: %w", err)
	}
	return nil
}

func (r *GamesRepository) ListAllAssets(gameID int64) ([]domain.GameAsset, error) {
	var assets []domain.GameAsset
	err := r.db.Select(&assets, `
		SELECT id, game_id, asset_uid, asset_type, path, sort_order, position_x, position_y, width_pct, created_at
		FROM game_assets
		WHERE game_id = ?
		ORDER BY asset_type ASC, sort_order ASC, id ASC
	`, gameID)
	if err != nil {
		return nil, fmt.Errorf("list all assets: %w", err)
	}
	return assets, nil
}

func (r *GamesRepository) listAssetsByType(gameID int64, assetType string) ([]domain.GameAsset, error) {
	var assets []domain.GameAsset
	err := r.db.Select(&assets, `
		SELECT id, game_id, asset_uid, asset_type, path, sort_order, position_x, position_y, width_pct, created_at
		FROM game_assets
		WHERE game_id = ? AND asset_type = ?
		ORDER BY sort_order ASC, id ASC
	`, gameID, assetType)
	if err != nil {
		return nil, fmt.Errorf("list %s assets: %w", assetType, err)
	}

	return assets, nil
}

func (r *GamesRepository) ListScreenshots(gameID int64) ([]domain.GameAsset, error) {
	return r.listAssetsByType(gameID, "screenshot")
}

func (r *GamesRepository) ListVideos(gameID int64) ([]domain.GameAsset, error) {
	return r.listAssetsByType(gameID, "video")
}

func (r *GamesRepository) ListCovers(gameID int64) ([]domain.GameAsset, error) {
	return r.listAssetsByType(gameID, "cover")
}

func (r *GamesRepository) ListLogos(gameID int64) ([]domain.GameAsset, error) {
	return r.listAssetsByType(gameID, "logo")
}

func (r *GamesRepository) ListBanners(gameID int64) ([]domain.GameAsset, error) {
	return r.listAssetsByType(gameID, "banner")
}

func (r *GamesRepository) GetSeriesMetadata(gameID int64) (*domain.MetadataItem, error) {
	const query = `
		SELECT s.id, s.name, s.slug, s.sort_order, s.created_at
		FROM games g
		INNER JOIN series s ON s.id = g.series_id
		WHERE g.id = ?
		LIMIT 1
	`

	var item domain.MetadataItem
	if err := r.db.Get(&item, query, gameID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get series metadata: %w", err)
	}

	return &item, nil
}

func (r *GamesRepository) ListMetadata(typ MetadataType, gameID int64) ([]domain.MetadataItem, error) {
	table := metadataTableName(typ)
	joinTable := metadataJoinTable(typ)
	joinColumn := metadataJoinColumn(typ)
	query := fmt.Sprintf(`
		SELECT m.id, m.name, m.slug, m.sort_order, m.created_at
		FROM %s m
		INNER JOIN %s gm ON gm.%s = m.id
		WHERE gm.game_id = ?
		ORDER BY gm.sort_order ASC, m.sort_order ASC, m.id ASC
	`, table, joinTable, joinColumn)

	var items []domain.MetadataItem
	if err := r.db.Select(&items, query, gameID); err != nil {
		return nil, fmt.Errorf("list metadata from %s: %w", table, err)
	}

	return items, nil
}
