package routes

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRegisterCustomDataRoutesAllowsUppercaseExtensions(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	dataDir := t.TempDir()
	assetPath := filepath.Join(dataDir, "ui", "FONT.WOFF2")
	if err := os.MkdirAll(filepath.Dir(assetPath), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(assetPath, []byte("font-data"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	router := gin.New()
	registerCustomDataRoutes(router, dataDir)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/data/ui/FONT.WOFF2", nil)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if recorder.Body.String() != "font-data" {
		t.Fatalf("body = %q, want %q", recorder.Body.String(), "font-data")
	}
}

func TestRegisterStaticRoutesFromDiskServesSPAIndexForUnknownPage(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	staticDir := t.TempDir()
	indexPath := filepath.Join(staticDir, "index.html")
	if err := os.WriteFile(indexPath, []byte("<html>spa</html>"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	router := gin.New()
	registerStaticRoutesFromDisk(router, staticDir, indexPath)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/games/unknown", nil)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if body := recorder.Body.String(); body != "<html>spa</html>" {
		t.Fatalf("body = %q, want %q", body, "<html>spa</html>")
	}
}

func TestRegisterStaticRoutesFromDiskReturnsJSON404ForUnknownAPIGet(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	staticDir := t.TempDir()
	indexPath := filepath.Join(staticDir, "index.html")
	if err := os.WriteFile(indexPath, []byte("<html>spa</html>"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	router := gin.New()
	registerStaticRoutesFromDisk(router, staticDir, indexPath)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/typo", nil)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusNotFound, recorder.Body.String())
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.Contains(contentType, "application/json") {
		t.Fatalf("content-type = %q, want application/json", contentType)
	}
	if body := recorder.Body.String(); !strings.Contains(body, "\"error\":\"route not found\"") {
		t.Fatalf("body = %q, want JSON 404 payload", body)
	}
}
