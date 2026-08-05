package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/hao/game/internal/config"
	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/files"
	"github.com/hao/game/internal/repositories"
)

type GameAggregateService struct {
	gamesRepo             *repositories.GamesRepository
	metadataService       *MetadataService
	assetCleanupTasksRepo *repositories.AssetCleanupTasksRepository
	fileGuard             *files.Guard
	assetStore            *files.AssetStore
	catalogService        *GameCatalogService
}

// NewGameAggregateService owns aggregate writes plus their follow-up filesystem/metadata cleanup.
// Read-side projections should stay in catalog/detail/timeline services even when they target the
// same underlying game rows.
func NewGameAggregateService(
	cfg config.Config,
	gamesRepo *repositories.GamesRepository,
	metadataService *MetadataService,
	catalogService *GameCatalogService,
	assetCleanupTasksRepo *repositories.AssetCleanupTasksRepository,
) *GameAggregateService {
	return &GameAggregateService{
		gamesRepo:             gamesRepo,
		metadataService:       metadataService,
		assetCleanupTasksRepo: assetCleanupTasksRepo,
		fileGuard:             files.NewGuard(cfg.PrimaryROMRoot),
		assetStore:            files.NewAssetStore(cfg.AssetsDir),
		catalogService:        catalogService,
	}
}

// ResolveGameID translates the public route id into the internal numeric id used by the games repository.
func (s *GameAggregateService) ResolveGameID(publicID string) (int64, error) {
	id, err := s.gamesRepo.ResolveIDByPublicID(publicID)
	if err != nil {
		return 0, normalizeRepoError(err)
	}
	return id, nil
}

// Create stays intentionally minimal so quick-add cannot drift into aggregate-edit semantics.
func (s *GameAggregateService) Create(input domain.GameCreateInput) (*domain.Game, error) {
	trimmedInput, err := validateAndTrimGameCreateInput(input)
	if err != nil {
		return nil, err
	}
	game, err := s.gamesRepo.Create(trimmedInput)
	if err != nil {
		return nil, err
	}
	if s.catalogService != nil {
		s.catalogService.InvalidateStatsCache()
	}
	return game, nil
}

