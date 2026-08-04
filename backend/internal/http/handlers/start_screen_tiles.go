package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/services"
)

type StartScreenTilesHandler struct {
	service *services.StartScreenTilesService
}

func NewStartScreenTilesHandler(service *services.StartScreenTilesService) *StartScreenTilesHandler {
	return &StartScreenTilesHandler{service: service}
}

func (h *StartScreenTilesHandler) Get(c *gin.Context) {
	tiles, err := h.service.List()
	if err != nil {
		writeServiceError(c, err, "获取开始屏幕磁贴失败")
		return
	}
	writeJSONSuccess(c, http.StatusOK, toStartScreenTileResponses(tiles))
}

func (h *StartScreenTilesHandler) Update(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}

	var request struct {
		Tiles []struct {
			GameID         int64   `json:"game_id"`
			TileSize       string  `json:"tile_size"`
			ImageSmallPath *string `json:"image_small_path"`
			ImageWidePath  *string `json:"image_wide_path"`
			ImageLargePath *string `json:"image_large_path"`
		} `json:"tiles"`
	}
	if err := decodeJSONStrict(c, &request); err != nil {
		writeJSONError(c, http.StatusBadRequest, "无效的开始屏幕磁贴数据")
		return
	}

	tiles := make([]domain.StartScreenTileWrite, 0, len(request.Tiles))
	for _, item := range request.Tiles {
		tiles = append(tiles, domain.StartScreenTileWrite{
			GameID:         item.GameID,
			TileSize:       item.TileSize,
			ImageSmallPath: item.ImageSmallPath,
			ImageWidePath:  item.ImageWidePath,
			ImageLargePath: item.ImageLargePath,
		})
	}

	result, err := h.service.Update(tiles)
	if err != nil {
		writeServiceError(c, err, "保存开始屏幕磁贴失败")
		return
	}
	writeJSONSuccess(c, http.StatusOK, toStartScreenTileResponses(result))
}

func (h *StartScreenTilesHandler) UploadImage(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		writeJSONError(c, http.StatusBadRequest, "需要上传图片文件")
		return
	}

	path, err := h.service.UploadTileImage(file)
	if err != nil {
		writeServiceError(c, err, "磁贴图片上传失败")
		return
	}

	writeJSONSuccess(c, http.StatusCreated, gin.H{
		"path": path,
	})
}

func toStartScreenTileResponses(tiles []domain.StartScreenTile) []startScreenTileResponse {
	result := make([]startScreenTileResponse, 0, len(tiles))
	for _, tile := range tiles {
		result = append(result, startScreenTileResponse{
			GameID:         tile.GameID,
			PublicID:       tile.PublicID,
			Title:          tile.Title,
			CoverImage:     tile.CoverImage,
			BannerImage:    tile.BannerImage,
			TileSize:       tile.TileSize,
			ImageSmallPath: tile.ImageSmallPath,
			ImageWidePath:  tile.ImageWidePath,
			ImageLargePath: tile.ImageLargePath,
			SortOrder:      tile.SortOrder,
		})
	}
	return result
}

type startScreenTileResponse struct {
	GameID         int64   `json:"game_id"`
	PublicID       string  `json:"public_id"`
	Title          string  `json:"title"`
	CoverImage     *string `json:"cover_image"`
	BannerImage    *string `json:"banner_image"`
	TileSize       string  `json:"tile_size"`
	ImageSmallPath *string `json:"image_small_path"`
	ImageWidePath  *string `json:"image_wide_path"`
	ImageLargePath *string `json:"image_large_path"`
	SortOrder      int     `json:"sort_order"`
}
