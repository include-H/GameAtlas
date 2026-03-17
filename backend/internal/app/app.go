package app

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jmoiron/sqlx"

	"github.com/hao/game/internal/config"
	"github.com/hao/game/internal/db"
	"github.com/hao/game/internal/http/routes"
)

type App struct {
	config config.Config
	db     *sqlx.DB
	server *http.Server
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
