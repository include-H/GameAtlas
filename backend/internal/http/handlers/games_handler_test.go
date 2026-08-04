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
	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/repositories"
)

func TestGamesHandlerListTimelineRejectsInvalidCursor(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/games/timeline?cursor=bad", nil)

	handler := NewSplitGamesHandler(nil, nil, nil, nil, nil)
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
	if response.Error != "无效的时间线游标" {
		t.Fatalf("error = %q, want 无效的时间线游标", response.Error)
	}
}

func TestGamesHandlerListTimelineRejectsInvalidLimit(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/games/timeline?limit=oops", nil)

	handler := NewSplitGamesHandler(nil, nil, nil, nil, nil)
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
	if response.Error != "无效的时间线查询参数: limit" {
		t.Fatalf("error = %q, want 无效的时间线查询参数: limit", response.Error)
	}
}

func TestGamesHandlerListTimelinePaginatesWithCursor(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	_ = insertGamesHandlerTestGame(t, db, "private-new", "Private New", "private", "2025-01-01")
	firstID := insertGamesHandlerTestGame(t, db, "public-new", "Public New", "public", "2024-02-01")
	secondID := insertGamesHandlerTestGame(t, db, "public-mid", "Public Mid", "public", "2023-05-01")
	thirdID := insertGamesHandlerTestGame(t, db, "public-old", "Public Old", "public", "2022-03-01")
	_ = insertGamesHandlerTestGame(t, db, "public-ancient", "Public Ancient", "public", "2019-06-01")

	handler := newSplitGamesHandlerForTest(config.Config{}, db)

	// First page: limit=2
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/games/timeline?limit=2", nil)
	handler.ListTimeline(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var page1 struct {
		Success bool `json:"success"`
		Data    []struct {
			ID       int64  `json:"id"`
			PublicID string `json:"public_id"`
		} `json:"data"`
		Pagination struct {
			Limit      int    `json:"limit"`
			HasMore    bool   `json:"hasMore"`
			NextCursor string `json:"nextCursor"`
		} `json:"pagination"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &page1); err != nil {
		t.Fatalf("decode page1: %v", err)
	}
	if !page1.Success {
		t.Fatalf("expected success=true")
	}
	if len(page1.Data) != 2 {
		t.Fatalf("len(data) = %d, want 2", len(page1.Data))
	}
	if page1.Data[0].ID != firstID || page1.Data[1].ID != secondID {
		t.Fatalf("page1 data = %+v, want first two public games", page1.Data)
	}
	if !page1.Pagination.HasMore {
		t.Fatalf("expected hasMore=true")
	}
	if page1.Pagination.NextCursor == "" {
		t.Fatalf("expected non-empty nextCursor")
	}

	// Second page: use cursor
	recorder2 := httptest.NewRecorder()
	context2, _ := gin.CreateTestContext(recorder2)
	context2.Request = httptest.NewRequest(http.MethodGet, "/api/games/timeline?limit=2&cursor="+page1.Pagination.NextCursor, nil)
	handler.ListTimeline(context2)

	var page2 struct {
		Success bool `json:"success"`
		Data    []struct {
			ID       int64  `json:"id"`
			PublicID string `json:"public_id"`
		} `json:"data"`
		Pagination struct {
			HasMore    bool   `json:"hasMore"`
			NextCursor string `json:"nextCursor"`
		} `json:"pagination"`
	}
	if err := json.Unmarshal(recorder2.Body.Bytes(), &page2); err != nil {
		t.Fatalf("decode page2: %v", err)
	}
	if len(page2.Data) != 2 {
		t.Fatalf("len(data) = %d, want 2", len(page2.Data))
	}
	if page2.Data[0].ID != thirdID {
		t.Fatalf("page2 data[0] = %+v, want public-old", page2.Data[0])
	}
	if page2.Pagination.HasMore {
		t.Fatalf("expected hasMore=false (no games after ancient)")
	}
}

func TestGamesHandlerStatsIncludesHomeFeedCollections(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	firstID := insertGamesHandlerTestGame(t, db, "stats-home-a", "Stats Home A", "public", "2024-01-01")
	secondID := insertGamesHandlerTestGame(t, db, "stats-home-b", "Stats Home B", "public", "2024-02-01")
	if _, err := db.Exec(`UPDATE games SET updated_at = ? WHERE id = ?`, "2026-01-02 00:00:00", firstID); err != nil {
		t.Fatalf("set first stats game updated_at: %v", err)
	}
	if _, err := db.Exec(`UPDATE games SET updated_at = ? WHERE id = ?`, "2026-01-03 00:00:00", secondID); err != nil {
		t.Fatalf("set second stats game updated_at: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO favorite_games (game_id) VALUES (?)`, secondID); err != nil {
		t.Fatalf("insert stats favorite game: %v", err)
	}

	handler := newSplitGamesHandlerForTest(config.Config{}, db)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/games/stats", nil)
	handler.Stats(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Success bool `json:"success"`
		Data    struct {
			TotalGames  int `json:"total_games"`
			RecentGames []struct {
				ID int64 `json:"id"`
			} `json:"recent_games"`
			RecentlyUpdatedGames []struct {
				ID int64 `json:"id"`
			} `json:"recently_updated_games"`
			PopularGames []struct {
				ID int64 `json:"id"`
			} `json:"popular_games"`
			FavoriteGames []struct {
				ID int64 `json:"id"`
			} `json:"favorite_games"`
			FavoriteCount  int `json:"favorite_count"`
			PendingReviews int `json:"pending_reviews"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if !response.Success {
		t.Fatalf("expected success=true")
	}
	if response.Data.TotalGames != 2 {
		t.Fatalf("total_games = %d, want 2", response.Data.TotalGames)
	}
	if len(response.Data.RecentGames) != 2 {
		t.Fatalf("recent_games = %d, want 2", len(response.Data.RecentGames))
	}
	if len(response.Data.RecentlyUpdatedGames) != 2 || response.Data.RecentlyUpdatedGames[0].ID != secondID {
		t.Fatalf("recently_updated_games = %+v, want second game first", response.Data.RecentlyUpdatedGames)
	}
	if len(response.Data.PopularGames) != 2 {
		t.Fatalf("popular_games = %d, want 2", len(response.Data.PopularGames))
	}
	if len(response.Data.FavoriteGames) != 1 || response.Data.FavoriteGames[0].ID != secondID {
		t.Fatalf("favorite_games = %+v, want only second game", response.Data.FavoriteGames)
	}
	if response.Data.FavoriteCount != 1 {
		t.Fatalf("favorite_count = %d, want 1", response.Data.FavoriteCount)
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

	handler := newSplitGamesHandlerForTest(config.Config{}, db)

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
			PreviewVideos []map[string]any `json:"preview_videos"`
			Files         []map[string]any `json:"files"`
		} `json:"data"`
	}
	if err := json.Unmarshal(publicRecorder.Body.Bytes(), &publicResponse); err != nil {
		t.Fatalf("decode public response: %v", err)
	}
	if len(publicResponse.Data.PreviewVideos) != 1 {
		t.Fatalf("len(public preview_videos) = %d, want 1", len(publicResponse.Data.PreviewVideos))
	}
	if publicResponse.Data.PreviewVideos[0]["path"] != "/assets/detail-paths/video-a.mp4" {
		t.Fatalf("public preview_videos[0] = %#v, want first sorted path included", publicResponse.Data.PreviewVideos[0])
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
		VALUES (?, 'resolved-logo', 'logo', '/assets/pending-resolved/logo.png', 0)
	`, resolvedID); err != nil {
		t.Fatalf("insert resolved logo: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO game_assets (game_id, asset_uid, asset_type, path, sort_order)
		VALUES (?, 'ignored-shot', 'screenshot', '/assets/pending-ignored/shot.png', 0)
	`, ignoredID); err != nil {
		t.Fatalf("insert ignored screenshot: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO game_assets (game_id, asset_uid, asset_type, path, sort_order)
		VALUES (?, 'ignored-logo', 'logo', '/assets/pending-ignored/logo.png', 0)
	`, ignoredID); err != nil {
		t.Fatalf("insert ignored logo: %v", err)
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

	developerID := insertGamesHandlerMetadataItem(t, db, "developers", "Studio", "studio")
	publisherID := insertGamesHandlerMetadataItem(t, db, "publishers", "Publisher", "publisher")
	linkGamesHandlerGameRelation(t, db, "game_developers", "developer_id", resolvedID, developerID)
	linkGamesHandlerGameRelation(t, db, "game_publishers", "publisher_id", resolvedID, publisherID)
	linkGamesHandlerGameRelation(t, db, "game_developers", "developer_id", ignoredID, developerID)
	linkGamesHandlerGameRelation(t, db, "game_publishers", "publisher_id", ignoredID, publisherID)

	if _, err := db.Exec(`
		INSERT INTO game_review_issue_overrides (game_id, issue_key, status)
		VALUES (?, 'missing-cover', 'ignored')
	`, ignoredID); err != nil {
		t.Fatalf("insert ignored override: %v", err)
	}

	handler := newSplitGamesHandlerForTest(config.Config{}, db)

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

	developerID := insertGamesHandlerMetadataItem(t, db, "developers", "Query Studio", "query-studio")
	publisherID := insertGamesHandlerMetadataItem(t, db, "publishers", "Query Publisher", "query-publisher")
	linkGamesHandlerGameRelation(t, db, "game_developers", "developer_id", recentID, developerID)
	linkGamesHandlerGameRelation(t, db, "game_publishers", "publisher_id", recentID, publisherID)
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

	handler := newSplitGamesHandlerForTest(config.Config{}, db)

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

	handler := newSplitGamesHandlerForTest(config.Config{}, db)

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
	if response.Error != "服务器内部错误" {
		t.Fatalf("error = %q, want 服务器内部错误", response.Error)
	}
}

func TestGamesHandlerListRejectsInvalidPageQuery(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	handler := newSplitGamesHandlerForTest(config.Config{}, db)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/games?page=abc", nil)

	handler.List(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"error":"无效的游戏查询参数: page"`) {
		t.Fatalf("body = %s, want 无效的游戏查询参数: page", recorder.Body.String())
	}
}

func TestGamesHandlerListRejectsInvalidPendingQueryBoolean(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	handler := newSplitGamesHandlerForTest(config.Config{}, db)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/games?pending=maybe", nil)

	handler.List(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"error":"无效的游戏查询参数: pending"`) {
		t.Fatalf("body = %s, want 无效的游戏查询参数: pending", recorder.Body.String())
	}
}

func TestGamesHandlerListRejectsFavoriteFalseQuery(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	handler := newSplitGamesHandlerForTest(config.Config{}, db)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/games?favorite=false", nil)

	handler.List(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"error":"无效的游戏查询参数: favorite"`) {
		t.Fatalf("body = %s, want 无效的游戏查询参数: favorite", recorder.Body.String())
	}
}

func TestGamesHandlerListRejectsRandomSortWithoutSeed(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	insertGamesHandlerTestGame(t, db, "random-seed-required", "Random Seed Required", "public", "")

	handler := newSplitGamesHandlerForTest(config.Config{}, db)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/games?sort=random&order=desc", nil)

	handler.List(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"error":"无效的游戏查询参数: seed"`) {
		t.Fatalf("body = %s, want 无效的游戏查询参数: seed", recorder.Body.String())
	}
}

func TestGamesHandlerCreateReturnsBadRequestWhenTitleMissing(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	handler := newSplitGamesHandlerForTest(config.Config{}, db)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/games", strings.NewReader(`{"title":"   "}`))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Set("is_admin", true)

	handler.Create(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"error":"标题为必填项"`) {
		t.Fatalf("body = %s, want 标题为必填项", recorder.Body.String())
	}
}

func TestGamesHandlerCreateRejectsInvalidJSON(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	handler := newSplitGamesHandlerForTest(config.Config{}, db)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/games", strings.NewReader("{"))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Set("is_admin", true)

	handler.Create(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"error":"无效的游戏请求"`) {
		t.Fatalf("body = %s, want 无效的游戏请求", recorder.Body.String())
	}
}

func TestGamesHandlerCreateRejectsUnknownFields(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	handler := newSplitGamesHandlerForTest(config.Config{}, db)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/games", strings.NewReader(`{"title":"Unknown Field","assets":{}}`))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Set("is_admin", true)

	handler.Create(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"error":"无效的游戏请求"`) {
		t.Fatalf("body = %s, want 无效的游戏请求", recorder.Body.String())
	}
}

func TestGamesHandlerFavoriteEndpointsPersistState(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	insertGamesHandlerTestGame(t, db, "favorite-toggle", "Favorite Toggle", "public", "")
	handler := newSplitGamesHandlerForTest(config.Config{}, db)

	favoriteRecorder := httptest.NewRecorder()
	favoriteContext, _ := gin.CreateTestContext(favoriteRecorder)
	favoriteContext.Request = httptest.NewRequest(http.MethodPut, "/api/games/favorite-toggle/favorite", nil)
	favoriteContext.Params = gin.Params{{Key: "publicId", Value: "favorite-toggle"}}

	handler.Favorite(favoriteContext)

	if favoriteRecorder.Code != http.StatusOK {
		t.Fatalf("favorite status = %d, want %d, body=%s", favoriteRecorder.Code, http.StatusOK, favoriteRecorder.Body.String())
	}
	if !strings.Contains(favoriteRecorder.Body.String(), `"is_favorite":true`) {
		t.Fatalf("favorite body = %s, want is_favorite true", favoriteRecorder.Body.String())
	}

	unfavoriteRecorder := httptest.NewRecorder()
	unfavoriteContext, _ := gin.CreateTestContext(unfavoriteRecorder)
	unfavoriteContext.Request = httptest.NewRequest(http.MethodDelete, "/api/games/favorite-toggle/favorite", nil)
	unfavoriteContext.Params = gin.Params{{Key: "publicId", Value: "favorite-toggle"}}

	handler.Unfavorite(unfavoriteContext)

	if unfavoriteRecorder.Code != http.StatusOK {
		t.Fatalf("unfavorite status = %d, want %d, body=%s", unfavoriteRecorder.Code, http.StatusOK, unfavoriteRecorder.Body.String())
	}
	if !strings.Contains(unfavoriteRecorder.Body.String(), `"is_favorite":false`) {
		t.Fatalf("unfavorite body = %s, want is_favorite false", unfavoriteRecorder.Body.String())
	}
}

func TestGamesHandlerUpdateAggregateIncludesAssetDeleteWarnings(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertGamesHandlerTestGame(t, db, "aggregate-warning", "Aggregate Warning", "public", "")
	if _, err := db.Exec(`INSERT INTO game_assets (game_id, asset_uid, asset_type, path, sort_order) VALUES (?, ?, 'cover', ?, 0)`, gameID, "bad-cover", "/assets/../bad-cover.png"); err != nil {
		t.Fatalf("insert cover asset: %v", err)
	}
	handler := newSplitGamesHandlerForTest(config.Config{AssetsDir: t.TempDir()}, db)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPut, "/api/games/aggregate-warning/aggregate", strings.NewReader(`{"game":{"title":"Aggregate Warning","visibility":"public"},"assets":{}}`))
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

func TestGamesHandlerUpdateAggregateReplacesRelations(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertGamesHandlerTestGame(t, db, "aggregate-replace-relations", "Aggregate Replace Relations", "public", "")
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
	_ = developerID

	handler := newSplitGamesHandlerForTest(config.Config{}, db)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPut, "/api/games/aggregate-replace-relations/aggregate", strings.NewReader(`{"game":{"title":"Aggregate Replace Relations Updated","visibility":"public","series_id":null,"developer_ids":[],"publisher_ids":[]},"assets":{}}`))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Params = gin.Params{{Key: "publicId", Value: "aggregate-replace-relations"}}
	context.Set("is_admin", true)

	handler.UpdateAggregate(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	developers, err := repositories.NewGamesRepository(db).ListMetadata(domain.MetadataDevelopers, gameID)
	if err != nil {
		t.Fatalf("ListMetadata(developers) returned error: %v", err)
	}
	if len(developers) != 0 {
		t.Fatalf("developers = %#v, want cleared developers", developers)
	}
}

func TestGamesHandlerDeleteReturnsNotFoundForMissingGame(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	handler := newSplitGamesHandlerForTest(config.Config{}, db)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodDelete, "/api/games/missing", nil)
	context.Params = gin.Params{{Key: "publicId", Value: "missing"}}
	context.Set("is_admin", true)

	handler.Delete(context)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusNotFound, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"error":"资源不存在"`) {
		t.Fatalf("body = %s, want 资源不存在", recorder.Body.String())
	}
}

func TestGamesHandlerDeleteRemovesExistingGame(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertGamesHandlerTestGame(t, db, "delete-existing", "Delete Existing", "public", "")
	handler := newSplitGamesHandlerForTest(config.Config{}, db)

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
			Deleted  bool `json:"deleted"`
			Warnings *struct {
				AssetDeletePaths []string `json:"asset_delete_paths"`
			} `json:"warnings"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success || !response.Data.Deleted {
		t.Fatalf("response = %s, want deleted=true", recorder.Body.String())
	}
	if response.Data.Warnings != nil {
		t.Fatalf("response warnings = %#v, want nil", response.Data.Warnings)
	}
	if _, err := repositories.NewGamesRepository(db).GetByID(gameID); err == nil {
		t.Fatalf("expected deleted game to be gone from repository")
	}
}

func TestGamesHandlerDeleteReturnsAssetDeleteWarnings(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertGamesHandlerTestGame(t, db, "delete-warning", "Delete Warning", "public", "")
	if _, err := db.Exec(`UPDATE games SET cover_image = ? WHERE id = ?`, "/assets/../bad-cover.png", gameID); err != nil {
		t.Fatalf("set cover image: %v", err)
	}

	handler := newSplitGamesHandlerForTest(config.Config{AssetsDir: filepath.Join(t.TempDir(), "assets")}, db)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodDelete, "/api/games/delete-warning", nil)
	context.Params = gin.Params{{Key: "publicId", Value: "delete-warning"}}
	context.Set("is_admin", true)

	handler.Delete(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Success bool `json:"success"`
		Data    struct {
			Deleted  bool `json:"deleted"`
			Warnings struct {
				AssetDeletePaths []string `json:"asset_delete_paths"`
			} `json:"warnings"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success || !response.Data.Deleted {
		t.Fatalf("response = %s, want deleted=true", recorder.Body.String())
	}
	if len(response.Data.Warnings.AssetDeletePaths) != 1 || response.Data.Warnings.AssetDeletePaths[0] != "/assets/../bad-cover.png" {
		t.Fatalf("warnings = %#v, want bad cover path", response.Data.Warnings.AssetDeletePaths)
	}
}

func TestGamesHandlerUpdateAggregateRejectsInvalidJSONAfterResolvingGame(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	insertGamesHandlerTestGame(t, db, "aggregate-invalid-json", "Aggregate Invalid JSON", "public", "")
	handler := newSplitGamesHandlerForTest(config.Config{}, db)

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
	if !strings.Contains(recorder.Body.String(), `"error":"无效的游戏请求"`) {
		t.Fatalf("body = %s, want 无效的游戏请求", recorder.Body.String())
	}
}

func TestGamesHandlerUpdateAggregateRejectsLegacyUnknownFields(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	insertGamesHandlerTestGame(t, db, "aggregate-unknown-field", "Aggregate Unknown Field", "public", "")
	handler := newSplitGamesHandlerForTest(config.Config{}, db)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPut, "/api/games/aggregate-unknown-field/aggregate", strings.NewReader(`{"game":{"title":"Aggregate Unknown Field","visibility":"public"},"assets":{"files":[{"file_path":"/tmp/demo.rom","sort_order":99}]}}`))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Params = gin.Params{{Key: "publicId", Value: "aggregate-unknown-field"}}
	context.Set("is_admin", true)

	handler.UpdateAggregate(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"error":"无效的游戏请求"`) {
		t.Fatalf("body = %s, want 无效的游戏请求", recorder.Body.String())
	}
}

func TestGamesHandlerUpdateAggregateReturnsNotFoundForMissingGame(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	handler := newSplitGamesHandlerForTest(config.Config{}, db)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPut, "/api/games/missing-aggregate/aggregate", strings.NewReader(`{"game":{"title":"Missing Aggregate","visibility":"public"},"assets":{}}`))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Params = gin.Params{{Key: "publicId", Value: "missing-aggregate"}}
	context.Set("is_admin", true)

	handler.UpdateAggregate(context)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusNotFound, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"error":"资源不存在"`) {
		t.Fatalf("body = %s, want 资源不存在", recorder.Body.String())
	}
}

func TestGamesHandlerUpdateAggregateReturnsBadRequestWhenPrimaryROMRootMissing(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	insertGamesHandlerTestGame(t, db, "aggregate-missing-root", "Aggregate Missing Root", "public", "")
	handler := newSplitGamesHandlerForTest(config.Config{}, db)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPut, "/api/games/aggregate-missing-root/aggregate", strings.NewReader(`{"game":{"title":"Aggregate Missing Root","visibility":"public"},"assets":{"files":[{"file_path":"/tmp/demo.rom"}]}}`))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Params = gin.Params{{Key: "publicId", Value: "aggregate-missing-root"}}
	context.Set("is_admin", true)

	handler.UpdateAggregate(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"error":"服务配置不完整"`) {
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

	handler := newSplitGamesHandlerForTest(config.Config{}, db)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPut, "/api/games/aggregate-missing-shot/aggregate", strings.NewReader(`{"game":{"title":"Aggregate Missing Shot","visibility":"public"},"assets":{"screenshot_order_asset_uids":["missing-shot"]}}`))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Params = gin.Params{{Key: "publicId", Value: "aggregate-missing-shot"}}
	context.Set("is_admin", true)

	handler.UpdateAggregate(context)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusNotFound, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"error":"资源不存在"`) {
		t.Fatalf("body = %s, want 资源不存在", recorder.Body.String())
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

	handler := newSplitGamesHandlerForTest(config.Config{}, db)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPut, "/api/games/aggregate-missing-video/aggregate", strings.NewReader(`{"game":{"title":"Aggregate Missing Video","visibility":"public"},"assets":{"video_order_asset_uids":["missing-video"]}}`))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Params = gin.Params{{Key: "publicId", Value: "aggregate-missing-video"}}
	context.Set("is_admin", true)

	handler.UpdateAggregate(context)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusNotFound, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"error":"资源不存在"`) {
		t.Fatalf("body = %s, want 资源不存在", recorder.Body.String())
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
	handler := newSplitGamesHandlerForTest(config.Config{PrimaryROMRoot: root}, db)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPut, "/api/games/aggregate-outside-root/aggregate", strings.NewReader(fmt.Sprintf(`{"game":{"title":"Aggregate Outside Root","visibility":"public"},"assets":{"files":[{"file_path":%q}]}}`, outsidePath)))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Params = gin.Params{{Key: "publicId", Value: "aggregate-outside-root"}}
	context.Set("is_admin", true)

	handler.UpdateAggregate(context)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusForbidden, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"error":"文件路径超出允许范围"`) {
		t.Fatalf("body = %s, want 文件路径超出允许范围", recorder.Body.String())
	}
}

func TestGamesHandlerListRejectsInvalidSortQuery(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	handler := newSplitGamesHandlerForTest(config.Config{}, db)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/games?sort=legacy_default", nil)

	handler.List(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if !strings.Contains(recorder.Body.String(), `"error":"无效的游戏查询参数: sort"`) {
		t.Fatalf("body = %s, want 无效的游戏查询参数: sort", recorder.Body.String())
	}
}

func TestGamesHandlerListRejectsInvalidOrderQuery(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	handler := newSplitGamesHandlerForTest(config.Config{}, db)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/games?sort=updated_at&order=sideways", nil)

	handler.List(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if !strings.Contains(recorder.Body.String(), `"error":"无效的游戏查询参数: order"`) {
		t.Fatalf("body = %s, want 无效的游戏查询参数: order", recorder.Body.String())
	}
}

func TestGamesHandlerListDefaultsAndClampsTransportListParamsBeforeService(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	firstID := insertGamesHandlerTestGame(t, db, "transport-default-a", "Transport Default A", "public", "")
	secondID := insertGamesHandlerTestGame(t, db, "transport-default-b", "Transport Default B", "public", "")
	if _, err := db.Exec(`
		UPDATE games
		SET updated_at = CASE id
			WHEN ? THEN '2026-04-22 12:00:00'
			WHEN ? THEN '2026-04-22 11:00:00'
		END
		WHERE id IN (?, ?)
	`, firstID, secondID, firstID, secondID); err != nil {
		t.Fatalf("set updated_at ordering: %v", err)
	}

	handler := newSplitGamesHandlerForTest(config.Config{}, db)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/games?page=0&limit=999", nil)

	handler.List(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Data []struct {
			ID int64 `json:"id"`
		} `json:"data"`
		Pagination struct {
			Page  int `json:"page"`
			Limit int `json:"limit"`
			Total int `json:"total"`
		} `json:"pagination"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Pagination.Page != 1 {
		t.Fatalf("pagination.page = %d, want 1 after transport fallback", response.Pagination.Page)
	}
	if response.Pagination.Limit != 100 {
		t.Fatalf("pagination.limit = %d, want 100 after transport clamp", response.Pagination.Limit)
	}
	if response.Pagination.Total != 2 || len(response.Data) != 2 {
		t.Fatalf("response = %+v, want both games in the first page", response)
	}
	if response.Data[0].ID != firstID || response.Data[1].ID != secondID {
		t.Fatalf("data = %+v, want updated_at desc transport defaults", response.Data)
	}
}

func TestGamesHandlerListRejectsInvalidPendingIssueQuery(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	handler := newSplitGamesHandlerForTest(config.Config{}, db)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/games?pending_issue=legacy-default", nil)

	handler.List(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if !strings.Contains(recorder.Body.String(), `"error":"无效的游戏查询参数: pending_issue"`) {
		t.Fatalf("body = %s, want 无效的游戏查询参数: pending_issue", recorder.Body.String())
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
