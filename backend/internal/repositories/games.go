package repositories

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/hao/game/internal/domain"
)

var allowedGameSortFields = map[string]string{
	"title":        "g.title",
	"created_at":   "g.created_at",
	"updated_at":   "g.updated_at",
	"release_date": "g.release_date",
	"downloads":    "g.downloads",
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

	if !params.IncludeAll {
		visibility := strings.TrimSpace(params.Visibility)
		if visibility == "" {
			visibility = domain.GameVisibilityPublic
		}
		where = append(where, "g.visibility = :visibility")
		args["visibility"] = visibility
	}

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
	if len(params.TagIDs) > 0 {
		tagFilters, tagArgs, err := r.buildTagFilters(params.TagIDs)
		if err != nil {
			return nil, 0, fmt.Errorf("build tag filters: %w", err)
		}
		where = append(where, tagFilters...)
		for key, value := range tagArgs {
			args[key] = value
		}
	}

	sortField := allowedGameSortFields[params.Sort]
	if params.Sort == "random" {
		sortField = "ABS((g.id * :sort_seed) % 2147483647)"
		args["sort_seed"] = params.SortSeed
	}
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
			g.downloads,
			(
				SELECT ga.path
				FROM game_assets ga
				WHERE ga.game_id = g.id AND ga.asset_type = 'screenshot'
				ORDER BY ga.sort_order ASC, ga.id ASC
				LIMIT 1
			) AS primary_screenshot,
			(
				SELECT COUNT(*)
				FROM game_assets ga
				WHERE ga.game_id = g.id AND ga.asset_type = 'screenshot'
			) AS screenshot_count,
			(
				SELECT COUNT(*)
				FROM game_files gf
				WHERE gf.game_id = g.id
			) AS file_count,
			(
				SELECT COUNT(*)
				FROM game_developers gd
				WHERE gd.game_id = g.id
			) AS developer_count,
			(
				SELECT COUNT(*)
				FROM game_publishers gp
				WHERE gp.game_id = g.id
			) AS publisher_count,
			(
				SELECT COUNT(*)
				FROM game_platforms gp
				WHERE gp.game_id = g.id
			) AS platform_count,
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
			visibility,
			summary,
			release_date,
			engine,
			cover_image,
			banner_image,
			wiki_content,
			wiki_content_html,
			needs_review,
			preview_video_asset_uid,
			downloads,
			NULL AS primary_screenshot,
			0 AS screenshot_count,
			0 AS file_count,
			0 AS developer_count,
			0 AS publisher_count,
			0 AS platform_count,
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

func (r *GamesRepository) ListTimeline(params domain.GamesTimelineParams) ([]domain.TimelineGame, bool, error) {
	where := []string{
		"g.release_date IS NOT NULL",
		"g.release_date != ''",
		"g.release_date >= :from_date",
		"g.release_date <= :to_date",
	}
	args := map[string]any{
		"from_date": params.FromDate,
		"to_date":   params.ToDate,
		"limit":     params.Limit + 1,
	}

	if !params.IncludeAll {
		visibility := strings.TrimSpace(params.Visibility)
		if visibility == "" {
			visibility = domain.GameVisibilityPublic
		}
		where = append(where, "g.visibility = :visibility")
		args["visibility"] = visibility
	}

	if params.CursorReleaseDate != "" && params.CursorID > 0 {
		where = append(where, "(g.release_date < :cursor_release_date OR (g.release_date = :cursor_release_date AND g.id < :cursor_id))")
		args["cursor_release_date"] = params.CursorReleaseDate
		args["cursor_id"] = params.CursorID
	}

	query := fmt.Sprintf(`
		SELECT
			g.id,
			g.title,
			g.release_date,
			g.cover_image
		FROM games g
		WHERE %s
		ORDER BY g.release_date DESC, g.id DESC
		LIMIT :limit
	`, strings.Join(where, " AND "))

	stmt, stmtArgs, err := sqlx.Named(query, args)
	if err != nil {
		return nil, false, fmt.Errorf("build games timeline query: %w", err)
	}
	stmt = r.db.Rebind(stmt)

	var games []domain.TimelineGame
	if err := r.db.Select(&games, stmt, stmtArgs...); err != nil {
		return nil, false, fmt.Errorf("list timeline games: %w", err)
	}

	hasMore := len(games) > params.Limit
	if hasMore {
		games = games[:params.Limit]
	}

	return games, hasMore, nil
}

func (r *GamesRepository) LatestTimelineReleaseDate(includeAll bool, visibility string) (*string, error) {
	where := []string{
		"g.release_date IS NOT NULL",
		"g.release_date != ''",
	}
	args := map[string]any{}

	if !includeAll {
		targetVisibility := strings.TrimSpace(visibility)
		if targetVisibility == "" {
			targetVisibility = domain.GameVisibilityPublic
		}
		where = append(where, "g.visibility = :visibility")
		args["visibility"] = targetVisibility
	}

	query := fmt.Sprintf(`
		SELECT g.release_date
		FROM games g
		WHERE %s
		ORDER BY g.release_date DESC, g.id DESC
		LIMIT 1
	`, strings.Join(where, " AND "))

	stmt, stmtArgs, err := sqlx.Named(query, args)
	if err != nil {
		return nil, fmt.Errorf("build latest timeline release date query: %w", err)
	}
	stmt = r.db.Rebind(stmt)

	var releaseDate string
	if err := r.db.Get(&releaseDate, stmt, stmtArgs...); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get latest timeline release date: %w", err)
	}

	trimmed := strings.TrimSpace(releaseDate)
	if trimmed == "" {
		return nil, nil
	}

	return &trimmed, nil
}

