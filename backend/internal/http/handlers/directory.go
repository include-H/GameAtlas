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

	writeJSONSuccess(c, http.StatusOK, gin.H{
		"path": path,
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

	writeJSONSuccess(c, http.StatusOK, toDirectoryListResponse(result))
}

func (h *DirectoryHandler) Search(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}

	query := strings.TrimSpace(c.Query("q"))
	if query == "" {
		writeJSONSuccess(c, http.StatusOK, []interface{}{})
		return
	}

	path := strings.TrimSpace(c.Query("path"))
	results, err := h.service.Search(query, path)
	if err != nil {
		writeServiceError(c, err, "搜索失败")
		return
	}

	writeJSONSuccess(c, http.StatusOK, results)
}
