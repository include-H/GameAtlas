package services

import (
	"strings"

	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/repositories"
)

type GameDetailService struct {
	gamesRepo                *repositories.GamesRepository
	gameFilesRepo            *repositories.GameFilesRepository
	reviewIssueOverridesRepo *repositories.ReviewIssueOverrideRepository
}

// NewGameDetailService wires the repositories required to assemble the full detail payload for one game.
func NewGameDetailService(
	gamesRepo *repositories.GamesRepository,
	gameFilesRepo *repositories.GameFilesRepository,
	reviewIssueOverridesRepo *repositories.ReviewIssueOverrideRepository,
) *GameDetailService {
	return &GameDetailService{
		gamesRepo:                gamesRepo,
		gameFilesRepo:            gameFilesRepo,
		reviewIssueOverridesRepo: reviewIssueOverridesRepo,
	}
}

// ResolveGameID translates the public id used by routes into the internal numeric id used by repositories.
func (s *GameDetailService) ResolveGameID(publicID string) (int64, error) {
	id, err := s.gamesRepo.ResolveIDByPublicID(publicID)
	if err != nil {
		return 0, normalizeRepoError(err)
	}
	return id, nil
}

// Get assembles the detail response from multiple repositories and applies visibility checks up front.
func (s *GameDetailService) Get(id int64, includeAll bool) (*GameDetail, error) {
	game, err := s.gamesRepo.GetByID(id)
	if err != nil {
		return nil, normalizeRepoError(err)
	}
	if !includeAll && game.Visibility == domain.GameVisibilityPrivate {
		// Public callers should observe private games as missing rather than leaking their existence.
		return nil, domain.ErrNotFound
	}

	allAssets, err := s.gamesRepo.ListAllAssets(id)
	if err != nil {
		return nil, err
	}
	assetsByType := make(map[string][]domain.GameAsset)
	for _, a := range allAssets {
		assetsByType[a.AssetType] = append(assetsByType[a.AssetType], a)
	}
	screenshots := assetsByType["screenshot"]
	videos := assetsByType["video"]
	covers := assetsByType["cover"]
	logos := assetsByType["logo"]
	banners := assetsByType["banner"]
	primarySeries, err := s.gamesRepo.GetSeriesMetadata(id)
	if err != nil {
		return nil, err
	}
	developers, err := s.gamesRepo.ListMetadata(domain.MetadataDevelopers, id)
	if err != nil {
		return nil, err
	}
	publishers, err := s.gamesRepo.ListMetadata(domain.MetadataPublishers, id)
	if err != nil {
		return nil, err
	}
	files, err := s.gameFilesRepo.ListByGameID(id)
	if err != nil {
		return nil, err
	}
	pendingIssues, err := getPendingIssueEvaluation(*game, int64(len(screenshots)), int64(len(logos)), int64(len(videos)), int64(len(files)), int64(len(developers)), int64(len(publishers)), s.reviewIssueOverridesRepo)
	if err != nil {
		return nil, err
	}

	return &GameDetail{
		Game:          game,
		PendingIssues: pendingIssues,
		// Videos are already sorted by the repository; the first item is the detail page's default playback target.
		PreviewVideos: videos,
		Screenshots:   screenshots,
		Covers:        covers,
		Banners:       banners,
		Logos:         logos,
		Series:        primarySeries,
		Developers:    emptyMetadata(developers),
		Publishers:    emptyMetadata(publishers),
		Files:         emptyFiles(files),
	}, nil
}

// ListPreviewVideos returns video assets for the given public ids in one round trip,
// filtering out private games unless includeAll is set. Games without videos are omitted.
func (s *GameDetailService) ListPreviewVideos(publicIDs []string, includeAll bool) ([]GamePreviewVideoBundle, error) {
	normalized := make([]string, 0, len(publicIDs))
	for _, id := range publicIDs {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, strings.ToLower(trimmed))
	}
	if len(normalized) == 0 {
		return nil, domain.ErrValidation
	}

	rows, err := s.gamesRepo.ListVideosByPublicIDs(normalized)
	if err != nil {
		return nil, err
	}

	bundlesByID := make(map[string]int, len(normalized))
	bundles := make([]GamePreviewVideoBundle, 0, len(normalized))
	for _, row := range rows {
		if !includeAll && row.Visibility == domain.GameVisibilityPrivate {
			continue
		}
		idx, ok := bundlesByID[row.PublicID]
		if !ok {
			idx = len(bundles)
			bundlesByID[row.PublicID] = idx
			bundles = append(bundles, GamePreviewVideoBundle{PublicID: row.PublicID})
		}
		bundles[idx].PreviewVideos = append(bundles[idx].PreviewVideos, row.GameAsset)
	}

	return bundles, nil
}
