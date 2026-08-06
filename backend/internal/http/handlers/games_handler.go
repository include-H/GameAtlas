package handlers

import (
	"net/http"
	"strings"

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
		// 2026-05-09: 统一为中文错误信息
		writeServiceError(c, err, "无效的游戏请求")
		return
	}

	data := make([]gameListItemResponse, 0, len(result.Games))
	for _, game := range result.Games {
		data = append(data, toGameListItemResponse(game))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
		"pagination": toGamesListPaginationResponse(result, params.PendingOnly),
	})
}

// ListTimeline returns games ordered by release date descending with cursor-based pagination.
func (h *GamesHandler) ListTimeline(c *gin.Context) {
	afterDate, afterID, ok := decodeGamesTimelineRequest(c)
	if !ok {
		return
	}
	limit, ok := parseGamesTimelineIntQuery(c, "limit", 60)
	if !ok {
		return
	}

	params := domain.GamesTimelineParams{
		Limit:      limit,
		AfterDate:  afterDate,
		AfterID:    afterID,
		IncludeAll: isAdminRequest(c),
	}

	result, err := h.timeline.List(params)
	if err != nil {
		writeServiceError(c, err, "无效的时间线参数")
		return
	}

	data := make([]timelineGameItemResponse, 0, len(result.Games))
	for _, game := range result.Games {
		data = append(data, toTimelineGameItemResponse(game))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
		"pagination": gin.H{
			"limit":      result.Limit,
			"hasMore":    result.HasMore,
			"nextCursor": result.NextCursor,
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
		// 2026-05-09: 统一为中文错误信息
		writeServiceError(c, err, "无效的游戏请求")
		return
	}

	writeJSONSuccess(c, http.StatusOK, gin.H{
		"total_games":            stats.TotalGames,
		"total_downloads":        stats.TotalDownloads,
		"recent_games":           toGameListItemResponses(stats.RecentGames),
		"recently_updated_games": toGameListItemResponses(stats.RecentlyUpdatedGames),
		"popular_games":          toGameListItemResponses(stats.PopularGames),
		"favorite_games":         toGameListItemResponses(stats.FavoriteGames),
		"favorite_count":         stats.FavoriteCount,
		"pending_reviews":        stats.PendingReviews,
		"pending_issue_counts":   toPendingIssueCountSummaryResponse(stats.PendingGroups),
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
		// 2026-05-09: 统一为中文错误信息
		writeServiceError(c, err, "无效的游戏请求")
		return
	}

	writeJSONSuccess(c, http.StatusOK, toGameDetailResponse(detail, isAdminRequest(c)))
}

// ListPreviewVideos returns preview videos for a batch of games (game store session).
func (h *GamesHandler) ListPreviewVideos(c *gin.Context) {
	raw := strings.TrimSpace(c.Query("public_ids"))
	if raw == "" {
		writeJSONError(c, http.StatusBadRequest, "无效的游戏请求")
		return
	}

	parts := strings.Split(raw, ",")
	publicIDs := make([]string, 0, len(parts))
	for _, part := range parts {
		if id := strings.TrimSpace(part); id != "" {
			publicIDs = append(publicIDs, id)
		}
	}
	if len(publicIDs) == 0 {
		writeJSONError(c, http.StatusBadRequest, "无效的游戏请求")
		return
	}
	if len(publicIDs) > 100 {
		writeJSONError(c, http.StatusBadRequest, "游戏数量超出上限")
		return
	}

	bundles, err := h.detail.ListPreviewVideos(publicIDs, isAdminRequest(c))
	if err != nil {
		writeServiceError(c, err, "无效的游戏请求")
		return
	}

	writeJSONSuccess(c, http.StatusOK, toGamePreviewVideosResponses(bundles))
}

func (h *GamesHandler) Create(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	var request gameCreateRequest
	if err := decodeJSONStrict(c, &request); err != nil {
		writeJSONError(c, http.StatusBadRequest, "无效的游戏请求")
		return
	}

	input := request.toInput()
	game, err := h.aggregate.Create(input)
	if err != nil {
		// 2026-05-09: 统一为中文错误信息
		writeServiceError(c, err, "标题为必填项")
		return
	}

	writeJSONSuccess(c, http.StatusCreated, toGameSummaryResponse(*game))
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
		writeJSONError(c, http.StatusBadRequest, "无效的游戏请求")
		return
	}

	game, deleteWarnings, err := h.aggregate.Update(id, request.toInput())
	if err != nil {
		// 2026-05-09: 统一为中文错误信息
		writeServiceError(c, err, "标题为必填项")
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

	writeJSONSuccess(c, http.StatusOK, data)
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
		// 2026-05-09: 统一为中文错误信息
		writeServiceError(c, err, "无效的游戏请求")
		return
	}

	data := gin.H{"deleted": true}
	if result != nil && len(result.Warnings) > 0 {
		data["warnings"] = gin.H{
			"asset_delete_paths": result.Warnings,
		}
	}

	writeJSONSuccess(c, http.StatusOK, data)
}

// Favorite/Unfavorite 有意不做 requireAdmin：favorite_games 是单用户/家庭场景下的
// 全局收藏状态，匿名访客也可切换（与"我的收藏"共享同一份数据）。
// 若未来引入多用户或需要管理员独占写权限，应先在这里加门禁并同步前端。
func (h *GamesHandler) Favorite(c *gin.Context) {
	id, ok := parseGamePublicIDParam(c, "publicId", h.favorites.ResolveGameID)
	if !ok {
		return
	}

	isFavorite, err := h.favorites.Set(id, true, isAdminRequest(c))
	if err != nil {
		// 2026-05-09: 统一为中文错误信息
		writeServiceError(c, err, "无效的游戏请求")
		return
	}

	writeJSONSuccess(c, http.StatusOK, gin.H{
		"is_favorite": isFavorite,
	})
}

func (h *GamesHandler) Unfavorite(c *gin.Context) {
	id, ok := parseGamePublicIDParam(c, "publicId", h.favorites.ResolveGameID)
	if !ok {
		return
	}

	isFavorite, err := h.favorites.Set(id, false, isAdminRequest(c))
	if err != nil {
		// 2026-05-09: 统一为中文错误信息
		writeServiceError(c, err, "无效的游戏请求")
		return
	}

	writeJSONSuccess(c, http.StatusOK, gin.H{
		"is_favorite": isFavorite,
	})
}
