package services

import (
	"strconv"

	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/repositories"
)

type GameTimelineService struct {
	timelineRepo *repositories.GameTimelineRepository
}

func NewGameTimelineService(timelineRepo *repositories.GameTimelineRepository) *GameTimelineService {
	return &GameTimelineService{timelineRepo: timelineRepo}
}

// List returns one page of the release timeline with a cursor for the next page.
func (s *GameTimelineService) List(params domain.GamesTimelineParams) (*GamesTimelineResult, error) {
	if err := normalizeTimelineParams(&params); err != nil {
		return nil, err
	}

	games, err := s.timelineRepo.List(params)
	if err != nil {
		return nil, err
	}

	// The repo fetches limit+1 rows; if we got more than limit, there's a next page.
	hasMore := len(games) > params.Limit
	if hasMore {
		games = games[:params.Limit]
	}

	var nextCursor string
	if hasMore && len(games) > 0 {
		last := games[len(games)-1]
		if last.ReleaseDate != nil {
			nextCursor = *last.ReleaseDate + "|" + strconv.FormatInt(last.ID, 10)
		}
	}

	return &GamesTimelineResult{
		Games:      games,
		Limit:      params.Limit,
		HasMore:    hasMore,
		NextCursor: nextCursor,
	}, nil
}
