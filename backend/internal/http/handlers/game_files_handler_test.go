package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/config"
	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/repositories"
	"github.com/hao/game/internal/services"
)

func TestGameFilesHandlerListHidesPathForPublicAndIncludesItForAdmin(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertGamesHandlerTestGame(t, db, "files-visible", "Files Visible", domain.GameVisibilityPublic, "")
	filePath := filepath.Join(t.TempDir(), "files-visible.rom")
	if err := os.WriteFile(filePath, []byte("demo"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	fileID := insertDownloadsHandlerGameFile(t, db, gameID, filePath, 0)
	if fileID <= 0 {
		t.Fatalf("fileID = %d, want > 0", fileID)
	}

	service := services.NewGameFilesService(
		config.Config{PrimaryROMRoot: t.TempDir()},
		repositories.NewGamesRepository(db),
		repositories.NewGameFilesRepository(db),
	)
	handler := NewGameFilesHandler(service)

	publicRecorder := httptest.NewRecorder()
	publicContext, _ := gin.CreateTestContext(publicRecorder)
	publicContext.Request = httptest.NewRequest(http.MethodGet, "/api/games/files-visible/files", nil)
	publicContext.Params = gin.Params{{Key: "publicId", Value: "files-visible"}}
	handler.List(publicContext)

	if publicRecorder.Code != http.StatusOK {
		t.Fatalf("public status = %d, want %d, body=%s", publicRecorder.Code, http.StatusOK, publicRecorder.Body.String())
	}

	var publicResponse struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(publicRecorder.Body.Bytes(), &publicResponse); err != nil {
		t.Fatalf("decode public response: %v", err)
	}
	if len(publicResponse.Data) != 1 {
		t.Fatalf("len(public data) = %d, want 1", len(publicResponse.Data))
	}
	if _, ok := publicResponse.Data[0]["file_path"]; ok {
		t.Fatalf("public response unexpectedly exposes file_path: %s", publicRecorder.Body.String())
	}
	if publicResponse.Data[0]["file_name"] != filepath.Base(filePath) {
		t.Fatalf("public file_name = %#v, want %q", publicResponse.Data[0]["file_name"], filepath.Base(filePath))
	}

	adminRecorder := httptest.NewRecorder()
	adminContext, _ := gin.CreateTestContext(adminRecorder)
	adminContext.Request = httptest.NewRequest(http.MethodGet, "/api/games/files-visible/files", nil)
	adminContext.Params = gin.Params{{Key: "publicId", Value: "files-visible"}}
	adminContext.Set("is_admin", true)
	handler.List(adminContext)

	if adminRecorder.Code != http.StatusOK {
		t.Fatalf("admin status = %d, want %d, body=%s", adminRecorder.Code, http.StatusOK, adminRecorder.Body.String())
	}

	var adminResponse struct {
		Data []struct {
			FileName string `json:"file_name"`
			FilePath string `json:"file_path"`
		} `json:"data"`
	}
	if err := json.Unmarshal(adminRecorder.Body.Bytes(), &adminResponse); err != nil {
		t.Fatalf("decode admin response: %v", err)
	}
	if len(adminResponse.Data) != 1 {
		t.Fatalf("len(admin data) = %d, want 1", len(adminResponse.Data))
	}
	if adminResponse.Data[0].FilePath != filePath {
		t.Fatalf("admin file_path = %q, want %q", adminResponse.Data[0].FilePath, filePath)
	}
	if adminResponse.Data[0].FileName != filepath.Base(filePath) {
		t.Fatalf("admin file_name = %q, want %q", adminResponse.Data[0].FileName, filepath.Base(filePath))
	}
}

func TestGameFilesHandlerListReturnsNotFoundForPrivateGameToPublic(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertGamesHandlerTestGame(t, db, "files-private", "Files Private", domain.GameVisibilityPrivate, "")
	insertDownloadsHandlerGameFile(t, db, gameID, "/roms/private.rom", 0)

	service := services.NewGameFilesService(
		config.Config{},
		repositories.NewGamesRepository(db),
		repositories.NewGameFilesRepository(db),
	)
	handler := NewGameFilesHandler(service)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/games/files-private/files", nil)
	context.Params = gin.Params{{Key: "publicId", Value: "files-private"}}

	handler.List(context)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusNotFound, recorder.Body.String())
	}
}
