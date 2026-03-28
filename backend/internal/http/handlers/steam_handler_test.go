package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/config"
	"github.com/hao/game/internal/services"
)

func TestSteamHandlerApplyUsesQueryGameIDFallback(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	handler := NewSteamHandler(services.NewSteamService(config.Config{}, nil))

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/steam/123/apply?game_id=7", strings.NewReader(`{}`))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Params = gin.Params{{Key: "appId", Value: "123"}}
	context.Set("is_admin", true)

	handler.Apply(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Success bool `json:"success"`
		Data    struct {
			AppID          int64    `json:"app_id"`
			ScreenshotURLs []string `json:"screenshot_urls"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success || response.Data.AppID != 123 {
		t.Fatalf("response = %s, want success with app_id 123", recorder.Body.String())
	}
	if len(response.Data.ScreenshotURLs) != 0 {
		t.Fatalf("ScreenshotURLs = %#v, want empty slice", response.Data.ScreenshotURLs)
	}
}

func TestSteamHandlerApplyRejectsInvalidJSON(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	handler := NewSteamHandler(services.NewSteamService(config.Config{}, nil))

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/steam/123/apply", strings.NewReader("{"))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Params = gin.Params{{Key: "appId", Value: "123"}}
	context.Set("is_admin", true)

	handler.Apply(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"error":"invalid steam asset payload"`) {
		t.Fatalf("body = %s, want invalid steam asset payload", recorder.Body.String())
	}
}

func TestSteamHandlerApplyReturnsBadRequestWhenGameIDMissing(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	handler := NewSteamHandler(services.NewSteamService(config.Config{}, nil))

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/steam/123/apply", strings.NewReader(`{}`))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Params = gin.Params{{Key: "appId", Value: "123"}}
	context.Set("is_admin", true)

	handler.Apply(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"error":"invalid steam asset payload"`) {
		t.Fatalf("body = %s, want invalid steam asset payload", recorder.Body.String())
	}
}

func TestSteamHandlerProxyRejectsInvalidURL(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	handler := NewSteamHandler(services.NewSteamService(config.Config{}, nil))

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/steam/proxy?url=ftp://example.com/demo.jpg", nil)
	context.Set("is_admin", true)

	handler.Proxy(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"error":"invalid steam proxy request"`) {
		t.Fatalf("body = %s, want invalid steam proxy request", recorder.Body.String())
	}
}

func TestSteamHandlerSearchRequiresQuery(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	handler := NewSteamHandler(services.NewSteamService(config.Config{}, nil))

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/steam/search?q=%20%20%20", nil)
	context.Set("is_admin", true)

	handler.Search(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"error":"search query is required"`) {
		t.Fatalf("body = %s, want search query is required", recorder.Body.String())
	}
}

func TestSteamHandlerSearchReturnsBadGatewayWhenUpstreamFails(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	handler := NewSteamHandler(services.NewSteamService(config.Config{Proxy: "http://127.0.0.1:1"}, nil))

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/steam/search?q=portal", nil)
	context.Set("is_admin", true)

	handler.Search(context)

	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusBadGateway, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"error":"steam search failed"`) {
		t.Fatalf("body = %s, want steam search failed", recorder.Body.String())
	}
}

func TestSteamHandlerPreviewReturnsDefaultFallbackPayloadWhenUpstreamFails(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	handler := NewSteamHandler(services.NewSteamService(config.Config{Proxy: "http://127.0.0.1:1"}, nil))

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/steam/321/preview", nil)
	context.Params = gin.Params{{Key: "appId", Value: "321"}}
	context.Set("is_admin", true)

	handler.Preview(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Success bool `json:"success"`
		Data    struct {
			AppID          int64    `json:"app_id"`
			Name           string   `json:"name"`
			CoverURL       *string  `json:"cover_url"`
			BannerURL      *string  `json:"banner_url"`
			ScreenshotURLs []string `json:"screenshot_urls"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success || response.Data.AppID != 321 || response.Data.Name != "Steam App 321" {
		t.Fatalf("response = %s, want default preview payload", recorder.Body.String())
	}
	if response.Data.CoverURL == nil || *response.Data.CoverURL != "https://steamcdn-a.akamaihd.net/steam/apps/321/library_600x900.jpg" {
		t.Fatalf("CoverURL = %v, want fallback cover candidate", response.Data.CoverURL)
	}
	if response.Data.BannerURL == nil || *response.Data.BannerURL != "https://steamcdn-a.akamaihd.net/steam/apps/321/library_hero.jpg" {
		t.Fatalf("BannerURL = %v, want fallback banner candidate", response.Data.BannerURL)
	}
	if len(response.Data.ScreenshotURLs) != 0 {
		t.Fatalf("ScreenshotURLs = %#v, want empty slice", response.Data.ScreenshotURLs)
	}
}

func TestSteamHandlerProxyRequiresURL(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	handler := NewSteamHandler(services.NewSteamService(config.Config{}, nil))

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/steam/proxy", nil)
	context.Set("is_admin", true)

	handler.Proxy(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"error":"url is required"`) {
		t.Fatalf("body = %s, want url is required", recorder.Body.String())
	}
}
