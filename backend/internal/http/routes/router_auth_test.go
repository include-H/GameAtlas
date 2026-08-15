package routes

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"

	"github.com/hao/game/internal/config"
	dbpkg "github.com/hao/game/internal/db"
	"github.com/hao/game/internal/services"
)

func openRouteAuthTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := dbpkg.OpenSQLite(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("OpenSQLite returned error: %v", err)
	}
	if err := dbpkg.RunMigrations(db); err != nil {
		_ = db.Close()
		t.Fatalf("RunMigrations returned error: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO games (public_id, title, visibility)
		VALUES ('known-game', 'Known Game', 'public')
	`); err != nil {
		_ = db.Close()
		t.Fatalf("insert known game: %v", err)
	}
	return db
}

func loginRouteAuthAdmin(t *testing.T, router *gin.Engine) *http.Cookie {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"password":"secret"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d, body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == services.AuthCookieName {
			return cookie
		}
	}
	t.Fatal("login response did not set admin cookie")
	return nil
}

func TestNewAPIRouteAuthMatrix(t *testing.T) {
	t.Setenv("GIN_MODE", gin.TestMode)

	db := openRouteAuthTestDB(t)
	defer func() { _ = db.Close() }()

	tempRoot := t.TempDir()
	staticDir := filepath.Join(tempRoot, "dist")
	if err := os.MkdirAll(staticDir, 0o755); err != nil {
		t.Fatalf("MkdirAll static dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(staticDir, "index.html"), []byte("<html>test</html>"), 0o644); err != nil {
		t.Fatalf("WriteFile index: %v", err)
	}
	assetsDir := filepath.Join(tempRoot, "assets")
	romRoot := filepath.Join(tempRoot, "rom")
	if err := os.MkdirAll(romRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll rom root: %v", err)
	}

	cfg := config.Config{
		AppEnv:           "test",
		StaticDir:        staticDir,
		AssetsDir:        assetsDir,
		PrimaryROMRoot:   romRoot,
		AdminPassword:    "secret",
		AdminDisplayName: "Admin",
	}
	router := New(cfg, db)
	adminCookie := loginRouteAuthAdmin(t, router)

	tests := []struct {
		name          string
		method        string
		path          string
		body          string
		requiresAdmin bool
		skipAdmin     bool
	}{
		{name: "health", method: http.MethodGet, path: "/api/health"},
		{name: "login", method: http.MethodPost, path: "/api/auth/login", body: `{"password":"secret"}`, skipAdmin: true},
		{name: "logout", method: http.MethodPost, path: "/api/auth/logout", skipAdmin: true},
		{name: "me", method: http.MethodGet, path: "/api/auth/me"},
		{name: "hitokoto", method: http.MethodGet, path: "/api/hitokoto"},
		{name: "pending issues", method: http.MethodGet, path: "/api/pending-issues"},
		{name: "games list", method: http.MethodGet, path: "/api/games"},
		{name: "games timeline", method: http.MethodGet, path: "/api/games/timeline"},
		{name: "games stats", method: http.MethodGet, path: "/api/games/stats"},
		{name: "games preview videos", method: http.MethodGet, path: "/api/games/preview-videos?public_ids=known-game"},
		{name: "game detail", method: http.MethodGet, path: "/api/games/known-game"},
		{name: "record download", method: http.MethodPost, path: "/api/games/known-game/files/1/downloads"},
		{name: "download", method: http.MethodGet, path: "/api/games/known-game/files/1/download"},
		{name: "launch script", method: http.MethodGet, path: "/api/games/known-game/files/1/launch-script"},
		{name: "wiki", method: http.MethodGet, path: "/api/games/known-game/wiki"},
		{name: "series list", method: http.MethodGet, path: "/api/series"},
		{name: "series detail", method: http.MethodGet, path: "/api/series/1"},
		{name: "developers list", method: http.MethodGet, path: "/api/developers"},
		{name: "publishers list", method: http.MethodGet, path: "/api/publishers"},
		{name: "publishers detail", method: http.MethodGet, path: "/api/publishers/1"},
		{name: "start screen tiles", method: http.MethodGet, path: "/api/start-screen/tiles"},
		{name: "steamgriddb available", method: http.MethodGet, path: "/api/steamgriddb/available"},

		{name: "create game", method: http.MethodPost, path: "/api/games", body: `{}`, requiresAdmin: true},
		{name: "update aggregate", method: http.MethodPut, path: "/api/games/unknown/aggregate", body: `{}`, requiresAdmin: true},
		{name: "delete game", method: http.MethodDelete, path: "/api/games/unknown", requiresAdmin: true},
		{name: "refresh sizes", method: http.MethodPost, path: "/api/games/refresh-sizes", requiresAdmin: true},
		{name: "update wiki", method: http.MethodPut, path: "/api/games/known-game/wiki", body: `{"content":"test","change_summary":"test"}`, requiresAdmin: true},
		{name: "wiki history", method: http.MethodGet, path: "/api/games/known-game/wiki/history", requiresAdmin: true},
		{name: "create series", method: http.MethodPost, path: "/api/series", body: `{}`, requiresAdmin: true},
		{name: "create developer", method: http.MethodPost, path: "/api/developers", body: `{}`, requiresAdmin: true},
		{name: "create publisher", method: http.MethodPost, path: "/api/publishers", body: `{}`, requiresAdmin: true},
		{name: "ignore review issue", method: http.MethodPut, path: "/api/games/unknown/review-issues/missing-cover/ignore", body: `{}`, requiresAdmin: true},
		{name: "restore review issue", method: http.MethodDelete, path: "/api/games/unknown/review-issues/missing-cover/ignore", requiresAdmin: true},
		{name: "upload cover", method: http.MethodPost, path: "/api/assets/cover", requiresAdmin: true},
		{name: "upload banner", method: http.MethodPost, path: "/api/assets/banner", requiresAdmin: true},
		{name: "upload video", method: http.MethodPost, path: "/api/assets/video", requiresAdmin: true},
		{name: "upload poster", method: http.MethodPost, path: "/api/assets/poster", requiresAdmin: true},
		{name: "upload screenshot", method: http.MethodPost, path: "/api/assets/screenshot", requiresAdmin: true},
		{name: "upload logo", method: http.MethodPost, path: "/api/assets/logo", requiresAdmin: true},
		{name: "directory default", method: http.MethodGet, path: "/api/directory/default", requiresAdmin: true},
		{name: "directory list", method: http.MethodGet, path: "/api/directory/list", requiresAdmin: true},
		{name: "directory search", method: http.MethodGet, path: "/api/directory/search", requiresAdmin: true},
		{name: "settings config", method: http.MethodGet, path: "/api/settings/config", requiresAdmin: true},
		{name: "update settings", method: http.MethodPut, path: "/api/settings/config", body: `{}`, requiresAdmin: true},
		{name: "upload background", method: http.MethodPost, path: "/api/settings/bg", requiresAdmin: true},
		{name: "remove background", method: http.MethodDelete, path: "/api/settings/bg", requiresAdmin: true},
		{name: "restart", method: http.MethodPost, path: "/api/settings/restart", body: `{}`, requiresAdmin: true, skipAdmin: true},
		{name: "start screen update", method: http.MethodPut, path: "/api/start-screen/tiles", body: `{}`, requiresAdmin: true},
		{name: "start screen add", method: http.MethodPost, path: "/api/start-screen/tiles", body: `{}`, requiresAdmin: true},
		{name: "start screen remove", method: http.MethodDelete, path: "/api/start-screen/tiles/1", requiresAdmin: true},
		{name: "steam search", method: http.MethodGet, path: "/api/steam/search", requiresAdmin: true},
		{name: "steam preview", method: http.MethodGet, path: "/api/steam/0/assets", requiresAdmin: true},
		{name: "steam proxy", method: http.MethodGet, path: "/api/steam/proxy", requiresAdmin: true},
		{name: "steamgriddb search", method: http.MethodGet, path: "/api/steamgriddb/search", requiresAdmin: true},
		{name: "steamgriddb game grids", method: http.MethodGet, path: "/api/steamgriddb/game/1/grids", requiresAdmin: true},
		{name: "steamgriddb game heroes", method: http.MethodGet, path: "/api/steamgriddb/game/1/heroes", requiresAdmin: true},
		{name: "steamgriddb game logos", method: http.MethodGet, path: "/api/steamgriddb/game/1/logos", requiresAdmin: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			anonymousRecorder := httptest.NewRecorder()
			anonymousRequest := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			if tt.body != "" {
				anonymousRequest.Header.Set("Content-Type", "application/json")
			}
			router.ServeHTTP(anonymousRecorder, anonymousRequest)

			if tt.requiresAdmin && anonymousRecorder.Code != http.StatusUnauthorized {
				t.Fatalf("anonymous status = %d, want %d, body=%s", anonymousRecorder.Code, http.StatusUnauthorized, anonymousRecorder.Body.String())
			}
			if !tt.requiresAdmin && anonymousRecorder.Code == http.StatusUnauthorized {
				t.Fatalf("anonymous status = %d, want non-401, body=%s", anonymousRecorder.Code, anonymousRecorder.Body.String())
			}

			if tt.skipAdmin {
				return
			}

			adminRecorder := httptest.NewRecorder()
			adminRequest := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			if tt.body != "" {
				adminRequest.Header.Set("Content-Type", "application/json")
			}
			adminRequest.AddCookie(adminCookie)
			router.ServeHTTP(adminRecorder, adminRequest)

			if adminRecorder.Code == http.StatusUnauthorized {
				t.Fatalf("admin status = %d, want non-401, body=%s", adminRecorder.Code, adminRecorder.Body.String())
			}
		})
	}
}
