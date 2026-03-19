package handlers

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/text/encoding/simplifiedchinese"

	"github.com/hao/game/internal/services"
)

type DownloadsHandler struct {
	service *services.GameFilesService
}

func NewDownloadsHandler(service *services.GameFilesService) *DownloadsHandler {
	return &DownloadsHandler{service: service}
}

func (h *DownloadsHandler) Download(c *gin.Context) {
	gameID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	fileID, ok := parseIDParam(c, "fileId")
	if !ok {
		return
	}

	downloadFile, err := h.service.GetDownloadFile(gameID, fileID, isAdminRequest(c))
	if err != nil {
		switch {
		case errors.Is(err, services.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "resource not found"})
		case errors.Is(err, services.ErrForbiddenPath):
			c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "file path is outside allowed roots"})
		case errors.Is(err, services.ErrMissingFile), errors.Is(err, services.ErrInvalidFile):
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "registered file is unavailable"})
		case errors.Is(err, services.ErrValidation), errors.Is(err, services.ErrMissingConfig):
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "internal server error"})
		}
		return
	}

	file, err := os.Open(downloadFile.ResolvedPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "registered file is unavailable",
		})
		return
	}
	defer file.Close()

	filename := filepath.Base(downloadFile.ResolvedPath)
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Header("Content-Length", int64ToString(downloadFile.SizeBytes))
	http.ServeContent(c.Writer, c.Request, filename, time.Unix(downloadFile.ModTime, 0), file)
}

func (h *DownloadsHandler) LaunchScript(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	gameID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	fileID, ok := parseIDParam(c, "fileId")
	if !ok {
		return
	}

	script, filename, err := h.service.BuildLaunchScript(gameID, fileID, isAdminRequest(c))
	if err != nil {
		switch {
		case errors.Is(err, services.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "resource not found"})
		case errors.Is(err, services.ErrForbiddenPath):
			c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "file path is outside allowed roots"})
		case errors.Is(err, services.ErrMissingFile), errors.Is(err, services.ErrInvalidFile):
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "registered file is unavailable"})
		case errors.Is(err, services.ErrInvalidLaunchFile), errors.Is(err, services.ErrMissingSMBConfig):
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		case errors.Is(err, services.ErrValidation), errors.Is(err, services.ErrMissingConfig):
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "internal server error"})
		}
		return
	}

	encodedScript, encodeErr := simplifiedchinese.GBK.NewEncoder().Bytes([]byte(script))
	if encodeErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "failed to encode launch script"})
		return
	}

	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Data(http.StatusOK, "application/octet-stream", encodedScript)
}
