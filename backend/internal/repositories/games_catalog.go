package repositories

import (
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/hao/game/internal/domain"
)

func (r *GamesRepository) List(params domain.GamesListParams) ([]domain.GameListItem, int, error) {
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
		%s
		SELECT
%s
		FROM page_games pg
		INNER JOIN games g ON g.id = pg.id
		LEFT JOIN screenshot_stats ss ON ss.game_id = g.id
		LEFT JOIN file_stats fs ON fs.game_id = g.id
		LEFT JOIN developer_stats ds ON ds.game_id = g.id
		LEFT JOIN publisher_stats ps ON ps.game_id = g.id
		LEFT JOIN platform_stats pls ON pls.game_id = g.id
		ORDER BY %s %s, g.id %s
	`, baseWhere, sortField, order, idOrder, gameListItemStatsCTEs("page_games"), gamesListItemSelectColumns(), sortField, order, idOrder)

	listStmt, listArgs, err := sqlx.Named(listQuery, args)
	if err != nil {
		return nil, 0, fmt.Errorf("build games list query: %w", err)
	}
	listStmt = r.db.Rebind(listStmt)

	var games []domain.GameListItem
	if err := r.db.Select(&games, listStmt, listArgs...); err != nil {
		return nil, 0, fmt.Errorf("list games: %w", err)
	}

	return games, total, nil
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
		pendingGroupCondition(domain.PendingIssueMissingAssets, params.PendingIncludeIgnored),
		pendingGroupCondition(domain.PendingIssueMissingWiki, params.PendingIncludeIgnored),
		pendingGroupCondition(domain.PendingIssueMissingFiles, params.PendingIncludeIgnored),
		pendingGroupCondition(domain.PendingIssueMissingMetadata, params.PendingIncludeIgnored),
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
			COALESCE(SUM(CASE WHEN (%s) THEN 1 ELSE 0 END), 0) AS pending_reviews
		FROM games g
		WHERE %s
	`, pendingAnyIssueCondition(false), baseWhere)

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

	loadGames := func(orderBy string) ([]domain.GameListItem, error) {
		query := fmt.Sprintf(`
			WITH stat_games AS (
				SELECT g.id
				FROM games g
				WHERE %s
				ORDER BY %s
				LIMIT 12
			),
			%s
			SELECT
%s
			FROM stat_games sg
			INNER JOIN games g ON g.id = sg.id
			LEFT JOIN screenshot_stats ss ON ss.game_id = g.id
			LEFT JOIN file_stats fs ON fs.game_id = g.id
			LEFT JOIN developer_stats ds ON ds.game_id = g.id
			LEFT JOIN publisher_stats ps ON ps.game_id = g.id
			LEFT JOIN platform_stats pls ON pls.game_id = g.id
			ORDER BY %s
		`, baseWhere, orderBy, gameListItemStatsCTEs("stat_games"), gamesListItemSelectColumns(), orderBy)
		stmt, queryArgs, err := sqlx.Named(query, args)
		if err != nil {
			return nil, fmt.Errorf("build stats games query: %w", err)
		}
		stmt = r.db.Rebind(stmt)

		var games []domain.GameListItem
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
