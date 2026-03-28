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
	for _, entry := range entries {
		itemPath := filepath.Join(dir.ResolvedPath, entry.Name())
		info, statErr := entry.Info()
		if statErr != nil {
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
		CurrentPath: dir.ResolvedPath,
		ParentPath:  s.guard.ParentDirectory(dir.ResolvedPath),
		Items:       items,
	}, nil
}

func normalizeDirectoryError(err error) error {
	switch {
	case err == nil:
		return nil
	case err == files.ErrNoPrimaryRoot:
		return ErrMissingConfig
	case err == files.ErrPathOutsideRoot:
		return ErrForbiddenPath
	case err == files.ErrFileNotFound:
		return ErrNotFound
	case err == files.ErrNotAFile:
		return ErrValidation
	default:
		return err
	}
}
