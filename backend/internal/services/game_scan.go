package services

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/repositories"
)

type GameScanService struct {
	gamesRepo     *repositories.GamesRepository
	gameFilesRepo *repositories.GameFilesRepository
	romRoot       string
}

func NewGameScanService(
	gamesRepo *repositories.GamesRepository,
	gameFilesRepo *repositories.GameFilesRepository,
	romRoot string,
) *GameScanService {
	return &GameScanService{
		gamesRepo:     gamesRepo,
		gameFilesRepo: gameFilesRepo,
		romRoot:       romRoot,
	}
}

var versionPattern = regexp.MustCompile(`\s*v?\d+[\.\d]*`)
var buildPattern = regexp.MustCompile(`\s*Build\.\d+`)

func (s *GameScanService) extractTitle(filename string) string {
	title := strings.TrimSuffix(filename, filepath.Ext(filename))
	title = versionPattern.ReplaceAllString(title, "")
	title = buildPattern.ReplaceAllString(title, "")
	return strings.TrimSpace(title)
}

func (s *GameScanService) Scan() (*domain.ScanResult, error) {
	result := &domain.ScanResult{
		Details: make([]domain.ScanDetail, 0),
	}

	// 获取所有已存在的文件路径
	existingPaths, err := s.gameFilesRepo.ListAllPaths()
	if err != nil {
		return nil, err
	}

	// 扫描目录
	err = filepath.Walk(s.romRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".vhd" && ext != ".vhdx" {
			return nil
		}

		result.Total++

		// 计算相对路径
		relPath, err := filepath.Rel(s.romRoot, path)
		if err != nil {
			return nil
		}

		// 检查是否已存在
		if existingPaths[relPath] {
			result.Skipped++
			result.Details = append(result.Details, domain.ScanDetail{
				Title:  s.extractTitle(info.Name()),
				Status: "skipped",
			})
			return nil
		}

		// 提取标题
		title := s.extractTitle(info.Name())

		// 创建游戏
		game, err := s.gamesRepo.Create(domain.GameCreateInput{
			Title:      title,
			Visibility: "public",
		})
		if err != nil {
			result.Errors++
			result.Details = append(result.Details, domain.ScanDetail{
				Title:  title,
				Status: "error",
				Reason: err.Error(),
			})
			return nil
		}

		// 添加文件
		_, err = s.gamesRepo.UpdateAggregate(game.ID, domain.GameAggregateUpdateInput{
			Game: domain.GameAggregateCoreUpdateInput{
				GameCoreInput: domain.GameCoreInput{
					Title:      title,
					Visibility: "public",
				},
			},
			Assets: domain.GameAggregateAssetsInput{
				Files: []domain.GameFileUpsertInput{
					{FilePath: relPath},
				},
			},
		})
		if err != nil {
			result.Errors++
			result.Details = append(result.Details, domain.ScanDetail{
				Title:  title,
				Status: "error",
				Reason: err.Error(),
			})
			return nil
		}

		result.Created++
		existingPaths[relPath] = true
		result.Details = append(result.Details, domain.ScanDetail{
			Title:  title,
			Status: "created",
		})

		return nil
	})

	return result, err
}
