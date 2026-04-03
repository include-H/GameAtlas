package services

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/hao/game/internal/domain"
)

var ErrNotFound = errors.New("resource not found")
var ErrValidation = errors.New("validation error")

func normalizeRepoError(err error) error {
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, sqlxErrNotFound()) {
		return ErrNotFound
	}
	return err
}

func emptyMetadata(items []domain.MetadataItem) []domain.MetadataItem {
	if items == nil {
		return []domain.MetadataItem{}
	}
	return items
}

func emptyTags(items []domain.Tag) []domain.Tag {
	if items == nil {
		return []domain.Tag{}
	}
	return items
}

func groupGameTags(tags []domain.Tag) []domain.GameTagGroup {
	if len(tags) == 0 {
		return []domain.GameTagGroup{}
	}

	groups := make([]domain.GameTagGroup, 0)
	indexByGroupID := map[int64]int{}

	for _, tag := range tags {
		index, ok := indexByGroupID[tag.GroupID]
		if !ok {
			groups = append(groups, domain.GameTagGroup{
				ID:            tag.GroupID,
				Key:           tag.GroupKey,
				Name:          tag.GroupName,
				AllowMultiple: tag.GroupAllowMultiple,
				IsFilterable:  tag.GroupIsFilterable,
				Tags:          []domain.Tag{},
			})
			index = len(groups) - 1
			indexByGroupID[tag.GroupID] = index
		}
		groups[index].Tags = append(groups[index].Tags, tag)
	}

	return groups
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

func sqlxErrNotFound() error {
	return sql.ErrNoRows
}
