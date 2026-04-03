package repositories

import "github.com/hao/game/internal/domain"

type GameTimelineRepository struct {
	games *GamesRepository
}

func NewGameTimelineRepository(games *GamesRepository) *GameTimelineRepository {
	return &GameTimelineRepository{games: games}
}

func (r *GameTimelineRepository) List(params domain.GamesTimelineParams) ([]domain.TimelineGame, bool, error) {
	return r.games.ListTimeline(params)
}

func (r *GameTimelineRepository) LatestReleaseDate(includeAll bool, visibility string) (*string, error) {
	return r.games.LatestTimelineReleaseDate(includeAll, visibility)
}

func (r *GameTimelineRepository) HasOlder(params domain.GamesTimelineParams, cursorReleaseDate string, cursorID int64) (bool, error) {
	return r.games.HasOlderTimelineGame(params, cursorReleaseDate, cursorID)
}
