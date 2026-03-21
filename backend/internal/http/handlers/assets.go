package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/domain"
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
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "valid game_id is required"})
			return
		}

		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "file is required"})
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

		data := gin.H{
			"path": result.Path,
		}
		if result.AssetID != nil {
			data["asset_id"] = *result.AssetID
		}
		if result.AssetUID != "" {
			data["asset_uid"] = result.AssetUID
		}

		c.JSON(http.StatusCreated, gin.H{
			"success": true,
			"data":    data,
		})
	}
}

func (h *AssetsHandler) Delete(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	var input domain.DeleteAssetInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid asset payload"})
		return
	}

	if err := h.service.Delete(input); err != nil {
		writeServiceError(c, err, "invalid asset payload")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{"deleted": true},
	})
}

func (h *AssetsHandler) ReorderScreenshots(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	var input domain.ScreenshotOrderUpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid screenshot reorder payload"})
		return
	}

	if err := h.service.ReorderScreenshots(input); err != nil {
		writeServiceError(c, err, "invalid screenshot reorder payload")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{"updated": true},
	})
}

func (h *AssetsHandler) ReorderVideos(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	var input domain.VideoOrderUpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid video reorder payload"})
		return
	}

	if err := h.service.ReorderVideos(input); err != nil {
		writeServiceError(c, err, "invalid video reorder payload")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{"updated": true},
	})
}

func (h *AssetsHandler) SetPrimaryVideo(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	var input domain.PrimaryVideoUpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid primary video payload"})
		return
	}
	if err := h.service.SetPrimaryVideo(input); err != nil {
		writeServiceError(c, err, "invalid primary video payload")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{"updated": true},
	})
}
