package repositories

import (
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/hao/game/internal/domain"
)

type MetadataRepository struct {
	db *sqlx.DB
}

func NewMetadataRepository(db *sqlx.DB) *MetadataRepository {
	return &MetadataRepository{db: db}
}

func (r *MetadataRepository) List(table string) ([]domain.MetadataItem, error) {
	query := fmt.Sprintf(`
		SELECT id, name, slug, sort_order, created_at
		FROM %s
		ORDER BY sort_order ASC, id ASC
	`, table)

	var items []domain.MetadataItem
	if err := r.db.Select(&items, query); err != nil {
		return nil, fmt.Errorf("list metadata from %s: %w", table, err)
	}

	return items, nil
}

func (r *MetadataRepository) CreateSeries(input domain.MetadataWriteInput, slug string, sortOrder int) (*domain.MetadataItem, error) {
	var item domain.MetadataItem
	err := r.db.Get(&item, `
		INSERT INTO series (name, slug, sort_order)
		VALUES (?, ?, ?)
		RETURNING id, name, slug, sort_order, created_at
	`, input.Name, slug, sortOrder)
	if err != nil {
		return nil, fmt.Errorf("create series: %w", err)
	}
	return &item, nil
}

func (r *MetadataRepository) CreateSimple(table string, input domain.MetadataWriteInput, slug string, sortOrder int) (*domain.MetadataItem, error) {
	query := fmt.Sprintf(`
		INSERT INTO %s (name, slug, sort_order)
		VALUES (?, ?, ?)
		RETURNING id, name, slug, sort_order, created_at
	`, table)

	var item domain.MetadataItem
	if err := r.db.Get(&item, query, input.Name, slug, sortOrder); err != nil {
		return nil, fmt.Errorf("create metadata in %s: %w", table, err)
	}
	return &item, nil
}

func (r *MetadataRepository) ListSeriesGames(seriesID int64) ([]domain.Game, error) {
	var games []domain.Game
	if err := r.db.Select(&games, `
		SELECT
			g.id,
			g.title,
			g.title_alt,
			g.visibility,
			g.summary,
			g.release_date,
			g.engine,
			g.cover_image,
			g.banner_image,
			g.wiki_content,
			g.wiki_content_html,
			g.needs_review,
			g.preview_video_asset_uid,
			g.views,
			g.downloads,
			(
				SELECT ga.path
				FROM game_assets ga
				WHERE ga.game_id = g.id AND ga.asset_type = 'screenshot'
				ORDER BY ga.sort_order ASC, ga.id ASC
				LIMIT 1
			) AS primary_screenshot,
			0 AS screenshot_count,
			0 AS file_count,
			0 AS developer_count,
			0 AS publisher_count,
			0 AS platform_count,
			g.created_at,
			g.updated_at
		FROM games g
		INNER JOIN game_series gs ON gs.game_id = g.id
		WHERE gs.series_id = ?
		ORDER BY g.updated_at DESC, gs.sort_order ASC, g.id DESC
	`, seriesID); err != nil {
		return nil, fmt.Errorf("list series games: %w", err)
	}

	return games, nil
}

func (r *MetadataRepository) DeleteUnused(table, joinTable, joinColumn string) error {
	query := fmt.Sprintf(`
		DELETE FROM %s
		WHERE id NOT IN (
			SELECT DISTINCT %s
			FROM %s
		)
	`, table, joinColumn, joinTable)

	if _, err := r.db.Exec(query); err != nil {
		return fmt.Errorf("delete unused metadata from %s: %w", table, err)
	}

	return nil
}