func (r *GamesRepository) HasOlderTimelineGame(params domain.GamesTimelineParams, cursorReleaseDate string, cursorID int64) (bool, error) {
	where := []string{
		"g.release_date IS NOT NULL",
		"g.release_date != ''",
		"g.release_date <= :to_date",
		"(g.release_date < :cursor_release_date OR (g.release_date = :cursor_release_date AND g.id < :cursor_id))",
	}
	args := map[string]any{
		"to_date":             params.ToDate,
		"cursor_release_date": cursorReleaseDate,
		"cursor_id":           cursorID,
	}

	if !params.IncludeAll {
		targetVisibility := strings.TrimSpace(params.Visibility)
		if targetVisibility == "" {
			targetVisibility = domain.GameVisibilityPublic
		}
		where = append(where, "g.visibility = :visibility")
		args["visibility"] = targetVisibility
	}

	query := fmt.Sprintf(`
		SELECT 1
		FROM games g
		WHERE %s
		LIMIT 1
	`, strings.Join(where, " AND "))

	stmt, stmtArgs, err := sqlx.Named(query, args)
	if err != nil {
		return false, fmt.Errorf("build older timeline exists query: %w", err)
	}
	stmt = r.db.Rebind(stmt)

	var value int
	if err := r.db.Get(&value, stmt, stmtArgs...); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("check older timeline game exists: %w", err)
	}

	return true, nil
}

