package handlers

import (
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/services"
)

type GamesHandler struct {
	service *services.GamesService
}

func NewGamesHandler(service *services.GamesService) *GamesHandler {
	return &GamesHandler{service: service}
}

func (h *GamesHandler) List(c *gin.Context) {
	params := domain.GamesListParams{
		Page:       parseQueryInt(c, "page", 1),
		Limit:      parseQueryInt(c, "limit", 20),
		Search:     c.Query("search"),
		SeriesID:   parseQueryInt64(c, "series", 0),
		PlatformID: parseQueryInt64(c, "platform", 0),
		TagIDs:     parseQueryInt64List(c, "tag"),
		Sort:       c.Query("sort"),
		Order:      c.Query("order"),
		SortSeed:   parseQueryInt64(c, "seed", 0),
		IncludeAll: isAdminRequest(c),
	}

	if raw := c.Query("needs_review"); raw != "" {
		if value, err := strconv.ParseBool(raw); err == nil {
			params.NeedsReview = &value
		}
	}
	if raw := c.Query("pending"); raw != "" {
		if value, err := strconv.ParseBool(raw); err == nil {
			params.PendingOnly = value
		}
	}

	result, err := h.service.List(params)
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
			"page":       result.Page,
			"limit":      result.Limit,
			"total":      result.Total,
			"totalPages": result.TotalPages,
		},
	})
}

func (h *GamesHandler) ListTimeline(c *gin.Context) {
	years := parseQueryInt(c, "years", 2)
	if years <= 0 {
		years = 2
	}
	if years > 10 {
		years = 10
	}

	now := time.Now().UTC()
	fromDate := c.Query("from")
	toDate := c.Query("to")
	cursorReleaseDate, cursorID, ok := parseTimelineCursor(c.Query("cursor"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid timeline cursor",
		})
		return
	}

	if strings.TrimSpace(toDate) == "" {
		latestReleaseDate, ok, err := h.service.LatestTimelineReleaseDate(isAdminRequest(c))
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

	result, err := h.service.ListTimeline(params)
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

	stats, err := h.service.Stats(params)
	if err != nil {
		writeServiceError(c, err, "invalid game payload")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"total_games":     stats.TotalGames,
			"total_downloads": stats.TotalDownloads,
			"total_size":      stats.TotalSize,
			"recent_games":    toGameListItemResponses(stats.RecentGames),
			"popular_games":   toGameListItemResponses(stats.PopularGames),
			"pending_reviews": stats.PendingReviews,
		},
	})
}

func (h *GamesHandler) Get(c *gin.Context) {
	id, ok := parseGamePublicIDParam(c, "publicId", h.service.ResolveGameID)
	if !ok {
		return
	}

	detail, err := h.service.GetDetail(id, isAdminRequest(c))
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
	var input domain.GameWriteInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid game payload",
		})
		return
	}

	game, err := h.service.Create(input)
	if err != nil {
		writeServiceError(c, err, "title is required")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    toGameListItemResponse(*game),
	})
}

func (h *GamesHandler) Update(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	id, ok := parseGamePublicIDParam(c, "publicId", h.service.ResolveGameID)
	if !ok {
		return
	}

	var input domain.GameWriteInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid game payload",
		})
		return
	}

	game, err := h.service.Update(id, input)
	if err != nil {
		writeServiceError(c, err, "title is required")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    toGameListItemResponse(*game),
	})
}

type gameAggregateUpdateRequest struct {
	Title                    string                        `json:"title"`
	TitleAlt                 *string                       `json:"title_alt"`
	Visibility               string                        `json:"visibility"`
	Summary                  *string                       `json:"summary"`
	ReleaseDate              *string                       `json:"release_date"`
	Engine                   *string                       `json:"engine"`
	CoverImage               *string                       `json:"cover_image"`
	BannerImage              *string                       `json:"banner_image"`
	NeedsReview              bool                          `json:"needs_review"`
	SeriesIDs                []int64                       `json:"series_ids"`
	PlatformIDs              []int64                       `json:"platform_ids"`
	DeveloperIDs             []int64                       `json:"developer_ids"`
	PublisherIDs             []int64                       `json:"publisher_ids"`
	TagIDs                   []int64                       `json:"tag_ids"`
	PreviewVideoAssetUID     *string                       `json:"preview_video_asset_uid"`
	Files                    []domain.GameFileUpsertInput  `json:"files"`
	DeleteAssets             []domain.GameAssetDeleteInput `json:"delete_assets"`
	ScreenshotOrderAssetUIDs []string                      `json:"screenshot_order_asset_uids"`
	VideoOrderAssetUIDs      []string                      `json:"video_order_asset_uids"`
}

