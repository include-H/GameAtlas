package services

import "github.com/hao/game/internal/domain"

type GameDeleteResult struct {
	Warnings []string
}

type GamesListResult struct {
	Games              []domain.GameListItem
	Page               int
	Limit              int
	Total              int
	TotalPages         int
	PendingIssueCounts *domain.PendingIssueCountSummary
}

type TimelineCursor struct {
	ReleaseDate string
	ID          int64
}

type GamesTimelineResult struct {
	Games      []domain.TimelineGame
	Limit      int
	FromDate   string
	ToDate     string
	HasMore    bool
	NextCursor *TimelineCursor
}

// GameDetail is the service-layer detail read model assembled for a single game response.
// Keep it separate from domain.Game so the detail endpoint can evolve without distorting
// aggregate writes or list summaries.
type GameDetail struct {
	Game          *domain.Game
	PendingIssues *domain.PendingIssueEvaluation
	PreviewVideos []domain.GameAsset
	Screenshots   []domain.GameAsset
	Series        *domain.MetadataItem
	Platforms     []domain.MetadataItem
	Developers    []domain.MetadataItem
	Publishers    []domain.MetadataItem
	Tags          []domain.Tag
	TagGroups     []domain.GameTagGroup
	Files         []domain.GameFile
}
