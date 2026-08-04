package routes

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"

	"github.com/hao/game/internal/config"
	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/files"
	"github.com/hao/game/internal/http/handlers"
	"github.com/hao/game/internal/repositories"
	"github.com/hao/game/internal/services"
	webassets "github.com/hao/game/web"
)

func New(cfg config.Config, db *sqlx.DB) *gin.Engine {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	healthHandler := handlers.NewHealthHandler(db)
	authAttemptsRepo := repositories.NewAuthAttemptRepository(db)
	authSessionsRepo := repositories.NewAuthSessionRepository(db)
	authService := services.NewAuthService(cfg, authAttemptsRepo, authSessionsRepo)
	authHandler := handlers.NewAuthHandler(authService, cfg)
	gamesRepo := repositories.NewGamesRepository(db)
	gameFilesRepo := repositories.NewGameFilesRepository(db)
	metadataRepo := repositories.NewMetadataRepository(db)
	favoriteGamesRepo := repositories.NewFavoriteGamesRepository(db)
	reviewIssueOverridesRepo := repositories.NewReviewIssueOverrideRepository(db)
	wikiRepo := repositories.NewWikiRepository(db)
	gameCatalogRepo := repositories.NewGameCatalogRepository(gamesRepo, favoriteGamesRepo)
	gameCatalogService := services.NewGameCatalogService(gameCatalogRepo, reviewIssueOverridesRepo)
	gameTimelineService := services.NewGameTimelineService(gamesRepo)
	gameDetailService := services.NewGameDetailService(gamesRepo, gameFilesRepo, reviewIssueOverridesRepo)
	assetCleanupTasksRepo := repositories.NewAssetCleanupTasksRepository(db)
	metadataService := services.NewMetadataService(metadataRepo)
	gameAggregateService := services.NewGameAggregateService(cfg, gamesRepo, metadataService, gameCatalogService, assetCleanupTasksRepo)
	gameFavoriteService := services.NewGameFavoriteService(gamesRepo, favoriteGamesRepo)
	gameFilesService := services.NewGameFilesService(cfg, gamesRepo, gameFilesRepo)
	windowsLaunchService := services.NewWindowsLaunchService(cfg, gamesRepo, gameFilesRepo)
	assetsService := services.NewAssetsService(cfg, gamesRepo)
	directoryService := services.NewDirectoryService(cfg)
	pendingIssuesService := services.NewPendingIssuesService()
	reviewIssueOverrideService := services.NewReviewIssueOverrideService(gamesRepo, reviewIssueOverridesRepo)
	steamService := services.NewSteamService(cfg, assetsService)
	wikiService := services.NewWikiService(gamesRepo, wikiRepo, cfg.WikiHistoryLimit)
	hitokotoService := services.NewHitokotoService()
	assetsHandler := handlers.NewAssetsHandler(assetsService)
	directoryHandler := handlers.NewDirectoryHandler(directoryService)
	gamesHandler := handlers.NewSplitGamesHandler(gameCatalogService, gameTimelineService, gameDetailService, gameAggregateService, gameFavoriteService)
	gameFilesHandler := handlers.NewGameFilesHandler(gameFilesService)
	downloadsHandler := handlers.NewDownloadsHandler(gameFilesService, windowsLaunchService, authService)
	// These endpoints are exposed as first-class resources for admin UX, but
	// they still point at metadata that is auto-pruned once unreferenced by any
	// game. The lightweight MetadataResource mapping keeps the transport layer
	// small while the actual lifecycle rule remains in aggregate-side cleanup.
	seriesHandler := handlers.NewMetadataHandler(metadataService, services.MetadataResource{Type: domain.MetadataSeries})
	developersHandler := handlers.NewMetadataHandler(metadataService, services.MetadataResource{Type: domain.MetadataDevelopers})
	publishersHandler := handlers.NewMetadataHandler(metadataService, services.MetadataResource{Type: domain.MetadataPublishers})
	reviewIssueOverrideHandler := handlers.NewReviewIssueOverrideHandler(reviewIssueOverrideService)
	pendingIssuesHandler := handlers.NewPendingIssuesHandler(pendingIssuesService)
	steamHandler := handlers.NewSteamHandler(steamService)
	steamGridDBHandler := handlers.NewSteamGridDBHandler(services.NewSteamGridDBService(cfg.SteamGridDBAPIKey, cfg.Proxy))
	wikiHandler := handlers.NewWikiHandler(wikiService)
	hitokotoHandler := handlers.NewHitokotoHandler(hitokotoService)
	settingsService := services.NewSettingsService(cfg, repositories.NewAppSettingsRepository(db))
	settingsHandler := handlers.NewSettingsHandler(settingsService)
	startScreenTilesService := services.NewStartScreenTilesService(
		repositories.NewStartScreenTilesRepository(db),
		repositories.NewStartScreenColumnsRepository(db),
		gamesRepo,
		files.NewAssetStore(cfg.AssetsDir),
	)
	startScreenTilesHandler := handlers.NewStartScreenTilesHandler(startScreenTilesService)
	gameFileRefreshService := services.NewGameFileRefreshService(gameFilesRepo, files.NewGuard(cfg.PrimaryROMRoot))
	gameFileRefreshHandler := handlers.NewGameFileRefreshHandler(gameFileRefreshService)

	router.Use(func(c *gin.Context) {
		session, _ := c.Cookie(services.AuthCookieName)
		c.Set("is_admin", authService.IsAdmin(session))
		c.Next()
	})

	api := router.Group("/api")
	api.GET("/health", healthHandler.Get)
	api.POST("/auth/login", authHandler.Login)
	api.POST("/auth/logout", authHandler.Logout)
	api.GET("/auth/me", authHandler.Me)
	api.GET("/hitokoto", hitokotoHandler.Get)
	api.GET("/pending-issues", pendingIssuesHandler.List)
	api.GET("/games", gamesHandler.List)
	api.GET("/games/timeline", gamesHandler.ListTimeline)
	api.GET("/games/stats", gamesHandler.Stats)
	api.GET("/games/preview-videos", gamesHandler.ListPreviewVideos)
	api.GET("/games/:publicId", gamesHandler.Get)
	api.PUT("/games/:publicId/favorite", gamesHandler.Favorite)
	api.DELETE("/games/:publicId/favorite", gamesHandler.Unfavorite)
	api.POST("/games", gamesHandler.Create)
	api.PUT("/games/:publicId/aggregate", gamesHandler.UpdateAggregate)
	api.DELETE("/games/:publicId", gamesHandler.Delete)
	api.POST("/games/refresh-sizes", gameFileRefreshHandler.RefreshSizes)
	api.GET("/games/:publicId/files", gameFilesHandler.List)
	api.POST("/games/:publicId/files/:fileId/downloads", downloadsHandler.RecordDownload)
	api.GET("/games/:publicId/files/:fileId/download", downloadsHandler.Download)
	api.GET("/games/:publicId/files/:fileId/launch-script", downloadsHandler.LaunchScript)
	api.GET("/games/:publicId/wiki", wikiHandler.Get)
	api.PUT("/games/:publicId/wiki", wikiHandler.Update)
	api.GET("/games/:publicId/wiki/history", wikiHandler.History)
	api.GET("/series", seriesHandler.List)
	api.GET("/series/:id", seriesHandler.Get)
	api.POST("/series", seriesHandler.Create)
	api.GET("/developers", developersHandler.List)
	api.POST("/developers", developersHandler.Create)
	api.GET("/publishers", publishersHandler.List)
	api.GET("/publishers/:id", publishersHandler.Get)
	api.POST("/publishers", publishersHandler.Create)
	api.PUT("/games/:publicId/review-issues/:issueKey/ignore", reviewIssueOverrideHandler.Ignore)
	api.DELETE("/games/:publicId/review-issues/:issueKey/ignore", reviewIssueOverrideHandler.Delete)
	api.POST("/assets/cover", assetsHandler.Upload("cover"))
	api.POST("/assets/banner", assetsHandler.Upload("banner"))
	api.POST("/assets/video", assetsHandler.Upload("video"))
	api.POST("/assets/screenshot", assetsHandler.Upload("screenshot"))
	api.POST("/assets/logo", assetsHandler.Upload("logo"))
	api.GET("/directory/default", directoryHandler.Default)
	api.GET("/directory/list", directoryHandler.List)
	api.GET("/directory/search", directoryHandler.Search)
	api.GET("/settings/config", settingsHandler.GetConfig)
	api.PUT("/settings/config", settingsHandler.UpdateConfig)
	api.POST("/settings/bg", settingsHandler.UploadBackground)
	api.DELETE("/settings/bg", settingsHandler.RemoveBackground)
	api.POST("/settings/restart", settingsHandler.Restart)
	api.GET("/start-screen/tiles", startScreenTilesHandler.Get)
	api.PUT("/start-screen/tiles", startScreenTilesHandler.Update)
	api.POST("/start-screen/tiles/image", startScreenTilesHandler.UploadImage)
	api.GET("/steam/search", steamHandler.Search)
	api.GET("/steam/:appId/assets", steamHandler.Preview)
	api.GET("/steam/proxy", steamHandler.Proxy)
	api.GET("/steamgriddb/available", steamGridDBHandler.Available)
	api.GET("/steamgriddb/search", steamGridDBHandler.Search)
	api.GET("/steamgriddb/:appId/grids", steamGridDBHandler.GetGrids)
	api.GET("/steamgriddb/:appId/heroes", steamGridDBHandler.GetHeroes)
	api.GET("/steamgriddb/:appId/logos", steamGridDBHandler.GetLogos)
	api.GET("/steamgriddb/:appId/icons", steamGridDBHandler.GetIcons)
	api.GET("/steamgriddb/game/:gameId/grids", steamGridDBHandler.GetGridsByGameID)
	api.GET("/steamgriddb/game/:gameId/heroes", steamGridDBHandler.GetHeroesByGameID)
	api.GET("/steamgriddb/game/:gameId/logos", steamGridDBHandler.GetLogosByGameID)

	registerAssetRoutes(router, cfg.AssetsDir, gamesRepo)
	registerCustomDataRoutes(router, filepath.Dir(cfg.AssetsDir))
	registerStaticRoutes(router, cfg.StaticDir)

	return router
}