// Update applies aggregate changes, then performs follow-up metadata and asset cleanup work.
// New assets are moved from staging to permanent storage before the DB transaction.
// Old assets not in the submitted form are auto-deleted by the repository layer.
func (s *GameAggregateService) Update(id int64, input domain.GameAggregateUpdateInput) (*domain.Game, []string, error) {
	trimmedInput, err := validateAndTrimGameAggregateCoreUpdateInput(input.Game)
	if err != nil {
		return nil, nil, err
	}

	normalizedFiles := make([]domain.GameFileUpsertInput, 0, len(input.Assets.Files))
	for index, item := range input.Assets.Files {
		fileInput := domain.GameFileWriteInput{
			FilePath:  item.FilePath,
			Label:     item.Label,
			Notes:     item.Notes,
			SortOrder: item.SortOrder,
		}
		if err := validateGameFileInput(fileInput); err != nil {
			return nil, nil, err
		}

		trimmedFileInput := trimGameFileInput(fileInput)
		resolved, err := s.fileGuard.ValidateFile(trimmedFileInput.FilePath)
		if err != nil {
			return nil, nil, normalizeFileError(err)
		}
		normalizedFiles = append(normalizedFiles, domain.GameFileUpsertInput{
			ID:        item.ID,
			FilePath:  resolved.ResolvedPath,
			Label:     trimmedFileInput.Label,
			Notes:     trimmedFileInput.Notes,
			SortOrder: index,
			SizeBytes: &resolved.SizeBytes,
			SourceCreatedAt: func() *string {
				t := time.Unix(resolved.ModTime, 0).UTC().Format("2006-01-02 15:04:05")
				return &t
			}(),
		})
	}

	// Move staging files to permanent storage before DB write.
	game, gameErr := s.gamesRepo.GetByID(id)
	if gameErr != nil {
		return nil, nil, normalizeRepoError(gameErr)
	}
	for _, asset := range input.Assets.NewAssets {
		if _, err := s.assetStore.MoveToPermanent(asset.Path, game.PublicID); err != nil {
			return nil, nil, fmt.Errorf("move staging asset to permanent: %w", err)
		}
		// 预告片封面帧同样从 staging 移入正式目录；否则会被启动时的
		// CleanStaging（1 小时过期）清理掉。
		if asset.PosterPath != nil && strings.TrimSpace(*asset.PosterPath) != "" {
			if _, err := s.assetStore.MoveToPermanent(*asset.PosterPath, game.PublicID); err != nil {
				return nil, nil, fmt.Errorf("move staging poster to permanent: %w", err)
			}
		}
	}

	deletedAssetPaths, err := s.gamesRepo.UpdateAggregate(id, domain.GameAggregateUpdateInput{
		Game: trimmedInput,
		Assets: domain.GameAggregateAssetsInput{
			Files:                    normalizedFiles,
			ScreenshotOrderAssetUIDs: input.Assets.ScreenshotOrderAssetUIDs,
			VideoOrderAssetUIDs:      input.Assets.VideoOrderAssetUIDs,
			CoverOrderAssetUIDs:      input.Assets.CoverOrderAssetUIDs,
			LogoOrderAssetUIDs:       input.Assets.LogoOrderAssetUIDs,
			BannerOrderAssetUIDs:     input.Assets.BannerOrderAssetUIDs,
			LogoPositions:            input.Assets.LogoPositions,
			NewAssets:                input.Assets.NewAssets,
		},
	})
	if err != nil {
		return nil, nil, normalizeRepoError(err)
	}

	if err := cleanupUnusedMetadata(s.metadataService); err != nil {
		return nil, nil, err
	}

	assetDeleteWarnings := make([]string, 0)
	for _, path := range deletedAssetPaths {
		// 2026-08-05: keep physical files that are still referenced elsewhere
		// (other games or start-screen tiles); only the detached DB row is gone.
		referenced, err := s.gamesRepo.IsAssetPathReferenced(path)
		if err != nil {
			return nil, nil, err
		}
		if referenced {
			continue
		}
		warning, err := cleanupAssetPath(s.assetStore, s.assetCleanupTasksRepo, path, "games.update_aggregate")
		if err != nil {
			return nil, nil, err
		}
		if warning {
			assetDeleteWarnings = append(assetDeleteWarnings, path)
		}
	}

	game, err = s.gamesRepo.GetByID(id)
	if err != nil {
		return nil, nil, normalizeRepoError(err)
	}

	if s.catalogService != nil {
		s.catalogService.InvalidateStatsCache()
	}

	return game, assetDeleteWarnings, nil
}

// Delete removes the game aggregate and then tries to clean up orphaned asset files and metadata rows.
func (s *GameAggregateService) Delete(id int64) (*GameDeleteResult, error) {
	deletedAssetPaths, deleted, err := s.gamesRepo.Delete(id)
	if err != nil {
		return nil, err
	}
	if !deleted {
		return nil, domain.ErrNotFound
	}

	warnings := make([]string, 0)
	for _, path := range deletedAssetPaths {
		// 2026-04-04: keep asset deletion best-effort because the game row has already been removed,
		// and leftover files can be retried from cleanup tasks without reviving the deleted resource.
		// Impact: only asset file removal is deferred; the game stays deleted.
		referenced, err := s.gamesRepo.IsAssetPathReferenced(path)
		if err != nil {
			return nil, err
		}
		if referenced {
			continue
		}
		warning, err := cleanupAssetPath(s.assetStore, s.assetCleanupTasksRepo, path, "games.delete")
		if err != nil {
			return nil, err
		}
		if warning {
			warnings = append(warnings, path)
		}
	}
	if err := cleanupUnusedMetadata(s.metadataService); err != nil {
		return nil, err
	}

	if s.catalogService != nil {
		s.catalogService.InvalidateStatsCache()
	}

	return &GameDeleteResult{Warnings: warnings}, nil
}

// ProcessPendingAssetCleanup retries deferred asset deletions recorded during earlier write operations.
func (s *GameAggregateService) ProcessPendingAssetCleanup(limit int) (int, error) {
	tasks, err := s.assetCleanupTasksRepo.ListPending(limit)
	if err != nil {
		return 0, err
	}

	processed := 0
	for _, task := range tasks {
		if _, err := cleanupAssetPath(s.assetStore, s.assetCleanupTasksRepo, task.AssetPath, "asset_cleanup.retry"); err != nil {
			return processed, err
		}
		processed++
	}

	return processed, nil
}
