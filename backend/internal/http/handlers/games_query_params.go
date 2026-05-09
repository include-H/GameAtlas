package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/domain"
)

func decodeGamesListParams(c *gin.Context) (domain.GamesListParams, bool) {
	page, ok := parseGamesListIntQuery(c, "page", 1)
	if !ok {
		return domain.GamesListParams{}, false
	}
	if page <= 0 {
		page = 1
	}
	limit, ok := parseGamesListIntQuery(c, "limit", 20)
	if !ok {
		return domain.GamesListParams{}, false
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	seriesID, ok := parseGamesListInt64Query(c, "series", 0)
	if !ok {
		return domain.GamesListParams{}, false
	}
	tagIDs, ok := parseGamesListInt64List(c, "tag")
	if !ok {
		return domain.GamesListParams{}, false
	}
	pendingRecentDays, ok := parseGamesListIntQuery(c, "pending_recent_days", 0)
	if !ok {
		return domain.GamesListParams{}, false
	}
	sortSeed, ok := parseGamesListInt64Query(c, "seed", 0)
	if !ok {
		return domain.GamesListParams{}, false
	}

	params := domain.GamesListParams{
		Page:              page,
		Limit:             limit,
		Search:            c.Query("search"),
		SeriesID:          seriesID,
		TagIDs:            tagIDs,
		PendingIssue:      strings.TrimSpace(c.Query("pending_issue")),
		PendingRecentDays: pendingRecentDays,
		Sort:              strings.TrimSpace(c.Query("sort")),
		Order:             strings.TrimSpace(c.Query("order")),
		SortSeed:          sortSeed,
		IncludeAll:        isAdminRequest(c),
	}

	// 2026-05-01: keep list query defaults and transport validation at decode time so
	// handler-visible semantics stay explicit and the catalog service no longer hides
	// page/limit/sort/order/pending_issue fallback behavior for HTTP callers. This is the
	// only layer that may supply default sort/order for the HTTP contract; deeper layers
	// must reject bad values instead of silently rewriting them.
	if params.Sort == "" {
		params.Sort = "updated_at"
	} else if !domain.IsAllowedGamesListSort(params.Sort) {
		writeGamesListQueryError(c, "sort")
		return domain.GamesListParams{}, false
	}
	if params.Order == "" {
		params.Order = "desc"
	} else if !domain.IsAllowedGamesListOrder(params.Order) {
		writeGamesListQueryError(c, "order")
		return domain.GamesListParams{}, false
	}
	if params.PendingIssue != "" && !domain.IsAllowedPendingIssueFilter(params.PendingIssue) {
		writeGamesListQueryError(c, "pending_issue")
		return domain.GamesListParams{}, false
	}

	// 2026-04-04: random list order now requires an explicit transport seed.
	// Impact: the frontend-owned route/query state remains the only place that creates this seed,
	// so service/repository code no longer carries hidden fallback randomness.
	if params.Sort == "random" && params.SortSeed <= 0 {
		writeGamesListQueryError(c, "seed")
		return domain.GamesListParams{}, false
	}

	if raw := c.Query("pending"); raw != "" {
		value, ok := parseGamesListBoolQuery(c, "pending")
		if !ok {
			return domain.GamesListParams{}, false
		}
		params.PendingOnly = value
	}
	if raw := c.Query("pending_include_ignored"); raw != "" {
		value, ok := parseGamesListBoolQuery(c, "pending_include_ignored")
		if !ok {
			return domain.GamesListParams{}, false
		}
		params.PendingIncludeIgnored = value
	}
	if raw := c.Query("pending_severe"); raw != "" {
		value, ok := parseGamesListBoolQuery(c, "pending_severe")
		if !ok {
			return domain.GamesListParams{}, false
		}
		params.PendingSevereOnly = value
	}
	if raw := c.Query("favorite"); raw != "" {
		value, ok := parseGamesListBoolQuery(c, "favorite")
		if !ok {
			return domain.GamesListParams{}, false
		}
		// 2026-05-01: games list transport only accepts favorite=true as a valid filter.
		// Impact: favorite=false is not a negative predicate; reject it here instead of
		// letting repository semantics silently collapse it into the same behavior as "missing".
		if !value {
			writeGamesListQueryError(c, "favorite")
			return domain.GamesListParams{}, false
		}
		params.FavoriteOnly = value
	}

	return params, true
}

func decodeGamesTimelineRequest(c *gin.Context) (int, string, string, string, int64, bool) {
	years, ok := parseGamesTimelineIntQuery(c, "years", 2)
	if !ok {
		return 0, "", "", "", 0, false
	}
	if years <= 0 {
		writeGamesTimelineQueryError(c, "years")
		return 0, "", "", "", 0, false
	}
	if years > 10 {
		years = 10
	}

	cursorReleaseDate, cursorID, ok := parseTimelineCursor(c.Query("cursor"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			// 2026-05-09: 统一为中文错误信息
		"error":   "无效的时间线游标",
		})
		return 0, "", "", "", 0, false
	}

	return years, c.Query("from"), c.Query("to"), cursorReleaseDate, cursorID, true
}

func parseGamesTimelineIntQuery(c *gin.Context, key string, fallback int) (int, bool) {
	raw := c.Query(key)
	if raw == "" {
		return fallback, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		writeGamesTimelineQueryError(c, key)
		return 0, false
	}
	return value, true
}

func writeGamesTimelineQueryError(c *gin.Context, key string) {
	c.JSON(http.StatusBadRequest, gin.H{
		"success": false,
		// 2026-05-09: 统一为中文错误信息
		"error":   "无效的时间线查询参数: " + key,
	})
}

func parseGamesListInt64List(c *gin.Context, key string) ([]int64, bool) {
	values := c.QueryArray(key)
	if len(values) == 0 {
		return []int64{}, true
	}

	result := make([]int64, 0, len(values))
	for _, raw := range values {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value <= 0 {
			writeGamesListQueryError(c, key)
			return nil, false
		}
		result = append(result, value)
	}
	return result, true
}

func parseGamesListIntQuery(c *gin.Context, key string, fallback int) (int, bool) {
	raw := c.Query(key)
	if raw == "" {
		return fallback, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		writeGamesListQueryError(c, key)
		return 0, false
	}
	return value, true
}

func parseGamesListInt64Query(c *gin.Context, key string, fallback int64) (int64, bool) {
	raw := c.Query(key)
	if raw == "" {
		return fallback, true
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		writeGamesListQueryError(c, key)
		return 0, false
	}
	return value, true
}

func parseGamesListBoolQuery(c *gin.Context, key string) (bool, bool) {
	value, err := strconv.ParseBool(c.Query(key))
	if err != nil {
		writeGamesListQueryError(c, key)
		return false, false
	}
	return value, true
}

func writeGamesListQueryError(c *gin.Context, key string) {
	c.JSON(http.StatusBadRequest, gin.H{
		"success": false,
		// 2026-05-09: 统一为中文错误信息
		"error":   "无效的游戏查询参数: " + key,
	})
}
