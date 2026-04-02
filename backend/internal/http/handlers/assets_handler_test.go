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

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}

	var response struct {
		Success bool `json:"success"`
		Data    struct {
			Path     string `json:"path"`
			AssetID  int64  `json:"asset_id"`
			AssetUID string `json:"asset_uid"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success {
		t.Fatalf("expected success=true")
	}
	if response.Data.AssetID <= 0 {
		t.Fatalf("asset_id = %d, want > 0", response.Data.AssetID)
	}
	if response.Data.AssetUID == "" {
		t.Fatalf("asset_uid should not be empty")
	}
	if !strings.HasPrefix(response.Data.Path, "/assets/upload-game/") || !strings.HasSuffix(response.Data.Path, ".mp4") {
		t.Fatalf("path = %q, want upload-game mp4 path", response.Data.Path)
	}

	asset := mustLoadHandlerAssetByUID(t, db, response.Data.AssetUID)
	if asset.SortOrder != 0 {
		t.Fatalf("SortOrder = %d, want 0 because negative form value should fallback", asset.SortOrder)
	}
	if _, err := os.Stat(filepath.Join(assetsDir, "upload-game", response.Data.AssetUID+".mp4")); err != nil {
		t.Fatalf("expected uploaded file on disk, got err=%v", err)
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
	if !strings.Contains(recorder.Body.String(), `"error":"file is required"`) {
		t.Fatalf("body = %s, want file is required error", recorder.Body.String())
	}
}

func TestAssetsHandlerDeleteRejectsInvalidPayload(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodDelete, "/api/assets", strings.NewReader("{"))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Set("is_admin", true)

	NewAssetsHandler(nil).Delete(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if !strings.Contains(recorder.Body.String(), `"error":"invalid asset payload"`) {
		t.Fatalf("body = %s, want invalid asset payload error", recorder.Body.String())
	}
}

func TestAssetsHandlerDeleteReturnsNotFoundWhenScreenshotMissing(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertGamesHandlerTestGame(t, db, "asset-delete-missing", "Asset Delete Missing", domain.GameVisibilityPublic, "")
	service := services.NewAssetsService(
		config.Config{AssetsDir: filepath.Join(t.TempDir(), "assets")},
		repositories.NewGamesRepository(db),
		repositories.NewAssetsRepository(db),
	)
	handler := NewAssetsHandler(service)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodDelete, "/api/assets", strings.NewReader(`{"game_id":`+strconv.FormatInt(gameID, 10)+`,"asset_type":"screenshot","path":"/assets/asset-delete-missing/missing.png"}`))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Set("is_admin", true)

	handler.Delete(context)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusNotFound, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"error":"resource not found"`) {
		t.Fatalf("body = %s, want resource not found error", recorder.Body.String())
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
	if !strings.Contains(recorder.Body.String(), `"error":"invalid asset upload"`) {
		t.Fatalf("body = %s, want invalid asset upload error", recorder.Body.String())
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
	if !strings.Contains(recorder.Body.String(), `"error":"resource not found"`) {
		t.Fatalf("body = %s, want resource not found error", recorder.Body.String())
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

func mustLoadHandlerAssetByUID(t *testing.T, db *sqlx.DB, assetUID string) *domain.GameAsset {
	t.Helper()

	var asset domain.GameAsset
	if err := db.Get(&asset, `
		SELECT id, game_id, asset_uid, asset_type, path, sort_order, created_at
		FROM game_assets
		WHERE asset_uid = ?
		LIMIT 1
	`, assetUID); err != nil {
		t.Fatalf("load asset by uid: %v", err)
	}
	return &asset
}
