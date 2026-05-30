package handlers

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
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

func TestAssetsHandlerUploadVideoPersistsAsset(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertGamesHandlerTestGame(t, db, "upload-game", "Upload Game", domain.GameVisibilityPublic, "")
	assetsDir := filepath.Join(t.TempDir(), "assets")
	service := services.NewAssetsService(
		config.Config{AssetsDir: assetsDir},
		repositories.NewGamesRepository(db),
		repositories.NewAssetsRepository(db),
		repositories.NewAssetCleanupTasksRepository(db),
	)
	handler := NewAssetsHandler(service)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("game_id", strconv.FormatInt(gameID, 10)); err != nil {
		t.Fatalf("WriteField game_id: %v", err)
	}
	if err := writer.WriteField("sort_order", "2"); err != nil {
		t.Fatalf("WriteField sort_order: %v", err)
	}
	partHeader := textproto.MIMEHeader{}
	partHeader.Set("Content-Disposition", `form-data; name="file"; filename="trailer.mp4"`)
	partHeader.Set("Content-Type", "video/mp4")
	part, err := writer.CreatePart(partHeader)
	if err != nil {
		t.Fatalf("CreatePart returned error: %v", err)
	}
	if _, err := part.Write([]byte("video-content")); err != nil {
		t.Fatalf("Write file part returned error: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close returned error: %v", err)
	}

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/assets/video", body)
	context.Request.Header.Set("Content-Type", writer.FormDataContentType())
	context.Set("is_admin", true)

	handler.Upload("video")(context)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}

	var response struct {
		Success bool `json:"success"`
		Data    struct {
			Path     string `json:"path"`
			AssetUID string `json:"asset_uid"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success {
		t.Fatalf("expected success=true")
	}
	if response.Data.AssetUID == "" {
		t.Fatalf("asset_uid should not be empty")
	}
	if !strings.HasPrefix(response.Data.Path, "/assets/upload-game/") || !strings.HasSuffix(response.Data.Path, ".mp4") {
		t.Fatalf("path = %q, want upload-game mp4 path", response.Data.Path)
	}

	// Upload writes to staging, not permanent location.
	if _, err := os.Stat(filepath.Join(assetsDir, "_staging", response.Data.AssetUID+".mp4")); err != nil {
		t.Fatalf("expected uploaded file in staging directory, got err=%v", err)
	}
}

func TestAssetsHandlerUploadRejectsInvalidSortOrder(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertGamesHandlerTestGame(t, db, "upload-invalid-sort", "Upload Invalid Sort", domain.GameVisibilityPublic, "")
	service := services.NewAssetsService(
		config.Config{AssetsDir: filepath.Join(t.TempDir(), "assets")},
		repositories.NewGamesRepository(db),
		repositories.NewAssetsRepository(db),
		repositories.NewAssetCleanupTasksRepository(db),
	)
	handler := NewAssetsHandler(service)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("game_id", strconv.FormatInt(gameID, 10)); err != nil {
		t.Fatalf("WriteField game_id: %v", err)
	}
	if err := writer.WriteField("sort_order", "-5"); err != nil {
		t.Fatalf("WriteField sort_order: %v", err)
	}
	partHeader := textproto.MIMEHeader{}
	partHeader.Set("Content-Disposition", `form-data; name="file"; filename="trailer.mp4"`)
	partHeader.Set("Content-Type", "video/mp4")
	part, err := writer.CreatePart(partHeader)
	if err != nil {
		t.Fatalf("CreatePart returned error: %v", err)
	}
	if _, err := part.Write([]byte("video-content")); err != nil {
		t.Fatalf("Write file part returned error: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close returned error: %v", err)
	}

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/assets/video", body)
	context.Request.Header.Set("Content-Type", writer.FormDataContentType())
	context.Set("is_admin", true)

	handler.Upload("video")(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"error":"需要有效的排序值"`) {
		t.Fatalf("body = %s, want 需要有效的排序值", recorder.Body.String())
	}
}

func TestAssetsHandlerUploadReturnsBadRequestWhenFileMissing(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("game_id", "1"); err != nil {
		t.Fatalf("WriteField returned error: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close returned error: %v", err)
	}

	context.Request = httptest.NewRequest(http.MethodPost, "/api/assets/video", body)
	context.Request.Header.Set("Content-Type", writer.FormDataContentType())
	context.Set("is_admin", true)

	NewAssetsHandler(nil).Upload("video")(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if !strings.Contains(recorder.Body.String(), `"error":"需要上传文件"`) {
		t.Fatalf("body = %s, want 需要上传文件", recorder.Body.String())
	}
}

func TestAssetsHandlerUploadReturnsBadRequestWhenContentTypeInvalid(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertGamesHandlerTestGame(t, db, "upload-invalid-type", "Upload Invalid Type", domain.GameVisibilityPublic, "")
	service := services.NewAssetsService(
		config.Config{AssetsDir: filepath.Join(t.TempDir(), "assets")},
		repositories.NewGamesRepository(db),
		repositories.NewAssetsRepository(db),
		repositories.NewAssetCleanupTasksRepository(db),
	)
	handler := NewAssetsHandler(service)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("game_id", strconv.FormatInt(gameID, 10)); err != nil {
		t.Fatalf("WriteField game_id: %v", err)
	}
	partHeader := textproto.MIMEHeader{}
	partHeader.Set("Content-Disposition", `form-data; name="file"; filename="bad.txt"`)
	partHeader.Set("Content-Type", "text/plain")
	part, err := writer.CreatePart(partHeader)
	if err != nil {
		t.Fatalf("CreatePart returned error: %v", err)
	}
	if _, err := part.Write([]byte("not-a-video")); err != nil {
		t.Fatalf("Write part returned error: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close returned error: %v", err)
	}

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/assets/video", body)
	context.Request.Header.Set("Content-Type", writer.FormDataContentType())
	context.Set("is_admin", true)

	handler.Upload("video")(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"error":"无效的资源上传"`) {
		t.Fatalf("body = %s, want 无效的资源上传", recorder.Body.String())
	}
}

func TestAssetsHandlerUploadReturnsNotFoundWhenGameMissing(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	service := services.NewAssetsService(
		config.Config{AssetsDir: filepath.Join(t.TempDir(), "assets")},
		repositories.NewGamesRepository(db),
		repositories.NewAssetsRepository(db),
		repositories.NewAssetCleanupTasksRepository(db),
	)
	handler := NewAssetsHandler(service)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("game_id", "999"); err != nil {
		t.Fatalf("WriteField game_id: %v", err)
	}
	partHeader := textproto.MIMEHeader{}
	partHeader.Set("Content-Disposition", `form-data; name="file"; filename="trailer.mp4"`)
	partHeader.Set("Content-Type", "video/mp4")
	part, err := writer.CreatePart(partHeader)
	if err != nil {
		t.Fatalf("CreatePart returned error: %v", err)
	}
	if _, err := part.Write([]byte("video")); err != nil {
		t.Fatalf("Write part returned error: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close returned error: %v", err)
	}

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/assets/video", body)
	context.Request.Header.Set("Content-Type", writer.FormDataContentType())
	context.Set("is_admin", true)

	handler.Upload("video")(context)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusNotFound, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"error":"资源不存在"`) {
		t.Fatalf("body = %s, want 资源不存在", recorder.Body.String())
	}
}

func mustLoadHandlerGame(t *testing.T, db *sqlx.DB, gameID int64) *domain.Game {
	t.Helper()

	game, err := repositories.NewGamesRepository(db).GetByID(gameID)
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	return game
}

