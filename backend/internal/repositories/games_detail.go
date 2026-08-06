package repositories

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

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
		SELECT id, game_id, asset_uid, asset_type, path, poster_path, sort_order, position_x, position_y, width_pct, created_at
		FROM game_assets
		WHERE game_id = ?
		ORDER BY asset_type ASC, sort_order ASC, id ASC
	`, gameID)
	if err != nil {
		return nil, fmt.Errorf("list all assets: %w", err)
	}
	return assets, nil
}

// videoAssetRow 是批量预告片查询的行映射：在 GameAsset 上附带游戏可见性，
// 供 service 层在返回前过滤私有游戏，避免把私有游戏的存在性泄露给匿名调用者。
type videoAssetRow struct {
	PublicID   string `db:"public_id"`
	Visibility string `db:"visibility"`
	domain.GameAsset
}

// ListVideosByPublicIDs 按 public_id 批量返回 video 资产，按 public_id 与
// sort_order 排序，供游戏店会话一次拉取多个游戏的预告片。
func (r *GamesRepository) ListVideosByPublicIDs(publicIDs []string) ([]videoAssetRow, error) {
	normalized := make([]string, 0, len(publicIDs))
	for _, id := range publicIDs {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, strings.ToLower(trimmed))
	}
	if len(normalized) == 0 {
		return nil, nil
	}

	query, boundArgs, err := sqlx.In(`
		SELECT g.public_id, g.visibility,
			ga.id, ga.game_id, ga.asset_uid, ga.asset_type, ga.path, ga.poster_path,
			ga.sort_order, ga.position_x, ga.position_y, ga.width_pct, ga.created_at
		FROM game_assets ga
		INNER JOIN games g ON g.id = ga.game_id
		WHERE ga.asset_type = 'video' AND g.public_id IN (?)
		ORDER BY g.public_id, ga.sort_order, ga.id
	`, normalized)
	if err != nil {
		return nil, fmt.Errorf("build videos by public ids query: %w", err)
	}
	query = r.db.Rebind(query)

	var rows []videoAssetRow
	if err := r.db.Select(&rows, query, boundArgs...); err != nil {
		return nil, fmt.Errorf("list videos by public ids: %w", err)
	}

	return rows, nil
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

func (r *GamesRepository) ListMetadata(typ domain.MetadataType, gameID int64) ([]domain.MetadataItem, error) {
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
