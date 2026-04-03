package services

import (
	"strings"
	"time"

	"github.com/hao/game/internal/config"
	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/files"
	"github.com/hao/game/internal/repositories"
)

type GameAggregateService struct {
	aggregateRepo         *repositories.GameAggregateRepository
	metadataRepo          *repositories.MetadataRepository
	tagsRepo              *repositories.TagsRepository
	assetCleanupTasksRepo *repositories.AssetCleanupTasksRepository
	fileGuard             *files.Guard
	assetStore            *files.AssetStore
}

// NewGameAggregateService owns aggregate writes plus their follow-up filesystem/metadata cleanup.
// Read-side projections should stay in catalog/detail/timeline services even when they target the
// same underlying game rows.
func NewGameAggregateService(
	cfg config.Config,
	aggregateRepo *repositories.GameAggregateRepository,
	metadataRepo *repositories.MetadataRepository,
	tagsRepo *repositories.TagsRepository,
) *GameAggregateService {
	return &GameAggregateService{
		aggregateRepo:         aggregateRepo,
		metadataRepo:          metadataRepo,
		tagsRepo:              tagsRepo,
		assetCleanupTasksRepo: repositories.NewAssetCleanupTasksRepository(aggregateRepo.DB()),
		fileGuard:             files.NewGuard(cfg.PrimaryROMRoot),
		assetStore:            files.NewAssetStore(cfg.AssetsDir, cfg.Proxy, 30*time.Second),
	}
}

// ResolveGameID translates the public route id into the internal numeric id used by the aggregate repository.
func (s *GameAggregateService) ResolveGameID(publicID string) (int64, error) {
	id, err := s.aggregateRepo.ResolveIDByPublicID(publicID)
	if err != nil {
		return 0, normalizeRepoError(err)
	}
	return id, nil
}

// Create validates write input before delegating the actual insert to the aggregate repository.
func (s *GameAggregateService) Create(input domain.GameWriteInput) (*domain.Game, error) {
	trimmedInput, err := validateAndTrimGameInput(input, s.tagsRepo)
	if err != nil {
		return nil, err
	}
	return s.aggregateRepo.Create(trimmedInput)
}

// Update applies aggregate changes, then performs follow-up metadata and asset cleanup work.
func (s *GameAggregateService) Update(id int64, input domain.GameAggregateUpdateInput) (*domain.Game, []string, error) {
	trimmedInput, err := validateAndTrimGameAggregatePatchInput(input.Game, s.tagsRepo)
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
		// File paths are normalized against the configured ROM root before they are persisted,
		// so later launches and scans do not depend on user-provided relative path variants.
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
		})
	}

	for _, item := range input.Assets.DeleteAssets {
		switch strings.TrimSpace(item.AssetType) {
		case "cover", "banner", "screenshot", "video":
		default:
			return nil, nil, ErrValidation
		}
	}

	deletedAssetPaths, err := s.aggregateRepo.Update(id, domain.GameAggregateUpdateInput{
		Game: trimmedInput,
		Assets: domain.GameAggregateAssetsInput{
			Files:                    normalizedFiles,
			DeleteAssets:             input.Assets.DeleteAssets,
			ScreenshotOrderAssetUIDs: input.Assets.ScreenshotOrderAssetUIDs,
			VideoOrderAssetUIDs:      input.Assets.VideoOrderAssetUIDs,
		},
	})
	if err != nil {
		return nil, nil, normalizeRepoError(err)
	}

	if err := cleanupUnusedMetadata(s.metadataRepo); err != nil {
		return nil, nil, err
	}

	assetDeleteWarnings := make([]string, 0)
	for _, path := range deletedAssetPaths {
		// Best-effort deletion still records warnings so the write succeeds even if filesystem cleanup
		// needs to be retried asynchronously.
		warning, err := cleanupAssetPath(s.assetStore, s.assetCleanupTasksRepo, path, "games.update_aggregate")
		if err != nil {
			return nil, nil, err
		}
		if warning {
			assetDeleteWarnings = append(assetDeleteWarnings, path)
		}
	}

	game, err := s.aggregateRepo.GetByID(id)
	if err != nil {
		return nil, nil, normalizeRepoError(err)
	}

	return game, assetDeleteWarnings, nil
}

// Delete removes the game aggregate and then tries to clean up orphaned asset files and metadata rows.
func (s *GameAggregateService) Delete(id int64) (*GameDeleteResult, error) {
	deletedAssetPaths, deleted, err := s.aggregateRepo.Delete(id)
	if err != nil {
		return nil, err
	}
	if !deleted {
		return nil, ErrNotFound
	}

	warnings := make([]string, 0)
	for _, path := range deletedAssetPaths {
		warning, err := cleanupAssetPath(s.assetStore, s.assetCleanupTasksRepo, path, "games.delete")
		if err != nil {
			return nil, err
		}
		if warning {
			warnings = append(warnings, path)
		}
	}
	if err := cleanupUnusedMetadata(s.metadataRepo); err != nil {
		return nil, err
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
