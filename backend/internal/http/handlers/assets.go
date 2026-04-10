package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/services"
)

type AssetsHandler struct {
	service *services.AssetsService
}

func NewAssetsHandler(service *services.AssetsService) *AssetsHandler {
	return &AssetsHandler{service: service}
}

func (h *AssetsHandler) Upload(assetType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireAdmin(c) {
			return
		}
		gameID, err := strconv.ParseInt(c.PostForm("game_id"), 10, 64)
		if err != nil || gameID <= 0 {
			writeJSONError(c, http.StatusBadRequest, "valid game_id is required")
			return
		}

		file, err := c.FormFile("file")
		if err != nil {
			writeJSONError(c, http.StatusBadRequest, "file is required")
			return
		}

		sortOrder, ok := parseAssetUploadSortOrder(c)
		if !ok {
			return
		}

		result, err := h.service.Upload(gameID, assetType, file, sortOrder)
		if err != nil {
			writeServiceError(c, err, "invalid asset upload")
			return
		}

		response := assetUploadResponse{
			Path: result.Path,
		}
		if result.AssetID != nil {
			response.AssetID = result.AssetID
		}
		if result.AssetUID != "" {
			response.AssetUID = result.AssetUID
		}

		writeJSONSuccess(c, http.StatusCreated, response)
	}
}

func parseAssetUploadSortOrder(c *gin.Context) (int, bool) {
	raw := c.PostForm("sort_order")
	if raw == "" {
		return 0, true
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		writeJSONError(c, http.StatusBadRequest, "valid sort_order is required")
		return 0, false
	}

	return value, true
}
