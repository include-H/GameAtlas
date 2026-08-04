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
			writeJSONError(c, http.StatusBadRequest, "需要有效的 game_id")
			return
		}

		file, err := c.FormFile("file")
		if err != nil {
			writeJSONError(c, http.StatusBadRequest, "需要上传文件")
			return
		}

		result, err := h.service.Upload(gameID, assetType, file)
		if err != nil {
			writeServiceError(c, err, "无效的资源上传")
			return
		}

		response := assetUploadResponse{
			Path:     result.Path,
			AssetUID: result.AssetUID,
		}

		writeJSONSuccess(c, http.StatusCreated, response)
	}
}
