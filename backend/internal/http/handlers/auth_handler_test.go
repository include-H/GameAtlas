package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/config"
	dbpkg "github.com/hao/game/internal/db"
	"github.com/hao/game/internal/services"
)

func TestAuthHandlerLoginReturnsBadRequestForInvalidPayload(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader("{"))
	context.Request.Header.Set("Content-Type", "application/json")

	handler := NewAuthHandler(services.NewAuthService(config.Config{}, nil), config.Config{AdminDisplayName: "Admin"})
	handler.Login(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestAuthHandlerLoginReturnsServiceUnavailableWhenDisabled(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"password":"secret"}`))
	context.Request.Header.Set("Content-Type", "application/json")

	handler := NewAuthHandler(services.NewAuthService(config.Config{}, nil), config.Config{AdminDisplayName: "Admin"})
	handler.Login(context)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
}

func TestAuthHandlerLoginSetsCookieOnSuccess(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"password":"secret"}`))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Request.Header.Set("User-Agent", "test-agent")
	context.Request.RemoteAddr = "127.0.0.1:1234"

	service := services.NewAuthService(config.Config{
		AdminPassword: "secret",
		SessionSecret: "session-secret",
	}, nil)
	handler := NewAuthHandler(service, config.Config{AdminDisplayName: "Admin"})
	handler.Login(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	cookies := recorder.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Name != services.AuthCookieName || cookies[0].Value == "" {
		t.Fatalf("expected auth cookie to be set, got %#v", cookies)
	}
}

func TestAuthHandlerLogoutClearsCookie(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)

	handler := NewAuthHandler(nil, config.Config{})
	handler.Logout(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	cookies := recorder.Result().Cookies()
	if len(cookies) == 0 || cookies[0].MaxAge >= 0 {
		t.Fatalf("expected cookie to be cleared, got %#v", cookies)
	}
}

func TestAuthHandlerLoginReturnsUnauthorizedForInvalidPassword(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db, err := dbpkg.OpenSQLite(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("OpenSQLite returned error: %v", err)
	}
	defer func() { _ = db.Close() }()
	if err := dbpkg.RunMigrations(db); err != nil {
		t.Fatalf("RunMigrations returned error: %v", err)
	}

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"password":"wrong"}`))
	context.Request.Header.Set("Content-Type", "application/json")

	service := services.NewAuthService(config.Config{
		AdminPassword: "secret",
		SessionSecret: "session-secret",
		AuthMaxFails:  3,
	}, db)
	handler := NewAuthHandler(service, config.Config{})
	handler.Login(context)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
	var response struct {
		Data struct {
			RemainingAttempts int `json:"remaining_attempts"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Data.RemainingAttempts != 2 {
		t.Fatalf("remaining_attempts = %d, want 2", response.Data.RemainingAttempts)
	}
}
