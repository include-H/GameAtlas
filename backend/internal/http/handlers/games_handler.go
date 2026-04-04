package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/services"
)

// 2026-04-03: split handler actions out of the old games.go transport file.
// Keep this file focused on route actions and service orchestration only.
type GamesHandler struct {
	catalog   *services.GameCatalogService
	timeline  *services.GameTimelineService
	detail    *services.GameDetailService
	aggregate *services.GameAggregateService
	favorites *services.GameFavoriteService
}

// NewSplitGamesHandler keeps HTTP routing aligned with the split application services.
// New games endpoints should depend on the smallest matching service instead of reintroducing
// a single catch-all games service.
func NewSplitGamesHandler(
	catalog *services.GameCatalogService,
	timeline *services.GameTimelineService,
	detail *services.GameDetailService,
	aggregate *services.GameAggregateService,
	favorites *services.GameFavoriteService,
) *GamesHandler {
	return &GamesHandler{
		catalog:   catalog,
		timeline:  timeline,
		detail:    detail,
		aggregate: aggregate,
		favorites: favorites,
	}
}

// List stays on the catalog read model boundary; avoid mixing aggregate write concerns here.
func (h *GamesHandler) List(c *gin.Context) {
	params, ok := decodeGamesListParams(c)
	if !ok {
		return
	}

	result, err := h.catalog.List(params)
	if err != nil {
		writeServiceError(c, err, "invalid game payload")
		return
	}

	data := make([]gameListItemResponse, 0, len(result.Games))
	for _, game := range result.Games {
		data = append(data, toGameListItemResponse(game))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
		"pagination": gin.H{
			"page":                 result.Page,
			"limit":                result.Limit,
			"total":                result.Total,
			"totalPages":           result.TotalPages,
			"pending_issue_counts": toPendingIssueCountSummaryResponse(result.PendingIssueCounts),
		},
	})
}

// ListTimeline is intentionally isolated from catalog/detail queries because it has its own cursor
// and release-date semantics.
func (h *GamesHandler) ListTimeline(c *gin.Context) {
	years, fromDate, toDate, cursorReleaseDate, cursorID, ok := decodeGamesTimelineRequest(c)
	now := time.Now().UTC()
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid timeline cursor",
		})
		return
	}

	if strings.TrimSpace(toDate) == "" {
		latestReleaseDate, ok, err := h.timeline.LatestReleaseDate(isAdminRequest(c))
		if err != nil {
			writeServiceError(c, err, "invalid timeline params")
			return
		}
		if ok {
			toDate = latestReleaseDate
		} else {
			toDate = now.Format("2006-01-02")
		}
	}
	if strings.TrimSpace(fromDate) == "" {
		baseDate, err := time.Parse("2006-01-02", toDate)
		if err != nil {
			baseDate = now
		}
		fromDate = baseDate.AddDate(-years, 0, 0).Format("2006-01-02")
	}

	params := domain.GamesTimelineParams{
		Limit:             parseQueryInt(c, "limit", 60),
		FromDate:          fromDate,
		ToDate:            toDate,
		CursorReleaseDate: cursorReleaseDate,
		CursorID:          cursorID,
		IncludeAll:        isAdminRequest(c),
	}

	result, err := h.timeline.List(params)
	if err != nil {
		writeServiceError(c, err, "invalid timeline params")
		return
	}

	data := make([]timelineGameItemResponse, 0, len(result.Games))
	for _, game := range result.Games {
		data = append(data, toTimelineGameItemResponse(game))
	}

	nextCursor := ""
	if result.NextCursor != nil {
		nextCursor = formatTimelineCursor(result.NextCursor.ReleaseDate, result.NextCursor.ID)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
		"pagination": gin.H{
			"limit":      result.Limit,
			"from":       result.FromDate,
			"to":         result.ToDate,
			"hasMore":    result.HasMore,
			"nextCursor": nextCursor,
		},
	})
}

func (h *GamesHandler) Stats(c *gin.Context) {
	params := domain.GamesListParams{
		Page:       1,
		Limit:      12,
		IncludeAll: isAdminRequest(c),
	}

	stats, err := h.catalog.Stats(params)
	if err != nil {
		writeServiceError(c, err, "invalid game payload")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"total_games":     stats.TotalGames,
			"total_downloads": stats.TotalDownloads,
			"recent_games":    toGameListItemResponses(stats.RecentGames),
			"popular_games":   toGameListItemResponses(stats.PopularGames),
			"favorite_count":  stats.FavoriteCount,
			"pending_reviews": stats.PendingReviews,
		},
	})
}

// Get serves the detail read model and should not absorb aggregate patch or timeline logic.
func (h *GamesHandler) Get(c *gin.Context) {
	id, ok := parseGamePublicIDParam(c, "publicId", h.detail.ResolveGameID)
	if !ok {
		return
	}

	detail, err := h.detail.Get(id, isAdminRequest(c))
	if err != nil {
		writeServiceError(c, err, "invalid game payload")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    toGameDetailResponse(detail, isAdminRequest(c)),
	})
}

func (h *GamesHandler) Create(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	var request gameCreateRequest
	if err := decodeJSONStrict(c, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid game payload",
		})
		return
	}

	input := request.toInput()
	game, err := h.aggregate.Create(input)
	if err != nil {
		writeServiceError(c, err, "title is required")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    toGameSummaryResponse(*game),
	})
}

func (h *GamesHandler) UpdateAggregate(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	id, ok := parseGamePublicIDParam(c, "publicId", h.aggregate.ResolveGameID)
	if !ok {
		return
	}

	var request gameAggregateUpdateRequest
	if err := decodeJSONStrict(c, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid game payload",
		})
		return
	}

	game, deleteWarnings, err := h.aggregate.Update(id, request.toInput())
	if err != nil {
		writeServiceError(c, err, "title is required")
		return
	}

	data := gin.H{
		"game": toGameSummaryResponse(*game),
	}
	if len(deleteWarnings) > 0 {
		data["warnings"] = gin.H{
			"asset_delete_paths": deleteWarnings,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}

func (h *GamesHandler) Delete(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	id, ok := parseGamePublicIDParam(c, "publicId", h.aggregate.ResolveGameID)
	if !ok {
		return
	}

	result, err := h.aggregate.Delete(id)
	if err != nil {
		writeServiceError(c, err, "invalid game payload")
		return
	}

	data := gin.H{"deleted": true}
	if result != nil && len(result.Warnings) > 0 {
		data["warnings"] = gin.H{
			"asset_delete_paths": result.Warnings,
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func (h *GamesHandler) Favorite(c *gin.Context) {
	id, ok := parseGamePublicIDParam(c, "publicId", h.favorites.ResolveGameID)
	if !ok {
		return
	}

	isFavorite, err := h.favorites.Set(id, true)
	if err != nil {
		writeServiceError(c, err, "invalid game payload")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"is_favorite": isFavorite,
		},
	})
}

func (h *GamesHandler) Unfavorite(c *gin.Context) {
	id, ok := parseGamePublicIDParam(c, "publicId", h.favorites.ResolveGameID)
	if !ok {
		return
	}

	isFavorite, err := h.favorites.Set(id, false)
	if err != nil {
		writeServiceError(c, err, "invalid game payload")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"is_favorite": isFavorite,
		},
	})
}
