package services

import (
	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/repositories"
)

type GameDetailService struct {
	detailRepo               *repositories.GameDetailRepository
	gameFilesRepo            *repositories.GameFilesRepository
	tagsRepo                 *repositories.TagsRepository
	reviewIssueOverridesRepo *repositories.ReviewIssueOverrideRepository
}

// NewGameDetailService wires the repositories required to assemble the full detail payload for one game.
func NewGameDetailService(
	detailRepo *repositories.GameDetailRepository,
	gameFilesRepo *repositories.GameFilesRepository,
	tagsRepo *repositories.TagsRepository,
	reviewIssueOverridesRepo *repositories.ReviewIssueOverrideRepository,
) *GameDetailService {
	return &GameDetailService{
		detailRepo:               detailRepo,
		gameFilesRepo:            gameFilesRepo,
		tagsRepo:                 tagsRepo,
		reviewIssueOverridesRepo: reviewIssueOverridesRepo,
	}
}

// ResolveGameID translates the public id used by routes into the internal numeric id used by repositories.
func (s *GameDetailService) ResolveGameID(publicID string) (int64, error) {
	id, err := s.detailRepo.ResolveIDByPublicID(publicID)
	if err != nil {
		return 0, normalizeRepoError(err)
	}
	return id, nil
}

// Get assembles the detail response from multiple repositories and applies visibility checks up front.
func (s *GameDetailService) Get(id int64, includeAll bool) (*GameDetail, error) {
	game, err := s.detailRepo.GetByID(id)
	if err != nil {
		return nil, normalizeRepoError(err)
	}
	if !includeAll && game.Visibility == domain.GameVisibilityPrivate {
		// Public callers should observe private games as missing rather than leaking their existence.
		return nil, ErrNotFound
	}

	screenshots, err := s.detailRepo.ListScreenshots(id)
	if err != nil {
		return nil, err
	}
	videos, err := s.detailRepo.ListVideos(id)
	if err != nil {
		return nil, err
	}
	primarySeries, err := s.detailRepo.GetSeriesMetadata(id)
	if err != nil {
		return nil, err
	}
	platforms, err := s.detailRepo.ListMetadata("platforms", "game_platforms", "platform_id", id)
	if err != nil {
		return nil, err
	}
	developers, err := s.detailRepo.ListMetadata("developers", "game_developers", "developer_id", id)
	if err != nil {
		return nil, err
	}
	publishers, err := s.detailRepo.ListMetadata("publishers", "game_publishers", "publisher_id", id)
	if err != nil {
		return nil, err
	}
	tags, err := s.tagsRepo.ListByGameID(id)
	if err != nil {
		return nil, err
	}
	files, err := s.gameFilesRepo.ListByGameID(id)
	if err != nil {
		return nil, err
	}
	pendingIssues, err := getPendingIssueEvaluation(*game, s.reviewIssueOverridesRepo)
	if err != nil {
		return nil, err
	}

	var previewVideo *domain.GameAsset
	if len(videos) > 0 {
		// Keep the first ordered video as the lightweight preview slot the frontend expects.
		previewVideo = &videos[0]
	}

	return &GameDetail{
		Game:          game,
		PendingIssues: pendingIssues,
		PreviewVideo:  previewVideo,
		PreviewVideos: videos,
		Screenshots:   screenshots,
		Series:        primarySeries,
		Platforms:     emptyMetadata(platforms),
		Developers:    emptyMetadata(developers),
		Publishers:    emptyMetadata(publishers),
		Tags:          emptyTags(tags),
		TagGroups:     groupGameTags(tags),
		Files:         emptyFiles(files),
	}, nil
}
