package repositories

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/hao/game/internal/domain"
)

func metadataTableName(typ domain.MetadataType) string {
	switch typ {
	case domain.MetadataDevelopers:
		return "developers"
	case domain.MetadataPublishers:
		return "publishers"
	case domain.MetadataSeries:
		return "series"
	default:
		return ""
	}
}

func metadataJoinTable(typ domain.MetadataType) string {
	switch typ {
	case domain.MetadataDevelopers:
		return "game_developers"
	case domain.MetadataPublishers:
		return "game_publishers"
	case domain.MetadataSeries:
		return "game_series"
	default:
		return ""
	}
}

func metadataJoinColumn(typ domain.MetadataType) string {
	switch typ {
	case domain.MetadataDevelopers:
		return "developer_id"
	case domain.MetadataPublishers:
		return "publisher_id"
	case domain.MetadataSeries:
		return "series_id"
	default:
		return ""
	}
}

type MetadataRepository struct {
	db *sqlx.DB
}

func NewMetadataRepository(db *sqlx.DB) *MetadataRepository {
	return &MetadataRepository{db: db}
}

func (r *MetadataRepository) List(typ domain.MetadataType) ([]domain.MetadataItem, error) {
	table := metadataTableName(typ)
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

func (r *MetadataRepository) Get(typ domain.MetadataType, id int64) (*domain.MetadataItem, error) {
	table := metadataTableName(typ)
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

func (r *MetadataRepository) FindSimpleByName(typ domain.MetadataType, name string) (*domain.MetadataItem, error) {
	table := metadataTableName(typ)
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

func (r *MetadataRepository) FindSimpleBySlug(typ domain.MetadataType, slug string) (*domain.MetadataItem, error) {
	table := metadataTableName(typ)
	query := fmt.Sprintf(`
		SELECT id, name, slug, sort_order, created_at
		FROM %s
		WHERE slug = ?
		LIMIT 1
	`, table)

	var item domain.MetadataItem
	if err := r.db.Get(&item, query, slug); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find metadata in %s by slug: %w", table, err)
	}

	return &item, nil
}

func (r *MetadataRepository) CreateOrGet(
	typ domain.MetadataType,
	input domain.MetadataWriteInput,
	slug string,
	sortOrder int,
) (*domain.MetadataItem, error) {
	table := metadataTableName(typ)
	if table == "" {
		return nil, fmt.Errorf("unsupported metadata resource type: %d", typ)
	}

	var item domain.MetadataItem
	query := fmt.Sprintf(`
		INSERT INTO %s (name, slug, sort_order)
		VALUES (?, ?, ?)
		ON CONFLICT DO NOTHING
		RETURNING id, name, slug, sort_order, created_at
	`, table)
	err := r.db.Get(&item, query, input.Name, slug, sortOrder)
	if err == nil {
		return &item, nil
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("create metadata in %s: %w", table, err)
	}

	existing, err := r.FindSimpleByName(typ, input.Name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}
	existing, err = r.FindSimpleBySlug(typ, slug)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}
	return nil, fmt.Errorf("metadata insert conflicted but no existing %s item was found", table)
}

type metadataGamesRelation struct {
	name        string
	join        string
	groupColumn string
}

func metadataGamesRelationFor(typ domain.MetadataType) (metadataGamesRelation, bool) {
	switch typ {
	case domain.MetadataSeries:
		return metadataGamesRelation{
			name:        "series",
			groupColumn: "g.series_id",
		}, true
	case domain.MetadataPublishers:
		return metadataGamesRelation{
			name:        "publisher",
			join:        "INNER JOIN game_publishers gp ON gp.game_id = g.id",
			groupColumn: "gp.publisher_id",
		}, true
	default:
		return metadataGamesRelation{}, false
	}
}

func metadataGameSummarySelect() string {
	return `
		g.id,
		g.public_id,
		g.title,
		g.title_alt,
		g.visibility,
		g.summary,
		g.release_date,
		g.cover_image,
		g.banner_image,
		g.wiki_content,
		g.downloads,
		(
			SELECT ga.path
			FROM game_assets ga
			WHERE ga.game_id = g.id AND ga.asset_type = 'screenshot'
			ORDER BY ga.sort_order ASC, ga.id ASC
			LIMIT 1
		) AS primary_screenshot,
		CASE WHEN EXISTS (SELECT 1 FROM favorite_games fg WHERE fg.game_id = g.id) THEN 1 ELSE 0 END AS is_favorite,
		g.series_id,
		s.name AS series_name,
		g.created_at,
		g.updated_at`
}

func (r *MetadataRepository) ListMetadataGames(typ domain.MetadataType, metadataID int64, includeAll bool) ([]domain.MetadataGameSummary, error) {
	relation, ok := metadataGamesRelationFor(typ)
	if !ok {
		return nil, fmt.Errorf("unsupported metadata games type: %d", typ)
	}

	where := []string{relation.groupColumn + " = ?"}
	args := []any{metadataID}
	if !includeAll {
		where = append(where, "g.visibility = ?")
		args = append(args, domain.GameVisibilityPublic)
	}

	query := fmt.Sprintf(`
		SELECT %s
		FROM games g
		LEFT JOIN series s ON s.id = g.series_id
		%s
		WHERE %s
		ORDER BY g.updated_at DESC, g.id DESC
	`, metadataGameSummarySelect(), relation.join, strings.Join(where, " AND "))

	var games []domain.MetadataGameSummary
	if err := r.db.Select(&games, query, args...); err != nil {
		return nil, fmt.Errorf("list %s games: %w", relation.name, err)
	}

	return games, nil
}

func (r *MetadataRepository) ListMetadataGamesByIDs(typ domain.MetadataType, metadataIDs []int64, includeAll bool) (map[int64][]domain.MetadataGameSummary, error) {
	relation, ok := metadataGamesRelationFor(typ)
	if !ok {
		return nil, fmt.Errorf("unsupported metadata games type: %d", typ)
	}

	normalized := uniquePositiveIDs(metadataIDs)
	gamesByMetadataID := make(map[int64][]domain.MetadataGameSummary, len(normalized))
	if len(normalized) == 0 {
		return gamesByMetadataID, nil
	}
	for _, metadataID := range normalized {
		gamesByMetadataID[metadataID] = []domain.MetadataGameSummary{}
	}

	where := []string{relation.groupColumn + " IN (?)"}
	args := []any{normalized}
	if !includeAll {
		where = append(where, "g.visibility = ?")
		args = append(args, domain.GameVisibilityPublic)
	}

	query, boundArgs, err := sqlx.In(fmt.Sprintf(`
		SELECT
			%s AS group_metadata_id,
			%s
		FROM games g
		LEFT JOIN series s ON s.id = g.series_id
		%s
		WHERE %s
		ORDER BY %s ASC, g.updated_at DESC, g.id DESC
	`, relation.groupColumn, metadataGameSummarySelect(), relation.join, strings.Join(where, " AND "), relation.groupColumn), args...)
	if err != nil {
		return nil, fmt.Errorf("build %s games by ids query: %w", relation.name, err)
	}
	query = r.db.Rebind(query)

	type metadataGameRow struct {
		MetadataID int64 `db:"group_metadata_id"`
		domain.MetadataGameSummary
	}

	var rows []metadataGameRow
	if err := r.db.Select(&rows, query, boundArgs...); err != nil {
		return nil, fmt.Errorf("list %s games by ids: %w", relation.name, err)
	}

	for _, row := range rows {
		gamesByMetadataID[row.MetadataID] = append(gamesByMetadataID[row.MetadataID], row.MetadataGameSummary)
	}

	return gamesByMetadataID, nil
}

func (r *MetadataRepository) ListSeriesGames(seriesID int64, includeAll bool) ([]domain.MetadataGameSummary, error) {
	return r.ListMetadataGames(domain.MetadataSeries, seriesID, includeAll)
}

func (r *MetadataRepository) ListSeriesGamesBySeriesIDs(seriesIDs []int64, includeAll bool) (map[int64][]domain.MetadataGameSummary, error) {
	return r.ListMetadataGamesByIDs(domain.MetadataSeries, seriesIDs, includeAll)
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

func (r *MetadataRepository) DeleteUnused(typ domain.MetadataType) error {
	table := metadataTableName(typ)
	joinTable := metadataJoinTable(typ)
	joinColumn := metadataJoinColumn(typ)
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
