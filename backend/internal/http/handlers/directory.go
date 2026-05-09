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
		// 2026-05-09: 统一为中文错误信息
		writeServiceError(c, err, "目录根路径未配置")
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
		// 2026-05-09: 统一为中文错误信息
		writeServiceError(c, err, "无效的目录路径")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    toDirectoryListResponse(result),
	})
}
