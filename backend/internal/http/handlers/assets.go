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

		sortOrder := 0
		if raw := c.PostForm("sort_order"); raw != "" {
			if parsed, parseErr := strconv.Atoi(raw); parseErr == nil && parsed >= 0 {
				sortOrder = parsed
			}
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
