package handlers

import (
	"github.com/jmoiron/sqlx"

	"github.com/hao/game/internal/config"
	"github.com/hao/game/internal/repositories"
	"github.com/hao/game/internal/services"
)

func newSplitGamesHandlerForTest(cfg config.Config, db *sqlx.DB) *GamesHandler {
	gamesRepo := repositories.NewGamesRepository(db)
	gameFilesRepo := repositories.NewGameFilesRepository(db)
	metadataRepo := repositories.NewMetadataRepository(db)
	reviewRepo := repositories.NewReviewIssueOverrideRepository(db)
	favoriteRepo := repositories.NewFavoriteGamesRepository(db)

	catalogService := services.NewGameCatalogService(repositories.NewGameCatalogRepository(gamesRepo), reviewRepo)
	return NewSplitGamesHandler(
		catalogService,
		services.NewGameTimelineService(repositories.NewGameTimelineRepository(gamesRepo)),
		services.NewGameDetailService(repositories.NewGameDetailRepository(gamesRepo), gameFilesRepo, reviewRepo),
		services.NewGameAggregateService(cfg, gamesRepo, metadataRepo, catalogService),
		services.NewGameFavoriteService(repositories.NewGameDetailRepository(gamesRepo), favoriteRepo),
	)
}
