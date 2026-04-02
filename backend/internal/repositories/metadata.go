package repositories

import (
	"database/sql"
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

func (r *MetadataRepository) Get(table string, id int64) (*domain.MetadataItem, error) {
	query := fmt.Sprintf(`
		SELECT id, name, slug, sort_order, created_at
		FROM %s
		WHERE id = ?
	`, table)

	var item domain.MetadataItem
	if err := r.db.Get(&item, query, id); err != nil {
		return nil, err
	}

	return &item, nil
}

func (r *MetadataRepository) FindSimpleByName(table string, name string) (*domain.MetadataItem, error) {
	query := fmt.Sprintf(`
		SELECT id, name, slug, sort_order, created_at
		FROM %s
		WHERE lower(trim(name)) = lower(trim(?))
		LIMIT 1
	`, table)

	var item domain.MetadataItem
	if err := r.db.Get(&item, query, name); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find metadata in %s by name: %w", table, err)
	}

	return &item, nil
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

func (r *MetadataRepository) ListSeriesGames(seriesID int64, includeAll bool) ([]domain.Game, error) {
	where := "WHERE g.series_id = ?"
	args := []any{seriesID}
	if !includeAll {
		where += " AND g.visibility = ?"
		args = append(args, domain.GameVisibilityPublic)
	}

	var games []domain.Game
	query := fmt.Sprintf(`
		SELECT
			g.id,
			g.public_id,
			g.title,
			g.title_alt,
			g.visibility,
			g.summary,
			g.release_date,
			g.engine,
			g.cover_image,
			g.banner_image,
			g.wiki_content,
			g.needs_review,
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
		%s
		ORDER BY g.updated_at DESC, g.id DESC
	`, where)

	if err := r.db.Select(&games, query, args...); err != nil {
		return nil, fmt.Errorf("list series games: %w", err)
	}

	return games, nil
}

func (r *MetadataRepository) ListSeriesGamesBySeriesIDs(seriesIDs []int64, includeAll bool) (map[int64][]domain.Game, error) {
	normalized := uniquePositiveIDs(seriesIDs)
	if len(normalized) == 0 {
		return map[int64][]domain.Game{}, nil
	}

	where := "WHERE g.series_id IN (?)"
	args := []any{normalized}
	if !includeAll {
		where += " AND g.visibility = ?"
		args = append(args, domain.GameVisibilityPublic)
	}

	query, boundArgs, err := sqlx.In(fmt.Sprintf(`
		SELECT
			g.series_id,
			g.id,
			g.public_id,
			g.title,
			g.title_alt,
			g.visibility,
			g.summary,
			g.release_date,
			g.engine,
			g.cover_image,
			g.banner_image,
			g.wiki_content,
			g.needs_review,
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
		%s
		ORDER BY g.series_id ASC, g.updated_at DESC, g.id DESC
	`, where), args...)
	if err != nil {
		return nil, fmt.Errorf("build series games by ids query: %w", err)
	}
	query = r.db.Rebind(query)

	type seriesGameRow struct {
		SeriesID int64 `db:"series_id"`
		domain.Game
	}

	var rows []seriesGameRow
	if err := r.db.Select(&rows, query, boundArgs...); err != nil {
		return nil, fmt.Errorf("list series games by ids: %w", err)
	}

	gamesBySeriesID := make(map[int64][]domain.Game, len(normalized))
	for _, seriesID := range normalized {
		gamesBySeriesID[seriesID] = []domain.Game{}
	}
	for _, row := range rows {
		gamesBySeriesID[row.SeriesID] = append(gamesBySeriesID[row.SeriesID], row.Game)
	}

	return gamesBySeriesID, nil
}

func (r *MetadataRepository) DeleteUnusedSeries() error {
	const query = `
		DELETE FROM series
		WHERE id NOT IN (
			SELECT DISTINCT series_id
			FROM games
			WHERE series_id IS NOT NULL
		)
	`

	if _, err := r.db.Exec(query); err != nil {
		return fmt.Errorf("delete unused series: %w", err)
	}

	return nil
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
