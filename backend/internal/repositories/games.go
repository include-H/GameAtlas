package repositories

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/hao/game/internal/domain"
)

var allowedGameSortFields = map[string]string{
	"title":        "g.title_sort_key",
	"created_at":   "g.created_at",
	"updated_at":   "g.updated_at",
	"release_date": "g.release_date",
	"downloads":    "g.downloads",
}

type pendingIssueDefinition struct {
	Key              string
	Group            string
	AnyCondition     string
	VisibleCondition string
}

var pendingIssueDefinitions = []pendingIssueDefinition{
	newPendingFieldIssue(string(domain.PendingIssueDetailMissingCover), string(domain.PendingIssueMissingAssets), "g.cover_image"),
	newPendingFieldIssue(string(domain.PendingIssueDetailMissingBanner), string(domain.PendingIssueMissingAssets), "g.banner_image"),
	newPendingRelationIssue(string(domain.PendingIssueDetailMissingScreenshots), string(domain.PendingIssueMissingAssets), "game_assets ga", "ga.game_id = g.id AND ga.asset_type = 'screenshot'"),
	newPendingWikiIssue(),
	newPendingRelationIssue(string(domain.PendingIssueDetailMissingFilesList), string(domain.PendingIssueMissingFiles), "game_files gf", "gf.game_id = g.id"),
	newPendingRelationIssue(string(domain.PendingIssueDetailMissingDeveloper), string(domain.PendingIssueMissingMetadata), "game_developers gd", "gd.game_id = g.id"),
	newPendingRelationIssue(string(domain.PendingIssueDetailMissingPublisher), string(domain.PendingIssueMissingMetadata), "game_publishers gp", "gp.game_id = g.id"),
	newPendingRelationIssue(string(domain.PendingIssueDetailMissingPlatform), string(domain.PendingIssueMissingMetadata), "game_platforms gp", "gp.game_id = g.id"),
	newPendingFieldIssue(string(domain.PendingIssueDetailMissingSummary), string(domain.PendingIssueMissingMetadata), "g.summary"),
}

type GamesRepository struct {
	db *sqlx.DB
}

var fallbackPublicIDCounter uint64

func NewGamesRepository(db *sqlx.DB) *GamesRepository {
	return &GamesRepository{db: db}
}

