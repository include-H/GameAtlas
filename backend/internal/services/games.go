package services

import (
	"database/sql"
	"errors"
	"math"
	"strings"

	"github.com/hao/game/internal/config"
	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/files"
	"github.com/hao/game/internal/repositories"
)

var ErrNotFound = errors.New("resource not found")
var ErrValidation = errors.New("validation error")

type GamesService struct {
	gamesRepo     *repositories.GamesRepository
	gameFilesRepo *repositories.GameFilesRepository
	fileGuard     *files.Guard
}

type GamesListResult struct {
	Games      []domain.Game
	Page       int
	Limit      int
	Total      int
	TotalPages int
}

type GameDetail struct {
	Game        *domain.Game
	Screenshots []domain.GameAsset
	Series      []domain.MetadataItem
	Platforms   []domain.MetadataItem
	Developers  []domain.MetadataItem
	Publishers  []domain.MetadataItem
	Files       []domain.GameFile
}

func NewGamesService(cfg config.Config, gamesRepo *repositories.GamesRepository, gameFilesRepo *repositories.GameFilesRepository) *GamesService {
	roots := cfg.AllowedRoots
	if len(roots) == 0 && strings.TrimSpace(cfg.PrimaryROMRoot) != "" {
		roots = append(roots, cfg.PrimaryROMRoot)
	}

	return &GamesService{
		gamesRepo:     gamesRepo,
		gameFilesRepo: gameFilesRepo,
		fileGuard:     files.NewGuard(roots),
	}
}

func (s *GamesService) List(params domain.GamesListParams) (*GamesListResult, error) {
	normalizeListParams(&params)
	games, total, err := s.gamesRepo.List(params)
	if err != nil {
		return nil, err
	}

	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(params.Limit)))
	}

	return &GamesListResult{
		Games:      games,
		Page:       params.Page,
		Limit:      params.Limit,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

func (s *GamesService) GetDetail(id int64) (*GameDetail, error) {
	game, err := s.gamesRepo.GetByID(id)
	if err != nil {
		return nil, normalizeRepoError(err)
	}

	screenshots, err := s.gamesRepo.ListScreenshots(id)
	if err != nil {
		return nil, err
	}
	series, err := s.gamesRepo.ListMetadata("series", "game_series", "series_id", id)
	if err != nil {
		return nil, err
	}
	platforms, err := s.gamesRepo.ListMetadata("platforms", "game_platforms", "platform_id", id)
	if err != nil {
		return nil, err
	}
	developers, err := s.gamesRepo.ListMetadata("developers", "game_developers", "developer_id", id)
	if err != nil {
		return nil, err
	}
	publishers, err := s.gamesRepo.ListMetadata("publishers", "game_publishers", "publisher_id", id)
	if err != nil {
		return nil, err
	}
	files, err := s.gameFilesRepo.ListByGameID(id)
	if err != nil {
		return nil, err
	}
	for index := range files {
		_ = s.refreshFileSize(id, &files[index])
	}

	return &GameDetail{
		Game:        game,
		Screenshots: screenshots,
		Series:      emptyMetadata(series),
		Platforms:   emptyMetadata(platforms),
		Developers:  emptyMetadata(developers),
		Publishers:  emptyMetadata(publishers),
		Files:       emptyFiles(files),
	}, nil
}

func (s *GamesService) Create(input domain.GameWriteInput) (*domain.Game, error) {
	if err := validateGameInput(input); err != nil {
		return nil, err
	}
	game, err := s.gamesRepo.Create(trimGameInput(input))
	if err != nil {
		return nil, err
	}
	return game, nil
}

func (s *GamesService) Update(id int64, input domain.GameWriteInput) (*domain.Game, error) {
	if err := validateGameInput(input); err != nil {
		return nil, err
	}
	game, err := s.gamesRepo.Update(id, trimGameInput(input))
	if err != nil {
		return nil, normalizeRepoError(err)
	}
	return game, nil
}

func (s *GamesService) Delete(id int64) error {
	deleted, err := s.gamesRepo.Delete(id)
	if err != nil {
		return err
	}
	if !deleted {
		return ErrNotFound
	}
	return nil
}

func validateGameInput(input domain.GameWriteInput) error {
	if strings.TrimSpace(input.Title) == "" {
		return ErrValidation
	}
	return nil
}

func trimGameInput(input domain.GameWriteInput) domain.GameWriteInput {
	input.Title = strings.TrimSpace(input.Title)
	input.TitleAlt = trimStringPtr(input.TitleAlt)
	input.Summary = trimStringPtr(input.Summary)
	input.ReleaseDate = trimStringPtr(input.ReleaseDate)
	input.Engine = trimStringPtr(input.Engine)
	input.CoverImage = trimStringPtr(input.CoverImage)
	input.BannerImage = trimStringPtr(input.BannerImage)
	return input
}

func normalizeListParams(params *domain.GamesListParams) {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.Limit <= 0 {
		params.Limit = 20
	}
	if params.Limit > 100 {
		params.Limit = 100
	}
	if params.Sort == "" {
		params.Sort = "updated_at"
	}
	if params.Order == "" {
		params.Order = "desc"
	}
}

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

func (s *GamesService) refreshFileSize(gameID int64, file *domain.GameFile) error {
	resolved, err := s.fileGuard.ValidateFile(file.FilePath)
	if err != nil {
		return nil
	}

	if file.SizeBytes != nil && *file.SizeBytes == resolved.SizeBytes {
		return nil
	}

	file.SizeBytes = &resolved.SizeBytes
	return s.gameFilesRepo.UpdateSizeBytes(gameID, file.ID, resolved.SizeBytes)
}
