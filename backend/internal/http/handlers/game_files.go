package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/services"
)

type GameFilesHandler struct {
	service *services.GameFilesService
}

func NewGameFilesHandler(service *services.GameFilesService) *GameFilesHandler {
	return &GameFilesHandler{service: service}
}

func (h *GameFilesHandler) List(c *gin.Context) {
	gameID, ok := parseIDParam(c, "id")
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

func (h *GameFilesHandler) Create(c *gin.Context) {
	gameID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if !requireAdmin(c) {
		return
	}

	var input domain.GameFileWriteInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid game file payload",
		})
		return
	}

	file, err := h.service.Create(gameID, input)
	if err != nil {
		writeServiceError(c, err, "file_path is required")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    toGameFileResponses([]domain.GameFile{*file}, true)[0],
	})
}

func (h *GameFilesHandler) Update(c *gin.Context) {
	gameID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if !requireAdmin(c) {
		return
	}
	fileID, ok := parseIDParam(c, "fileId")
	if !ok {
		return
	}

	var input domain.GameFileWriteInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid game file payload",
		})
		return
	}

	file, err := h.service.Update(gameID, fileID, input)
	if err != nil {
		writeServiceError(c, err, "file_path is required")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    toGameFileResponses([]domain.GameFile{*file}, true)[0],
	})
}

func (h *GameFilesHandler) Delete(c *gin.Context) {
	gameID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if !requireAdmin(c) {
		return
	}
	fileID, ok := parseIDParam(c, "fileId")
	if !ok {
		return
	}

	if err := h.service.Delete(gameID, fileID); err != nil {
		writeServiceError(c, err, "file_path is required")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"deleted": true,
		},
	})
}
