package services

import (
	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/repositories"
)

type GameTimelineService struct {
	timelineRepo *repositories.GameTimelineRepository
}

// NewGameTimelineService wires the release-timeline read model.
func NewGameTimelineService(timelineRepo *repositories.GameTimelineRepository) *GameTimelineService {
	return &GameTimelineService{timelineRepo: timelineRepo}
}

// List returns one window of the release timeline plus the cursor needed to request the next window.
func (s *GameTimelineService) List(params domain.GamesTimelineParams) (*GamesTimelineResult, error) {
	if err := normalizeTimelineParams(&params); err != nil {
		return nil, err
	}

	games, hasMoreInWindow, err := s.timelineRepo.List(params)
	if err != nil {
		return nil, err
	}

	hasOlderWindow := false
	if !hasMoreInWindow && params.CursorReleaseDate == "" && len(games) > 0 {
		last := games[len(games)-1]
		if last.ReleaseDate != nil {
			// The first window can end exactly at the archive boundary; probe once more so the UI can
			// still show a "load more" affordance when older data exists beyond the current month range.
			exists, err := s.timelineRepo.HasOlder(params, *last.ReleaseDate, last.ID)
			if err != nil {
				return nil, err
			}
			hasOlderWindow = exists
		}
	}

	var nextCursor *TimelineCursor
	if hasMoreInWindow && len(games) > 0 {
		last := games[len(games)-1]
		if last.ReleaseDate != nil {
			// Cursor pagination stays stable by pairing the release date with the last seen game id.
			nextCursor = &TimelineCursor{
				ReleaseDate: *last.ReleaseDate,
				ID:          last.ID,
			}
		}
	}

	return &GamesTimelineResult{
		Games:      games,
		Limit:      params.Limit,
		FromDate:   params.FromDate,
		ToDate:     params.ToDate,
		HasMore:    hasMoreInWindow || hasOlderWindow,
		NextCursor: nextCursor,
	}, nil
}

// LatestReleaseDate returns the newest release date in timeline format, if any public data exists.
func (s *GameTimelineService) LatestReleaseDate(includeAll bool) (string, bool, error) {
	releaseDate, err := s.timelineRepo.LatestReleaseDate(includeAll, domain.GameVisibilityPublic)
	if err != nil {
		return "", false, err
	}
	if releaseDate == nil {
		return "", false, nil
	}

	normalized, err := normalizeTimelineDate(*releaseDate)
	if err != nil {
		return "", false, nil
	}

	return normalized, true, nil
}
