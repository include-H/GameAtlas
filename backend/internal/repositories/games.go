package repositories

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/hao/game/internal/domain"
)

var allowedGameSortFields = map[string]string{
	"title":      "g.title",
	"created_at": "g.created_at",
	"updated_at": "g.updated_at",
	"views":      "g.views",
	"downloads":  "g.downloads",
}

type GamesRepository struct {
	db *sqlx.DB
}

func NewGamesRepository(db *sqlx.DB) *GamesRepository {
	return &GamesRepository{db: db}
}

func (r *GamesRepository) List(params domain.GamesListParams) ([]domain.Game, int, error) {
	where := []string{"1 = 1"}
	args := map[string]any{}

	if params.Search != "" {
		where = append(where, "(g.title LIKE :search OR COALESCE(g.title_alt, '') LIKE :search)")
		args["search"] = "%" + params.Search + "%"
	}
	if params.NeedsReview != nil {
		where = append(where, "g.needs_review = :needs_review")
		if *params.NeedsReview {
			args["needs_review"] = 1
		} else {
			args["needs_review"] = 0
		}
	}
	if params.SeriesID > 0 {
		where = append(where, "EXISTS (SELECT 1 FROM game_series gs WHERE gs.game_id = g.id AND gs.series_id = :series_id)")
		args["series_id"] = params.SeriesID
	}
	if params.PlatformID > 0 {
		where = append(where, "EXISTS (SELECT 1 FROM game_platforms gp WHERE gp.game_id = g.id AND gp.platform_id = :platform_id)")
		args["platform_id"] = params.PlatformID
	}

	sortField := allowedGameSortFields[params.Sort]
	if sortField == "" {
		sortField = allowedGameSortFields["updated_at"]
	}
	order := strings.ToUpper(params.Order)
	if order != "ASC" && order != "DESC" {
		order = "DESC"
	}

	baseWhere := strings.Join(where, " AND ")

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM games g WHERE %s", baseWhere)
	countStmt, countArgs, err := sqlx.Named(countQuery, args)
	if err != nil {
		return nil, 0, fmt.Errorf("build games count query: %w", err)
	}
	countStmt = r.db.Rebind(countStmt)

	var total int
	if err := r.db.Get(&total, countStmt, countArgs...); err != nil {
		return nil, 0, fmt.Errorf("count games: %w", err)
	}

	offset := (params.Page - 1) * params.Limit
	args["limit"] = params.Limit
	args["offset"] = offset

	listQuery := fmt.Sprintf(`
		SELECT
			g.id,
			g.title,
			g.title_alt,
			g.summary,
			g.release_date,
			g.engine,
			g.cover_image,
			g.banner_image,
			g.wiki_content,
			g.wiki_content_html,
			g.needs_review,
			g.views,
			g.downloads,
			g.created_at,
			g.updated_at
		FROM games g
		WHERE %s
		ORDER BY %s %s, g.id DESC
		LIMIT :limit OFFSET :offset
	`, baseWhere, sortField, order)

	listStmt, listArgs, err := sqlx.Named(listQuery, args)
	if err != nil {
		return nil, 0, fmt.Errorf("build games list query: %w", err)
	}
	listStmt = r.db.Rebind(listStmt)

	var games []domain.Game
	if err := r.db.Select(&games, listStmt, listArgs...); err != nil {
		return nil, 0, fmt.Errorf("list games: %w", err)
	}

	return games, total, nil
}

