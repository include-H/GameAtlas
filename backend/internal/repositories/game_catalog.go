package repositories

import (
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/hao/game/internal/domain"
)

type GameCatalogRepository struct {
	games     *GamesRepository
	favorites *FavoriteGamesRepository
}

func NewGameCatalogRepository(games *GamesRepository, favorites *FavoriteGamesRepository) *GameCatalogRepository {
	return &GameCatalogRepository{games: games, favorites: favorites}
}

func (r *GameCatalogRepository) List(params domain.GamesListParams) ([]domain.GameListItem, int, error) {
	where, args, err := r.games.buildGamesListWhere(params, false)
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
	// 2026-05-01: this repository assumes sort/order were already validated by transport/service.
	// Do not restore fallback-to-updated_at or fallback-to-DESC behavior here; invalid callers
	// must fail earlier so unsupported sort semantics stay auditably explicit.
	order := strings.ToUpper(params.Order)
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
	countStmt = r.games.db.Rebind(countStmt)

	var total int
	if err := r.games.db.Get(&total, countStmt, countArgs...); err != nil {
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
		LEFT JOIN favorite_games fg ON fg.game_id = g.id
		LEFT JOIN screenshot_stats ss ON ss.game_id = g.id
		LEFT JOIN logo_stats ls ON ls.game_id = g.id
		LEFT JOIN video_stats vs ON vs.game_id = g.id
		LEFT JOIN file_stats fs ON fs.game_id = g.id
		LEFT JOIN developer_stats ds ON ds.game_id = g.id
		LEFT JOIN publisher_stats ps ON ps.game_id = g.id
		LEFT JOIN series s ON s.id = g.series_id
		ORDER BY %s %s, g.id %s
	`, baseWhere, sortField, order, idOrder, gameListItemStatsCTEs("page_games"), gamesListItemSelectColumns(), sortField, order, idOrder)

	listStmt, listArgs, err := sqlx.Named(listQuery, args)
	if err != nil {
		return nil, 0, fmt.Errorf("build games list query: %w", err)
	}
	listStmt = r.games.db.Rebind(listStmt)

	var games []domain.GameListItem
	if err := r.games.db.Select(&games, listStmt, listArgs...); err != nil {
		return nil, 0, fmt.Errorf("list games: %w", err)
	}

	return games, total, nil
}

func (r *GameCatalogRepository) CountPendingGroups(params domain.GamesListParams) (*domain.PendingGroupCounts, error) {
	where, args, err := r.games.buildGamesListWhere(params, true)
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
	stmt = r.games.db.Rebind(stmt)

	var counts domain.PendingGroupCounts
	if err := r.games.db.Get(&counts, stmt, queryArgs...); err != nil {
		return nil, fmt.Errorf("count pending groups: %w", err)
	}

	return &counts, nil
}

func (r *GameCatalogRepository) Stats(params domain.GamesListParams) (*domain.GameStats, error) {
	where := []string{"1 = 1"}
	args := map[string]any{}

	if !params.IncludeAll {
		where = append(where, "g.visibility = :visibility")
		args["visibility"] = domain.DefaultVisibility(params.Visibility, params.IncludeAll)
	}

	baseWhere := strings.Join(where, " AND ")

	summaryQuery := fmt.Sprintf(`
		SELECT
			COUNT(*) AS total_games,
			COALESCE(SUM(CASE WHEN (%s) THEN 1 ELSE 0 END), 0) AS pending_reviews
		FROM games g
		WHERE %s
	`, pendingAnyIssueCondition(false), baseWhere)

	type statsRow struct {
		TotalGames     int `db:"total_games"`
		PendingReviews int `db:"pending_reviews"`
	}

	summaryStmt, summaryArgs, err := sqlx.Named(summaryQuery, args)
	if err != nil {
		return nil, fmt.Errorf("build games stats query: %w", err)
	}
	summaryStmt = r.games.db.Rebind(summaryStmt)

	var summary statsRow
	if err := r.games.db.Get(&summary, summaryStmt, summaryArgs...); err != nil {
		return nil, fmt.Errorf("get games stats: %w", err)
	}

	loadGames := func(orderBy string, favoriteOnly bool, extraWhere string) ([]domain.GameListItem, error) {
		favoriteJoin := ""
		if favoriteOnly {
			favoriteJoin = "INNER JOIN favorite_games fg ON fg.game_id = g.id"
		}
		where := baseWhere + extraWhere
		query := fmt.Sprintf(`
			WITH stat_games AS (
				SELECT g.id
				FROM games g
				%s
				WHERE %s
				ORDER BY %s
				LIMIT 12
			),
			%s
			SELECT
%s
			FROM stat_games sg
			INNER JOIN games g ON g.id = sg.id
			LEFT JOIN favorite_games fg ON fg.game_id = g.id
			LEFT JOIN screenshot_stats ss ON ss.game_id = g.id
			LEFT JOIN logo_stats ls ON ls.game_id = g.id
			LEFT JOIN video_stats vs ON vs.game_id = g.id
			LEFT JOIN file_stats fs ON fs.game_id = g.id
			LEFT JOIN developer_stats ds ON ds.game_id = g.id
			LEFT JOIN publisher_stats ps ON ps.game_id = g.id
			LEFT JOIN series s ON s.id = g.series_id
			ORDER BY %s
		`, favoriteJoin, where, orderBy, gameListItemStatsCTEs("stat_games"), gamesListItemSelectColumns(), orderBy)
		stmt, queryArgs, err := sqlx.Named(query, args)
		if err != nil {
			return nil, fmt.Errorf("build stats games query: %w", err)
		}
		stmt = r.games.db.Rebind(stmt)

		var games []domain.GameListItem
		if err := r.games.db.Select(&games, stmt, queryArgs...); err != nil {
			return nil, fmt.Errorf("list stats games: %w", err)
		}
		return games, nil
	}

	recentGames, err := loadGames("g.created_at DESC, g.id DESC", false, "")
	if err != nil {
		return nil, err
	}
	// 最近完善：只返回更新明显晚于创建（间隔超过 1 天）的游戏，避免与“最近添加”重复。
	recentlyUpdatedGames, err := loadGames(
		"g.updated_at DESC, g.id DESC",
		false,
		" AND julianday(g.updated_at) - julianday(g.created_at) > 1",
	)
	if err != nil {
		return nil, err
	}
	popularGames, err := loadGames("g.downloads DESC, g.id DESC", false, "")
	if err != nil {
		return nil, err
	}
	favoriteGames, err := loadGames("fg.created_at DESC, g.id DESC", true, "")
	if err != nil {
		return nil, err
	}
	favoriteCount, err := r.favorites.Count(params.IncludeAll, params.Visibility)
	if err != nil {
		return nil, err
	}

	return &domain.GameStats{
		TotalGames:           summary.TotalGames,
		RecentGames:          recentGames,
		RecentlyUpdatedGames: recentlyUpdatedGames,
		PopularGames:         popularGames,
		FavoriteGames:        favoriteGames,
		FavoriteCount:        favoriteCount,
		PendingReviews:       summary.PendingReviews,
	}, nil
}
