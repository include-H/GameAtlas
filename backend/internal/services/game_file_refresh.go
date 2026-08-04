package services

import (
	"time"

	"github.com/hao/game/internal/files"
	"github.com/hao/game/internal/repositories"
)

type GameFileRefreshService struct {
	gameFilesRepo *repositories.GameFilesRepository
	fileGuard     *files.Guard
}

func NewGameFileRefreshService(
	gameFilesRepo *repositories.GameFilesRepository,
	fileGuard *files.Guard,
) *GameFileRefreshService {
	return &GameFileRefreshService{
		gameFilesRepo: gameFilesRepo,
		fileGuard:     fileGuard,
	}
}

type RefreshResult struct {
	Updated int `json:"updated"`
	Errors  int `json:"errors"`
}

// RefreshFileSizes 重新读取所有已登记文件的实际大小与修改时间并写回数据库，
// 覆盖文件被替换/增大后数据库中仍保留旧大小的情况。
func (s *GameFileRefreshService) RefreshFileSizes() (*RefreshResult, error) {
	result := &RefreshResult{}

	files, err := s.gameFilesRepo.ListAll()
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		resolved, err := s.fileGuard.ValidateFile(file.FilePath)
		if err != nil {
			result.Errors++
			continue
		}

		sourceCreatedAt := time.Unix(resolved.ModTime, 0).UTC().Format("2006-01-02 15:04:05")
		if err := s.gameFilesRepo.UpdateFileSizeAndDate(file.ID, resolved.SizeBytes, sourceCreatedAt); err != nil {
			result.Errors++
			continue
		}

		result.Updated++
	}

	return result, nil
}
