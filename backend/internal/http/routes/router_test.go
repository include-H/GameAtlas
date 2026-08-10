package routes

import (
	"errors"
	"image"
	"image/color"
	"image/jpeg"
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

type assetRouteGameRepositoryPublicStub struct{}

func (s assetRouteGameRepositoryPublicStub) GetByPublicID(publicID string) (*domain.Game, error) {
	return &domain.Game{PublicID: publicID, Visibility: domain.GameVisibilityPublic}, nil
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
	stagingDir := filepath.Join(assetsDir, "_staging", "start-screen")
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
	stagingDir := filepath.Join(assetsDir, "_staging", "start-screen")
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
	registerCustomDataRoutes(router.Group("/api"), dataDir)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/data/bg.jpg", nil)

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
	registerCustomDataRoutes(router.Group("/api"), dataDir)

	for _, path := range []string{
		"/api/data/ui/FONT.WOFF2",
		"/api/data/fonts/CUSTOM.woff2",
		"/api/data/CUSTOM.woff2",
		"/api/data/gamelist/private-game/cover.png",
		"/api/data/custom/cover.png",
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

func TestRegisterAssetRoutesServesWebPVariantForWidthQuery(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	assetsDir := t.TempDir()
	gameDir := filepath.Join(assetsDir, "variant-game")
	if err := os.MkdirAll(gameDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	srcImage := image.NewRGBA(image.Rect(0, 0, 1600, 900))
	for y := 0; y < 900; y++ {
		for x := 0; x < 1600; x++ {
			srcImage.Set(x, y, color.RGBA{R: uint8(x % 255), G: uint8(y % 255), B: 128, A: 255})
		}
	}
	srcPath := filepath.Join(gameDir, "shot.jpg")
	file, err := os.Create(srcPath)
	if err != nil {
		t.Fatalf("create source image: %v", err)
	}
	if err := jpeg.Encode(file, srcImage, nil); err != nil {
		_ = file.Close()
		t.Fatalf("encode source image: %v", err)
	}
	_ = file.Close()

	router := gin.New()
	registerAssetRoutes(
		router,
		assetsDir,
		assetRouteGameRepositoryPublicStub{},
		assetRouteStartScreenRepositoryStub{},
	)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/assets/variant-game/shot.jpg?w=480", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.Contains(contentType, "image/webp") {
		t.Fatalf("content-type = %q, want image/webp", contentType)
	}
	if _, err := os.Stat(filepath.Join(gameDir, "shot.w480.webp")); err != nil {
		t.Fatalf("variant file not persisted: %v", err)
	}

	recorderSecond := httptest.NewRecorder()
	requestSecond := httptest.NewRequest(http.MethodGet, "/assets/variant-game/shot.jpg?w=480", nil)
	router.ServeHTTP(recorderSecond, requestSecond)
	if recorderSecond.Code != http.StatusOK {
		t.Fatalf("second status = %d, want %d", recorderSecond.Code, http.StatusOK)
	}
	if cacheControl := recorderSecond.Header().Get("Cache-Control"); !strings.Contains(cacheControl, "immutable") {
		t.Fatalf("cache-control = %q, want immutable for cached variant", cacheControl)
	}
}
