package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/services"
)

type SteamGridDBHandler struct {
	service *services.SteamGridDBService
}

func NewSteamGridDBHandler(service *services.SteamGridDBService) *SteamGridDBHandler {
	return &SteamGridDBHandler{service: service}
}

func (h *SteamGridDBHandler) GetGrids(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	h.handleImages(c, h.service.GetGridsBySteamAppID)
}

func (h *SteamGridDBHandler) GetHeroes(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	h.handleImages(c, h.service.GetHeroesBySteamAppID)
}

func (h *SteamGridDBHandler) GetLogos(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	h.handleImages(c, h.service.GetLogosBySteamAppID)
}

func (h *SteamGridDBHandler) GetIcons(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	h.handleImages(c, h.service.GetIconsBySteamAppID)
}

func (h *SteamGridDBHandler) Available(c *gin.Context) {
	writeJSONSuccess(c, http.StatusOK, h.service.Available())
}

func (h *SteamGridDBHandler) Search(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	if !h.service.Available() {
		writeJSONError(c, http.StatusServiceUnavailable, "SteamGridDB API 未配置")
		return
	}
	query := c.Query("q")
	if query == "" {
		writeJSONError(c, http.StatusBadRequest, "搜索关键词为必填项")
		return
	}
	results, err := h.service.Search(query)
	if err != nil {
		writeServiceError(c, err, "SteamGridDB 搜索失败")
		return
	}
	writeJSONSuccess(c, http.StatusOK, toSteamGridDBGameResponses(results))
}

func (h *SteamGridDBHandler) GetGridsByGameID(c *gin.Context) {
	h.handleImagesByGameID(c, h.service.GetGridsByGameID)
}

func (h *SteamGridDBHandler) GetHeroesByGameID(c *gin.Context) {
	h.handleImagesByGameID(c, h.service.GetHeroesByGameID)
}

func (h *SteamGridDBHandler) GetLogosByGameID(c *gin.Context) {
	h.handleImagesByGameID(c, h.service.GetLogosByGameID)
}

func (h *SteamGridDBHandler) handleImagesByGameID(c *gin.Context, fetch func(int) ([]services.SteamGridDBImage, error)) {
	if !requireAdmin(c) {
		return
	}
	if !h.service.Available() {
		writeJSONError(c, http.StatusServiceUnavailable, "SteamGridDB API 未配置")
		return
	}
	gameID, ok := parseIDParam(c, "gameId")
	if !ok {
		return
	}
	images, err := fetch(int(gameID))
	if err != nil {
		writeServiceError(c, err, "SteamGridDB 请求失败")
		return
	}
	writeJSONSuccess(c, http.StatusOK, toSteamGridDBImageResponses(images))
}

func (h *SteamGridDBHandler) handleImages(c *gin.Context, fetch func(int64) ([]services.SteamGridDBImage, error)) {
	if !h.service.Available() {
		writeJSONError(c, http.StatusServiceUnavailable, "SteamGridDB API 未配置")
		return
	}

	appID, ok := parseIDParam(c, "appId")
	if !ok {
		return
	}

	images, err := fetch(appID)
	if err != nil {
		writeServiceError(c, err, "SteamGridDB 请求失败")
		return
	}

	writeJSONSuccess(c, http.StatusOK, toSteamGridDBImageResponses(images))
}
