package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/files"
	"github.com/hao/game/internal/repositories"
	"github.com/hao/game/internal/services"
)

func TestGameFileRefreshHandlerHidesRepositoryError(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	if err := db.Close(); err != nil {
		t.Fatalf("close test db: %v", err)
	}

	handler := NewGameFileRefreshHandler(services.NewGameFileRefreshService(
		repositories.NewGameFilesRepository(db),
		files.NewGuard(t.TempDir()),
	))

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/games/refresh-sizes", nil)
	context.Set("is_admin", true)

	handler.RefreshSizes(context)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	var response struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Error != "刷新文件大小失败" {
		t.Fatalf("error = %q, want 刷新文件大小失败", response.Error)
	}
}
