package repositories

import "github.com/hao/game/internal/domain"

type GameTimelineRepository struct {
	games *GamesRepository
}

func NewGameTimelineRepository(games *GamesRepository) *GameTimelineRepository {
	return &GameTimelineRepository{games: games}
}

func (r *GameTimelineRepository) List(params domain.GamesTimelineParams) ([]domain.TimelineGame, error) {
	return r.games.ListTimeline(params)
}
