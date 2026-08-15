package services

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hao/game/internal/config"
	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/files"
)

type DirectoryService struct {
	guard *files.Guard
}

func NewDirectoryService(cfg config.Config) *DirectoryService {
	return &DirectoryService{
		guard: files.NewGuard(cfg.PrimaryROMRoot),
	}
}

func (s *DirectoryService) Default() (string, error) {
	path, err := s.guard.DefaultDirectory()
	if err != nil {
		return "", normalizeDirectoryError(err)
	}
	return path, nil
}

func (s *DirectoryService) List(path string) (*domain.DirectoryListResponse, error) {
	dir, err := s.guard.ValidateDirectory(path)
	if err != nil {
		return nil, normalizeDirectoryError(err)
	}

	entries, err := os.ReadDir(dir.ResolvedPath)
	if err != nil {
		return nil, err
	}

	items := make([]domain.DirectoryItem, 0, len(entries))
	skippedCount := 0
	for _, entry := range entries {
		itemPath := filepath.Join(dir.ResolvedPath, entry.Name())
		info, statErr := entry.Info()
		if statErr != nil {
			skippedCount++
			continue
		}

		isDir := info.IsDir()
		var sizeBytes *int64
		if !isDir && info.Mode().IsRegular() {
			size := info.Size()
			sizeBytes = &size
		}

		items = append(items, domain.DirectoryItem{
			Name:        entry.Name(),
			Path:        itemPath,
			IsDirectory: isDir,
			SizeBytes:   sizeBytes,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].IsDirectory != items[j].IsDirectory {
			return items[i].IsDirectory
		}
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})

	return &domain.DirectoryListResponse{
		CurrentPath:  dir.ResolvedPath,
		ParentPath:   s.guard.ParentDirectory(dir.ResolvedPath),
		Items:        items,
		Incomplete:   skippedCount > 0,
		SkippedCount: skippedCount,
	}, nil
}

const maxSearchResults = 100

type SearchResult struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	IsDirectory bool   `json:"is_directory"`
	SizeBytes   *int64 `json:"size_bytes,omitempty"`
	ParentPath  string `json:"parent_path"`
}

func (s *DirectoryService) Search(query string, fromPath string) ([]SearchResult, error) {
	// Validate and resolve the starting directory
	dir, err := s.guard.ValidateDirectory(fromPath)
	if err != nil {
		return nil, normalizeDirectoryError(err)
	}

	normalizedQuery := files.NormalizeForSearch(query)
	if normalizedQuery == "" {
		return []SearchResult{}, nil
	}

	results := make([]SearchResult, 0)
	err = filepath.Walk(dir.ResolvedPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip inaccessible entries
		}

		// Skip the root directory itself
		if path == dir.ResolvedPath {
			return nil
		}

		// Limit results
		if len(results) >= maxSearchResults {
			return filepath.SkipAll
		}

		name := info.Name()
		normalizedName := files.NormalizeForSearch(name)

		// Check similarity
		if files.FuzzyMatch(normalizedQuery, normalizedName) {
			var sizeBytes *int64
			if !info.IsDir() && info.Mode().IsRegular() {
				size := info.Size()
				sizeBytes = &size
			}

			parent := filepath.Dir(path)

			results = append(results, SearchResult{
				Name:        name,
				Path:        path,
				IsDirectory: info.IsDir(),
				SizeBytes:   sizeBytes,
				ParentPath:  parent,
			})
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return results, nil
}

func normalizeDirectoryError(err error) error {
	switch {
	case err == nil:
		return nil
	case err == files.ErrNoPrimaryRoot:
		return domain.ErrMissingConfig
	case err == files.ErrPathOutsideRoot:
		return domain.ErrForbiddenPath
	case err == files.ErrFileNotFound:
		return domain.ErrNotFound
	case err == files.ErrNotAFile:
		return domain.ErrValidation
	default:
		return err
	}
}
