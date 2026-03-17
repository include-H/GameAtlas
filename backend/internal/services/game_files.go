package services

import (
	"errors"
	"strings"

	"github.com/hao/game/internal/config"
	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/files"
	"github.com/hao/game/internal/repositories"
)

var ErrForbiddenPath = errors.New("file path is outside allowed roots")
var ErrMissingFile = errors.New("registered file is unavailable")
var ErrInvalidFile = errors.New("registered path is not a file")
var ErrMissingConfig = errors.New("allowed library roots are not configured")

type GameFilesService struct {
	gamesRepo     *repositories.GamesRepository
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

func NewGameFilesService(cfg config.Config, gamesRepo *repositories.GamesRepository, gameFilesRepo *repositories.GameFilesRepository) *GameFilesService {
	roots := cfg.AllowedRoots
	if len(roots) == 0 && strings.TrimSpace(cfg.PrimaryROMRoot) != "" {
		roots = append(roots, cfg.PrimaryROMRoot)
	}

	return &GameFilesService{
		gamesRepo:     gamesRepo,
		gameFilesRepo: gameFilesRepo,
		fileGuard:     files.NewGuard(roots),
	}
}

func (s *GameFilesService) List(gameID int64) ([]domain.GameFile, error) {
	if _, err := s.gamesRepo.GetByID(gameID); err != nil {
		return nil, normalizeRepoError(err)
	}
	files, err := s.gameFilesRepo.ListByGameID(gameID)
	if err != nil {
		return nil, err
	}
	for index := range files {
		_ = s.refreshFileSize(gameID, &files[index])
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

func (s *GameFilesService) GetDownloadFile(gameID, fileID int64) (*DownloadFile, error) {
	if _, err := s.gamesRepo.GetByID(gameID); err != nil {
		return nil, normalizeRepoError(err)
	}

	file, err := s.gameFilesRepo.GetByID(gameID, fileID)
	if err != nil {
		return nil, normalizeRepoError(err)
	}

	resolved, err := s.fileGuard.ValidateFile(file.FilePath)
	if err != nil {
		return nil, normalizeFileError(err)
	}

	if file.SizeBytes == nil || *file.SizeBytes != resolved.SizeBytes {
		if err := s.gameFilesRepo.UpdateSizeBytes(gameID, fileID, resolved.SizeBytes); err != nil {
			return nil, err
		}
	}
	if err := s.gamesRepo.IncrementDownloads(gameID); err != nil {
		return nil, err
	}

	return &DownloadFile{
		GameID:       gameID,
		FileID:       fileID,
		ResolvedPath: resolved.ResolvedPath,
		SizeBytes:    resolved.SizeBytes,
		ModTime:      resolved.ModTime,
	}, nil
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

func (s *GameFilesService) refreshFileSize(gameID int64, file *domain.GameFile) error {
	resolved, err := s.fileGuard.ValidateFile(file.FilePath)
	if err != nil {
		return normalizeFileError(err)
	}

	if file.SizeBytes != nil && *file.SizeBytes == resolved.SizeBytes {
		return nil
	}

	file.SizeBytes = &resolved.SizeBytes
	return s.gameFilesRepo.UpdateSizeBytes(gameID, file.ID, resolved.SizeBytes)
}

func normalizeFileError(err error) error {
	switch {
	case errors.Is(err, files.ErrPathOutsideRoots):
		return ErrForbiddenPath
	case errors.Is(err, files.ErrFileNotFound):
		return ErrMissingFile
	case errors.Is(err, files.ErrNotAFile):
		return ErrInvalidFile
	case errors.Is(err, files.ErrNoAllowedRoots):
		return ErrMissingConfig
	default:
		return err
	}
}
