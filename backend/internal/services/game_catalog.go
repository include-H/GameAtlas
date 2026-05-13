package services

import (
	"math"
	"sync"
	"time"

	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/repositories"
)

type GameCatalogService struct {
	catalogRepo              *repositories.GameCatalogRepository
	reviewIssueOverridesRepo *repositories.ReviewIssueOverrideRepository
	statsCache               sync.Map // key: statsCacheKey -> cachedStatsEntry
}

type statsCacheKey struct {
	Visibility string
	IncludeAll bool
}

type cachedStatsEntry struct {
	stats    *domain.GameStats
	cachedAt time.Time
}

// NewGameCatalogService wires the read-only catalog boundary used by list, stats, and pending
// review summaries. Keep write workflows and timeline behavior out of this service.
func NewGameCatalogService(
	catalogRepo *repositories.GameCatalogRepository,
	reviewIssueOverridesRepo *repositories.ReviewIssueOverrideRepository,
) *GameCatalogService {
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
	// 2026-05-01: normalizeListParams only applies domain-level shaping such as visibility
	// and pending-day clamps. Transport/query defaults belong upstream, so list callers that
	// arrive here with bad sort/order values must fail fast instead of relying on repository fallback.
	if err := validateListParamsContract(params); err != nil {
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

	key := statsCacheKey{Visibility: params.Visibility, IncludeAll: params.IncludeAll}
	if cached, ok := s.statsCache.Load(key); ok {
		entry := cached.(cachedStatsEntry)
		if time.Since(entry.cachedAt) < 30*time.Second {
			return entry.stats, nil
		}
	}

	stats, err := s.catalogRepo.Stats(params)
	if err != nil {
		return nil, err
	}
	s.statsCache.Store(key, cachedStatsEntry{stats: stats, cachedAt: time.Now()})
	return stats, nil
}

// InvalidateStatsCache clears the stats cache. Call this after game create/update/delete.
func (s *GameCatalogService) InvalidateStatsCache() {
	s.statsCache = sync.Map{}
}
