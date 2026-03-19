package handlers

import (
	"net/http"
	"strconv"

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
		Sort:       c.Query("sort"),
		Order:      c.Query("order"),
	}

	if raw := c.Query("needs_review"); raw != "" {
		if value, err := strconv.ParseBool(raw); err == nil {
			params.NeedsReview = &value
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

func (h *GamesHandler) Get(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	detail, err := h.service.GetDetail(id)
	if err != nil {
		writeServiceError(c, err, "invalid game payload")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    toGameDetailResponse(detail),
	})
}

func (h *GamesHandler) Create(c *gin.Context) {
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
	id, ok := parseIDParam(c, "id")
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

func (h *GamesHandler) Delete(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
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
	Title             string  `json:"title"`
	TitleAlt          *string `json:"title_alt"`
	Summary           *string `json:"summary"`
	ReleaseDate       *string `json:"release_date"`
	Engine            *string `json:"engine"`
	CoverImage        *string `json:"cover_image"`
	BannerImage       *string `json:"banner_image"`
	NeedsReview       bool    `json:"needs_review"`
	Views             int64   `json:"views"`
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
	ID        int64   `json:"id"`
	GameID    int64   `json:"game_id"`
	FilePath  string  `json:"file_path"`
	Label     *string `json:"label"`
	Notes     *string `json:"notes"`
	SizeBytes *int64  `json:"size_bytes"`
	SortOrder int     `json:"sort_order"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

type gameDetailResponse struct {
	ID              int64                  `json:"id"`
	Title           string                 `json:"title"`
	TitleAlt        *string                `json:"title_alt"`
	Summary         *string                `json:"summary"`
	ReleaseDate     *string                `json:"release_date"`
	Engine          *string                `json:"engine"`
	CoverImage      *string                `json:"cover_image"`
	BannerImage     *string                `json:"banner_image"`
	WikiContent     *string                `json:"wiki_content"`
	WikiContentHTML *string                `json:"wiki_content_html"`
	NeedsReview     bool                   `json:"needs_review"`
	Views           int64                  `json:"views"`
	Downloads       int64                  `json:"downloads"`
	PreviewVideo    *gameAssetResponse     `json:"preview_video"`
	PreviewVideos   []gameAssetResponse    `json:"preview_videos"`
	Screenshots     []gameAssetResponse    `json:"screenshots"`
	Series          []metadataItemResponse `json:"series"`
	Platforms       []metadataItemResponse `json:"platforms"`
	Developers      []metadataItemResponse `json:"developers"`
	Publishers      []metadataItemResponse `json:"publishers"`
	Files           []gameFileResponse     `json:"files"`
	CreatedAt       string                 `json:"created_at"`
	UpdatedAt       string                 `json:"updated_at"`
}

func toGameListItemResponse(game domain.Game) gameListItemResponse {
	return gameListItemResponse{
		ID:                game.ID,
		Title:             game.Title,
		TitleAlt:          game.TitleAlt,
		Summary:           game.Summary,
		ReleaseDate:       game.ReleaseDate,
		Engine:            game.Engine,
		CoverImage:        game.CoverImage,
		BannerImage:       game.BannerImage,
		NeedsReview:       game.NeedsReview,
		Views:             game.Views,
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

func toGameDetailResponse(detail *services.GameDetail) gameDetailResponse {
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
		Title:           detail.Game.Title,
		TitleAlt:        detail.Game.TitleAlt,
		Summary:         detail.Game.Summary,
		ReleaseDate:     detail.Game.ReleaseDate,
		Engine:          detail.Game.Engine,
		CoverImage:      detail.Game.CoverImage,
		BannerImage:     detail.Game.BannerImage,
		WikiContent:     detail.Game.WikiContent,
		WikiContentHTML: detail.Game.WikiContentHTML,
		NeedsReview:     detail.Game.NeedsReview,
		Views:           detail.Game.Views,
		Downloads:       detail.Game.Downloads,
		PreviewVideo:    previewVideo,
		PreviewVideos:   previewVideos,
		Screenshots:     screenshots,
		Series:          toMetadataResponses(detail.Series),
		Platforms:       toMetadataResponses(detail.Platforms),
		Developers:      toMetadataResponses(detail.Developers),
		Publishers:      toMetadataResponses(detail.Publishers),
		Files:           toGameFileResponses(detail.Files),
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

func toGameFileResponses(items []domain.GameFile) []gameFileResponse {
	result := make([]gameFileResponse, 0, len(items))
	for _, item := range items {
		result = append(result, gameFileResponse{
			ID:        item.ID,
			GameID:    item.GameID,
			FilePath:  item.FilePath,
			Label:     item.Label,
			Notes:     item.Notes,
			SizeBytes: item.SizeBytes,
			SortOrder: item.SortOrder,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
		})
	}
	return result
}