func (h *GamesHandler) UpdateAggregate(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	id, ok := parseGamePublicIDParam(c, "publicId", h.service.ResolveGameID)
	if !ok {
		return
	}

	var request gameAggregateUpdateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid game payload",
		})
		return
	}

	game, deleteWarnings, err := h.service.UpdateAggregate(id, domain.GameAggregateUpdateInput{
		Game: domain.GameWriteInput{
			Title:                request.Title,
			TitleAlt:             request.TitleAlt,
			Visibility:           request.Visibility,
			Summary:              request.Summary,
			ReleaseDate:          request.ReleaseDate,
			Engine:               request.Engine,
			CoverImage:           request.CoverImage,
			BannerImage:          request.BannerImage,
			NeedsReview:          request.NeedsReview,
			SeriesIDs:            request.SeriesIDs,
			PlatformIDs:          request.PlatformIDs,
			DeveloperIDs:         request.DeveloperIDs,
			PublisherIDs:         request.PublisherIDs,
			TagIDs:               request.TagIDs,
			PreviewVideoAssetUID: request.PreviewVideoAssetUID,
		},
		Files:                    request.Files,
		DeleteAssets:             request.DeleteAssets,
		ScreenshotOrderAssetUIDs: request.ScreenshotOrderAssetUIDs,
		VideoOrderAssetUIDs:      request.VideoOrderAssetUIDs,
	})
	if err != nil {
		writeServiceError(c, err, "title is required")
		return
	}

	data := gin.H{
		"game": toGameListItemResponse(*game),
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
	id, ok := parseGamePublicIDParam(c, "publicId", h.service.ResolveGameID)
	if !ok {
		return
	}

	if err := h.service.Delete(id); err != nil {
		writeServiceError(c, err, "invalid game payload")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"deleted": true,
		},
	})
}

