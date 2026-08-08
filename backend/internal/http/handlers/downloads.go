package handlers

import (
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/services"
)

type DownloadsHandler struct {
	service       *services.GameFilesService
	windowsLaunch *services.WindowsLaunchService
	authService   *services.AuthService
}

func NewDownloadsHandler(service *services.GameFilesService, windowsLaunch *services.WindowsLaunchService, authService *services.AuthService) *DownloadsHandler {
	return &DownloadsHandler{
		service:       service,
		windowsLaunch: windowsLaunch,
		authService:   authService,
	}
}

func (h *DownloadsHandler) Download(c *gin.Context) {
	gameID, ok := parseGamePublicIDParam(c, "publicId", h.service.ResolveGameID)
	if !ok {
		return
	}
	fileID, ok := parseIDParam(c, "fileId")
	if !ok {
		return
	}

	downloadFile, err := h.service.GetDownloadFile(gameID, fileID, isAdminRequest(c))
	if err != nil {
		writeServiceError(c, err, "无效的下载请求")
		return
	}

	// 2026-05-09: os.Open + http.ServeContent is the standard Go pattern for
	// serving files with Content-Disposition and range support. This stays in the
	// handler because it is HTTP response formatting, not business logic.
	file, err := os.Open(downloadFile.ResolvedPath)
	if err != nil {
		// 2026-05-09: 统一为中文错误信息
		writeJSONError(c, http.StatusNotFound, "注册文件不可用")
		return
	}
	defer file.Close()

	filename := filepath.Base(downloadFile.ResolvedPath)
	c.Header("Content-Disposition", buildAttachmentDisposition(filename))
	c.Header("Content-Length", int64ToString(downloadFile.SizeBytes))
	http.ServeContent(c.Writer, c.Request, filename, time.Unix(downloadFile.ModTime, 0), file)
}

func (h *DownloadsHandler) RecordDownload(c *gin.Context) {
	gameID, ok := parseGamePublicIDParam(c, "publicId", h.service.ResolveGameID)
	if !ok {
		return
	}
	fileID, ok := parseIDParam(c, "fileId")
	if !ok {
		return
	}

	sourceKey := h.authService.SourceKey(c.ClientIP(), c.Request.UserAgent())
	if !h.service.ShouldRecordDownload(sourceKey, gameID, fileID) {
		writeJSONSuccess(c, http.StatusOK, operationStatusResponse{Recorded: false})
		return
	}

	if err := h.service.RecordDownload(gameID, fileID, isAdminRequest(c)); err != nil {
		writeServiceError(c, err, "无效的下载记录请求")
		return
	}

	writeJSONSuccess(c, http.StatusOK, operationStatusResponse{Recorded: true})
}

func (h *DownloadsHandler) LaunchScript(c *gin.Context) {
	if h.windowsLaunch == nil {
		// 2026-05-09: 统一为中文错误信息
		writeJSONError(c, http.StatusInternalServerError, "启动脚本服务不可用")
		return
	}

	// This endpoint intentionally follows the same visibility boundary as normal downloads instead of
	// requiring admin. The current product assumption is single-user / trusted deployment, and the
	// configured SMB account is expected to have read-only access only. If that deployment model changes,
	// revisit this endpoint first and move it behind stricter authorization.
	gameID, ok := parseGamePublicIDParam(c, "publicId", h.service.ResolveGameID)
	if !ok {
		return
	}
	fileID, ok := parseIDParam(c, "fileId")
	if !ok {
		return
	}

	script, filename, err := h.windowsLaunch.BuildLaunchScript(gameID, fileID, isAdminRequest(c))
	if err != nil {
		writeServiceError(c, err, "无效的启动脚本请求")
		return
	}

	// 2026-08-08: script 是 ASCII bat 引导壳（PS 主脚本以 UTF-16LE base64 内嵌），
	// 不再需要 GBK 转码；base64 载荷对传输编码免疫。
	// no-store：脚本随后端代码更新，禁止浏览器缓存旧版本（URL 不变会命中缓存）。
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", buildAttachmentDisposition(filename))
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "application/octet-stream", []byte(script))
}

// Use the standard library to serialize Content-Disposition so Chinese
// filenames and quoted characters do not produce an invalid header.
func buildAttachmentDisposition(filename string) string {
	value := mime.FormatMediaType("attachment", map[string]string{
		"filename": filename,
	})
	if value == "" {
		return "attachment"
	}
	return value
}
