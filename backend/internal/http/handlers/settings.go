package handlers

import (
	"log"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/services"
)

type SettingsHandler struct {
	service *services.SettingsService
}

func NewSettingsHandler(service *services.SettingsService) *SettingsHandler {
	return &SettingsHandler{service: service}
}

func (h *SettingsHandler) GetConfig(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}

	entries, err := h.service.GetConfig()
	if err != nil {
		writeServiceError(c, err, "读取配置失败")
		return
	}

	writeJSONSuccess(c, http.StatusOK, entries)
}

func (h *SettingsHandler) UpdateConfig(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}

	var updates map[string]string
	if err := decodeJSONStrict(c, &updates); err != nil {
		writeJSONError(c, http.StatusBadRequest, "无效的配置数据")
		return
	}

	if err := h.service.UpdateConfig(updates); err != nil {
		writeServiceError(c, err, "保存配置失败")
		return
	}

	writeJSONSuccess(c, http.StatusOK, gin.H{
		"message": "配置已保存，重启服务后生效",
	})
}

func (h *SettingsHandler) UploadBackground(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}

	file, header, err := c.Request.FormFile("bg")
	if err != nil {
		writeJSONError(c, http.StatusBadRequest, "请选择图片文件")
		return
	}
	defer file.Close()

	if err := h.service.SaveBackgroundImage(file, header); err != nil {
		writeServiceError(c, err, "上传背景图片失败")
		return
	}

	writeJSONSuccess(c, http.StatusOK, gin.H{
		"message": "背景图片已上传",
	})
}

func (h *SettingsHandler) RemoveBackground(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}

	if err := h.service.RemoveBackgroundImage(); err != nil {
		writeServiceError(c, err, "删除背景图片失败")
		return
	}

	writeJSONSuccess(c, http.StatusOK, gin.H{
		"message": "背景图片已删除",
	})
}

func (h *SettingsHandler) Restart(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}

	go func() {
		time.Sleep(200 * time.Millisecond)
		log.Println("[restart] re-executing process...")
		execErr := syscall.Exec(os.Args[0], os.Args, os.Environ())
		if execErr != nil {
			log.Printf("[restart] re-exec failed: %v", execErr)
		}
	}()

	writeJSONSuccess(c, http.StatusOK, gin.H{
		"message": "正在重启服务...",
	})
}
