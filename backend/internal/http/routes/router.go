package routes

import (
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"

	"github.com/hao/game/internal/config"
	"github.com/hao/game/internal/domain"
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
	gameDetailRepo := repositories.NewGameDetailRepository(gamesRepo)
	gameFilesRepo := repositories.NewGameFilesRepository(db)
	assetsRepo := repositories.NewAssetsRepository(db)
	metadataRepo := repositories.NewMetadataRepository(db)
	tagsRepo := repositories.NewTagsRepository(db)
	favoriteGamesRepo := repositories.NewFavoriteGamesRepository(db)
	reviewIssueOverridesRepo := repositories.NewReviewIssueOverrideRepository(db)
	wikiRepo := repositories.NewWikiRepository(db)
	gameCatalogRepo := repositories.NewGameCatalogRepository(gamesRepo)
	gameTimelineRepo := repositories.NewGameTimelineRepository(gamesRepo)
	gameAggregateRepo := repositories.NewGameAggregateRepository(gamesRepo)
	gameCatalogService := services.NewGameCatalogService(gameCatalogRepo, reviewIssueOverridesRepo)
	gameTimelineService := services.NewGameTimelineService(gameTimelineRepo)
	gameDetailService := services.NewGameDetailService(gameDetailRepo, gameFilesRepo, tagsRepo, reviewIssueOverridesRepo)
	gameAggregateService := services.NewGameAggregateService(cfg, gameAggregateRepo, metadataRepo, tagsRepo)
	gameFavoriteService := services.NewGameFavoriteService(gameDetailRepo, favoriteGamesRepo)
	gameFilesService := services.NewGameFilesService(cfg, gameDetailRepo, gameFilesRepo)
	windowsLaunchService := services.NewWindowsLaunchService(cfg, gameDetailRepo, gameFilesRepo)
	assetsService := services.NewAssetsService(cfg, gameDetailRepo, assetsRepo)
	directoryService := services.NewDirectoryService(cfg)
	metadataService := services.NewMetadataService(metadataRepo)
	pendingIssuesService := services.NewPendingIssuesService()
	tagsService := services.NewTagsService(tagsRepo)
	reviewIssueOverrideService := services.NewReviewIssueOverrideService(gameDetailRepo, reviewIssueOverridesRepo)
	steamService := services.NewSteamService(cfg, assetsService)
	wikiService := services.NewWikiService(gameDetailRepo, wikiRepo, cfg.WikiHistoryLimit)
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
	seriesHandler := handlers.NewMetadataHandler(metadataService, services.MetadataResource{Table: "series", ResourceName: "series"})
	platformsHandler := handlers.NewMetadataHandler(metadataService, services.MetadataResource{Table: "platforms", ResourceName: "platforms"})
	developersHandler := handlers.NewMetadataHandler(metadataService, services.MetadataResource{Table: "developers", ResourceName: "developers"})
	publishersHandler := handlers.NewMetadataHandler(metadataService, services.MetadataResource{Table: "publishers", ResourceName: "publishers"})
	reviewIssueOverrideHandler := handlers.NewReviewIssueOverrideHandler(reviewIssueOverrideService)
	pendingIssuesHandler := handlers.NewPendingIssuesHandler(pendingIssuesService)
	steamHandler := handlers.NewSteamHandler(steamService)
	wikiHandler := handlers.NewWikiHandler(wikiService)
	tagsHandler := handlers.NewTagsHandler(tagsService)
	hitokotoHandler := handlers.NewHitokotoHandler(hitokotoService)

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
	api.GET("/games/:publicId", gamesHandler.Get)
	api.PUT("/games/:publicId/favorite", gamesHandler.Favorite)
	api.DELETE("/games/:publicId/favorite", gamesHandler.Unfavorite)
	api.POST("/games", gamesHandler.Create)
	api.PUT("/games/:publicId/aggregate", gamesHandler.UpdateAggregate)
	api.DELETE("/games/:publicId", gamesHandler.Delete)
	api.GET("/games/:publicId/files", gameFilesHandler.List)
	api.POST("/games/:publicId/files", gameFilesHandler.Create)
	api.PUT("/games/:publicId/files/:fileId", gameFilesHandler.Update)
	api.DELETE("/games/:publicId/files/:fileId", gameFilesHandler.Delete)
	api.POST("/games/:publicId/files/:fileId/downloads", downloadsHandler.RecordDownload)
	api.GET("/games/:publicId/files/:fileId/download", downloadsHandler.Download)
	api.GET("/games/:publicId/files/:fileId/launch-script", downloadsHandler.LaunchScript)
	api.GET("/games/:publicId/wiki", wikiHandler.Get)
	api.PUT("/games/:publicId/wiki", wikiHandler.Update)
	api.GET("/games/:publicId/wiki/history", wikiHandler.History)
	api.GET("/series", seriesHandler.List)
	api.GET("/series/:id", seriesHandler.Get)
	api.POST("/series", seriesHandler.Create)
	api.GET("/platforms", platformsHandler.List)
	api.POST("/platforms", platformsHandler.Create)
	api.GET("/developers", developersHandler.List)
	api.POST("/developers", developersHandler.Create)
	api.GET("/publishers", publishersHandler.List)
	api.POST("/publishers", publishersHandler.Create)
	api.GET("/tag-groups", tagsHandler.ListGroups)
	api.POST("/tag-groups", tagsHandler.CreateGroup)
	api.GET("/tags", tagsHandler.ListTags)
	api.POST("/tags", tagsHandler.CreateTag)
	api.GET("/review-issue-overrides", reviewIssueOverrideHandler.List)
	api.PUT("/games/:publicId/review-issues/:issueKey/ignore", reviewIssueOverrideHandler.Ignore)
	api.DELETE("/games/:publicId/review-issues/:issueKey/ignore", reviewIssueOverrideHandler.Delete)
	api.POST("/assets/cover", assetsHandler.Upload("cover"))
	api.POST("/assets/banner", assetsHandler.Upload("banner"))
	api.POST("/assets/video", assetsHandler.Upload("video"))
	api.POST("/assets/screenshot", assetsHandler.Upload("screenshot"))
	api.PUT("/assets/screenshot/order", assetsHandler.ReorderScreenshots)
	api.PUT("/assets/video/order", assetsHandler.ReorderVideos)
	api.DELETE("/assets", assetsHandler.Delete)
	api.GET("/directory/default", directoryHandler.Default)
	api.GET("/directory/list", directoryHandler.List)
	api.GET("/steam/search", steamHandler.Search)
	api.GET("/steam/:appId/assets", steamHandler.Preview)
	api.POST("/steam/:appId/apply-assets", steamHandler.Apply)
	api.GET("/steam/proxy", steamHandler.Proxy)

	registerAssetRoutes(router, cfg.AssetsDir, gameDetailRepo)
	registerCustomDataRoutes(router, filepath.Dir(cfg.AssetsDir))
	registerStaticRoutes(router, cfg.StaticDir)

	return router
}

