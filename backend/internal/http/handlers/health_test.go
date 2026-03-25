package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"

	dbpkg "github.com/hao/game/internal/db"
)

func TestHealthHandlerGet(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db, err := dbpkg.OpenSQLite(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("OpenSQLite returned error: %v", err)
	}
	defer func() { _ = db.Close() }()

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/health", nil)

	handler := NewHealthHandler(db)
	handler.Get(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response struct {
		Success bool `json:"success"`
		Data    struct {
			Status    string `json:"status"`
			Timestamp string `json:"timestamp"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success || response.Data.Status != "ok" || response.Data.Timestamp == "" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestHealthHandlerGetReturnsServiceUnavailableWhenDatabaseClosed(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db, err := dbpkg.OpenSQLite(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("OpenSQLite returned error: %v", err)
	}
	_ = db.Close()

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/health", nil)

	handler := NewHealthHandler(db)
	handler.Get(context)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
}
