package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/services"
)

type GameScanHandler struct {
	scanService *services.GameScanService
}

func NewGameScanHandler(scanService *services.GameScanService) *GameScanHandler {
	return &GameScanHandler{scanService: scanService}
}

func (h *GameScanHandler) Scan(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}

	result, err := h.scanService.Scan()
	if err != nil {
		writeJSONError(c, http.StatusInternalServerError, "扫描失败: "+err.Error())
		return
	}

	writeJSONSuccess(c, http.StatusOK, result)
}