type assetRouteGameRepository interface {
	GetByPublicID(publicID string) (*domain.Game, error)
}

type assetRouteCacheEntry struct {
	exists     bool
	visibility string
	loadedAt   time.Time
}

func registerAssetRoutes(router *gin.Engine, assetsDir string, gamesRepo assetRouteGameRepository) {
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		return
	}

	var assetCache sync.Map

	serveAssetFile := func(c *gin.Context, rawPath string) {
		targetPath := filepath.Join(assetsDir, filepath.FromSlash(rawPath))
		relative, err := filepath.Rel(assetsDir, targetPath)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			c.Status(http.StatusNotFound)
			return
		}

		if _, err := os.Stat(targetPath); err != nil {
			// Fallback: check staging directory for files not yet moved to permanent.
			stagingPath := filepath.Join(assetsDir, "_staging", filepath.Base(rawPath))
			if _, statErr := os.Stat(stagingPath); statErr == nil {
				targetPath = stagingPath
			} else {
				c.Status(http.StatusNotFound)
				return
			}
		}

		c.Header("Cache-Control", "public, max-age=86400")
		c.File(targetPath)
	}

	router.GET("/assets/*filepath", func(c *gin.Context) {
		rawPath := strings.TrimPrefix(c.Param("filepath"), "/")
		if rawPath == "" {
			c.Status(http.StatusNotFound)
			return
		}

		segments := strings.Split(rawPath, "/")
		if len(segments) < 2 {
			c.Status(http.StatusNotFound)
			return
		}

		gamePublicID := strings.TrimSpace(segments[0])
		if gamePublicID == "" {
			c.Status(http.StatusNotFound)
			return
		}

		// 开始屏幕磁贴裁剪图存放在 assets/start-screen/ 下，不属于某个游戏，
		// 跳过游戏可见性校验，直接按文件服务。
		if gamePublicID == "start-screen" {
			serveAssetFile(c, rawPath)
			return
		}

		// Check cache first
		var gameExists bool
		var gameVisibility string
		if cached, ok := assetCache.Load(gamePublicID); ok {
			entry := cached.(assetRouteCacheEntry)
			if time.Since(entry.loadedAt) < 5*time.Minute {
				gameExists = entry.exists
				gameVisibility = entry.visibility
			}
		}

		if !gameExists || gameVisibility == "" {
			game, err := gamesRepo.GetByPublicID(gamePublicID)
			if err != nil {
				assetCache.Store(gamePublicID, assetRouteCacheEntry{exists: false, loadedAt: time.Now()})
				c.Status(http.StatusNotFound)
				return
			}
			gameExists = true
			gameVisibility = game.Visibility
			assetCache.Store(gamePublicID, assetRouteCacheEntry{exists: true, visibility: game.Visibility, loadedAt: time.Now()})
		}

		isAdmin, _ := c.Get("is_admin")
		admin, _ := isAdmin.(bool)
		if !admin && gameVisibility == domain.GameVisibilityPrivate {
			c.Status(http.StatusNotFound)
			return
		}

		serveAssetFile(c, rawPath)
	})
}

