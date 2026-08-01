package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/services"
)

type GameFileRefreshHandler struct {
	refreshService *services.GameFileRefreshService
}

func NewGameFileRefreshHandler(refreshService *services.GameFileRefreshService) *GameFileRefreshHandler {
	return &GameFileRefreshHandler{refreshService: refreshService}
}

func (h *GameFileRefreshHandler) RefreshSizes(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}

	result, err := h.refreshService.RefreshFileSizes()
	if err != nil {
		writeJSONError(c, http.StatusInternalServerError, "刷新文件大小失败: "+err.Error())
		return
	}

	writeJSONSuccess(c, http.StatusOK, result)
}
