package services

import (
	"github.com/hao/game/internal/domain"
)

// cleanupUnusedMetadata applies the product rule that game metadata is lightweight helper data,
// not standalone master data. Aggregate mutations may create it opportunistically, so writes must
// also prune unreferenced rows once the aggregate update finishes.
func cleanupUnusedMetadata(metadataService *MetadataService) error {
	return metadataService.CleanupUnusedGameMetadata()
}

func (s *MetadataService) CleanupUnusedGameMetadata() error {
	if err := s.repo.DeleteUnusedSeries(); err != nil {
		return err
	}

	targets := []domain.MetadataType{
		domain.MetadataDevelopers,
		domain.MetadataPublishers,
	}

	for _, typ := range targets {
		if err := s.repo.DeleteUnused(typ); err != nil {
			return err
		}
	}

	return nil
}
