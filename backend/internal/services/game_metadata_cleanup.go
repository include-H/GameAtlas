package services

import (
	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/repositories"
)

// cleanupUnusedMetadata applies the product rule that game metadata is lightweight helper data,
// not standalone master data. Aggregate mutations may create it opportunistically, so writes must
// also prune unreferenced rows once the aggregate update finishes.
func cleanupUnusedMetadata(metadataRepo *repositories.MetadataRepository) error {
	if err := metadataRepo.DeleteUnusedSeries(); err != nil {
		return err
	}

	targets := []domain.MetadataType{
		domain.MetadataDevelopers,
		domain.MetadataPublishers,
	}

	for _, typ := range targets {
		if err := metadataRepo.DeleteUnused(typ); err != nil {
			return err
		}
	}

	return nil
}
