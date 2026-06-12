package services

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/hao/game/internal/domain"
)

func normalizeRepoError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrNotFound
	}
	return err
}

func emptyMetadata(items []domain.MetadataItem) []domain.MetadataItem {
	if items == nil {
		return []domain.MetadataItem{}
	}
	return items
}

func uniqueIDs(ids []int64) []int64 {
	if len(ids) == 0 {
		return []int64{}
	}

	seen := make(map[int64]struct{}, len(ids))
	result := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}

	return result
}

func emptyFiles(items []domain.GameFile) []domain.GameFile {
	if items == nil {
		return []domain.GameFile{}
	}
	return items
}

func trimStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