func registerCustomDataRoutes(router *gin.Engine, dataDir string) {
	allowedExtensions := map[string]struct{}{
		".jpg":   {},
		".jpeg":  {},
		".png":   {},
		".webp":  {},
		".avif":  {},
		".gif":   {},
		".svg":   {},
		".ttf":   {},
		".otf":   {},
		".woff":  {},
		".woff2": {},
	}

	dataHandler := func(c *gin.Context) {
		rawPath := strings.TrimPrefix(c.Param("filepath"), "/")
		if rawPath == "" {
			c.Status(http.StatusNotFound)
			return
		}

		cleanPath := filepath.Clean(filepath.FromSlash(rawPath))
		if cleanPath == "." || cleanPath == ".." || strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) {
			c.Status(http.StatusNotFound)
			return
		}

		extension := strings.ToLower(filepath.Ext(cleanPath))
		if _, ok := allowedExtensions[extension]; !ok {
			c.Status(http.StatusNotFound)
			return
		}

		assetPath := filepath.Join(dataDir, cleanPath)
		relative, err := filepath.Rel(dataDir, assetPath)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			c.Status(http.StatusNotFound)
			return
		}

		if _, err := os.Stat(assetPath); err != nil {
			c.Status(http.StatusNotFound)
			return
		}

		c.Header("Cache-Control", "no-cache, must-revalidate")
		if strings.HasSuffix(cleanPath, "bg.jpg") {
			fmt.Printf("[data] serving bg.jpg from: %s (exists: %v)\n", assetPath, true)
		}
		c.File(assetPath)
	}
	router.GET("/data/*filepath", dataHandler)
	router.HEAD("/data/*filepath", dataHandler)
}

