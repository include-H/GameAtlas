package services

import "github.com/hao/game/internal/repositories"

// cleanupUnusedMetadata applies the product rule that game metadata is lightweight helper data,
// not standalone master data. Aggregate mutations may create it opportunistically, so writes must
// also prune unreferenced rows once the aggregate update finishes.
func cleanupUnusedMetadata(metadataRepo *repositories.MetadataRepository) error {
	if err := metadataRepo.DeleteUnusedSeries(); err != nil {
		return err
	}

	targets := []struct {
		table      string
		joinTable  string
		joinColumn string
	}{
		{table: "platforms", joinTable: "game_platforms", joinColumn: "platform_id"},
		{table: "developers", joinTable: "game_developers", joinColumn: "developer_id"},
		{table: "publishers", joinTable: "game_publishers", joinColumn: "publisher_id"},
	}

	for _, target := range targets {
		if err := metadataRepo.DeleteUnused(target.table, target.joinTable, target.joinColumn); err != nil {
			return err
		}
	}

	return nil
}
