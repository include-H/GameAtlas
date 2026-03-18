package routes

import (
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"

	"github.com/hao/game/internal/config"
	"github.com/hao/game/internal/http/handlers"
	"github.com/hao/game/internal/markdown"
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
	gamesRepo := repositories.NewGamesRepository(db)
	gameFilesRepo := repositories.NewGameFilesRepository(db)
	assetsRepo := repositories.NewAssetsRepository(db)
	metadataRepo := repositories.NewMetadataRepository(db)
	reviewIssueOverridesRepo := repositories.NewReviewIssueOverrideRepository(db)
	wikiRepo := repositories.NewWikiRepository(db)
	markdownRenderer := markdown.NewRenderer()
	gamesService := services.NewGamesService(cfg, gamesRepo, gameFilesRepo)
	gameFilesService := services.NewGameFilesService(cfg, gamesRepo, gameFilesRepo)
	assetsService := services.NewAssetsService(cfg, gamesRepo, assetsRepo)
	directoryService := services.NewDirectoryService(cfg)
	metadataService := services.NewMetadataService(metadataRepo)
	reviewIssueOverrideService := services.NewReviewIssueOverrideService(gamesRepo, reviewIssueOverridesRepo)
	steamService := services.NewSteamService(cfg, assetsService)
	wikiService := services.NewWikiService(gamesRepo, wikiRepo, markdownRenderer)
	assetsHandler := handlers.NewAssetsHandler(assetsService)
	directoryHandler := handlers.NewDirectoryHandler(directoryService)
	gamesHandler := handlers.NewGamesHandler(gamesService)
	gameFilesHandler := handlers.NewGameFilesHandler(gameFilesService)
	downloadsHandler := handlers.NewDownloadsHandler(gameFilesService)
	seriesHandler := handlers.NewMetadataHandler(metadataService, services.MetadataResource{Table: "series", ResourceName: "series"})
	platformsHandler := handlers.NewMetadataHandler(metadataService, services.MetadataResource{Table: "platforms", ResourceName: "platforms"})
	developersHandler := handlers.NewMetadataHandler(metadataService, services.MetadataResource{Table: "developers", ResourceName: "developers"})
	publishersHandler := handlers.NewMetadataHandler(metadataService, services.MetadataResource{Table: "publishers", ResourceName: "publishers"})
	reviewIssueOverrideHandler := handlers.NewReviewIssueOverrideHandler(reviewIssueOverrideService)
	steamHandler := handlers.NewSteamHandler(steamService)
	wikiHandler := handlers.NewWikiHandler(wikiService)

	api := router.Group("/api")
	api.GET("/health", healthHandler.Get)
	api.GET("/games", gamesHandler.List)
	api.GET("/games/:id", gamesHandler.Get)
	api.POST("/games", gamesHandler.Create)
	api.PUT("/games/:id", gamesHandler.Update)
	api.DELETE("/games/:id", gamesHandler.Delete)
	api.GET("/games/:id/files", gameFilesHandler.List)
	api.POST("/games/:id/files", gameFilesHandler.Create)
	api.PUT("/games/:id/files/:fileId", gameFilesHandler.Update)
	api.DELETE("/games/:id/files/:fileId", gameFilesHandler.Delete)
	api.GET("/games/:id/files/:fileId/download", downloadsHandler.Download)
	api.GET("/games/:id/wiki", wikiHandler.Get)
	api.PUT("/games/:id/wiki", wikiHandler.Update)
	api.GET("/games/:id/wiki/history", wikiHandler.History)
	api.GET("/series", seriesHandler.List)
	api.POST("/series", seriesHandler.Create)
	api.GET("/platforms", platformsHandler.List)
	api.POST("/platforms", platformsHandler.Create)
	api.GET("/developers", developersHandler.List)
	api.POST("/developers", developersHandler.Create)
	api.GET("/publishers", publishersHandler.List)
	api.POST("/publishers", publishersHandler.Create)
	api.GET("/review-issue-overrides", reviewIssueOverrideHandler.List)
	api.PUT("/games/:id/review-issues/:issueKey/ignore", reviewIssueOverrideHandler.Ignore)
	api.DELETE("/games/:id/review-issues/:issueKey/ignore", reviewIssueOverrideHandler.Delete)
	api.POST("/assets/cover", assetsHandler.Upload("cover"))
	api.POST("/assets/banner", assetsHandler.Upload("banner"))
	api.POST("/assets/video", assetsHandler.Upload("video"))
	api.POST("/assets/screenshot", assetsHandler.Upload("screenshot"))
	api.PUT("/assets/screenshot/order", assetsHandler.ReorderScreenshots)
	api.PUT("/assets/video/primary", assetsHandler.SetPrimaryVideo)
	api.DELETE("/assets", assetsHandler.Delete)
	api.GET("/directory/default", directoryHandler.Default)
	api.GET("/directory/list", directoryHandler.List)
	api.GET("/steam/search", steamHandler.Search)
	api.GET("/steam/:appId/assets", steamHandler.Preview)
	api.POST("/steam/:appId/apply-assets", steamHandler.Apply)
	api.GET("/steam/proxy", steamHandler.Proxy)

	registerAssetRoutes(router, cfg.AssetsDir)
	registerStaticRoutes(router, cfg.StaticDir)

	return router
}

func registerAssetRoutes(router *gin.Engine, assetsDir string) {
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		return
	}

	router.Static("/assets", assetsDir)
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
		if c.Request.Method != http.MethodGet {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "route not found",
			})
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
		if c.Request.Method != http.MethodGet {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "route not found",
			})
			return
		}

		content, readErr := fs.ReadFile(distFS, "index.html")
		if readErr != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "route not found",
			})
			return
		}

		c.Data(http.StatusOK, "text/html; charset=utf-8", content)
	})
}
