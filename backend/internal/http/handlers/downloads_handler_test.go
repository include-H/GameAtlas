package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"

	"github.com/hao/game/internal/config"
	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/repositories"
	"github.com/hao/game/internal/services"
)

func TestDownloadsHandlerDownloadServesRegisteredFile(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	root := t.TempDir()
	romPath := filepath.Join(root, "downloads", "demo.rom")
	if err := os.MkdirAll(filepath.Dir(romPath), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	content := []byte("rom-data")
	if err := os.WriteFile(romPath, content, 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	gameID := insertGamesHandlerTestGame(t, db, "download-game", "Download Game", domain.GameVisibilityPublic, "")
	fileID := insertDownloadsHandlerGameFile(t, db, gameID, romPath, 0)
	service := services.NewGameFilesService(
		config.Config{PrimaryROMRoot: root},
		repositories.NewGamesRepository(db),
		repositories.NewGameFilesRepository(db),
	)
	handler := NewDownloadsHandler(service, services.NewAuthService(config.Config{}, nil))

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/downloads/download-game/files/"+strconv.FormatInt(fileID, 10), nil)
	context.Params = gin.Params{
		{Key: "publicId", Value: "download-game"},
		{Key: "fileId", Value: strconv.FormatInt(fileID, 10)},
	}

	handler.Download(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if recorder.Body.String() != string(content) {
		t.Fatalf("body = %q, want %q", recorder.Body.String(), string(content))
	}
	if got := recorder.Header().Get("Content-Disposition"); got != `attachment; filename="demo.rom"` {
		t.Fatalf("Content-Disposition = %q, want attachment filename", got)
	}
	if got := recorder.Header().Get("Content-Length"); got != strconv.Itoa(len(content)) {
		t.Fatalf("Content-Length = %q, want %d", got, len(content))
	}
}

func TestDownloadsHandlerRecordDownloadDedupesWithinWindow(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertGamesHandlerTestGame(t, db, "record-download", "Record Download", domain.GameVisibilityPublic, "")
	fileID := insertDownloadsHandlerGameFile(t, db, gameID, "/roms/record-download.rom", 0)
	service := services.NewGameFilesService(
		config.Config{},
		repositories.NewGamesRepository(db),
		repositories.NewGameFilesRepository(db),
	)
	handler := NewDownloadsHandler(service, services.NewAuthService(config.Config{SessionSecret: "dedupe-secret"}, nil))

	firstRecorder := httptest.NewRecorder()
	firstContext, _ := gin.CreateTestContext(firstRecorder)
	firstContext.Request = httptest.NewRequest(http.MethodPost, "/api/downloads/record-download/files/"+strconv.FormatInt(fileID, 10)+"/record", nil)
	firstContext.Request.Header.Set("User-Agent", "test-agent")
	firstContext.Request.RemoteAddr = "127.0.0.1:4567"
	firstContext.Params = gin.Params{
		{Key: "publicId", Value: "record-download"},
		{Key: "fileId", Value: strconv.FormatInt(fileID, 10)},
	}

	handler.RecordDownload(firstContext)

	secondRecorder := httptest.NewRecorder()
	secondContext, _ := gin.CreateTestContext(secondRecorder)
	secondContext.Request = httptest.NewRequest(http.MethodPost, "/api/downloads/record-download/files/"+strconv.FormatInt(fileID, 10)+"/record", nil)
	secondContext.Request.Header.Set("User-Agent", "test-agent")
	secondContext.Request.RemoteAddr = "127.0.0.1:4567"
	secondContext.Params = gin.Params{
		{Key: "publicId", Value: "record-download"},
		{Key: "fileId", Value: strconv.FormatInt(fileID, 10)},
	}

	handler.RecordDownload(secondContext)

	if firstRecorder.Code != http.StatusOK {
		t.Fatalf("first status = %d, want %d", firstRecorder.Code, http.StatusOK)
	}
	if secondRecorder.Code != http.StatusOK {
		t.Fatalf("second status = %d, want %d", secondRecorder.Code, http.StatusOK)
	}

	var firstResponse struct {
		Data struct {
			Recorded bool `json:"recorded"`
		} `json:"data"`
	}
	if err := json.Unmarshal(firstRecorder.Body.Bytes(), &firstResponse); err != nil {
		t.Fatalf("decode first response: %v", err)
	}
	if !firstResponse.Data.Recorded {
		t.Fatalf("first response = %s, want recorded=true", firstRecorder.Body.String())
	}

	var secondResponse struct {
		Data struct {
			Recorded bool `json:"recorded"`
		} `json:"data"`
	}
	if err := json.Unmarshal(secondRecorder.Body.Bytes(), &secondResponse); err != nil {
		t.Fatalf("decode second response: %v", err)
	}
	if secondResponse.Data.Recorded {
		t.Fatalf("second response = %s, want recorded=false", secondRecorder.Body.String())
	}

	if got := loadDownloadsCount(t, db, gameID); got != 1 {
		t.Fatalf("downloads = %d, want 1 after dedupe", got)
	}
}

func TestDownloadsHandlerLaunchScriptReturnsBadRequestWhenSMBConfigMissing(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertGamesHandlerTestGame(t, db, "launch-missing-config", "Launch Missing Config", domain.GameVisibilityPublic, "")
	fileID := insertDownloadsHandlerGameFile(t, db, gameID, "/roms/launch.vhdx", 0)
	service := services.NewGameFilesService(
		config.Config{},
		repositories.NewGamesRepository(db),
		repositories.NewGameFilesRepository(db),
	)
	handler := NewDownloadsHandler(service, services.NewAuthService(config.Config{}, nil))

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/downloads/launch-missing-config/files/"+strconv.FormatInt(fileID, 10)+"/launch", nil)
	context.Params = gin.Params{
		{Key: "publicId", Value: "launch-missing-config"},
		{Key: "fileId", Value: strconv.FormatInt(fileID, 10)},
	}

	handler.LaunchScript(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), services.ErrMissingSMBConfig.Error()) {
		t.Fatalf("body = %s, want missing SMB config error", recorder.Body.String())
	}
}

func insertDownloadsHandlerGameFile(t *testing.T, db *sqlx.DB, gameID int64, filePath string, sortOrder int) int64 {
	t.Helper()

	result, err := db.Exec(`
		INSERT INTO game_files (game_id, file_path, sort_order)
		VALUES (?, ?, ?)
	`, gameID, filePath, sortOrder)
	if err != nil {
		t.Fatalf("insert downloads handler game file: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId returned error: %v", err)
	}

	return id
}

func loadDownloadsCount(t *testing.T, db *sqlx.DB, gameID int64) int64 {
	t.Helper()

	var downloads int64
	if err := db.Get(&downloads, `SELECT downloads FROM games WHERE id = ?`, gameID); err != nil {
		t.Fatalf("load downloads count: %v", err)
	}
	return downloads
}
