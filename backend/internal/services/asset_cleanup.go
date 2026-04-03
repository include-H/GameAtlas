package services

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/hao/game/internal/files"
	"github.com/hao/game/internal/repositories"
)

func cleanupAssetPath(
	store *files.AssetStore,
	tasksRepo *repositories.AssetCleanupTasksRepository,
	assetPath string,
	source string,
) (bool, error) {
	trimmedPath := strings.TrimSpace(assetPath)
	if trimmedPath == "" {
		return false, nil
	}

	err := store.DeleteAsset(trimmedPath)
	if err == nil || errors.Is(err, os.ErrNotExist) {
		if tasksRepo != nil {
			if cleanupErr := tasksRepo.DeleteByPath(trimmedPath); cleanupErr != nil {
				log.Printf("%s: failed to clear cleanup task for %s: %v", source, trimmedPath, cleanupErr)
			}
		}
		return false, nil
	}

	if tasksRepo == nil {
		return false, fmt.Errorf("delete asset file %s: %w", trimmedPath, err)
	}
	if enqueueErr := tasksRepo.Enqueue(trimmedPath, source, err.Error()); enqueueErr != nil {
		return false, fmt.Errorf("enqueue cleanup task for %s: %w", trimmedPath, enqueueErr)
	}

	log.Printf("%s: queued asset cleanup task for %s after delete failure: %v", source, trimmedPath, err)
	return true, nil
}
