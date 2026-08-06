package routes

import (
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/domain"
)

type assetRouteGameRepositoryStub struct {
	err error
}

func (s assetRouteGameRepositoryStub) GetByPublicID(publicID string) (*domain.Game, error) {
	return nil, s.err
}

type assetRouteStartScreenRepositoryStub struct {
	visibility string
	err        error
}

func (s assetRouteStartScreenRepositoryStub) GetGameVisibilityByImagePath(imagePath string) (string, error) {
	return s.visibility, s.err
}

func TestRegisterAssetRoutesServesPublicStartScreenTileImages(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	assetsDir := t.TempDir()
	imageDir := filepath.Join(assetsDir, "start-screen")
	if err := os.MkdirAll(imageDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	imagePath := filepath.Join(imageDir, "66dbcee2-3512-4139-ad10-65898b8f0cfb.png")
	if err := os.WriteFile(imagePath, []byte("tile-image"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	router := gin.New()
	registerAssetRoutes(
		router,
		assetsDir,
		assetRouteGameRepositoryStub{err: errors.New("game lookup must be skipped")},
		assetRouteStartScreenRepositoryStub{visibility: domain.GameVisibilityPublic},
	)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/assets/start-screen/66dbcee2-3512-4139-ad10-65898b8f0cfb.png", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if recorder.Body.String() != "tile-image" {
		t.Fatalf("body = %q, want %q", recorder.Body.String(), "tile-image")
	}
}

func TestRegisterAssetRoutesHidesPrivateStartScreenTileImagesFromPublicCallers(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	assetsDir := t.TempDir()
	imageDir := filepath.Join(assetsDir, "start-screen")
	if err := os.MkdirAll(imageDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	imagePath := filepath.Join(imageDir, "66dbcee2-3512-4139-ad10-65898b8f0cfb.png")
	if err := os.WriteFile(imagePath, []byte("tile-image"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	router := gin.New()
	registerAssetRoutes(
		router,
		assetsDir,
		assetRouteGameRepositoryStub{err: errors.New("game lookup must be skipped")},
		assetRouteStartScreenRepositoryStub{visibility: domain.GameVisibilityPrivate},
	)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/assets/start-screen/66dbcee2-3512-4139-ad10-65898b8f0cfb.png", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

func TestRegisterAssetRoutesAllowsAdminToReadPrivateStartScreenTileImages(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	assetsDir := t.TempDir()
	imageDir := filepath.Join(assetsDir, "start-screen")
	if err := os.MkdirAll(imageDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	imagePath := filepath.Join(imageDir, "66dbcee2-3512-4139-ad10-65898b8f0cfb.png")
	if err := os.WriteFile(imagePath, []byte("tile-image"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("is_admin", true)
		c.Next()
	})
	registerAssetRoutes(
		router,
		assetsDir,
		assetRouteGameRepositoryStub{err: errors.New("game lookup must be skipped")},
		assetRouteStartScreenRepositoryStub{visibility: domain.GameVisibilityPrivate},
	)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/assets/start-screen/66dbcee2-3512-4139-ad10-65898b8f0cfb.png", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if recorder.Body.String() != "tile-image" {
		t.Fatalf("body = %q, want %q", recorder.Body.String(), "tile-image")
	}
}

func TestRegisterAssetRoutesHidesUnknownStartScreenImagesFromPublicCallers(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	assetsDir := t.TempDir()
	stagingDir := filepath.Join(assetsDir, "_staging")
	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	stagingPath := filepath.Join(stagingDir, "66dbcee2-3512-4139-ad10-65898b8f0cfb.png")
	if err := os.WriteFile(stagingPath, []byte("tile-image"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	router := gin.New()
	registerAssetRoutes(
		router,
		assetsDir,
		assetRouteGameRepositoryStub{err: errors.New("game lookup must be skipped")},
		assetRouteStartScreenRepositoryStub{err: errors.New("tile not saved")},
	)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/assets/start-screen/66dbcee2-3512-4139-ad10-65898b8f0cfb.png", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

func TestRegisterAssetRoutesAllowsAdminToPreviewUnregisteredStartScreenImages(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	assetsDir := t.TempDir()
	stagingDir := filepath.Join(assetsDir, "_staging")
	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	stagingPath := filepath.Join(stagingDir, "66dbcee2-3512-4139-ad10-65898b8f0cfb.png")
	if err := os.WriteFile(stagingPath, []byte("tile-image"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("is_admin", true)
		c.Next()
	})
	registerAssetRoutes(
		router,
		assetsDir,
		assetRouteGameRepositoryStub{err: errors.New("game lookup must be skipped")},
		assetRouteStartScreenRepositoryStub{err: errors.New("tile not saved")},
	)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/assets/start-screen/66dbcee2-3512-4139-ad10-65898b8f0cfb.png", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if recorder.Body.String() != "tile-image" {
		t.Fatalf("body = %q, want %q", recorder.Body.String(), "tile-image")
	}
}

func TestRegisterAssetRoutesStillHidesUnknownGameAssets(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	assetsDir := t.TempDir()
	router := gin.New()
	registerAssetRoutes(
		router,
		assetsDir,
		assetRouteGameRepositoryStub{err: errors.New("no such game")},
		assetRouteStartScreenRepositoryStub{},
	)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/assets/unknown-game/cover.png", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

func TestRegisterCustomDataRoutesServesBackground(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	dataDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dataDir, "bg.jpg"), []byte("bg-data"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	router := gin.New()
	registerCustomDataRoutes(router, dataDir)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/data/bg.jpg", nil)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if recorder.Body.String() != "bg-data" {
		t.Fatalf("body = %q, want %q", recorder.Body.String(), "bg-data")
	}
}

func TestRegisterCustomDataRoutesRejectsUnlistedDataPaths(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	dataDir := t.TempDir()
	for _, relPath := range []string{
		filepath.Join("ui", "FONT.WOFF2"),
		filepath.Join("fonts", "CUSTOM.woff2"),
		"CUSTOM.woff2",
		filepath.Join("gamelist", "private-game", "cover.png"),
		filepath.Join("custom", "cover.png"),
	} {
		fullPath := filepath.Join(dataDir, relPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatalf("MkdirAll returned error: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte("private-asset"), 0o644); err != nil {
			t.Fatalf("WriteFile returned error: %v", err)
		}
	}

	router := gin.New()
	registerCustomDataRoutes(router, dataDir)

	for _, path := range []string{
		"/data/ui/FONT.WOFF2",
		"/data/fonts/CUSTOM.woff2",
		"/data/CUSTOM.woff2",
		"/data/gamelist/private-game/cover.png",
		"/data/custom/cover.png",
	} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, path, nil)
		router.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusNotFound {
			t.Fatalf("%s status = %d, want %d", path, recorder.Code, http.StatusNotFound)
		}
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
	if body := recorder.Body.String(); !strings.Contains(body, "\"error\":\"路由不存在\"") {
		t.Fatalf("body = %q, want JSON 404 payload", body)
	}
}

func TestResolveEmbeddedDistFSRequiresIndexHTML(t *testing.T) {
	_, err := resolveEmbeddedDistFSFrom(fstest.MapFS{
		"ui/app.js": {Data: []byte("console.log('ok')")},
	})
	if err == nil {
		t.Fatal("expected resolveEmbeddedDistFSFrom to fail when index.html is missing")
	}
	if !strings.Contains(err.Error(), "index.html") {
		t.Fatalf("err = %v, want missing index.html", err)
	}
}

func TestResolveEmbeddedDistFSFromReturnsFSWhenIndexExists(t *testing.T) {
	distFS, err := resolveEmbeddedDistFSFrom(fstest.MapFS{
		"index.html": {Data: []byte("<html>spa</html>")},
		"ui/app.js":  {Data: []byte("console.log('ok')")},
	})
	if err != nil {
		t.Fatalf("resolveEmbeddedDistFSFrom returned error: %v", err)
	}

	content, readErr := fs.ReadFile(distFS, "index.html")
	if readErr != nil {
		t.Fatalf("ReadFile returned error: %v", readErr)
	}
	if string(content) != "<html>spa</html>" {
		t.Fatalf("index.html = %q, want %q", string(content), "<html>spa</html>")
	}
}
