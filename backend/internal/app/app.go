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
	config config.Config
	db     *sqlx.DB
	server *http.Server
}

func New(cfg config.Config) (*App, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	sqliteDB, err := db.OpenSQLite(cfg.DBPath)
	if err != nil {
		return nil, err
	}

	if err := db.RunMigrations(sqliteDB); err != nil {
		_ = sqliteDB.Close()
		return nil, err
	}

	gamesRepo := repositories.NewGamesRepository(sqliteDB)
	assetReconcileService := services.NewAssetReconcileService(cfg, sqliteDB)
	gameAggregateService := services.NewGameAggregateService(
		cfg,
		gamesRepo,
		repositories.NewMetadataRepository(sqliteDB),
	)
	if processed, err := gameAggregateService.ProcessPendingAssetCleanup(100); err != nil {
		log.Printf("asset cleanup retry failed after %d task(s): %v", processed, err)
	} else if processed > 0 {
		log.Printf("asset cleanup retry processed %d task(s)", processed)
	}
	if reconciled, err := assetReconcileService.ReconcileAllMissingAssets(); err != nil {
		log.Printf("asset reconcile failed: %v", err)
	} else if reconciled > 0 {
		log.Printf("asset reconcile removed stale references for %d game(s)", reconciled)
	}

	go func() {
		orphaned, err := assetReconcileService.CleanOrphanedAssetFiles()
		if err != nil {
			log.Printf("orphaned asset cleanup failed: %v", err)
		} else if orphaned > 0 {
			log.Printf("orphaned asset cleanup deleted %d file(s)", orphaned)
		}
	}()

	router := routes.New(cfg, sqliteDB)

	server := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler:           router,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
	}

	return &App{
		config: cfg,
		db:     sqliteDB,
		server: server,
	}, nil
}

func (a *App) Run() error {
	return a.server.ListenAndServe()
}

func (a *App) Shutdown(ctx context.Context) error {
	return a.server.Shutdown(ctx)
}

func (a *App) Close() error {
	if a.db == nil {
		return nil
	}

	return a.db.Close()
}
