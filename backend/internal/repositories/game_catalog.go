package repositories

import "github.com/hao/game/internal/domain"

type GameCatalogRepository struct {
	games *GamesRepository
}

func NewGameCatalogRepository(games *GamesRepository) *GameCatalogRepository {
	return &GameCatalogRepository{games: games}
}

func (r *GameCatalogRepository) List(params domain.GamesListParams) ([]domain.GameListItem, int, error) {
	return r.games.List(params)
}

func (r *GameCatalogRepository) CountPendingGroups(params domain.GamesListParams) (*domain.PendingGroupCounts, error) {
	return r.games.CountPendingGroups(params)
}

func (r *GameCatalogRepository) Stats(params domain.GamesListParams) (*domain.GameStats, error) {
	return r.games.Stats(params)
}
