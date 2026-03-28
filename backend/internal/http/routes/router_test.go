package routes

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
