package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/services"
)

type GameFilesHandler struct {
	service *services.GameFilesService
}

func NewGameFilesHandler(service *services.GameFilesService) *GameFilesHandler {
	return &GameFilesHandler{service: service}
}

func (h *GameFilesHandler) List(c *gin.Context) {
	gameID, ok := parseGamePublicIDParam(c, "publicId", h.service.ResolveGameID)
	if !ok {
		return
	}

	files, err := h.service.List(gameID, isAdminRequest(c))
	if err != nil {
		// 2026-05-09: 统一为中文错误信息
		writeServiceError(c, err, "无效的文件列表请求")
		return
	}

	writeJSONSuccess(c, http.StatusOK, toGameFileResponses(files, isAdminRequest(c)))
}
