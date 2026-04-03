package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"

	"github.com/hao/game/internal/config"
	dbpkg "github.com/hao/game/internal/db"
	"github.com/hao/game/internal/repositories"
	"github.com/hao/game/internal/services"
)

func TestGamesHandlerListTimelineRejectsInvalidCursor(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/games/timeline?cursor=bad", nil)

	handler := NewGamesHandler(nil)
	handler.ListTimeline(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}

	var response struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Error != "invalid timeline cursor" {
		t.Fatalf("error = %q, want invalid timeline cursor", response.Error)
	}
}

func TestGamesHandlerListTimelineUsesLatestPublicReleaseDateAndFormatsCursor(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	_ = insertGamesHandlerTestGame(t, db, "private-new", "Private New", "private", "2025-01-01")
	firstID := insertGamesHandlerTestGame(t, db, "public-new", "Public New", "public", "2024-02-01")
	secondID := insertGamesHandlerTestGame(t, db, "public-mid", "Public Mid", "public", "2023-05-01")
	_ = insertGamesHandlerTestGame(t, db, "public-old", "Public Old", "public", "2022-03-01")

	service := services.NewGamesService(
		config.Config{},
		repositories.NewGamesRepository(db),
		repositories.NewGameFilesRepository(db),
		repositories.NewMetadataRepository(db),
		repositories.NewTagsRepository(db),
	)
	handler := NewGamesHandler(service)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/games/timeline?limit=2", nil)

	handler.ListTimeline(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response struct {
		Success bool `json:"success"`
		Data    []struct {
			ID          int64   `json:"id"`
			PublicID    string  `json:"public_id"`
			ReleaseDate *string `json:"release_date"`
		} `json:"data"`
		Pagination struct {
			Limit      int    `json:"limit"`
			From       string `json:"from"`
			To         string `json:"to"`
			HasMore    bool   `json:"hasMore"`
			NextCursor string `json:"nextCursor"`
		} `json:"pagination"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if !response.Success {
		t.Fatalf("expected success=true")
	}
	if len(response.Data) != 2 {
		t.Fatalf("len(data) = %d, want 2", len(response.Data))
	}
	if response.Data[0].ID != firstID || response.Data[0].PublicID != "public-new" {
		t.Fatalf("data[0] = %+v, want first public game", response.Data[0])
	}
	if response.Data[1].ID != secondID || response.Data[1].PublicID != "public-mid" {
		t.Fatalf("data[1] = %+v, want second public game", response.Data[1])
	}
	if response.Pagination.Limit != 2 {
		t.Fatalf("pagination.limit = %d, want 2", response.Pagination.Limit)
	}
	if response.Pagination.To != "2024-02-01" {
		t.Fatalf("pagination.to = %q, want 2024-02-01", response.Pagination.To)
	}
	if response.Pagination.From != "2022-02-01" {
		t.Fatalf("pagination.from = %q, want 2022-02-01", response.Pagination.From)
	}
	if !response.Pagination.HasMore {
		t.Fatalf("expected hasMore=true")
	}
	wantCursor := fmt.Sprintf("2023-05-01|%d", secondID)
	if response.Pagination.NextCursor != wantCursor {
		t.Fatalf("pagination.nextCursor = %q, want %q", response.Pagination.NextCursor, wantCursor)
	}
}

func TestGamesHandlerGetHidesFilePathsForPublicAndIncludesThemForAdmin(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertGamesHandlerTestGame(t, db, "detail-paths", "Detail Paths", "public", "2024-02-01")
	if _, err := db.Exec(`
		INSERT INTO game_assets (game_id, asset_uid, asset_type, path, sort_order)
		VALUES (?, 'video-a', 'video', '/assets/detail-paths/video-a.mp4', 0)
	`, gameID); err != nil {
		t.Fatalf("insert game asset: %v", err)
	}
	romPath := filepath.Join(t.TempDir(), "detail-paths.rom")
	if _, err := db.Exec(`
		INSERT INTO game_files (game_id, file_path, sort_order)
		VALUES (?, ?, 0)
	`, gameID, romPath); err != nil {
		t.Fatalf("insert game file: %v", err)
	}

	service := services.NewGamesService(
		config.Config{},
		repositories.NewGamesRepository(db),
		repositories.NewGameFilesRepository(db),
		repositories.NewMetadataRepository(db),
		repositories.NewTagsRepository(db),
	)
	handler := NewGamesHandler(service)

	publicRecorder := httptest.NewRecorder()
	publicContext, _ := gin.CreateTestContext(publicRecorder)
	publicContext.Request = httptest.NewRequest(http.MethodGet, "/api/games/detail-paths", nil)
	publicContext.Params = gin.Params{{Key: "publicId", Value: "detail-paths"}}
	handler.Get(publicContext)

	if publicRecorder.Code != http.StatusOK {
		t.Fatalf("public status = %d, want %d, body=%s", publicRecorder.Code, http.StatusOK, publicRecorder.Body.String())
	}

	var publicResponse struct {
		Data struct {
			PreviewVideo map[string]any   `json:"preview_video"`
			Files        []map[string]any `json:"files"`
		} `json:"data"`
	}
	if err := json.Unmarshal(publicRecorder.Body.Bytes(), &publicResponse); err != nil {
		t.Fatalf("decode public response: %v", err)
	}
	if publicResponse.Data.PreviewVideo["path"] != "/assets/detail-paths/video-a.mp4" {
		t.Fatalf("public preview_video = %#v, want path included", publicResponse.Data.PreviewVideo)
	}
	if len(publicResponse.Data.Files) != 1 {
		t.Fatalf("len(public files) = %d, want 1", len(publicResponse.Data.Files))
	}
	if _, ok := publicResponse.Data.Files[0]["file_path"]; ok {
		t.Fatalf("public files unexpectedly expose file_path: %s", publicRecorder.Body.String())
	}

	adminRecorder := httptest.NewRecorder()
	adminContext, _ := gin.CreateTestContext(adminRecorder)
	adminContext.Request = httptest.NewRequest(http.MethodGet, "/api/games/detail-paths", nil)
	adminContext.Params = gin.Params{{Key: "publicId", Value: "detail-paths"}}
	adminContext.Set("is_admin", true)
	handler.Get(adminContext)

	if adminRecorder.Code != http.StatusOK {
		t.Fatalf("admin status = %d, want %d, body=%s", adminRecorder.Code, http.StatusOK, adminRecorder.Body.String())
	}

	var adminResponse struct {
		Data struct {
			Files []struct {
				FilePath string `json:"file_path"`
				FileName string `json:"file_name"`
			} `json:"files"`
		} `json:"data"`
	}
	if err := json.Unmarshal(adminRecorder.Body.Bytes(), &adminResponse); err != nil {
		t.Fatalf("decode admin response: %v", err)
	}
	if len(adminResponse.Data.Files) != 1 {
		t.Fatalf("len(admin files) = %d, want 1", len(adminResponse.Data.Files))
	}
	if adminResponse.Data.Files[0].FilePath != romPath {
		t.Fatalf("admin file_path = %q, want %q", adminResponse.Data.Files[0].FilePath, romPath)
	}
	if adminResponse.Data.Files[0].FileName != filepath.Base(romPath) {
		t.Fatalf("admin file_name = %q, want %q", adminResponse.Data.Files[0].FileName, filepath.Base(romPath))
	}
}

func TestGamesHandlerListPendingUsesNativePendingFilter(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	pendingID := insertGamesHandlerTestGame(t, db, "pending-visible", "Pending Visible", "public", "")
	resolvedID := insertGamesHandlerTestGame(t, db, "pending-resolved", "Pending Resolved", "public", "")
	ignoredID := insertGamesHandlerTestGame(t, db, "pending-ignored", "Pending Ignored", "public", "")

	if _, err := db.Exec(`
		UPDATE games
		SET cover_image = ?, banner_image = ?, summary = ?, wiki_content = ?
		WHERE id IN (?, ?)
	`, "/assets/cover.png", "/assets/banner.png", "Ready", "# Ready", resolvedID, ignoredID); err != nil {
		t.Fatalf("seed handler pending games: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO game_assets (game_id, asset_uid, asset_type, path, sort_order)
		VALUES (?, 'resolved-shot', 'screenshot', '/assets/pending-resolved/shot.png', 0)
	`, resolvedID); err != nil {
		t.Fatalf("insert resolved screenshot: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO game_assets (game_id, asset_uid, asset_type, path, sort_order)
		VALUES (?, 'ignored-shot', 'screenshot', '/assets/pending-ignored/shot.png', 0)
	`, ignoredID); err != nil {
		t.Fatalf("insert ignored screenshot: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO game_files (game_id, file_path, sort_order)
		VALUES (?, '/roms/pending-resolved.rom', 0)
	`, resolvedID); err != nil {
		t.Fatalf("insert resolved file: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO game_files (game_id, file_path, sort_order)
		VALUES (?, '/roms/pending-ignored.rom', 0)
	`, ignoredID); err != nil {
		t.Fatalf("insert ignored file: %v", err)
	}

	platformID := insertGamesHandlerMetadataItem(t, db, "platforms", "Windows", "windows")
	developerID := insertGamesHandlerMetadataItem(t, db, "developers", "Studio", "studio")
	publisherID := insertGamesHandlerMetadataItem(t, db, "publishers", "Publisher", "publisher")
	linkGamesHandlerGameRelation(t, db, "game_platforms", "platform_id", resolvedID, platformID)
	linkGamesHandlerGameRelation(t, db, "game_developers", "developer_id", resolvedID, developerID)
	linkGamesHandlerGameRelation(t, db, "game_publishers", "publisher_id", resolvedID, publisherID)
	linkGamesHandlerGameRelation(t, db, "game_platforms", "platform_id", ignoredID, platformID)
	linkGamesHandlerGameRelation(t, db, "game_developers", "developer_id", ignoredID, developerID)
	linkGamesHandlerGameRelation(t, db, "game_publishers", "publisher_id", ignoredID, publisherID)

	if _, err := db.Exec(`
		INSERT INTO game_review_issue_overrides (game_id, issue_key, status)
		VALUES (?, 'missing-cover', 'ignored')
	`, ignoredID); err != nil {
		t.Fatalf("insert ignored override: %v", err)
	}

	service := services.NewGamesService(
		config.Config{},
		repositories.NewGamesRepository(db),
		repositories.NewGameFilesRepository(db),
		repositories.NewMetadataRepository(db),
		repositories.NewTagsRepository(db),
	)
	handler := NewGamesHandler(service)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/games?pending=true&limit=10", nil)

	handler.List(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Success bool `json:"success"`
		Data    []struct {
			ID       int64  `json:"id"`
			PublicID string `json:"public_id"`
		} `json:"data"`
		Pagination struct {
			Total int `json:"total"`
		} `json:"pagination"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if !response.Success {
		t.Fatalf("expected success=true")
	}
	if response.Pagination.Total != 1 {
		t.Fatalf("pagination.total = %d, want 1", response.Pagination.Total)
	}
	if len(response.Data) != 1 || response.Data[0].ID != pendingID || response.Data[0].PublicID != "pending-visible" {
		t.Fatalf("data = %+v, want only native pending game", response.Data)
	}
}

func TestGamesHandlerListPendingUsesNativePendingQueryOptions(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	severeID := insertGamesHandlerTestGame(t, db, "pending-severe", "Pending Severe", "public", "")
	recentID := insertGamesHandlerTestGame(t, db, "pending-recent", "Pending Recent", "public", "")
	ignoredID := insertGamesHandlerTestGame(t, db, "pending-ignored-all", "Pending Ignored All", "public", "")

	if _, err := db.Exec(`
		UPDATE games
		SET banner_image = ?, summary = ?
		WHERE id = ?
	`, "/assets/recent-banner.png", "Ready", recentID); err != nil {
		t.Fatalf("seed handler native pending recent game: %v", err)
	}
	if _, err := db.Exec(`
		UPDATE games
		SET banner_image = ?, summary = ?, wiki_content = ?
		WHERE id = ?
	`, "/assets/ignored-banner.png", "Ready", "# Ready", ignoredID); err != nil {
		t.Fatalf("seed handler native pending query games: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO game_assets (game_id, asset_uid, asset_type, path, sort_order)
		VALUES (?, 'recent-shot', 'screenshot', '/assets/pending-recent/shot.png', 0),
		       (?, 'ignored-shot', 'screenshot', '/assets/pending-ignored-all/shot.png', 0)
	`, recentID, ignoredID); err != nil {
		t.Fatalf("insert pending query screenshots: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO game_files (game_id, file_path, sort_order)
		VALUES (?, '/roms/pending-recent.rom', 0),
		       (?, '/roms/pending-ignored-all.rom', 0)
	`, recentID, ignoredID); err != nil {
		t.Fatalf("insert pending query files: %v", err)
	}

	platformID := insertGamesHandlerMetadataItem(t, db, "platforms", "Query Windows", "query-windows")
	developerID := insertGamesHandlerMetadataItem(t, db, "developers", "Query Studio", "query-studio")
	publisherID := insertGamesHandlerMetadataItem(t, db, "publishers", "Query Publisher", "query-publisher")
	linkGamesHandlerGameRelation(t, db, "game_platforms", "platform_id", recentID, platformID)
	linkGamesHandlerGameRelation(t, db, "game_developers", "developer_id", recentID, developerID)
	linkGamesHandlerGameRelation(t, db, "game_publishers", "publisher_id", recentID, publisherID)
	linkGamesHandlerGameRelation(t, db, "game_platforms", "platform_id", ignoredID, platformID)
	linkGamesHandlerGameRelation(t, db, "game_developers", "developer_id", ignoredID, developerID)
	linkGamesHandlerGameRelation(t, db, "game_publishers", "publisher_id", ignoredID, publisherID)

	now := time.Now().UTC()
	for _, item := range []struct {
		id        int64
		createdAt string
	}{
		{id: severeID, createdAt: now.Format("2006-01-02 15:04:05")},
		{id: recentID, createdAt: now.AddDate(0, 0, -1).Format("2006-01-02 15:04:05")},
		{id: ignoredID, createdAt: now.AddDate(0, 0, -1).Format("2006-01-02 15:04:05")},
	} {
		if _, err := db.Exec(`UPDATE games SET created_at = ?, updated_at = ? WHERE id = ?`, item.createdAt, item.createdAt, item.id); err != nil {
			t.Fatalf("set handler pending query timestamps: %v", err)
		}
	}

	for _, issueKey := range []string{"missing-cover", "missing-banner", "missing-summary"} {
		if _, err := db.Exec(`
			INSERT INTO game_review_issue_overrides (game_id, issue_key, status)
			VALUES (?, ?, 'ignored')
		`, ignoredID, issueKey); err != nil {
			t.Fatalf("insert ignored override %s: %v", issueKey, err)
		}
	}

	service := services.NewGamesService(
		config.Config{},
		repositories.NewGamesRepository(db),
		repositories.NewGameFilesRepository(db),
		repositories.NewMetadataRepository(db),
		repositories.NewTagsRepository(db),
	)
	handler := NewGamesHandler(service)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(
		http.MethodGet,
		"/api/games?pending=true&pending_include_ignored=true&pending_issue=missing-assets&pending_severe=true&pending_recent_days=30&sort=pending_issue_count&order=desc&search=Pending",
		nil,
	)

	handler.List(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Data []struct {
			ID int64 `json:"id"`
		} `json:"data"`
		Pagination struct {
			Total              int `json:"total"`
			PendingIssueCounts struct {
				Groups       map[string]int `json:"groups"`
				IgnoredTotal int            `json:"ignored_total"`
			} `json:"pending_issue_counts"`
		} `json:"pagination"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Pagination.Total != 2 {
		t.Fatalf("pagination.total = %d, want 2", response.Pagination.Total)
	}
	if len(response.Data) != 2 {
		t.Fatalf("len(data) = %d, want 2", len(response.Data))
	}
	if response.Data[0].ID != severeID || response.Data[1].ID != recentID {
		t.Fatalf("data = %+v, want severe then recent", response.Data)
	}
	if response.Pagination.PendingIssueCounts.Groups["missing-assets"] != 2 ||
		response.Pagination.PendingIssueCounts.Groups["missing-wiki"] != 2 ||
		response.Pagination.PendingIssueCounts.Groups["missing-files"] != 1 ||
		response.Pagination.PendingIssueCounts.Groups["missing-metadata"] != 1 {
		t.Fatalf("pending_issue_counts = %+v, want aggregated native counts", response.Pagination.PendingIssueCounts)
	}
	if response.Pagination.PendingIssueCounts.IgnoredTotal != 0 {
		t.Fatalf("pending_issue_counts.ignored_total = %d, want 0 after native queue filters", response.Pagination.PendingIssueCounts.IgnoredTotal)
	}
}

func TestGamesHandlerListReturnsInternalServerErrorWhenPendingOverridesLookupFails(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	insertGamesHandlerTestGame(t, db, "pending-overrides-error", "Pending Overrides Error", "public", "")
	if _, err := db.Exec(`DROP TABLE game_review_issue_overrides`); err != nil {
		t.Fatalf("drop override table: %v", err)
	}

	service := services.NewGamesService(
		config.Config{},
		repositories.NewGamesRepository(db),
		repositories.NewGameFilesRepository(db),
		repositories.NewMetadataRepository(db),
		repositories.NewTagsRepository(db),
		repositories.NewReviewIssueOverrideRepository(db),
	)
	handler := NewGamesHandler(service)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/games?limit=10", nil)

	handler.List(context)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusInternalServerError, recorder.Body.String())
	}

	var response struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Success {
		t.Fatalf("expected success=false")
	}
	if response.Error != "internal server error" {
		t.Fatalf("error = %q, want internal server error", response.Error)
	}
}

func TestGamesHandlerCreateReturnsBadRequestWhenTitleMissing(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	service := services.NewGamesService(
		config.Config{},
		repositories.NewGamesRepository(db),
		repositories.NewGameFilesRepository(db),
		repositories.NewMetadataRepository(db),
		repositories.NewTagsRepository(db),
	)
	handler := NewGamesHandler(service)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/games", strings.NewReader(`{"title":"   "}`))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Set("is_admin", true)

	handler.Create(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"error":"title is required"`) {
		t.Fatalf("body = %s, want title is required error", recorder.Body.String())
	}
}

func TestGamesHandlerCreateRejectsInvalidJSON(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	service := services.NewGamesService(
		config.Config{},
		repositories.NewGamesRepository(db),
		repositories.NewGameFilesRepository(db),
		repositories.NewMetadataRepository(db),
		repositories.NewTagsRepository(db),
	)
	handler := NewGamesHandler(service)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/games", strings.NewReader("{"))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Set("is_admin", true)

	handler.Create(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"error":"invalid game payload"`) {
		t.Fatalf("body = %s, want invalid game payload", recorder.Body.String())
	}
}

func TestGamesHandlerUpdateAggregateIncludesAssetDeleteWarnings(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertGamesHandlerTestGame(t, db, "aggregate-warning", "Aggregate Warning", "public", "")
	service := services.NewGamesService(
		config.Config{AssetsDir: t.TempDir()},
		repositories.NewGamesRepository(db),
		repositories.NewGameFilesRepository(db),
		repositories.NewMetadataRepository(db),
		repositories.NewTagsRepository(db),
	)
	handler := NewGamesHandler(service)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPut, "/api/games/aggregate-warning/aggregate", strings.NewReader(`{"game":{"title":"Aggregate Warning"},"assets":{"delete_assets":[{"asset_type":"cover","path":"/assets/../bad-cover.png"}]}}`))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Params = gin.Params{{Key: "publicId", Value: "aggregate-warning"}}
	context.Set("is_admin", true)

	handler.UpdateAggregate(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Success bool `json:"success"`
		Data    struct {
			Game struct {
				ID int64 `json:"id"`
			} `json:"game"`
			Warnings struct {
				AssetDeletePaths []string `json:"asset_delete_paths"`
			} `json:"warnings"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success || response.Data.Game.ID != gameID {
		t.Fatalf("response = %s, want updated game %d", recorder.Body.String(), gameID)
	}
	if len(response.Data.Warnings.AssetDeletePaths) != 1 || response.Data.Warnings.AssetDeletePaths[0] != "/assets/../bad-cover.png" {
		t.Fatalf("warnings = %#v, want asset delete warning", response.Data.Warnings.AssetDeletePaths)
	}
}

func TestGamesHandlerUpdateAggregatePreservesOmittedRelations(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertGamesHandlerTestGame(t, db, "aggregate-preserve-relations", "Aggregate Preserve Relations", "public", "")
	developerResult, err := db.Exec(`INSERT INTO developers (name, slug) VALUES (?, ?)`, "Nintendo", "nintendo")
	if err != nil {
		t.Fatalf("insert developer: %v", err)
	}
	developerID, err := developerResult.LastInsertId()
	if err != nil {
		t.Fatalf("developer LastInsertId returned error: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO game_developers (game_id, developer_id, sort_order) VALUES (?, ?, 0)`, gameID, developerID); err != nil {
		t.Fatalf("link game developer: %v", err)
	}

	service := services.NewGamesService(
		config.Config{},
		repositories.NewGamesRepository(db),
		repositories.NewGameFilesRepository(db),
		repositories.NewMetadataRepository(db),
		repositories.NewTagsRepository(db),
	)
	handler := NewGamesHandler(service)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPut, "/api/games/aggregate-preserve-relations/aggregate", strings.NewReader(`{"game":{"title":"Aggregate Preserve Relations Updated"},"assets":{}}`))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Params = gin.Params{{Key: "publicId", Value: "aggregate-preserve-relations"}}
	context.Set("is_admin", true)

	handler.UpdateAggregate(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	developers, err := repositories.NewGamesRepository(db).ListMetadata("developers", "game_developers", "developer_id", gameID)
	if err != nil {
		t.Fatalf("ListMetadata(developers) returned error: %v", err)
	}
	if len(developers) != 1 || developers[0].ID != developerID {
		t.Fatalf("developers = %#v, want existing developer preserved", developers)
	}
}

func TestGamesHandlerUpdateAggregateRejectsNullRelationPatch(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	insertGamesHandlerTestGame(t, db, "aggregate-null-relations", "Aggregate Null Relations", "public", "")
	service := services.NewGamesService(
		config.Config{},
		repositories.NewGamesRepository(db),
		repositories.NewGameFilesRepository(db),
		repositories.NewMetadataRepository(db),
		repositories.NewTagsRepository(db),
	)
	handler := NewGamesHandler(service)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPut, "/api/games/aggregate-null-relations/aggregate", strings.NewReader(`{"game":{"title":"Aggregate Null Relations","developer_ids":null},"assets":{}}`))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Params = gin.Params{{Key: "publicId", Value: "aggregate-null-relations"}}
	context.Set("is_admin", true)

	handler.UpdateAggregate(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"error":"invalid game payload"`) {
		t.Fatalf("body = %s, want invalid game payload", recorder.Body.String())
	}
}

func TestGamesHandlerDeleteReturnsNotFoundForMissingGame(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	service := services.NewGamesService(
		config.Config{},
		repositories.NewGamesRepository(db),
		repositories.NewGameFilesRepository(db),
		repositories.NewMetadataRepository(db),
		repositories.NewTagsRepository(db),
	)
	handler := NewGamesHandler(service)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodDelete, "/api/games/missing", nil)
	context.Params = gin.Params{{Key: "publicId", Value: "missing"}}
	context.Set("is_admin", true)

	handler.Delete(context)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusNotFound, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"error":"resource not found"`) {
		t.Fatalf("body = %s, want resource not found error", recorder.Body.String())
	}
}

func TestGamesHandlerDeleteRemovesExistingGame(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertGamesHandlerTestGame(t, db, "delete-existing", "Delete Existing", "public", "")
	service := services.NewGamesService(
		config.Config{},
		repositories.NewGamesRepository(db),
		repositories.NewGameFilesRepository(db),
		repositories.NewMetadataRepository(db),
		repositories.NewTagsRepository(db),
	)
	handler := NewGamesHandler(service)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodDelete, "/api/games/delete-existing", nil)
	context.Params = gin.Params{{Key: "publicId", Value: "delete-existing"}}
	context.Set("is_admin", true)

	handler.Delete(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Success bool `json:"success"`
		Data    struct {
			Deleted bool `json:"deleted"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success || !response.Data.Deleted {
		t.Fatalf("response = %s, want deleted=true", recorder.Body.String())
	}
	if _, err := repositories.NewGamesRepository(db).GetByID(gameID); err == nil {
		t.Fatalf("expected deleted game to be gone from repository")
	}
}

func TestGamesHandlerUpdateAggregateReturnsBadRequestForInvalidDeleteAssetType(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	insertGamesHandlerTestGame(t, db, "aggregate-invalid-type", "Aggregate Invalid Type", "public", "")
	service := services.NewGamesService(
		config.Config{},
		repositories.NewGamesRepository(db),
		repositories.NewGameFilesRepository(db),
		repositories.NewMetadataRepository(db),
		repositories.NewTagsRepository(db),
	)
	handler := NewGamesHandler(service)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPut, "/api/games/aggregate-invalid-type/aggregate", strings.NewReader(`{"game":{"title":"Aggregate Invalid Type"},"assets":{"delete_assets":[{"asset_type":"manual","path":"/assets/manual.pdf"}]}}`))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Params = gin.Params{{Key: "publicId", Value: "aggregate-invalid-type"}}
	context.Set("is_admin", true)

	handler.UpdateAggregate(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"error":"title is required"`) {
		t.Fatalf("body = %s, want current validation mapping", recorder.Body.String())
	}
}

func TestGamesHandlerUpdateAggregateRejectsInvalidJSONAfterResolvingGame(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	insertGamesHandlerTestGame(t, db, "aggregate-invalid-json", "Aggregate Invalid JSON", "public", "")
	service := services.NewGamesService(
		config.Config{},
		repositories.NewGamesRepository(db),
		repositories.NewGameFilesRepository(db),
		repositories.NewMetadataRepository(db),
		repositories.NewTagsRepository(db),
	)
	handler := NewGamesHandler(service)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPut, "/api/games/aggregate-invalid-json/aggregate", strings.NewReader("{"))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Params = gin.Params{{Key: "publicId", Value: "aggregate-invalid-json"}}
	context.Set("is_admin", true)

	handler.UpdateAggregate(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"error":"invalid game payload"`) {
		t.Fatalf("body = %s, want invalid game payload", recorder.Body.String())
	}
}

func TestGamesHandlerUpdateAggregateReturnsNotFoundForMissingGame(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	service := services.NewGamesService(
		config.Config{},
		repositories.NewGamesRepository(db),
		repositories.NewGameFilesRepository(db),
		repositories.NewMetadataRepository(db),
		repositories.NewTagsRepository(db),
	)
	handler := NewGamesHandler(service)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPut, "/api/games/missing-aggregate/aggregate", strings.NewReader(`{"game":{"title":"Missing Aggregate"},"assets":{}}`))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Params = gin.Params{{Key: "publicId", Value: "missing-aggregate"}}
	context.Set("is_admin", true)

	handler.UpdateAggregate(context)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusNotFound, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"error":"resource not found"`) {
		t.Fatalf("body = %s, want resource not found error", recorder.Body.String())
	}
}

func TestGamesHandlerUpdateAggregateReturnsBadRequestWhenPrimaryROMRootMissing(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	insertGamesHandlerTestGame(t, db, "aggregate-missing-root", "Aggregate Missing Root", "public", "")
	service := services.NewGamesService(
		config.Config{},
		repositories.NewGamesRepository(db),
		repositories.NewGameFilesRepository(db),
		repositories.NewMetadataRepository(db),
		repositories.NewTagsRepository(db),
	)
	handler := NewGamesHandler(service)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPut, "/api/games/aggregate-missing-root/aggregate", strings.NewReader(`{"game":{"title":"Aggregate Missing Root"},"assets":{"files":[{"file_path":"/tmp/demo.rom"}]}}`))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Params = gin.Params{{Key: "publicId", Value: "aggregate-missing-root"}}
	context.Set("is_admin", true)

	handler.UpdateAggregate(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"error":"PRIMARY_ROM_ROOT is not configured"`) {
		t.Fatalf("body = %s, want missing PRIMARY_ROM_ROOT error", recorder.Body.String())
	}
}

func TestGamesHandlerUpdateAggregateReturnsNotFoundForMissingScreenshotReorderUID(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertGamesHandlerTestGame(t, db, "aggregate-missing-shot", "Aggregate Missing Shot", "public", "")
	if _, err := db.Exec(`
		INSERT INTO game_assets (game_id, asset_uid, asset_type, path, sort_order)
		VALUES (?, 'shot-a', 'screenshot', '/assets/aggregate-missing-shot/shot-a.png', 0)
	`, gameID); err != nil {
		t.Fatalf("insert screenshot asset: %v", err)
	}

	service := services.NewGamesService(
		config.Config{},
		repositories.NewGamesRepository(db),
		repositories.NewGameFilesRepository(db),
		repositories.NewMetadataRepository(db),
		repositories.NewTagsRepository(db),
	)
	handler := NewGamesHandler(service)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPut, "/api/games/aggregate-missing-shot/aggregate", strings.NewReader(`{"game":{"title":"Aggregate Missing Shot"},"assets":{"screenshot_order_asset_uids":["missing-shot"]}}`))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Params = gin.Params{{Key: "publicId", Value: "aggregate-missing-shot"}}
	context.Set("is_admin", true)

	handler.UpdateAggregate(context)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusNotFound, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"error":"resource not found"`) {
		t.Fatalf("body = %s, want resource not found error", recorder.Body.String())
	}
}

func TestGamesHandlerUpdateAggregateReturnsNotFoundForMissingVideoReorderUID(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertGamesHandlerTestGame(t, db, "aggregate-missing-video", "Aggregate Missing Video", "public", "")
	if _, err := db.Exec(`
		INSERT INTO game_assets (game_id, asset_uid, asset_type, path, sort_order)
		VALUES (?, 'video-a', 'video', '/assets/aggregate-missing-video/video-a.mp4', 0)
	`, gameID); err != nil {
		t.Fatalf("insert video asset: %v", err)
	}

	service := services.NewGamesService(
		config.Config{},
		repositories.NewGamesRepository(db),
		repositories.NewGameFilesRepository(db),
		repositories.NewMetadataRepository(db),
		repositories.NewTagsRepository(db),
	)
	handler := NewGamesHandler(service)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPut, "/api/games/aggregate-missing-video/aggregate", strings.NewReader(`{"game":{"title":"Aggregate Missing Video"},"assets":{"video_order_asset_uids":["missing-video"]}}`))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Params = gin.Params{{Key: "publicId", Value: "aggregate-missing-video"}}
	context.Set("is_admin", true)

	handler.UpdateAggregate(context)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusNotFound, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"error":"resource not found"`) {
		t.Fatalf("body = %s, want resource not found error", recorder.Body.String())
	}
}

func TestGamesHandlerUpdateAggregateReturnsForbiddenForFileOutsidePrimaryROMRoot(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	root := t.TempDir()
	outsideDir := t.TempDir()
	outsidePath := filepath.Join(outsideDir, "outside.rom")
	if err := os.WriteFile(outsidePath, []byte("outside"), 0o644); err != nil {
		t.Fatalf("WriteFile(outside) returned error: %v", err)
	}

	insertGamesHandlerTestGame(t, db, "aggregate-outside-root", "Aggregate Outside Root", "public", "")
	service := services.NewGamesService(
		config.Config{PrimaryROMRoot: root},
		repositories.NewGamesRepository(db),
		repositories.NewGameFilesRepository(db),
		repositories.NewMetadataRepository(db),
		repositories.NewTagsRepository(db),
	)
	handler := NewGamesHandler(service)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPut, "/api/games/aggregate-outside-root/aggregate", strings.NewReader(fmt.Sprintf(`{"game":{"title":"Aggregate Outside Root"},"assets":{"files":[{"file_path":%q}]}}`, outsidePath)))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Params = gin.Params{{Key: "publicId", Value: "aggregate-outside-root"}}
	context.Set("is_admin", true)

	handler.UpdateAggregate(context)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusForbidden, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"error":"file path is outside PRIMARY_ROM_ROOT"`) {
		t.Fatalf("body = %s, want forbidden path error", recorder.Body.String())
	}
}

func openGamesHandlerTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := dbpkg.OpenSQLite(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("OpenSQLite returned error: %v", err)
	}
	if err := dbpkg.RunMigrations(db); err != nil {
		_ = db.Close()
		t.Fatalf("RunMigrations returned error: %v", err)
	}

	return db
}

func insertGamesHandlerTestGame(t *testing.T, db *sqlx.DB, publicID string, title string, visibility string, releaseDate string) int64 {
	t.Helper()

	result, err := db.Exec(`
		INSERT INTO games (public_id, title, visibility, release_date)
		VALUES (?, ?, ?, ?)
	`, publicID, title, visibility, releaseDate)
	if err != nil {
		t.Fatalf("insert test game: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId returned error: %v", err)
	}

	return id
}

func insertGamesHandlerMetadataItem(t *testing.T, db *sqlx.DB, table string, name string, slug string) int64 {
	t.Helper()

	result, err := db.Exec(fmt.Sprintf(`
		INSERT INTO %s (name, slug)
		VALUES (?, ?)
	`, table), name, slug)
	if err != nil {
		t.Fatalf("insert %s item: %v", table, err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId returned error: %v", err)
	}

	return id
}

func linkGamesHandlerGameRelation(t *testing.T, db *sqlx.DB, table string, column string, gameID int64, relationID int64) {
	t.Helper()

	if _, err := db.Exec(fmt.Sprintf(`
		INSERT INTO %s (game_id, %s, sort_order)
		VALUES (?, ?, 0)
	`, table, column), gameID, relationID); err != nil {
		t.Fatalf("link %s relation: %v", table, err)
	}
}