func (r *GamesRepository) Create(input domain.GameWriteInput) (*domain.Game, error) {
	const query = `
		INSERT INTO games (
			title, title_alt, visibility, summary, release_date, engine, cover_image, banner_image, needs_review, preview_video_asset_uid
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING
			id, title, title_alt, visibility, summary, release_date, engine, cover_image, banner_image,
			wiki_content, wiki_content_html, needs_review, preview_video_asset_uid, downloads, created_at, updated_at`

	var game domain.Game
	if err := r.db.Get(
		&game,
		query,
		input.Title,
		input.TitleAlt,
		input.Visibility,
		input.Summary,
		input.ReleaseDate,
		input.Engine,
		input.CoverImage,
		input.BannerImage,
		boolToInt(input.NeedsReview),
		input.PreviewVideoAssetUID,
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
			visibility = ?,
			summary = ?,
			release_date = ?,
			engine = ?,
			cover_image = ?,
			banner_image = ?,
			needs_review = ?,
			preview_video_asset_uid = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`

	result, err := r.db.Exec(
		query,
		input.Title,
		input.TitleAlt,
		input.Visibility,
		input.Summary,
		input.ReleaseDate,
		input.Engine,
		input.CoverImage,
		input.BannerImage,
		boolToInt(input.NeedsReview),
		input.PreviewVideoAssetUID,
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

func (r *GamesRepository) UpdateAggregate(id int64, input domain.GameAggregateUpdateInput) ([]string, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return nil, fmt.Errorf("begin aggregate update tx: %w", err)
	}
	defer tx.Rollback()

	if err := r.updateGameRowTx(tx, id, input.Game); err != nil {
		return nil, err
	}
	if err := r.replaceRelationsTx(tx, id, input.Game); err != nil {
		return nil, err
	}
	if err := r.syncGameFilesTx(tx, id, input.Files); err != nil {
		return nil, err
	}

	deletedAssetPaths, err := r.deleteAssetsTx(tx, id, input.DeleteAssets)
	if err != nil {
		return nil, err
	}
	if err := r.reorderAssetsTx(tx, id, "screenshot", input.ScreenshotOrderAssetUIDs); err != nil {
		return nil, err
	}
	if err := r.reorderAssetsTx(tx, id, "video", input.VideoOrderAssetUIDs); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit aggregate update tx: %w", err)
	}

	return uniqueNonEmptyStrings(deletedAssetPaths), nil
}

func (r *GamesRepository) updateGameRowTx(tx *sqlx.Tx, id int64, input domain.GameWriteInput) error {
	result, err := tx.Exec(`
		UPDATE games
		SET
			title = ?,
			title_alt = ?,
			visibility = ?,
			summary = ?,
			release_date = ?,
			engine = ?,
			cover_image = ?,
			banner_image = ?,
			needs_review = ?,
			preview_video_asset_uid = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`,
		input.Title,
		input.TitleAlt,
		input.Visibility,
		input.Summary,
		input.ReleaseDate,
		input.Engine,
		input.CoverImage,
		input.BannerImage,
		boolToInt(input.NeedsReview),
		input.PreviewVideoAssetUID,
		id,
	)
	if err != nil {
		return fmt.Errorf("update game: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read updated rows: %w", err)
	}
	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *GamesRepository) replaceRelationsTx(tx *sqlx.Tx, gameID int64, input domain.GameWriteInput) error {
	if err := replaceRelationRows(tx, "game_series", "series_id", gameID, input.SeriesIDs); err != nil {
		return err
	}
	if err := replaceRelationRows(tx, "game_platforms", "platform_id", gameID, input.PlatformIDs); err != nil {
		return err
	}
	if err := replaceRelationRows(tx, "game_developers", "developer_id", gameID, input.DeveloperIDs); err != nil {
		return err
	}
	if err := replaceRelationRows(tx, "game_publishers", "publisher_id", gameID, input.PublisherIDs); err != nil {
		return err
	}
	if err := replaceRelationRows(tx, "game_tags", "tag_id", gameID, input.TagIDs); err != nil {
		return err
	}
	return nil
}

func (r *GamesRepository) syncGameFilesTx(tx *sqlx.Tx, gameID int64, files []domain.GameFileUpsertInput) error {
	type existingGameFile struct {
		ID int64 `db:"id"`
	}

	var existingFiles []existingGameFile
	if err := tx.Select(&existingFiles, "SELECT id FROM game_files WHERE game_id = ?", gameID); err != nil {
		return fmt.Errorf("list game files before sync: %w", err)
	}

	keepFileIDs := make(map[int64]struct{}, len(files))
	for index, item := range files {
		sortOrder := item.SortOrder
		if sortOrder < 0 {
			sortOrder = index
		}

		if item.ID != nil && *item.ID > 0 {
			result, err := tx.Exec(`
				UPDATE game_files
				SET file_path = ?, label = ?, notes = ?, sort_order = ?, updated_at = CURRENT_TIMESTAMP
				WHERE game_id = ? AND id = ?
			`, item.FilePath, item.Label, item.Notes, sortOrder, gameID, *item.ID)
			if err != nil {
				return fmt.Errorf("update game file: %w", err)
			}
			rows, err := result.RowsAffected()
			if err != nil {
				return fmt.Errorf("read updated game file rows: %w", err)
			}
			if rows == 0 {
				return sql.ErrNoRows
			}
			keepFileIDs[*item.ID] = struct{}{}
			continue
		}

		if _, err := tx.Exec(`
			INSERT INTO game_files (game_id, file_path, label, notes, sort_order)
			VALUES (?, ?, ?, ?, ?)
		`, gameID, item.FilePath, item.Label, item.Notes, sortOrder); err != nil {
			return fmt.Errorf("create game file: %w", err)
		}
	}

	for _, file := range existingFiles {
		if _, keep := keepFileIDs[file.ID]; keep {
			continue
		}
		if _, err := tx.Exec("DELETE FROM game_files WHERE game_id = ? AND id = ?", gameID, file.ID); err != nil {
			return fmt.Errorf("delete game file: %w", err)
		}
	}

	return nil
}

func (r *GamesRepository) deleteAssetsTx(tx *sqlx.Tx, gameID int64, deleteAssets []domain.GameAssetDeleteInput) ([]string, error) {
	assetPaths := make([]string, 0, len(deleteAssets))

	var currentPrimaryVideoUID sql.NullString
	if err := tx.Get(&currentPrimaryVideoUID, "SELECT preview_video_asset_uid FROM games WHERE id = ?", gameID); err != nil {
		return nil, fmt.Errorf("load game preview video uid: %w", err)
	}
	needsPrimaryVideoFallback := false

	for _, item := range deleteAssets {
		switch strings.TrimSpace(item.AssetType) {
		case "cover":
			if _, err := tx.Exec("UPDATE games SET cover_image = NULL, updated_at = CURRENT_TIMESTAMP WHERE id = ?", gameID); err != nil {
				return nil, fmt.Errorf("delete cover image: %w", err)
			}
			assetPaths = append(assetPaths, strings.TrimSpace(item.Path))
		case "banner":
			if _, err := tx.Exec("UPDATE games SET banner_image = NULL, updated_at = CURRENT_TIMESTAMP WHERE id = ?", gameID); err != nil {
				return nil, fmt.Errorf("delete banner image: %w", err)
			}
			assetPaths = append(assetPaths, strings.TrimSpace(item.Path))
		case "screenshot":
			deletedPath, _, deleted, err := r.deleteSingleAssetTx(tx, gameID, "screenshot", item)
			if err != nil {
				return nil, err
			}
			if deleted {
				assetPaths = append(assetPaths, deletedPath)
			}
		case "video":
			deletedPath, deletedUID, deleted, err := r.deleteSingleAssetTx(tx, gameID, "video", item)
			if err != nil {
				return nil, err
			}
			if deleted {
				assetPaths = append(assetPaths, deletedPath)
				if currentPrimaryVideoUID.Valid && strings.TrimSpace(currentPrimaryVideoUID.String) == strings.TrimSpace(deletedUID) {
					needsPrimaryVideoFallback = true
					currentPrimaryVideoUID = sql.NullString{}
				}
			}
		default:
			return nil, fmt.Errorf("invalid asset type: %s", strings.TrimSpace(item.AssetType))
		}
	}

	if needsPrimaryVideoFallback {
		var fallbackUID sql.NullString
		if err := tx.Get(&fallbackUID, `
			SELECT asset_uid
			FROM game_assets
			WHERE game_id = ? AND asset_type = 'video'
			ORDER BY sort_order ASC, id ASC
			LIMIT 1
		`, gameID); err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return nil, fmt.Errorf("load fallback primary video: %w", err)
			}
			fallbackUID = sql.NullString{}
		}

		if fallbackUID.Valid {
			if _, err := tx.Exec(
				"UPDATE games SET preview_video_asset_uid = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
				fallbackUID.String,
				gameID,
			); err != nil {
				return nil, fmt.Errorf("set fallback primary video: %w", err)
			}
		} else {
			if _, err := tx.Exec(
				"UPDATE games SET preview_video_asset_uid = NULL, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
				gameID,
			); err != nil {
				return nil, fmt.Errorf("clear primary video: %w", err)
			}
		}
	}

	return assetPaths, nil
}

func (r *GamesRepository) deleteSingleAssetTx(
	tx *sqlx.Tx,
	gameID int64,
	assetType string,
	item domain.GameAssetDeleteInput,
) (string, string, bool, error) {
	trimmedUID := strings.TrimSpace(item.AssetUID)
	if trimmedUID != "" {
		var deleted struct {
			Path     string         `db:"path"`
			AssetUID sql.NullString `db:"asset_uid"`
		}
		if err := tx.Get(&deleted, `
			DELETE FROM game_assets
			WHERE game_id = ? AND asset_type = ? AND asset_uid = ?
			RETURNING path, asset_uid
		`, gameID, assetType, trimmedUID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return "", "", false, nil
			}
			return "", "", false, fmt.Errorf("delete %s by uid: %w", assetType, err)
		}
		return deleted.Path, deleted.AssetUID.String, true, nil
	}

	if item.AssetID != nil && *item.AssetID > 0 {
		var deleted struct {
			Path     string         `db:"path"`
			AssetUID sql.NullString `db:"asset_uid"`
		}
		if err := tx.Get(&deleted, `
			DELETE FROM game_assets
			WHERE game_id = ? AND asset_type = ? AND id = ?
			RETURNING path, asset_uid
		`, gameID, assetType, *item.AssetID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return "", "", false, nil
			}
			return "", "", false, fmt.Errorf("delete %s by id: %w", assetType, err)
		}
		return deleted.Path, deleted.AssetUID.String, true, nil
	}

	trimmedPath := strings.TrimSpace(item.Path)
	if trimmedPath == "" {
		return "", "", false, nil
	}
	var deleted struct {
		Path     string         `db:"path"`
		AssetUID sql.NullString `db:"asset_uid"`
	}
	if err := tx.Get(&deleted, `
		DELETE FROM game_assets
		WHERE game_id = ? AND asset_type = ? AND path = ?
		RETURNING path, asset_uid
	`, gameID, assetType, trimmedPath); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", false, nil
		}
		return "", "", false, fmt.Errorf("delete %s by path: %w", assetType, err)
	}
	return deleted.Path, deleted.AssetUID.String, true, nil
}

func (r *GamesRepository) reorderAssetsTx(tx *sqlx.Tx, gameID int64, assetType string, assetUIDs []string) error {
	if len(assetUIDs) == 0 {
		return nil
	}

	for index, assetUID := range assetUIDs {
		trimmedUID := strings.TrimSpace(assetUID)
		if trimmedUID == "" {
			return fmt.Errorf("empty %s asset uid", assetType)
		}

		result, err := tx.Exec(`
			UPDATE game_assets
			SET sort_order = ?
			WHERE game_id = ? AND asset_type = ? AND asset_uid = ?
		`, index, gameID, assetType, trimmedUID)
		if err != nil {
			return fmt.Errorf("update %s sort order: %w", assetType, err)
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("read %s reorder rows: %w", assetType, err)
		}
		if rows == 0 {
			return sql.ErrNoRows
		}
	}

	return nil
}

func uniqueNonEmptyStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}

	return result
}

func (r *GamesRepository) Stats(params domain.GamesListParams) (*domain.GameStats, error) {
	where := []string{"1 = 1"}
	args := map[string]any{}

	if !params.IncludeAll {
		visibility := strings.TrimSpace(params.Visibility)
		if visibility == "" {
			visibility = domain.GameVisibilityPublic
		}
		where = append(where, "g.visibility = :visibility")
		args["visibility"] = visibility
	}

	baseWhere := strings.Join(where, " AND ")

	const baseSelect = `
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
			g.downloads,
			(
				SELECT ga.path
				FROM game_assets ga
				WHERE ga.game_id = g.id AND ga.asset_type = 'screenshot'
				ORDER BY ga.sort_order ASC, ga.id ASC
				LIMIT 1
			) AS primary_screenshot,
			(
				SELECT COUNT(*)
				FROM game_assets ga
				WHERE ga.game_id = g.id AND ga.asset_type = 'screenshot'
			) AS screenshot_count,
			(
				SELECT COUNT(*)
				FROM game_files gf
				WHERE gf.game_id = g.id
			) AS file_count,
			(
				SELECT COUNT(*)
				FROM game_developers gd
				WHERE gd.game_id = g.id
			) AS developer_count,
			(
				SELECT COUNT(*)
				FROM game_publishers gp
				WHERE gp.game_id = g.id
			) AS publisher_count,
			(
				SELECT COUNT(*)
				FROM game_platforms gp
				WHERE gp.game_id = g.id
			) AS platform_count,
			g.created_at,
			g.updated_at
		FROM games g
		WHERE %s
	`

	summaryQuery := fmt.Sprintf(`
		SELECT
			COUNT(*) AS total_games,
			COALESCE(SUM(g.downloads), 0) AS total_downloads,
			COALESCE(SUM(CASE WHEN g.needs_review = 1 THEN 1 ELSE 0 END), 0) AS pending_reviews
		FROM games g
		WHERE %s
	`, baseWhere)

	type statsRow struct {
		TotalGames     int   `db:"total_games"`
		TotalDownloads int64 `db:"total_downloads"`
		PendingReviews int   `db:"pending_reviews"`
	}

	summaryStmt, summaryArgs, err := sqlx.Named(summaryQuery, args)
	if err != nil {
		return nil, fmt.Errorf("build games stats query: %w", err)
	}
	summaryStmt = r.db.Rebind(summaryStmt)

	var summary statsRow
	if err := r.db.Get(&summary, summaryStmt, summaryArgs...); err != nil {
		return nil, fmt.Errorf("get games stats: %w", err)
	}

	loadGames := func(orderBy string) ([]domain.Game, error) {
		query := fmt.Sprintf(baseSelect+`
			ORDER BY %s
			LIMIT 12
		`, baseWhere, orderBy)
		stmt, queryArgs, err := sqlx.Named(query, args)
		if err != nil {
			return nil, fmt.Errorf("build stats games query: %w", err)
		}
		stmt = r.db.Rebind(stmt)

		var games []domain.Game
		if err := r.db.Select(&games, stmt, queryArgs...); err != nil {
			return nil, fmt.Errorf("list stats games: %w", err)
		}
		return games, nil
	}

	recentGames, err := loadGames("g.created_at DESC, g.id DESC")
	if err != nil {
		return nil, err
	}
	popularGames, err := loadGames("g.downloads DESC, g.id DESC")
	if err != nil {
		return nil, err
	}

	return &domain.GameStats{
		TotalGames:     summary.TotalGames,
		TotalDownloads: summary.TotalDownloads,
		TotalSize:      0,
		RecentGames:    recentGames,
		PopularGames:   popularGames,
		PendingReviews: summary.PendingReviews,
	}, nil
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
		SET downloads = downloads + 1
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
	if err := replaceRelationRows(tx, "game_tags", "tag_id", gameID, input.TagIDs); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit relations update: %w", err)
	}

	return nil
}

func (r *GamesRepository) buildTagFilters(tagIDs []int64) ([]string, map[string]any, error) {
	normalized := uniquePositiveIDs(tagIDs)
	if len(normalized) == 0 {
		return nil, map[string]any{}, nil
	}

	query, queryArgs, err := sqlx.In(`
		SELECT id, group_id
		FROM tags
		WHERE is_active = 1 AND id IN (?)
	`, normalized)
	if err != nil {
		return nil, nil, fmt.Errorf("build tag grouping query: %w", err)
	}
	query = r.db.Rebind(query)

	type row struct {
		ID      int64 `db:"id"`
		GroupID int64 `db:"group_id"`
	}

	var rows []row
	if err := r.db.Select(&rows, query, queryArgs...); err != nil {
		return nil, nil, fmt.Errorf("load tag groups: %w", err)
	}
	if len(rows) != len(normalized) {
		return []string{"1 = 0"}, map[string]any{}, nil
	}

	grouped := map[int64][]int64{}
	for _, item := range rows {
		grouped[item.GroupID] = append(grouped[item.GroupID], item.ID)
	}

	groupIDs := make([]int64, 0, len(grouped))
	for groupID := range grouped {
		groupIDs = append(groupIDs, groupID)
	}
	sort.Slice(groupIDs, func(i, j int) bool {
		return groupIDs[i] < groupIDs[j]
	})

	filters := make([]string, 0, len(groupIDs))
	args := map[string]any{}
	for groupIndex, groupID := range groupIDs {
		placeholders := make([]string, 0, len(grouped[groupID]))
		for tagIndex, tagID := range grouped[groupID] {
			argKey := fmt.Sprintf("tag_%d_%d", groupIndex, tagIndex)
			args[argKey] = tagID
			placeholders = append(placeholders, ":"+argKey)
		}
		filters = append(filters, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM game_tags gt WHERE gt.game_id = g.id AND gt.tag_id IN (%s))",
			strings.Join(placeholders, ", "),
		))
	}

	return filters, args, nil
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