func registerStaticRoutes(router *gin.Engine, staticDir string) {
	indexPath := filepath.Join(staticDir, "index.html")
	if _, err := os.Stat(indexPath); err == nil {
		registerStaticRoutesFromDisk(router, staticDir, indexPath)
		return
	}

	registerStaticRoutesFromEmbedded(router)
}

func resolveEmbeddedDistFSFrom(distFS fs.FS) (fs.FS, error) {
	if _, err := fs.Stat(distFS, "index.html"); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, errors.New("embedded frontend is missing index.html")
		}
		return nil, err
	}
	return distFS, nil
}

func resolveEmbeddedDistFS() (fs.FS, error) {
	distFS, err := webassets.DistFS()
	if err != nil {
		return nil, err
	}
	return resolveEmbeddedDistFSFrom(distFS)
}

func registerStaticRoutesFromDisk(router *gin.Engine, staticDir string, indexPath string) {
	uiAssetsDir := filepath.Join(staticDir, "ui")
	if _, err := os.Stat(uiAssetsDir); err == nil {
		router.Static("/ui", uiAssetsDir)
	}
	registerLive2DStaticRoutesFromDisk(router, staticDir)

	router.NoRoute(func(c *gin.Context) {
		if !shouldServeSPAIndex(c) {
			renderRouteNotFound(c)
			return
		}

		c.File(indexPath)
	})
}

func registerStaticRoutesFromEmbedded(router *gin.Engine) {
	distFS, err := resolveEmbeddedDistFS()
	if err != nil {
		// 2026-04-08: README defines embedded frontend as the fallback hosting contract.
		// Impact: if both disk assets and embedded assets are missing, startup must fail fast
		// instead of quietly degrading into route-not-found responses for every SPA page.
		panic(err)
	}

	if uiFS, err := fs.Sub(distFS, "ui"); err == nil {
		router.StaticFS("/ui", http.FS(uiFS))
	}
	registerLive2DStaticRoutesFromFS(router, distFS)

	router.NoRoute(func(c *gin.Context) {
		if !shouldServeSPAIndex(c) {
			renderRouteNotFound(c)
			return
		}

		content, readErr := fs.ReadFile(distFS, "index.html")
		if readErr != nil {
			renderRouteNotFound(c)
			return
		}

		c.Data(http.StatusOK, "text/html; charset=utf-8", content)
	})
}

func registerLive2DStaticRoutesFromDisk(router *gin.Engine, staticDir string) {
	for _, name := range []string{"live2d-widget", "live2d-models", "live2d-config"} {
		dir := filepath.Join(staticDir, name)
		if _, err := os.Stat(dir); err == nil {
			router.Static("/"+name, dir)
		}
	}
}

func registerLive2DStaticRoutesFromFS(router *gin.Engine, distFS fs.FS) {
	for _, name := range []string{"live2d-widget", "live2d-models", "live2d-config"} {
		sub, err := fs.Sub(distFS, name)
		if err == nil {
			router.StaticFS("/"+name, http.FS(sub))
		}
	}
}

func shouldServeSPAIndex(c *gin.Context) bool {
	if c.Request.Method != http.MethodGet {
		return false
	}

	path := c.Request.URL.Path
	if path == "/api" || strings.HasPrefix(path, "/api/") {
		return false
	}

	return true
}

func renderRouteNotFound(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{
		"success": false,
		"error":   "路由不存在",
	})
}
