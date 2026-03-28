package services

import (
	"strings"
	"time"

	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/files"
)

func populateSourceCreatedAt(fileGuard *files.Guard, file *domain.GameFile) bool {
	if fileGuard == nil || file == nil {
		return false
	}

	if file.SourceCreatedAt != nil && strings.TrimSpace(*file.SourceCreatedAt) != "" {
		return false
	}

	resolved, err := fileGuard.ValidateFile(file.FilePath)
	if err != nil {
		return false
	}

	sourceCreatedAt, err := files.ReadSourceCreatedAt(resolved.ResolvedPath)
	if err != nil {
		return false
	}

	if sourceCreatedAt == nil {
		fallback := time.Unix(resolved.ModTime, 0).UTC().Format(time.RFC3339)
		sourceCreatedAt = &fallback
	}

	file.SourceCreatedAt = sourceCreatedAt
	return true
}

func populateSourceCreatedAtForFiles(fileGuard *files.Guard, items []domain.GameFile) []domain.GameFile {
	for index := range items {
		populateSourceCreatedAt(fileGuard, &items[index])
	}
	return items
}
