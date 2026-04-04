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
		writeServiceError(c, err, "file_path is required")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    toGameFileResponses(files, isAdminRequest(c)),
	})
}
