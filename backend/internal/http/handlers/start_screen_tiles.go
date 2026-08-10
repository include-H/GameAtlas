package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/files"
	"github.com/hao/game/internal/services"
)

type StartScreenTilesHandler struct {
	service *services.StartScreenTilesService
}

func NewStartScreenTilesHandler(service *services.StartScreenTilesService) *StartScreenTilesHandler {
	return &StartScreenTilesHandler{service: service}
}

func (h *StartScreenTilesHandler) Get(c *gin.Context) {
	layout, err := h.service.List(isAdminRequest(c))
	if err != nil {
		writeServiceError(c, err, "获取开始屏幕磁贴失败")
		return
	}
	writeJSONSuccess(c, http.StatusOK, toStartScreenLayoutResponse(layout))
}

func (h *StartScreenTilesHandler) Update(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}

	var request struct {
		Columns []struct {
			Name string `json:"name"`
		} `json:"columns"`
		Tiles []struct {
			GameID         int64   `json:"game_id"`
			TileSize       string  `json:"tile_size"`
			ImageSmallPath *string `json:"image_small_path"`
			ImageWidePath  *string `json:"image_wide_path"`
			ImageLargePath *string `json:"image_large_path"`
			ColumnIndex    *int    `json:"column_index"`
			GridRow        *int    `json:"grid_row"`
			GridCol        *int    `json:"grid_col"`
		} `json:"tiles"`
	}
	if err := decodeJSONStrict(c, &request); err != nil {
		writeJSONError(c, http.StatusBadRequest, "无效的开始屏幕磁贴数据")
		return
	}

	columns := make([]domain.StartScreenColumnWrite, 0, len(request.Columns))
	for _, item := range request.Columns {
		columns = append(columns, domain.StartScreenColumnWrite{Name: item.Name})
	}

	tiles := make([]domain.StartScreenTileWrite, 0, len(request.Tiles))
	for _, item := range request.Tiles {
		tiles = append(tiles, domain.StartScreenTileWrite{
			GameID:         item.GameID,
			TileSize:       item.TileSize,
			ImageSmallPath: item.ImageSmallPath,
			ImageWidePath:  item.ImageWidePath,
			ImageLargePath: item.ImageLargePath,
			ColumnIndex:    intValueOrZero(item.ColumnIndex),
			GridRow:        intValueOrZero(item.GridRow),
			GridCol:        intValueOrZero(item.GridCol),
		})
	}

	result, err := h.service.Update(columns, tiles)
	if err != nil {
		writeServiceError(c, err, "保存开始屏幕磁贴失败")
		return
	}
	writeJSONSuccess(c, http.StatusOK, toStartScreenLayoutResponse(result))
}

func (h *StartScreenTilesHandler) AddTile(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}

	var request struct {
		GameID         int64   `json:"game_id"`
		TileSize       string  `json:"tile_size"`
		ImageSmallPath *string `json:"image_small_path"`
		ImageWidePath  *string `json:"image_wide_path"`
		ImageLargePath *string `json:"image_large_path"`
		ColumnIndex    *int    `json:"column_index"`
		GridRow        *int    `json:"grid_row"`
		GridCol        *int    `json:"grid_col"`
	}
	if err := decodeJSONStrict(c, &request); err != nil {
		writeJSONError(c, http.StatusBadRequest, "无效的开始屏幕磁贴数据")
		return
	}

	result, err := h.service.AddTile(domain.StartScreenTileWrite{
		GameID:         request.GameID,
		TileSize:       request.TileSize,
		ImageSmallPath: request.ImageSmallPath,
		ImageWidePath:  request.ImageWidePath,
		ImageLargePath: request.ImageLargePath,
		ColumnIndex:    intValueOrZero(request.ColumnIndex),
		GridRow:        intValueOrZero(request.GridRow),
		GridCol:        intValueOrZero(request.GridCol),
	})
	if err != nil {
		writeServiceError(c, err, "添加到开始屏幕失败")
		return
	}
	writeJSONSuccess(c, http.StatusCreated, toStartScreenLayoutResponse(result))
}

func intValueOrZero(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func (h *StartScreenTilesHandler) RemoveTile(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}

	gameID, err := strconv.ParseInt(c.Param("gameId"), 10, 64)
	if err != nil || gameID <= 0 {
		writeJSONError(c, http.StatusBadRequest, "无效的游戏 ID")
		return
	}

	result, err := h.service.RemoveTile(gameID)
	if err != nil {
		writeServiceError(c, err, "从开始屏幕移除失败")
		return
	}
	writeJSONSuccess(c, http.StatusOK, toStartScreenLayoutResponse(result))
}

func (h *StartScreenTilesHandler) UploadImage(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	limitMultipartBody(c, files.MaxImageUploadBytes)
	if err := parseMultipartForm(c); err != nil {
		writeMultipartParseError(c, err, "需要上传图片文件")
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		writeMultipartParseError(c, err, "需要上传图片文件")
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

func toStartScreenLayoutResponse(layout *domain.StartScreenLayout) startScreenLayoutResponse {
	return startScreenLayoutResponse{
		Columns: toStartScreenColumnResponses(layout.Columns),
		Tiles:   toStartScreenTileResponses(layout.Tiles),
	}
}

func toStartScreenColumnResponses(columns []domain.StartScreenColumn) []startScreenColumnResponse {
	result := make([]startScreenColumnResponse, 0, len(columns))
	for _, column := range columns {
		result = append(result, startScreenColumnResponse{
			ID:        column.ID,
			Name:      column.Name,
			SortOrder: column.SortOrder,
		})
	}
	return result
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
			ColumnIndex:    tile.ColumnIndex,
			GridRow:        tile.GridRow,
			GridCol:        tile.GridCol,
		})
	}
	return result
}

type startScreenLayoutResponse struct {
	Columns []startScreenColumnResponse `json:"columns"`
	Tiles   []startScreenTileResponse   `json:"tiles"`
}

type startScreenColumnResponse struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
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
	ColumnIndex    int     `json:"column_index"`
	GridRow        int     `json:"grid_row"`
	GridCol        int     `json:"grid_col"`
}