func parseQueryInt(c *gin.Context, key string, fallback int) int {
	raw := c.Query(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func parseQueryInt64(c *gin.Context, key string, fallback int64) int64 {
	raw := c.Query(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return fallback
	}
	return value
}

type gameListItemResponse struct {
	ID                int64   `json:"id"`
	PublicID          string  `json:"public_id"`
	Title             string  `json:"title"`
	TitleAlt          *string `json:"title_alt"`
	Visibility        string  `json:"visibility"`
	Summary           *string `json:"summary"`
	ReleaseDate       *string `json:"release_date"`
	Engine            *string `json:"engine"`
	CoverImage        *string `json:"cover_image"`
	BannerImage       *string `json:"banner_image"`
	NeedsReview       bool    `json:"needs_review"`
	Downloads         int64   `json:"downloads"`
	PrimaryScreenshot *string `json:"primary_screenshot"`
	ScreenshotCount   int64   `json:"screenshot_count"`
	FileCount         int64   `json:"file_count"`
	DeveloperCount    int64   `json:"developer_count"`
	PublisherCount    int64   `json:"publisher_count"`
	PlatformCount     int64   `json:"platform_count"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
}

type timelineGameItemResponse struct {
	ID          int64   `json:"id"`
	PublicID    string  `json:"public_id"`
	Title       string  `json:"title"`
	ReleaseDate *string `json:"release_date"`
	CoverImage  *string `json:"cover_image"`
}

type gameAssetResponse struct {
	ID        int64  `json:"id"`
	AssetUID  string `json:"asset_uid"`
	Path      string `json:"path"`
	SortOrder int    `json:"sort_order"`
}

type metadataItemResponse struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	SortOrder int    `json:"sort_order"`
	CreatedAt string `json:"created_at"`
}

type gameFileResponse struct {
	ID              int64   `json:"id"`
	GameID          int64   `json:"game_id"`
	FileName        string  `json:"file_name"`
	FilePath        string  `json:"file_path,omitempty"`
	Label           *string `json:"label"`
	Notes           *string `json:"notes"`
	SizeBytes       *int64  `json:"size_bytes"`
	SortOrder       int     `json:"sort_order"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
	SourceCreatedAt *string `json:"source_created_at"`
}

type gameDetailResponse struct {
	ID              int64                  `json:"id"`
	PublicID        string                 `json:"public_id"`
	Title           string                 `json:"title"`
	TitleAlt        *string                `json:"title_alt"`
	Visibility      string                 `json:"visibility"`
	Summary         *string                `json:"summary"`
	ReleaseDate     *string                `json:"release_date"`
	Engine          *string                `json:"engine"`
	CoverImage      *string                `json:"cover_image"`
	BannerImage     *string                `json:"banner_image"`
	WikiContent     *string                `json:"wiki_content"`
	WikiContentHTML *string                `json:"wiki_content_html"`
	NeedsReview     bool                   `json:"needs_review"`
	Downloads       int64                  `json:"downloads"`
	PreviewVideo    *gameAssetResponse     `json:"preview_video"`
	PreviewVideos   []gameAssetResponse    `json:"preview_videos"`
	Screenshots     []gameAssetResponse    `json:"screenshots"`
	Series          []metadataItemResponse `json:"series"`
	Platforms       []metadataItemResponse `json:"platforms"`
	Developers      []metadataItemResponse `json:"developers"`
	Publishers      []metadataItemResponse `json:"publishers"`
	Tags            []tagResponse          `json:"tags"`
	TagGroups       []gameTagGroupResponse `json:"tag_groups"`
	Files           []gameFileResponse     `json:"files"`
	CreatedAt       string                 `json:"created_at"`
	UpdatedAt       string                 `json:"updated_at"`
}

type tagResponse struct {
	ID        int64  `json:"id"`
	GroupID   int64  `json:"group_id"`
	GroupKey  string `json:"group_key"`
	GroupName string `json:"group_name"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	ParentID  *int64 `json:"parent_id"`
	SortOrder int    `json:"sort_order"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type gameTagGroupResponse struct {
	ID            int64         `json:"id"`
	Key           string        `json:"key"`
	Name          string        `json:"name"`
	AllowMultiple bool          `json:"allow_multiple"`
	IsFilterable  bool          `json:"is_filterable"`
	Tags          []tagResponse `json:"tags"`
}

func toGameListItemResponse(game domain.Game) gameListItemResponse {
	return gameListItemResponse{
		ID:                game.ID,
		PublicID:          game.PublicID,
		Title:             game.Title,
		TitleAlt:          game.TitleAlt,
		Visibility:        game.Visibility,
		Summary:           game.Summary,
		ReleaseDate:       game.ReleaseDate,
		Engine:            game.Engine,
		CoverImage:        game.CoverImage,
		BannerImage:       game.BannerImage,
		NeedsReview:       game.NeedsReview,
		Downloads:         game.Downloads,
		PrimaryScreenshot: game.PrimaryScreenshot,
		ScreenshotCount:   game.ScreenshotCount,
		FileCount:         game.FileCount,
		DeveloperCount:    game.DeveloperCount,
		PublisherCount:    game.PublisherCount,
		PlatformCount:     game.PlatformCount,
		CreatedAt:         game.CreatedAt,
		UpdatedAt:         game.UpdatedAt,
	}
}

func toTimelineGameItemResponse(game domain.TimelineGame) timelineGameItemResponse {
	return timelineGameItemResponse{
		ID:          game.ID,
		PublicID:    game.PublicID,
		Title:       game.Title,
		ReleaseDate: game.ReleaseDate,
		CoverImage:  game.CoverImage,
	}
}

func toGameListItemResponses(games []domain.Game) []gameListItemResponse {
	result := make([]gameListItemResponse, 0, len(games))
	for _, game := range games {
		result = append(result, toGameListItemResponse(game))
	}
	return result
}

func parseTimelineCursor(raw string) (string, int64, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", 0, true
	}

	parts := strings.Split(trimmed, "|")
	if len(parts) != 2 {
		return "", 0, false
	}

	releaseDate := strings.TrimSpace(parts[0])
	id, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
	if err != nil || id <= 0 {
		return "", 0, false
	}

	return releaseDate, id, true
}

func formatTimelineCursor(releaseDate string, id int64) string {
	return releaseDate + "|" + strconv.FormatInt(id, 10)
}

func toGameDetailResponse(detail *services.GameDetail, includePaths bool) gameDetailResponse {
	screenshots := make([]gameAssetResponse, 0, len(detail.Screenshots))
	for _, asset := range detail.Screenshots {
		screenshots = append(screenshots, gameAssetResponse{
			ID:        asset.ID,
			AssetUID:  asset.AssetUID,
			Path:      asset.Path,
			SortOrder: asset.SortOrder,
		})
	}

	var previewVideo *gameAssetResponse
	if detail.PreviewVideo != nil {
		previewVideo = &gameAssetResponse{
			ID:        detail.PreviewVideo.ID,
			AssetUID:  detail.PreviewVideo.AssetUID,
			Path:      detail.PreviewVideo.Path,
			SortOrder: detail.PreviewVideo.SortOrder,
		}
	}

	previewVideos := make([]gameAssetResponse, 0, len(detail.PreviewVideos))
	for _, asset := range detail.PreviewVideos {
		previewVideos = append(previewVideos, gameAssetResponse{
			ID:        asset.ID,
			AssetUID:  asset.AssetUID,
			Path:      asset.Path,
			SortOrder: asset.SortOrder,
		})
	}

	return gameDetailResponse{
		ID:              detail.Game.ID,
		PublicID:        detail.Game.PublicID,
		Title:           detail.Game.Title,
		TitleAlt:        detail.Game.TitleAlt,
		Visibility:      detail.Game.Visibility,
		Summary:         detail.Game.Summary,
		ReleaseDate:     detail.Game.ReleaseDate,
		Engine:          detail.Game.Engine,
		CoverImage:      detail.Game.CoverImage,
		BannerImage:     detail.Game.BannerImage,
		WikiContent:     detail.Game.WikiContent,
		WikiContentHTML: detail.Game.WikiContentHTML,
		NeedsReview:     detail.Game.NeedsReview,
		Downloads:       detail.Game.Downloads,
		PreviewVideo:    previewVideo,
		PreviewVideos:   previewVideos,
		Screenshots:     screenshots,
		Series:          toMetadataResponses(detail.Series),
		Platforms:       toMetadataResponses(detail.Platforms),
		Developers:      toMetadataResponses(detail.Developers),
		Publishers:      toMetadataResponses(detail.Publishers),
		Tags:            toTagResponses(detail.Tags),
		TagGroups:       toGameTagGroupResponses(detail.TagGroups),
		Files:           toGameFileResponses(detail.Files, includePaths),
		CreatedAt:       detail.Game.CreatedAt,
		UpdatedAt:       detail.Game.UpdatedAt,
	}
}

func toMetadataResponses(items []domain.MetadataItem) []metadataItemResponse {
	result := make([]metadataItemResponse, 0, len(items))
	for _, item := range items {
		result = append(result, metadataItemResponse{
			ID:        item.ID,
			Name:      item.Name,
			Slug:      item.Slug,
			SortOrder: item.SortOrder,
			CreatedAt: item.CreatedAt,
		})
	}
	return result
}

func toGameFileResponses(items []domain.GameFile, includePaths bool) []gameFileResponse {
	result := make([]gameFileResponse, 0, len(items))
	for _, item := range items {
		response := gameFileResponse{
			ID:              item.ID,
			GameID:          item.GameID,
			FileName:        filepath.Base(item.FilePath),
			Label:           item.Label,
			Notes:           item.Notes,
			SizeBytes:       item.SizeBytes,
			SortOrder:       item.SortOrder,
			CreatedAt:       item.CreatedAt,
			UpdatedAt:       item.UpdatedAt,
			SourceCreatedAt: item.SourceCreatedAt,
		}
		if includePaths {
			response.FilePath = item.FilePath
		}
		result = append(result, response)
	}
	return result
}

func toTagResponses(items []domain.Tag) []tagResponse {
	result := make([]tagResponse, 0, len(items))
	for _, item := range items {
		result = append(result, tagResponse{
			ID:        item.ID,
			GroupID:   item.GroupID,
			GroupKey:  item.GroupKey,
			GroupName: item.GroupName,
			Name:      item.Name,
			Slug:      item.Slug,
			ParentID:  item.ParentID,
			SortOrder: item.SortOrder,
			IsActive:  item.IsActive,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
		})
	}
	return result
}

func toGameTagGroupResponses(items []domain.GameTagGroup) []gameTagGroupResponse {
	result := make([]gameTagGroupResponse, 0, len(items))
	for _, item := range items {
		result = append(result, gameTagGroupResponse{
			ID:            item.ID,
			Key:           item.Key,
			Name:          item.Name,
			AllowMultiple: item.AllowMultiple,
			IsFilterable:  item.IsFilterable,
			Tags:          toTagResponses(item.Tags),
		})
	}
	return result
}

func parseQueryInt64List(c *gin.Context, key string) []int64 {
	values := c.QueryArray(key)
	if len(values) == 0 {
		return []int64{}
	}

	result := make([]int64, 0, len(values))
	for _, raw := range values {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value <= 0 {
			continue
		}
		result = append(result, value)
	}
	return result
}
