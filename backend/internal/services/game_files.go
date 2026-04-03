package services

import (
	"errors"
	"strings"

	"github.com/hao/game/internal/config"
	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/files"
	"github.com/hao/game/internal/repositories"
)

var ErrForbiddenPath = errors.New("file path is outside primary ROM root")
var ErrMissingFile = errors.New("registered file is unavailable")
var ErrInvalidFile = errors.New("registered path is not a file")
var ErrMissingConfig = errors.New("PRIMARY_ROM_ROOT is not configured")

type gameFilesGameRepository interface {
	ResolveIDByPublicID(publicID string) (int64, error)
	GetByID(id int64) (*domain.Game, error)
	IncrementDownloads(id int64) error
}

type GameFilesService struct {
	gamesRepo     gameFilesGameRepository
	gameFilesRepo *repositories.GameFilesRepository
	fileGuard     *files.Guard
}

type DownloadFile struct {
	GameID       int64
	FileID       int64
	ResolvedPath string
	SizeBytes    int64
	ModTime      int64
}

func NewGameFilesService(cfg config.Config, gamesRepo gameFilesGameRepository, gameFilesRepo *repositories.GameFilesRepository) *GameFilesService {
	return &GameFilesService{
		gamesRepo:     gamesRepo,
		gameFilesRepo: gameFilesRepo,
		fileGuard:     files.NewGuard(cfg.PrimaryROMRoot),
	}
}

func (s *GameFilesService) ResolveGameID(publicID string) (int64, error) {
	id, err := s.gamesRepo.ResolveIDByPublicID(publicID)
	if err != nil {
		return 0, normalizeRepoError(err)
	}
	return id, nil
}

func (s *GameFilesService) List(gameID int64, includeAll bool) ([]domain.GameFile, error) {
	game, err := s.gamesRepo.GetByID(gameID)
	if err != nil {
		return nil, normalizeRepoError(err)
	}
	if !includeAll && game.Visibility == domain.GameVisibilityPrivate {
		return nil, ErrNotFound
	}
	files, err := s.gameFilesRepo.ListByGameID(gameID)
	if err != nil {
		return nil, err
	}
	if files == nil {
		return []domain.GameFile{}, nil
	}
	return files, nil
}

func (s *GameFilesService) Create(gameID int64, input domain.GameFileWriteInput) (*domain.GameFile, error) {
	if _, err := s.gamesRepo.GetByID(gameID); err != nil {
		return nil, normalizeRepoError(err)
	}
	if err := validateGameFileInput(input); err != nil {
		return nil, err
	}
	input = trimGameFileInput(input)
	resolved, err := s.fileGuard.ValidateFile(input.FilePath)
	if err != nil {
		return nil, normalizeFileError(err)
	}
	input.FilePath = resolved.ResolvedPath

	file, err := s.gameFilesRepo.Create(gameID, input)
	if err != nil {
		return nil, err
	}

	file.SizeBytes = &resolved.SizeBytes
	if err := s.gameFilesRepo.UpdateSizeBytes(gameID, file.ID, resolved.SizeBytes); err != nil {
		return nil, err
	}
	if populateSourceCreatedAt(s.fileGuard, file) {
		if err := s.gameFilesRepo.UpdateSourceCreatedAt(gameID, file.ID, file.SourceCreatedAt); err != nil {
			return nil, err
		}
	}

	return file, nil
}

func (s *GameFilesService) Update(gameID, fileID int64, input domain.GameFileWriteInput) (*domain.GameFile, error) {
	if _, err := s.gamesRepo.GetByID(gameID); err != nil {
		return nil, normalizeRepoError(err)
	}
	if err := validateGameFileInput(input); err != nil {
		return nil, err
	}
	input = trimGameFileInput(input)
	resolved, err := s.fileGuard.ValidateFile(input.FilePath)
	if err != nil {
		return nil, normalizeFileError(err)
	}
	input.FilePath = resolved.ResolvedPath

	file, err := s.gameFilesRepo.Update(gameID, fileID, input)
	if err != nil {
		return nil, normalizeRepoError(err)
	}

	file.SizeBytes = &resolved.SizeBytes
	if err := s.gameFilesRepo.UpdateSizeBytes(gameID, fileID, resolved.SizeBytes); err != nil {
		return nil, err
	}
	file.SourceCreatedAt = nil
	if populateSourceCreatedAt(s.fileGuard, file) {
		if err := s.gameFilesRepo.UpdateSourceCreatedAt(gameID, fileID, file.SourceCreatedAt); err != nil {
			return nil, err
		}
	}

	return file, nil
}

func (s *GameFilesService) Delete(gameID, fileID int64) error {
	if _, err := s.gamesRepo.GetByID(gameID); err != nil {
		return normalizeRepoError(err)
	}
	deleted, err := s.gameFilesRepo.Delete(gameID, fileID)
	if err != nil {
		return err
	}
	if !deleted {
		return ErrNotFound
	}
	return nil
}

func (s *GameFilesService) GetDownloadFile(gameID, fileID int64, includeAll bool) (*DownloadFile, error) {
	game, err := s.gamesRepo.GetByID(gameID)
	if err != nil {
		return nil, normalizeRepoError(err)
	}
	if !includeAll && game.Visibility == domain.GameVisibilityPrivate {
		return nil, ErrNotFound
	}

	file, err := s.gameFilesRepo.GetByID(gameID, fileID)
	if err != nil {
		return nil, normalizeRepoError(err)
	}

	resolved, err := s.fileGuard.ValidateFile(file.FilePath)
	if err != nil {
		return nil, normalizeFileError(err)
	}

	return &DownloadFile{
		GameID:       gameID,
		FileID:       fileID,
		ResolvedPath: resolved.ResolvedPath,
		SizeBytes:    resolved.SizeBytes,
		ModTime:      resolved.ModTime,
	}, nil
}

func (s *GameFilesService) RecordDownload(gameID, fileID int64, includeAll bool) error {
	game, err := s.gamesRepo.GetByID(gameID)
	if err != nil {
		return normalizeRepoError(err)
	}
	if !includeAll && game.Visibility == domain.GameVisibilityPrivate {
		return ErrNotFound
	}

	if _, err := s.gameFilesRepo.GetByID(gameID, fileID); err != nil {
		return normalizeRepoError(err)
	}

	return s.gamesRepo.IncrementDownloads(gameID)
}

func validateGameFileInput(input domain.GameFileWriteInput) error {
	if strings.TrimSpace(input.FilePath) == "" {
		return ErrValidation
	}
	return nil
}

func trimGameFileInput(input domain.GameFileWriteInput) domain.GameFileWriteInput {
	input.FilePath = strings.TrimSpace(input.FilePath)
	input.Label = trimStringPtr(input.Label)
	input.Notes = trimStringPtr(input.Notes)
	return input
}

func normalizeFileError(err error) error {
	switch {
	case errors.Is(err, files.ErrPathOutsideRoot):
		return ErrForbiddenPath
	case errors.Is(err, files.ErrFileNotFound):
		return ErrMissingFile
	case errors.Is(err, files.ErrNotAFile):
		return ErrInvalidFile
	case errors.Is(err, files.ErrNoPrimaryRoot):
		return ErrMissingConfig
	default:
		return err
	}
}
