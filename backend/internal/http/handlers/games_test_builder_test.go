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
	tagsRepo := repositories.NewTagsRepository(db)
	reviewRepo := repositories.NewReviewIssueOverrideRepository(db)
	favoriteRepo := repositories.NewFavoriteGamesRepository(db)

	return NewSplitGamesHandler(
		services.NewGameCatalogService(repositories.NewGameCatalogRepository(gamesRepo), reviewRepo),
		services.NewGameTimelineService(repositories.NewGameTimelineRepository(gamesRepo)),
		services.NewGameDetailService(repositories.NewGameDetailRepository(gamesRepo), gameFilesRepo, tagsRepo, reviewRepo),
		services.NewGameAggregateService(cfg, gamesRepo, metadataRepo, tagsRepo),
		services.NewGameFavoriteService(repositories.NewGameDetailRepository(gamesRepo), favoriteRepo),
	)
}