func (r *GamesRepository) GetByID(id int64) (*domain.Game, error) {
	const query = `
		SELECT
			id,
			title,
			title_alt,
			summary,
			release_date,
			engine,
			cover_image,
			banner_image,
			wiki_content,
			wiki_content_html,
			needs_review,
			views,
			downloads,
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

func (r *GamesRepository) Create(input domain.GameWriteInput) (*domain.Game, error) {
	const query = `
		INSERT INTO games (
			title, title_alt, summary, release_date, engine, cover_image, banner_image, needs_review
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING
			id, title, title_alt, summary, release_date, engine, cover_image, banner_image,
			wiki_content, wiki_content_html, needs_review, views, downloads, created_at, updated_at`

	var game domain.Game
	if err := r.db.Get(
		&game,
		query,
		input.Title,
		input.TitleAlt,
		input.Summary,
		input.ReleaseDate,
		input.Engine,
		input.CoverImage,
		input.BannerImage,
		boolToInt(input.NeedsReview),
	); err != nil {
		return nil, fmt.Errorf("create game: %w", err)
	}

	if err := r.replaceRelations(game.ID, input); err != nil {
		return nil, err
	}

	return r.GetByID(game.ID)
}

func (r *GamesRepository) Update(id int64, input domain.GameWriteInput) (*domain.Game, error) {
	const query = `
		UPDATE games
		SET
			title = ?,
			title_alt = ?,
			summary = ?,
			release_date = ?,
			engine = ?,
			cover_image = ?,
			banner_image = ?,
			needs_review = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`

	result, err := r.db.Exec(
		query,
		input.Title,
		input.TitleAlt,
		input.Summary,
		input.ReleaseDate,
		input.Engine,
		input.CoverImage,
		input.BannerImage,
		boolToInt(input.NeedsReview),
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("update game: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("read updated rows: %w", err)
	}
	if rows == 0 {
		return nil, sql.ErrNoRows
	}

	if err := r.replaceRelations(id, input); err != nil {
		return nil, err
	}

	return r.GetByID(id)
}

func (r *GamesRepository) Delete(id int64) (bool, error) {
	result, err := r.db.Exec("DELETE FROM games WHERE id = ?", id)
	if err != nil {
		return false, fmt.Errorf("delete game: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read deleted rows: %w", err)
	}

	return rows > 0, nil
}

func (r *GamesRepository) IncrementDownloads(id int64) error {
	_, err := r.db.Exec(`
		UPDATE games
		SET downloads = downloads + 1, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, id)
	if err != nil {
		return fmt.Errorf("increment game downloads: %w", err)
	}
	return nil
}

func (r *GamesRepository) listAssetsByType(gameID int64, assetType string) ([]domain.GameAsset, error) {
	var assets []domain.GameAsset
	err := r.db.Select(&assets, `
		SELECT id, game_id, asset_uid, asset_type, path, sort_order, created_at
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

func (r *GamesRepository) ListMetadata(table, joinTable, joinColumn string, gameID int64) ([]domain.MetadataItem, error) {
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

func (r *GamesRepository) replaceRelations(gameID int64, input domain.GameWriteInput) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return fmt.Errorf("begin relations update: %w", err)
	}

	if err := replaceRelationRows(tx, "game_series", "series_id", gameID, input.SeriesIDs); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := replaceRelationRows(tx, "game_platforms", "platform_id", gameID, input.PlatformIDs); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := replaceRelationRows(tx, "game_developers", "developer_id", gameID, input.DeveloperIDs); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := replaceRelationRows(tx, "game_publishers", "publisher_id", gameID, input.PublisherIDs); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit relations update: %w", err)
	}

	return nil
}

func replaceRelationRows(tx *sqlx.Tx, table, column string, gameID int64, ids []int64) error {
	if _, err := tx.Exec(fmt.Sprintf("DELETE FROM %s WHERE game_id = ?", table), gameID); err != nil {
		return fmt.Errorf("clear %s: %w", table, err)
	}

	for index, id := range ids {
		if _, err := tx.Exec(
			fmt.Sprintf("INSERT INTO %s (game_id, %s, sort_order) VALUES (?, ?, ?)", table, column),
			gameID,
			id,
			index,
		); err != nil {
			return fmt.Errorf("insert %s relation: %w", table, err)
		}
	}

	return nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
