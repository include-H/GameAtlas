package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/config"
	"github.com/hao/game/internal/services"
)

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

func TestSteamHandlerPreviewReturnsBadGatewayWhenUpstreamFails(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	handler := NewSteamHandler(services.NewSteamService(config.Config{Proxy: "http://127.0.0.1:1"}, nil))

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/steam/321/preview", nil)
	context.Params = gin.Params{{Key: "appId", Value: "321"}}
	context.Set("is_admin", true)

	handler.Preview(context)

	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusBadGateway, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"error":"upstream request failed:`) {
		t.Fatalf("body = %s, want upstream error payload", recorder.Body.String())
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

func TestSteamHandlerSearchRejectsInvalidProxy(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	handler := NewSteamHandler(services.NewSteamService(config.Config{}, nil))

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/steam/search?q=portal&proxy=%3A%3A%3Abad%3A%3A", nil)
	context.Set("is_admin", true)

	handler.Search(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"error":"valid steam proxy is required"`) {
		t.Fatalf("body = %s, want valid steam proxy is required", recorder.Body.String())
	}
}

func TestSteamHandlerPreviewRejectsInvalidProxy(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	handler := NewSteamHandler(services.NewSteamService(config.Config{}, nil))

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/steam/321/preview?proxy=socks5://127.0.0.1:9000", nil)
	context.Params = gin.Params{{Key: "appId", Value: "321"}}
	context.Set("is_admin", true)

	handler.Preview(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"error":"valid steam proxy is required"`) {
		t.Fatalf("body = %s, want valid steam proxy is required", recorder.Body.String())
	}
}

func TestSteamHandlerProxyRejectsInvalidProxyOverride(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	handler := NewSteamHandler(services.NewSteamService(config.Config{}, nil))

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/steam/proxy?url=https://cdn.cloudflare.steamstatic.com/demo.jpg&proxy=%3A%3A%3Abad%3A%3A", nil)
	context.Set("is_admin", true)

	handler.Proxy(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"error":"valid steam proxy is required"`) {
		t.Fatalf("body = %s, want valid steam proxy is required", recorder.Body.String())
	}
}