type assetRouteGameRepository interface {
	GetByPublicID(publicID string) (*domain.Game, error)
}

func registerAssetRoutes(router *gin.Engine, assetsDir string, gamesRepo assetRouteGameRepository) {
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		return
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

		game, err := gamesRepo.GetByPublicID(gamePublicID)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}

		isAdmin, _ := c.Get("is_admin")
		admin, _ := isAdmin.(bool)
		if !admin && game.Visibility == domain.GameVisibilityPrivate {
			c.Status(http.StatusNotFound)
			return
		}

		targetPath := filepath.Join(assetsDir, filepath.FromSlash(rawPath))
		relative, err := filepath.Rel(assetsDir, targetPath)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			c.Status(http.StatusNotFound)
			return
		}

		if _, err := os.Stat(targetPath); err != nil {
			c.Status(http.StatusNotFound)
			return
		}

		c.File(targetPath)
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

	router.GET("/data/*filepath", func(c *gin.Context) {
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

		c.File(assetPath)
	})
}

func registerStaticRoutes(router *gin.Engine, staticDir string) {
	indexPath := filepath.Join(staticDir, "index.html")
	if _, err := os.Stat(indexPath); err == nil {
		registerStaticRoutesFromDisk(router, staticDir, indexPath)
		return
	}

	registerStaticRoutesFromEmbedded(router)
}

func registerStaticRoutesFromDisk(router *gin.Engine, staticDir string, indexPath string) {
	uiAssetsDir := filepath.Join(staticDir, "ui")
	if _, err := os.Stat(uiAssetsDir); err == nil {
		router.Static("/ui", uiAssetsDir)
	}

	router.NoRoute(func(c *gin.Context) {
		if !shouldServeSPAIndex(c) {
			renderRouteNotFound(c)
			return
		}

		c.File(indexPath)
	})
}

func registerStaticRoutesFromEmbedded(router *gin.Engine) {
	distFS, err := webassets.DistFS()
	if err != nil {
		return
	}
	if _, err := fs.Stat(distFS, "index.html"); err != nil {
		return
	}

	if uiFS, err := fs.Sub(distFS, "ui"); err == nil {
		router.StaticFS("/ui", http.FS(uiFS))
	}

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
		"error":   "route not found",
	})
}
