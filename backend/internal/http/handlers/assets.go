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
			// 2026-05-09: 统一为中文错误信息
			writeJSONError(c, http.StatusBadRequest, "需要有效的 game_id")
			return
		}

		file, err := c.FormFile("file")
		if err != nil {
			// 2026-05-09: 统一为中文错误信息
			writeJSONError(c, http.StatusBadRequest, "需要上传文件")
			return
		}

		sortOrder, ok := parseAssetUploadSortOrder(c)
		if !ok {
			return
		}

		result, err := h.service.Upload(gameID, assetType, file, sortOrder)
		if err != nil {
			// 2026-05-09: 统一为中文错误信息
			writeServiceError(c, err, "无效的资源上传")
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

func (h *AssetsHandler) Delete(assetType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireAdmin(c) {
			return
		}
		gameID, err := strconv.ParseInt(c.Query("game_id"), 10, 64)
		if err != nil || gameID <= 0 {
			writeJSONError(c, http.StatusBadRequest, "需要有效的 game_id")
			return
		}

		assetUID := c.Query("asset_uid")
		if assetUID == "" {
			writeJSONError(c, http.StatusBadRequest, "需要 asset_uid")
			return
		}

		if err := h.service.Delete(gameID, assetType, assetUID); err != nil {
			writeServiceError(c, err, "删除资源失败")
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func parseAssetUploadSortOrder(c *gin.Context) (int, bool) {
	raw := c.PostForm("sort_order")
	if raw == "" {
		return 0, true
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		// 2026-05-09: 统一为中文错误信息
		writeJSONError(c, http.StatusBadRequest, "需要有效的排序值")
		return 0, false
	}

	return value, true
}
