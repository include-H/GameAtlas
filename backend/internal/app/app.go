package app

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/jmoiron/sqlx"

	"github.com/hao/game/internal/config"
	"github.com/hao/game/internal/db"
	"github.com/hao/game/internal/http/routes"
	"github.com/hao/game/internal/repositories"
	"github.com/hao/game/internal/services"
)

type App struct {
	config       config.Config
	db           *sqlx.DB
	server       *http.Server
	backupCancel context.CancelFunc
}

func New(cfg config.Config) (*App, error) {
	sqliteDB, err := db.OpenSQLite(cfg.DBPath)
	if err != nil {
		return nil, err
	}

	if err := db.RunMigrations(sqliteDB); err != nil {
		_ = sqliteDB.Close()
		return nil, err
	}

	settingsRepo := repositories.NewAppSettingsRepository(sqliteDB)
	if err := settingsRepo.EnsureDefaults(cfg.RuntimeSettings()); err != nil {
		_ = sqliteDB.Close()
		return nil, err
	}
	settings, err := settingsRepo.List()
	if err != nil {
		_ = sqliteDB.Close()
		return nil, err
	}
	cfg, err = cfg.ApplyRuntimeSettings(settings)
	if err != nil {
		_ = sqliteDB.Close()
		return nil, err
	}
	if normalized := cfg.NormalizeStoredRuntimePaths(); len(normalized) > 0 {
		if err := settingsRepo.UpsertMany(normalized); err != nil {
			_ = sqliteDB.Close()
			return nil, err
		}
	}
	if err := cfg.Validate(); err != nil {
		_ = sqliteDB.Close()
		return nil, err
	}
	if cfg.Proxy == "" {
		log.Printf("outbound proxy: disabled")
	} else {
		log.Printf("outbound proxy: %s", cfg.ProxyLogValue())
	}

	gamesRepo := repositories.NewGamesRepository(sqliteDB)
	backupService := services.NewDatabaseBackupService(cfg, sqliteDB)
	if path, err := backupService.BackupNow("startup"); err != nil {
		_ = sqliteDB.Close()
		return nil, err
	} else if path != "" {
		log.Printf("database startup backup created: %s", cfg.RuntimeRelativePath(path))
	}
	if removed, err := backupService.CleanupOldBackups(); err != nil {
		log.Printf("database backup retention cleanup failed: %v", err)
	} else if removed > 0 {
		log.Printf("database backup retention removed %d file(s)", removed)
	}

	assetReconcileService := services.NewAssetReconcileService(cfg, sqliteDB)
	favoriteGamesRepo := repositories.NewFavoriteGamesRepository(sqliteDB)
	catalogRepo := repositories.NewGameCatalogRepository(gamesRepo, favoriteGamesRepo)
	reviewIssueOverridesRepo := repositories.NewReviewIssueOverrideRepository(sqliteDB)
	catalogService := services.NewGameCatalogService(catalogRepo, reviewIssueOverridesRepo)
	assetCleanupTasksRepo := repositories.NewAssetCleanupTasksRepository(sqliteDB)
	metadataService := services.NewMetadataService(repositories.NewMetadataRepository(sqliteDB))
	gameAggregateService := services.NewGameAggregateService(
		cfg,
		gamesRepo,
		metadataService,
		catalogService,
		assetCleanupTasksRepo,
	)
	if processed, err := gameAggregateService.ProcessPendingAssetCleanup(100); err != nil {
		log.Printf("asset cleanup retry failed after %d task(s): %v", processed, err)
	} else if processed > 0 {
		log.Printf("asset cleanup retry processed %d task(s)", processed)
	}
	if cleaned, err := assetReconcileService.CleanStaging(); err != nil {
		log.Printf("staging cleanup failed: %v", err)
	} else if cleaned > 0 {
		log.Printf("staging cleanup removed %d expired file(s)", cleaned)
	}
	if reconciled, err := assetReconcileService.ReconcileAllMissingAssets(); err != nil {
		log.Printf("asset reconcile failed: %v", err)
	} else if reconciled > 0 {
		log.Printf("asset reconcile removed stale references for %d game(s)", reconciled)
	}
	if backfilled, err := assetReconcileService.BackfillThumbnails(); err != nil {
		log.Printf("thumbnail backfill failed: %v", err)
	} else if backfilled > 0 {
		log.Printf("thumbnail backfill generated %d thumbnail(s)", backfilled)
	}

	go func() {
		orphaned, err := assetReconcileService.CleanOrphanedAssetFiles()
		if err != nil {
			log.Printf("orphaned asset cleanup failed: %v", err)
		} else if orphaned > 0 {
			log.Printf("orphaned asset cleanup processed %d file(s)", orphaned)
		}
	}()

	router := routes.New(cfg, sqliteDB)

	server := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler:           router,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
	}

	backupCtx, backupCancel := context.WithCancel(context.Background())
	backupService.StartPeriodic(backupCtx)

	return &App{
		config:       cfg,
		db:           sqliteDB,
		server:       server,
		backupCancel: backupCancel,
	}, nil
}

func (a *App) Run() error {
	return a.server.ListenAndServe()
}

func (a *App) Shutdown(ctx context.Context) error {
	if a.backupCancel != nil {
		a.backupCancel()
	}
	return a.server.Shutdown(ctx)
}

func (a *App) Close() error {
	if a.backupCancel != nil {
		a.backupCancel()
	}
	if a.db == nil {
		return nil
	}

	return a.db.Close()
}
