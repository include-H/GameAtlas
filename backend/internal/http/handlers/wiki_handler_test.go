package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/repositories"
	"github.com/hao/game/internal/services"
)

func TestWikiHandlerHistoryReturnsFormattedAdminResponse(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertGamesHandlerTestGame(t, db, "wiki-history", "Wiki History", "public", "")
	firstSummary := "first edit"
	secondSummary := "second edit"
	if _, err := db.Exec(`
		INSERT INTO wiki_history (game_id, content, change_summary)
		VALUES (?, ?, ?), (?, ?, ?)
	`, gameID, "first", &firstSummary, gameID, "second", &secondSummary); err != nil {
		t.Fatalf("insert wiki history: %v", err)
	}

	service := services.NewWikiService(
		repositories.NewGamesRepository(db),
		repositories.NewWikiRepository(db),
		10,
	)
	handler := NewWikiHandler(service)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/games/wiki-history/wiki/history", nil)
	context.Params = gin.Params{{Key: "publicId", Value: "wiki-history"}}
	context.Set("is_admin", true)

	handler.History(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response struct {
		Success bool `json:"success"`
		Data    []struct {
			GameID        int64   `json:"game_id"`
			Content       string  `json:"content"`
			ChangeSummary *string `json:"change_summary"`
		} `json:"data"`
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
	if response.Data[0].GameID != gameID || response.Data[0].Content != "second" {
		t.Fatalf("data[0] = %+v, want latest history row", response.Data[0])
	}
	if response.Data[0].ChangeSummary == nil || *response.Data[0].ChangeSummary != secondSummary {
		t.Fatalf("data[0].change_summary = %v, want %q", response.Data[0].ChangeSummary, secondSummary)
	}
	if response.Data[1].Content != "first" {
		t.Fatalf("data[1] = %+v, want older history row", response.Data[1])
	}
}

func TestWikiHandlerGetReturnsNotFoundForUnknownGame(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	service := services.NewWikiService(
		repositories.NewGamesRepository(db),
		repositories.NewWikiRepository(db),
		10,
	)
	handler := NewWikiHandler(service)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/games/missing/wiki", nil)
	context.Params = gin.Params{{Key: "publicId", Value: "missing"}}

	handler.Get(context)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
	if !strings.Contains(recorder.Body.String(), `"error":"resource not found"`) {
		t.Fatalf("body = %s, want resource not found error", recorder.Body.String())
	}
}

func TestWikiHandlerUpdateRejectsInvalidJSONAfterResolvingGame(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertGamesHandlerTestGame(t, db, "wiki-update", "Wiki Update", domain.GameVisibilityPublic, "")
	if gameID <= 0 {
		t.Fatalf("expected inserted game id > 0")
	}

	service := services.NewWikiService(
		repositories.NewGamesRepository(db),
		repositories.NewWikiRepository(db),
		10,
	)
	handler := NewWikiHandler(service)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPut, "/api/games/wiki-update/wiki", strings.NewReader("{"))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Params = gin.Params{{Key: "publicId", Value: "wiki-update"}}
	context.Set("is_admin", true)

	handler.Update(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if !strings.Contains(recorder.Body.String(), `"error":"invalid wiki payload"`) {
		t.Fatalf("body = %s, want invalid wiki payload error", recorder.Body.String())
	}
}
