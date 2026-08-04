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

type GamesTimelineResult struct {
	Games      []domain.TimelineGame
	Limit      int
	HasMore    bool
	NextCursor string // "date|id" cursor for the next page; empty when no more data
}

// GameDetail is the service-layer detail read model assembled for a single game response.
// Keep it separate from domain.Game so the detail endpoint can evolve without distorting
// aggregate writes or list summaries.
type GameDetail struct {
	Game          *domain.Game
	PendingIssues *domain.PendingIssueEvaluation
	PreviewVideos []domain.GameAsset
	Screenshots   []domain.GameAsset
	Covers        []domain.GameAsset
	Banners       []domain.GameAsset
	Logos         []domain.GameAsset
	Series        *domain.MetadataItem
	Developers    []domain.MetadataItem
	Publishers    []domain.MetadataItem
	Files         []domain.GameFile
}

// GamePreviewVideoBundle is the batch preview-video read model: one game's video
// assets keyed by its public id, used by the game store session to fetch trailers
// for many games in a single round trip.
type GamePreviewVideoBundle struct {
	PublicID     string
	PreviewVideos []domain.GameAsset
}
