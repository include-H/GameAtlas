package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/services"
)

type DirectoryHandler struct {
	service *services.DirectoryService
}

func NewDirectoryHandler(service *services.DirectoryService) *DirectoryHandler {
	return &DirectoryHandler{service: service}
}

func (h *DirectoryHandler) Default(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	path, err := h.service.Default()
	if err != nil {
		writeServiceError(c, err, "directory roots are not configured")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"path": path,
		},
	})
}

func (h *DirectoryHandler) List(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	path := strings.TrimSpace(c.Query("path"))
	result, err := h.service.List(path)
	if err != nil {
		writeServiceError(c, err, "invalid directory path")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}