func (r *GamesRepository) List(params domain.GamesListParams) ([]domain.Game, int, error) {
	where, args, err := r.buildGamesListWhere(params, false)
	if err != nil {
		return nil, 0, err
	}

	sortField := allowedGameSortFields[params.Sort]
	if params.Sort == "random" {
		sortField = "ABS((g.id * :sort_seed) % 2147483647)"
		args["sort_seed"] = params.SortSeed
	} else if params.Sort == "pending_issue_count" {
		sortField = pendingVisibleIssueCountExpression()
	}
	if sortField == "" {
		sortField = allowedGameSortFields["updated_at"]
	}
	order := strings.ToUpper(params.Order)
	if order != "ASC" && order != "DESC" {
		order = "DESC"
	}
	idOrder := "DESC"
	if order == "ASC" {
		idOrder = "ASC"
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
		WITH page_games AS (
			SELECT g.id
			FROM games g
			WHERE %s
			ORDER BY %s %s, g.id %s
			LIMIT :limit OFFSET :offset
		),
		ranked_screenshots AS (
			SELECT
				ga.game_id,
				ga.path,
				ROW_NUMBER() OVER (
					PARTITION BY ga.game_id
					ORDER BY ga.sort_order ASC, ga.id ASC
				) AS row_num
			FROM game_assets ga
			INNER JOIN page_games pg ON pg.id = ga.game_id
			WHERE ga.asset_type = 'screenshot'
		),
		screenshot_stats AS (
			SELECT
				rs.game_id,
				COUNT(*) AS screenshot_count,
				MAX(CASE WHEN rs.row_num = 1 THEN rs.path END) AS primary_screenshot
			FROM ranked_screenshots rs
			GROUP BY rs.game_id
		),
		file_stats AS (
			SELECT gf.game_id, COUNT(*) AS file_count
			FROM game_files gf
			INNER JOIN page_games pg ON pg.id = gf.game_id
			GROUP BY gf.game_id
		),
		developer_stats AS (
			SELECT gd.game_id, COUNT(*) AS developer_count
			FROM game_developers gd
			INNER JOIN page_games pg ON pg.id = gd.game_id
			GROUP BY gd.game_id
		),
		publisher_stats AS (
			SELECT gp.game_id, COUNT(*) AS publisher_count
			FROM game_publishers gp
			INNER JOIN page_games pg ON pg.id = gp.game_id
			GROUP BY gp.game_id
		),
		platform_stats AS (
			SELECT gp.game_id, COUNT(*) AS platform_count
			FROM game_platforms gp
			INNER JOIN page_games pg ON pg.id = gp.game_id
			GROUP BY gp.game_id
		)
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
			ss.primary_screenshot,
			COALESCE(ss.screenshot_count, 0) AS screenshot_count,
			COALESCE(fs.file_count, 0) AS file_count,
			COALESCE(ds.developer_count, 0) AS developer_count,
			COALESCE(ps.publisher_count, 0) AS publisher_count,
			COALESCE(pls.platform_count, 0) AS platform_count,
			g.created_at,
			g.updated_at
		FROM page_games pg
		INNER JOIN games g ON g.id = pg.id
		LEFT JOIN screenshot_stats ss ON ss.game_id = g.id
		LEFT JOIN file_stats fs ON fs.game_id = g.id
		LEFT JOIN developer_stats ds ON ds.game_id = g.id
		LEFT JOIN publisher_stats ps ON ps.game_id = g.id
		LEFT JOIN platform_stats pls ON pls.game_id = g.id
		ORDER BY %s %s, g.id %s
	`, baseWhere, sortField, order, idOrder, sortField, order, idOrder)

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

func (r *GamesRepository) buildGamesListWhere(params domain.GamesListParams, excludePendingIssueFilter bool) ([]string, map[string]any, error) {
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
		where = append(where, "(g.title LIKE :search OR COALESCE(g.title_alt, '') LIKE :search OR COALESCE(g.summary, '') LIKE :search)")
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
	if params.PendingOnly {
		where = append(where, "("+pendingAnyIssueCondition(params.PendingIncludeIgnored)+")")
		if !excludePendingIssueFilter && params.PendingIssue != "" {
			pendingIssueConditions := pendingIssueConditionsForFilter(params.PendingIssue, params.PendingIncludeIgnored)
			if len(pendingIssueConditions) == 0 {
				where = append(where, "1 = 0")
			} else {
				where = append(where, "("+strings.Join(pendingIssueConditions, " OR ")+")")
			}
		}
		if params.PendingSevereOnly {
			where = append(where, "("+pendingSevereCondition()+")")
		}
		if params.PendingRecentDays > 0 {
			args["pending_recent_days"] = fmt.Sprintf("-%d days", params.PendingRecentDays)
			where = append(where, "datetime(g.created_at) >= datetime('now', :pending_recent_days)")
		}
	}
	if params.SeriesID > 0 {
		where = append(where, "g.series_id = :series_id")
		args["series_id"] = params.SeriesID
	}
	if params.PlatformID > 0 {
		where = append(where, "EXISTS (SELECT 1 FROM game_platforms gp WHERE gp.game_id = g.id AND gp.platform_id = :platform_id)")
		args["platform_id"] = params.PlatformID
	}
	if len(params.TagIDs) > 0 {
		tagFilters, tagArgs, err := r.buildTagFilters(params.TagIDs)
		if err != nil {
			return nil, nil, fmt.Errorf("build tag filters: %w", err)
		}
		where = append(where, tagFilters...)
		for key, value := range tagArgs {
			args[key] = value
		}
	}

	return where, args, nil
}

func (r *GamesRepository) CountPendingGroups(params domain.GamesListParams) (*domain.PendingGroupCounts, error) {
	where, args, err := r.buildGamesListWhere(params, true)
	if err != nil {
		return nil, err
	}
	baseWhere := strings.Join(where, " AND ")

	query := fmt.Sprintf(`
		SELECT
			COALESCE(SUM(CASE WHEN %s THEN 1 ELSE 0 END), 0) AS missing_assets,
			COALESCE(SUM(CASE WHEN %s THEN 1 ELSE 0 END), 0) AS missing_wiki,
			COALESCE(SUM(CASE WHEN %s THEN 1 ELSE 0 END), 0) AS missing_files,
			COALESCE(SUM(CASE WHEN %s THEN 1 ELSE 0 END), 0) AS missing_metadata,
			COALESCE((
				SELECT COUNT(*)
				FROM game_review_issue_overrides gio
				WHERE gio.status = 'ignored'
					AND EXISTS (SELECT 1 FROM games g WHERE %s AND g.id = gio.game_id)
			), 0) AS ignored_total
		FROM games g
		WHERE %s
	`,
		pendingGroupCondition(string(domain.PendingIssueMissingAssets), params.PendingIncludeIgnored),
		pendingGroupCondition(string(domain.PendingIssueMissingWiki), params.PendingIncludeIgnored),
		pendingGroupCondition(string(domain.PendingIssueMissingFiles), params.PendingIncludeIgnored),
		pendingGroupCondition(string(domain.PendingIssueMissingMetadata), params.PendingIncludeIgnored),
		baseWhere,
		baseWhere,
	)

	stmt, queryArgs, err := sqlx.Named(query, args)
	if err != nil {
		return nil, fmt.Errorf("build pending group counts query: %w", err)
	}
	stmt = r.db.Rebind(stmt)

	var counts domain.PendingGroupCounts
	if err := r.db.Get(&counts, stmt, queryArgs...); err != nil {
		return nil, fmt.Errorf("count pending groups: %w", err)
	}

	return &counts, nil
}

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
			engine,
			cover_image,
			banner_image,
			wiki_content,
			needs_review,
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
			engine,
			cover_image,
			banner_image,
			wiki_content,
			needs_review,
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
		WHERE lower(public_id) = lower(?)`

	var game domain.Game
	if err := r.db.Get(&game, query, strings.TrimSpace(publicID)); err != nil {
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
	if err := r.db.Get(&id, "SELECT id FROM games WHERE lower(public_id) = lower(?) LIMIT 1", trimmed); err != nil {
		return 0, err
	}
	return id, nil
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
			g.public_id,
			g.title,
			g.release_date,
			g.cover_image,
			g.banner_image
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
	tx, err := r.db.Beginx()
	if err != nil {
		return nil, fmt.Errorf("begin create game tx: %w", err)
	}
	defer tx.Rollback()

	const query = `
		INSERT INTO games (
			public_id, title, title_alt, title_sort_key, visibility, summary, release_date, engine, cover_image, banner_image, needs_review, series_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING
			id, public_id, title, title_alt, visibility, summary, release_date, engine, cover_image, banner_image,
			wiki_content, needs_review, downloads, created_at, updated_at`

	var game domain.Game
	if err := tx.Get(
		&game,
		query,
		newGamePublicID(),
		input.Title,
		input.TitleAlt,
		buildTitleSortKey(input.Title, input.TitleAlt),
		input.Visibility,
		input.Summary,
		input.ReleaseDate,
		input.Engine,
		input.CoverImage,
		input.BannerImage,
		boolToInt(input.NeedsReview),
		input.SeriesID,
	); err != nil {
		return nil, fmt.Errorf("create game: %w", err)
	}

	if err := r.replaceRelationsTx(tx, game.ID, domain.GameAggregatePatchInput{
		GameCoreInput: domain.GameCoreInput{},
		PlatformIDs:   domain.Int64SlicePatch{Present: true, Values: input.PlatformIDs},
		DeveloperIDs:  domain.Int64SlicePatch{Present: true, Values: input.DeveloperIDs},
		PublisherIDs:  domain.Int64SlicePatch{Present: true, Values: input.PublisherIDs},
		TagIDs:        domain.Int64SlicePatch{Present: true, Values: input.TagIDs},
	}); err != nil {
		return nil, fmt.Errorf("create game relations: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit create game tx: %w", err)
	}

	return r.GetByID(game.ID)
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
	if err := r.syncGameFilesTx(tx, id, input.Assets.Files); err != nil {
		return nil, err
	}

	deletedAssetPaths, err := r.deleteAssetsTx(tx, id, input.Assets.DeleteAssets)
	if err != nil {
		return nil, err
	}
	if err := r.reorderAssetsTx(tx, id, "screenshot", input.Assets.ScreenshotOrderAssetUIDs); err != nil {
		return nil, err
	}
	if err := r.reorderAssetsTx(tx, id, "video", input.Assets.VideoOrderAssetUIDs); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit aggregate update tx: %w", err)
	}

	return uniqueNonEmptyStrings(deletedAssetPaths), nil
}

func (r *GamesRepository) RebuildTitleSortKeys() error {
	type gameTitleRow struct {
		ID           int64   `db:"id"`
		Title        string  `db:"title"`
		TitleAlt     *string `db:"title_alt"`
		TitleSortKey string  `db:"title_sort_key"`
	}

	var rows []gameTitleRow
	if err := r.db.Select(&rows, "SELECT id, title, title_alt, title_sort_key FROM games"); err != nil {
		return fmt.Errorf("list games for title sort key rebuild: %w", err)
	}
	if len(rows) == 0 {
		return nil
	}

	tx, err := r.db.Beginx()
	if err != nil {
		return fmt.Errorf("begin title sort key rebuild tx: %w", err)
	}
	defer tx.Rollback()

	const updateQuery = "UPDATE games SET title_sort_key = ? WHERE id = ?"
	for _, row := range rows {
		nextKey := buildTitleSortKey(row.Title, row.TitleAlt)
		if row.TitleSortKey == nextKey {
			continue
		}
		if _, err := tx.Exec(updateQuery, nextKey, row.ID); err != nil {
			return fmt.Errorf("update title sort key for game %d: %w", row.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit title sort key rebuild tx: %w", err)
	}

	return nil
}

func (r *GamesRepository) updateGameRowTx(tx *sqlx.Tx, id int64, input domain.GameAggregatePatchInput) error {
	setClauses := []string{
		"title = ?",
		"title_alt = ?",
		"title_sort_key = ?",
		"visibility = ?",
		"summary = ?",
		"release_date = ?",
		"engine = ?",
		"cover_image = ?",
		"banner_image = ?",
		"needs_review = ?",
	}
	args := []any{
		input.Title,
		input.TitleAlt,
		buildTitleSortKey(input.Title, input.TitleAlt),
		input.Visibility,
		input.Summary,
		input.ReleaseDate,
		input.Engine,
		input.CoverImage,
		input.BannerImage,
		boolToInt(input.NeedsReview),
	}
	if input.SeriesID.Present {
		setClauses = append(setClauses, "series_id = ?")
		args = append(args, input.SeriesID.Value)
	}
	setClauses = append(setClauses, "updated_at = CURRENT_TIMESTAMP")
	args = append(args, id)

	query := fmt.Sprintf(`
		UPDATE games
		SET
			%s
		WHERE id = ?
	`, strings.Join(setClauses, ",\n\t\t\t"))

	result, err := tx.Exec(query, args...)
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

func (r *GamesRepository) replaceRelationsTx(tx *sqlx.Tx, gameID int64, input domain.GameAggregatePatchInput) error {
	if input.PlatformIDs.Present {
		if err := replaceRelationRows(tx, "game_platforms", "platform_id", gameID, input.PlatformIDs.Values); err != nil {
			return err
		}
	}
	if input.DeveloperIDs.Present {
		if err := replaceRelationRows(tx, "game_developers", "developer_id", gameID, input.DeveloperIDs.Values); err != nil {
			return err
		}
	}
	if input.PublisherIDs.Present {
		if err := replaceRelationRows(tx, "game_publishers", "publisher_id", gameID, input.PublisherIDs.Values); err != nil {
			return err
		}
	}
	if input.TagIDs.Present {
		if err := replaceRelationRows(tx, "game_tags", "tag_id", gameID, input.TagIDs.Values); err != nil {
			return err
		}
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
			deletedPath, _, deleted, err := r.deleteSingleAssetTx(tx, gameID, "video", item)
			if err != nil {
				return nil, err
			}
			if deleted {
				assetPaths = append(assetPaths, deletedPath)
			}
		default:
			return nil, fmt.Errorf("invalid asset type: %s", strings.TrimSpace(item.AssetType))
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
		query := fmt.Sprintf(`
			WITH stat_games AS (
				SELECT g.id
				FROM games g
				WHERE %s
				ORDER BY %s
				LIMIT 12
			),
			ranked_screenshots AS (
				SELECT
					ga.game_id,
					ga.path,
					ROW_NUMBER() OVER (
						PARTITION BY ga.game_id
						ORDER BY ga.sort_order ASC, ga.id ASC
					) AS row_num
				FROM game_assets ga
				INNER JOIN stat_games sg ON sg.id = ga.game_id
				WHERE ga.asset_type = 'screenshot'
			),
			screenshot_stats AS (
				SELECT
					rs.game_id,
					COUNT(*) AS screenshot_count,
					MAX(CASE WHEN rs.row_num = 1 THEN rs.path END) AS primary_screenshot
				FROM ranked_screenshots rs
				GROUP BY rs.game_id
			),
			file_stats AS (
				SELECT gf.game_id, COUNT(*) AS file_count
				FROM game_files gf
				INNER JOIN stat_games sg ON sg.id = gf.game_id
				GROUP BY gf.game_id
			),
			developer_stats AS (
				SELECT gd.game_id, COUNT(*) AS developer_count
				FROM game_developers gd
				INNER JOIN stat_games sg ON sg.id = gd.game_id
				GROUP BY gd.game_id
			),
			publisher_stats AS (
				SELECT gp.game_id, COUNT(*) AS publisher_count
				FROM game_publishers gp
				INNER JOIN stat_games sg ON sg.id = gp.game_id
				GROUP BY gp.game_id
			),
			platform_stats AS (
				SELECT gp.game_id, COUNT(*) AS platform_count
				FROM game_platforms gp
				INNER JOIN stat_games sg ON sg.id = gp.game_id
				GROUP BY gp.game_id
			)
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
				ss.primary_screenshot,
				COALESCE(ss.screenshot_count, 0) AS screenshot_count,
				COALESCE(fs.file_count, 0) AS file_count,
				COALESCE(ds.developer_count, 0) AS developer_count,
				COALESCE(ps.publisher_count, 0) AS publisher_count,
				COALESCE(pls.platform_count, 0) AS platform_count,
				g.created_at,
				g.updated_at
			FROM stat_games sg
			INNER JOIN games g ON g.id = sg.id
			LEFT JOIN screenshot_stats ss ON ss.game_id = g.id
			LEFT JOIN file_stats fs ON fs.game_id = g.id
			LEFT JOIN developer_stats ds ON ds.game_id = g.id
			LEFT JOIN publisher_stats ps ON ps.game_id = g.id
			LEFT JOIN platform_stats pls ON pls.game_id = g.id
			ORDER BY %s
		`, baseWhere, orderBy, orderBy)
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

func newGamePublicID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fallbackGamePublicID()
	}

	// UUIDv4 bits.
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80

	hexText := hex.EncodeToString(buf)
	return fmt.Sprintf(
		"%s-%s-%s-%s-%s",
		hexText[0:8],
		hexText[8:12],
		hexText[12:16],
		hexText[16:20],
		hexText[20:32],
	)
}

func fallbackGamePublicID() string {
	now := time.Now().UnixNano()
	sequence := atomic.AddUint64(&fallbackPublicIDCounter, 1)
	return fmt.Sprintf(
		"f%07x-%04x-4%03x-a%03x-%010x%02x",
		now&0x0fffffff,
		now&0xffff,
		now&0x0fff,
		now&0x0fff,
		now&0x0fffffffff,
		sequence&0xff,
	)
}

func newPendingFieldIssue(key string, group string, fieldExpr string) pendingIssueDefinition {
	condition := pendingMissingFieldCondition(fieldExpr)
	return pendingIssueDefinition{
		Key:              key,
		Group:            group,
		AnyCondition:     condition,
		VisibleCondition: pendingVisibleIssueCondition(condition, key),
	}
}

func newPendingRelationIssue(key string, group string, table string, predicate string) pendingIssueDefinition {
	condition := pendingMissingRelationCondition(table, predicate)
	return pendingIssueDefinition{
		Key:              key,
		Group:            group,
		AnyCondition:     condition,
		VisibleCondition: pendingVisibleIssueCondition(condition, key),
	}
}

func newPendingWikiIssue() pendingIssueDefinition {
	condition := pendingMissingWikiCondition()
	return pendingIssueDefinition{
		Key:              string(domain.PendingIssueDetailMissingWikiContent),
		Group:            string(domain.PendingIssueMissingWiki),
		AnyCondition:     condition,
		VisibleCondition: pendingVisibleIssueCondition(condition, string(domain.PendingIssueDetailMissingWikiContent)),
	}
}

func pendingMissingFieldCondition(fieldExpr string) string {
	return fmt.Sprintf("COALESCE(TRIM(%s), '') = ''", fieldExpr)
}

func pendingMissingRelationCondition(table string, predicate string) string {
	return fmt.Sprintf("NOT EXISTS (SELECT 1 FROM %s WHERE %s)", table, predicate)
}

func pendingMissingWikiCondition() string {
	return "COALESCE(TRIM(g.wiki_content), '') = ''"
}

func pendingVisibleIssueCondition(condition string, issueKey string) string {
	return fmt.Sprintf("(%s AND %s)", condition, pendingIssueNotIgnoredCondition(issueKey))
}

func pendingIssueNotIgnoredCondition(issueKey string) string {
	return fmt.Sprintf(
		"NOT EXISTS (SELECT 1 FROM game_review_issue_overrides gio WHERE gio.game_id = g.id AND gio.issue_key = '%s' AND gio.status = 'ignored')",
		issueKey,
	)
}

func pendingAnyIssueCondition(includeIgnored bool) string {
	conditions := make([]string, 0, len(pendingIssueDefinitions))
	for _, definition := range pendingIssueDefinitions {
		if includeIgnored {
			conditions = append(conditions, definition.AnyCondition)
			continue
		}
		conditions = append(conditions, definition.VisibleCondition)
	}
	return strings.Join(conditions, " OR ")
}

func pendingIssueConditionsForFilter(filterKey string, includeIgnored bool) []string {
	conditions := make([]string, 0)
	for _, definition := range pendingIssueDefinitions {
		if definition.Key != filterKey && definition.Group != filterKey {
			continue
		}
		if includeIgnored {
			conditions = append(conditions, definition.AnyCondition)
		} else {
			conditions = append(conditions, definition.VisibleCondition)
		}
	}
	return conditions
}

func pendingGroupCondition(groupKey string, includeIgnored bool) string {
	conditions := pendingIssueConditionsForFilter(groupKey, includeIgnored)
	if len(conditions) == 0 {
		return "0 = 1"
	}
	return "(" + strings.Join(conditions, " OR ") + ")"
}

func pendingVisibleIssueCountExpression() string {
	parts := make([]string, 0, len(pendingIssueDefinitions))
	for _, definition := range pendingIssueDefinitions {
		parts = append(parts, fmt.Sprintf("CASE WHEN %s THEN 1 ELSE 0 END", definition.VisibleCondition))
	}
	return "(" + strings.Join(parts, " + ") + ")"
}

func pendingSevereCondition() string {
	missingFiles := strings.Join(pendingIssueConditionsForFilter(string(domain.PendingIssueMissingFiles), false), " OR ")
	missingAssets := strings.Join(pendingIssueConditionsForFilter(string(domain.PendingIssueMissingAssets), false), " OR ")
	missingWiki := strings.Join(pendingIssueConditionsForFilter(string(domain.PendingIssueMissingWiki), false), " OR ")
	return fmt.Sprintf(
		"(%s >= 3 OR (%s) OR ((%s) AND (%s)))",
		pendingVisibleIssueCountExpression(),
		missingFiles,
		missingAssets,
		missingWiki,
	)
}
