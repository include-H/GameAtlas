package services

import (
	"math"

	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/repositories"
)

type GameCatalogService struct {
	catalogRepo              *repositories.GameCatalogRepository
	reviewIssueOverridesRepo *repositories.ReviewIssueOverrideRepository
}

// NewGameCatalogService wires the read-only catalog boundary used by list, stats, and pending
// review summaries. Keep write workflows and timeline behavior out of this service.
func NewGameCatalogService(catalogRepo *repositories.GameCatalogRepository, reviewIssueOverridesRepo *repositories.ReviewIssueOverrideRepository) *GameCatalogService {
	return &GameCatalogService{
		catalogRepo:              catalogRepo,
		reviewIssueOverridesRepo: reviewIssueOverridesRepo,
	}
}

// List returns the paginated game list and, when requested, pending-issue aggregates for the current filter.
func (s *GameCatalogService) List(params domain.GamesListParams) (*GamesListResult, error) {
	if err := normalizeListParams(&params); err != nil {
		return nil, err
	}
	games, total, err := s.catalogRepo.List(params)
	if err != nil {
		return nil, err
	}

	var pendingIssueCounts *domain.PendingIssueCountSummary
	if params.PendingOnly {
		// 2026-04-04: keep pending group aggregation aligned with the current queue filters
		// across the pending workbench badges; do not narrow it by the selected issue key.
		counts, err := s.catalogRepo.CountPendingGroups(params)
		if err != nil {
			return nil, err
		}
		pendingIssueCounts = &domain.PendingIssueCountSummary{
			Groups: map[domain.PendingIssueKey]int{
				domain.PendingIssueMissingAssets:   counts.MissingAssets,
				domain.PendingIssueMissingWiki:     counts.MissingWiki,
				domain.PendingIssueMissingFiles:    counts.MissingFiles,
				domain.PendingIssueMissingMetadata: counts.MissingMetadata,
			},
			IgnoredTotal: counts.IgnoredTotal,
		}
	}

	if err := attachPendingIssues(games, s.reviewIssueOverridesRepo); err != nil {
		return nil, err
	}

	totalPages := 0
	if total > 0 {
		// normalizeListParams guarantees a positive limit, so we can safely derive the last page here.
		totalPages = int(math.Ceil(float64(total) / float64(params.Limit)))
	}

	return &GamesListResult{
		Games:              games,
		Page:               params.Page,
		Limit:              params.Limit,
		Total:              total,
		TotalPages:         totalPages,
		PendingIssueCounts: pendingIssueCounts,
	}, nil
}

// Stats returns summary counters for the same filter shape used by the catalog list.
func (s *GameCatalogService) Stats(params domain.GamesListParams) (*domain.GameStats, error) {
	if err := normalizeListParams(&params); err != nil {
		return nil, err
	}
	return s.catalogRepo.Stats(params)
}
