package routes

import (
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

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
	startScreenTilesRepo := repositories.NewStartScreenTilesRepository(db)
	startScreenTilesService := services.NewStartScreenTilesService(
		startScreenTilesRepo,
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
	api.POST("/assets/poster", assetsHandler.Upload("poster"))
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
	api.POST("/start-screen/tiles", startScreenTilesHandler.AddTile)
	api.DELETE("/start-screen/tiles/:gameId", startScreenTilesHandler.RemoveTile)
	api.PUT("/start-screen/tiles", startScreenTilesHandler.Update)
	api.POST("/start-screen/tiles/image", startScreenTilesHandler.UploadImage)
	api.GET("/steam/search", steamHandler.Search)
	api.GET("/steam/:appId/assets", steamHandler.Preview)
	api.GET("/steam/proxy", steamHandler.Proxy)
	api.GET("/steamgriddb/available", steamGridDBHandler.Available)
	api.GET("/steamgriddb/search", steamGridDBHandler.Search)
	api.GET("/steamgriddb/game/:gameId/grids", steamGridDBHandler.GetGridsByGameID)
	api.GET("/steamgriddb/game/:gameId/heroes", steamGridDBHandler.GetHeroesByGameID)
	api.GET("/steamgriddb/game/:gameId/logos", steamGridDBHandler.GetLogosByGameID)

	registerAssetRoutes(router, cfg.AssetsDir, gamesRepo, startScreenTilesRepo)
	// /data/bg.jpg 挂载到 /api 前缀：浏览器直连 URL 统一走 api-url.ts 的 buildApiUrl。
	registerCustomDataRoutes(api, filepath.Dir(cfg.AssetsDir))
	registerStaticRoutes(router, cfg.StaticDir)

	return router
}

type assetRouteGameRepository interface {
	GetByPublicID(publicID string) (*domain.Game, error)
}

type assetRouteStartScreenRepository interface {
	GetGameVisibilityByImagePath(imagePath string) (string, error)
}

func registerAssetRoutes(
	router *gin.Engine,
	assetsDir string,
	gamesRepo assetRouteGameRepository,
	startScreenTilesRepo assetRouteStartScreenRepository,
) {
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		return
	}

	assetStore := files.NewAssetStore(assetsDir)
	requestedVariantWidth := func(c *gin.Context) int {
		raw := strings.TrimSpace(c.Query("w"))
		if raw == "" {
			return 0
		}
		width, err := strconv.Atoi(raw)
		if err != nil || !files.IsAllowedVariantWidth(width) {
			return 0
		}
		return width
	}

	serveAssetFile := func(c *gin.Context, rawPath string) {
		targetPath := filepath.Join(assetsDir, filepath.FromSlash(rawPath))
		relative, err := filepath.Rel(assetsDir, targetPath)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			c.Status(http.StatusNotFound)
			return
		}

		permanentFile := true
		if _, err := os.Stat(targetPath); err != nil {
			// Fallback: check the exact game-scoped staging path for files not yet
			// moved to permanent storage.
			stagingPath, stagingErr := assetStore.StagingPath("/assets/" + rawPath)
			if stagingErr == nil {
				if _, statErr := os.Stat(stagingPath); statErr == nil {
					targetPath = stagingPath
					permanentFile = false
				} else {
					c.Status(http.StatusNotFound)
					return
				}
			} else {
				c.Status(http.StatusNotFound)
				return
			}
		}

		c.Header("Cache-Control", "public, max-age=86400")

		if permanentFile {
			if variant, variantErr := assetStore.EnsureVariant("/assets/"+rawPath, requestedVariantWidth(c)); variantErr == nil && variant != targetPath {
				targetPath = variant
				c.Header("Cache-Control", "public, max-age=31536000, immutable")
			}
		}

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

		isAdmin, _ := c.Get("is_admin")
		admin, _ := isAdmin.(bool)

		// 开始屏幕磁贴裁剪图先反查所属游戏，复用同一可见性规则；
		// 未登记的暂存图只允许管理员访问，保证保存前裁剪预览仍可用。
		if gamePublicID == "start-screen" {
			visibility, err := startScreenTilesRepo.GetGameVisibilityByImagePath("/assets/" + rawPath)
			if err != nil {
				if !admin {
					c.Status(http.StatusNotFound)
					return
				}
				serveAssetFile(c, rawPath)
				return
			}
			if !admin && visibility == domain.GameVisibilityPrivate {
				c.Status(http.StatusNotFound)
				return
			}
			serveAssetFile(c, rawPath)
			return
		}

		game, err := gamesRepo.GetByPublicID(gamePublicID)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		gameVisibility := game.Visibility

		if !admin && gameVisibility == domain.GameVisibilityPrivate {
			c.Status(http.StatusNotFound)
			return
		}

		serveAssetFile(c, rawPath)
	})
}

func registerCustomDataRoutes(api *gin.RouterGroup, dataDir string) {
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

		if cleanPath != "bg.jpg" {
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
		c.File(assetPath)
	}
	api.GET("/data/*filepath", dataHandler)
	api.HEAD("/data/*filepath", dataHandler)
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
