package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/config"
)

func TestAuthHandlerMeReturnsGuestRoleForAnonymousRequest(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)

	handler := NewAuthHandler(nil, config.Config{AdminDisplayName: "Admin"})
	handler.Me(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response struct {
		Success bool `json:"success"`
		Data    struct {
			IsAdmin          bool   `json:"is_admin"`
			Role             string `json:"role"`
			AdminDisplayName string `json:"admin_display_name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if !response.Success {
		t.Fatalf("expected success=true")
	}
	if response.Data.IsAdmin {
		t.Fatalf("expected is_admin=false")
	}
	if response.Data.Role != "guest" {
		t.Fatalf("expected role=guest, got %q", response.Data.Role)
	}
	if response.Data.AdminDisplayName != "Admin" {
		t.Fatalf("expected admin_display_name=Admin, got %q", response.Data.AdminDisplayName)
	}
}

func TestAuthHandlerMeReturnsAdminRoleForAdminRequest(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	context.Set("is_admin", true)

	handler := NewAuthHandler(nil, config.Config{AdminDisplayName: "Boss"})
	handler.Me(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response struct {
		Data struct {
			IsAdmin          bool   `json:"is_admin"`
			Role             string `json:"role"`
			AdminDisplayName string `json:"admin_display_name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if !response.Data.IsAdmin {
		t.Fatalf("expected is_admin=true")
	}
	if response.Data.Role != "admin" {
		t.Fatalf("expected role=admin, got %q", response.Data.Role)
	}
	if response.Data.AdminDisplayName != "Boss" {
		t.Fatalf("expected admin_display_name=Boss, got %q", response.Data.AdminDisplayName)
	}
}
