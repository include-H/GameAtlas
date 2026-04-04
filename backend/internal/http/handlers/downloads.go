package handlers

import (
	"errors"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/text/encoding/simplifiedchinese"

	"github.com/hao/game/internal/services"
)

type DownloadsHandler struct {
	service       *services.GameFilesService
	windowsLaunch *services.WindowsLaunchService
	authService   *services.AuthService
	// Download stats only need lightweight single-process dedupe for this app's
	// main use case: preventing accidental double-counts from repeated clicks by
	// the same person. This is intentionally in-memory and approximate, not a
	// cross-restart or multi-instance guarantee.
	downloadDedupeMu sync.Mutex
	downloadDedupe   map[string]time.Time
}

const downloadRecordWindow = 10 * time.Minute

func NewDownloadsHandler(service *services.GameFilesService, windowsLaunch *services.WindowsLaunchService, authService *services.AuthService) *DownloadsHandler {
	return &DownloadsHandler{
		service:        service,
		windowsLaunch:  windowsLaunch,
		authService:    authService,
		downloadDedupe: make(map[string]time.Time),
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
		writeDownloadLookupError(c, err)
		return
	}

	file, err := os.Open(downloadFile.ResolvedPath)
	if err != nil {
		writeJSONError(c, http.StatusNotFound, "registered file is unavailable")
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
	if !h.shouldRecordDownload(gameID, fileID, sourceKey, time.Now().UTC()) {
		writeJSONSuccess(c, http.StatusOK, operationStatusResponse{Recorded: false})
		return
	}

	if err := h.service.RecordDownload(gameID, fileID, isAdminRequest(c)); err != nil {
		writeDownloadRecordError(c, err)
		return
	}

	writeJSONSuccess(c, http.StatusOK, operationStatusResponse{Recorded: true})
}

func (h *DownloadsHandler) shouldRecordDownload(gameID, fileID int64, sourceKey string, now time.Time) bool {
	h.downloadDedupeMu.Lock()
	defer h.downloadDedupeMu.Unlock()

	// Best-effort cleanup for the in-memory dedupe window. We do not persist this
	// state because the goal is only to absorb local click bursts, not to enforce
	// stable rate limiting semantics across process restarts or deployments.
	for key, expiresAt := range h.downloadDedupe {
		if !expiresAt.After(now) {
			delete(h.downloadDedupe, key)
		}
	}

	recordKey := sourceKey + ":" + int64ToString(gameID) + ":" + int64ToString(fileID)
	if expiresAt, exists := h.downloadDedupe[recordKey]; exists && expiresAt.After(now) {
		return false
	}

	h.downloadDedupe[recordKey] = now.Add(downloadRecordWindow)
	return true
}

func (h *DownloadsHandler) LaunchScript(c *gin.Context) {
	if h.windowsLaunch == nil {
		writeJSONError(c, http.StatusInternalServerError, "launch script service is unavailable")
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
		writeLaunchScriptError(c, err)
		return
	}

	encodedScript, encodeErr := simplifiedchinese.GBK.NewEncoder().Bytes([]byte(script))
	if encodeErr != nil {
		writeJSONError(c, http.StatusInternalServerError, "failed to encode launch script")
		return
	}

	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", buildAttachmentDisposition(filename))
	c.Data(http.StatusOK, "application/octet-stream", encodedScript)
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

func writeDownloadLookupError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrNotFound):
		writeJSONError(c, http.StatusNotFound, "resource not found")
	case errors.Is(err, services.ErrForbiddenPath):
		writeJSONError(c, http.StatusForbidden, "file path is outside PRIMARY_ROM_ROOT")
	case errors.Is(err, services.ErrMissingFile), errors.Is(err, services.ErrInvalidFile):
		writeJSONError(c, http.StatusNotFound, "registered file is unavailable")
	case errors.Is(err, services.ErrValidation), errors.Is(err, services.ErrMissingConfig):
		writeJSONError(c, http.StatusBadRequest, err.Error())
	default:
		writeJSONError(c, http.StatusInternalServerError, "internal server error")
	}
}

func writeDownloadRecordError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrNotFound):
		writeJSONError(c, http.StatusNotFound, "resource not found")
	case errors.Is(err, services.ErrValidation), errors.Is(err, services.ErrMissingConfig):
		writeJSONError(c, http.StatusBadRequest, err.Error())
	default:
		writeJSONError(c, http.StatusInternalServerError, "internal server error")
	}
}

func writeLaunchScriptError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrNotFound):
		writeJSONError(c, http.StatusNotFound, "resource not found")
	case errors.Is(err, services.ErrForbiddenPath):
		writeJSONError(c, http.StatusForbidden, "file path is outside PRIMARY_ROM_ROOT")
	case errors.Is(err, services.ErrMissingFile), errors.Is(err, services.ErrInvalidFile):
		writeJSONError(c, http.StatusNotFound, "registered file is unavailable")
	case errors.Is(err, services.ErrInvalidLaunchFile), errors.Is(err, services.ErrMissingSMBConfig):
		writeJSONError(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, services.ErrValidation), errors.Is(err, services.ErrMissingConfig):
		writeJSONError(c, http.StatusBadRequest, err.Error())
	default:
		writeJSONError(c, http.StatusInternalServerError, "internal server error")
	}
}
