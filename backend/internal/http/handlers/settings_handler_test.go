package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/config"
	"github.com/hao/game/internal/repositories"
	"github.com/hao/game/internal/services"
)

func TestSettingsHandlerUpdateConfigRejectsInvalidRuntimeValue(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openGamesHandlerTestDB(t)
	defer func() { _ = db.Close() }()

	cfg := config.Config{AdminPassword: "secret"}
	repo := repositories.NewAppSettingsRepository(db)
	if err := repo.EnsureDefaults(cfg.RuntimeSettings()); err != nil {
		t.Fatalf("EnsureDefaults returned error: %v", err)
	}

	handler := NewSettingsHandler(services.NewSettingsService(cfg, repo))

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPut, "/api/settings/config", strings.NewReader(`{"PORT":"abc"}`))
	context.Set("is_admin", true)

	handler.UpdateConfig(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	var response struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Error != "保存配置失败" {
		t.Fatalf("error = %q, want 保存配置失败", response.Error)
	}
}
