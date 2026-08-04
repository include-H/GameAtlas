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
			GameID   int64  `json:"game_id"`
			TileSize string `json:"tile_size"`
		} `json:"tiles"`
	}
	if err := decodeJSONStrict(c, &request); err != nil {
		writeJSONError(c, http.StatusBadRequest, "无效的开始屏幕磁贴数据")
		return
	}

	tiles := make([]domain.StartScreenTileWrite, 0, len(request.Tiles))
	for _, item := range request.Tiles {
		tiles = append(tiles, domain.StartScreenTileWrite{
			GameID:   item.GameID,
			TileSize: item.TileSize,
		})
	}

	result, err := h.service.Update(tiles)
	if err != nil {
		writeServiceError(c, err, "保存开始屏幕磁贴失败")
		return
	}
	writeJSONSuccess(c, http.StatusOK, toStartScreenTileResponses(result))
}

func toStartScreenTileResponses(tiles []domain.StartScreenTile) []startScreenTileResponse {
	result := make([]startScreenTileResponse, 0, len(tiles))
	for _, tile := range tiles {
		result = append(result, startScreenTileResponse{
			GameID:     tile.GameID,
			PublicID:   tile.PublicID,
			Title:      tile.Title,
			CoverImage: tile.CoverImage,
			TileSize:   tile.TileSize,
			SortOrder:  tile.SortOrder,
		})
	}
	return result
}

type startScreenTileResponse struct {
	GameID     int64   `json:"game_id"`
	PublicID   string  `json:"public_id"`
	Title      string  `json:"title"`
	CoverImage *string `json:"cover_image"`
	TileSize   string  `json:"tile_size"`
	SortOrder  int     `json:"sort_order"`
}
