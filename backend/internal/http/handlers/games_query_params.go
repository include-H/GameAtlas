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
	limit, ok := parseGamesListIntQuery(c, "limit", 20)
	if !ok {
		return domain.GamesListParams{}, false
	}
	seriesID, ok := parseGamesListInt64Query(c, "series", 0)
	if !ok {
		return domain.GamesListParams{}, false
	}
	platformID, ok := parseGamesListInt64Query(c, "platform", 0)
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
		PlatformID:        platformID,
		TagIDs:            tagIDs,
		PendingIssue:      strings.TrimSpace(c.Query("pending_issue")),
		PendingRecentDays: pendingRecentDays,
		Sort:              c.Query("sort"),
		Order:             c.Query("order"),
		SortSeed:          sortSeed,
		IncludeAll:        isAdminRequest(c),
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
		params.FavoriteOnly = value
	}

	return params, true
}

func decodeGamesTimelineRequest(c *gin.Context) (int, string, string, string, int64, bool) {
	years := parseQueryInt(c, "years", 2)
	if years <= 0 {
		years = 2
	}
	if years > 10 {
		years = 10
	}

	cursorReleaseDate, cursorID, ok := parseTimelineCursor(c.Query("cursor"))
	if !ok {
		return 0, "", "", "", 0, false
	}

	return years, c.Query("from"), c.Query("to"), cursorReleaseDate, cursorID, true
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
		"error":   "invalid games query parameter: " + key,
	})
}
