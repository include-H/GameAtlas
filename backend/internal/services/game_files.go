package services

import (
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hao/game/internal/config"
	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/files"
	"github.com/hao/game/internal/repositories"
)

type gameFilesGameRepository interface {
	ResolveIDByPublicID(publicID string) (int64, error)
	GetByID(id int64) (*domain.Game, error)
	IncrementDownloads(id int64) error
}

const downloadRecordWindow = 10 * time.Minute

type GameFilesService struct {
	gamesRepo     gameFilesGameRepository
	gameFilesRepo *repositories.GameFilesRepository
	fileGuard     *files.Guard
	// Download stats only need lightweight single-process dedupe for this app's
	// main use case: preventing accidental double-counts from repeated clicks by
	// the same person. This is intentionally in-memory and approximate, not a
	// cross-restart or multi-instance guarantee.
	downloadDedupeMu sync.Mutex
	downloadDedupe   map[string]time.Time
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
		gamesRepo:      gamesRepo,
		gameFilesRepo:  gameFilesRepo,
		fileGuard:      files.NewGuard(cfg.PrimaryROMRoot),
		downloadDedupe: make(map[string]time.Time),
	}
}

func (s *GameFilesService) ResolveGameID(publicID string) (int64, error) {
	id, err := s.gamesRepo.ResolveIDByPublicID(publicID)
	if err != nil {
		return 0, normalizeRepoError(err)
	}
	return id, nil
}

func (s *GameFilesService) GetDownloadFile(gameID, fileID int64, includeAll bool) (*DownloadFile, error) {
	game, err := s.gamesRepo.GetByID(gameID)
	if err != nil {
		return nil, normalizeRepoError(err)
	}
	if !includeAll && game.Visibility == domain.GameVisibilityPrivate {
		return nil, domain.ErrNotFound
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

// ShouldRecordDownload returns true if this download should be counted, using an
// in-memory time-window dedupe keyed by source + game + file. Repeated requests
// within the window are suppressed to absorb accidental double-clicks.
func (s *GameFilesService) ShouldRecordDownload(sourceKey string, gameID, fileID int64) bool {
	s.downloadDedupeMu.Lock()
	defer s.downloadDedupeMu.Unlock()

	now := time.Now().UTC()

	// Best-effort cleanup for the in-memory dedupe window. We do not persist this
	// state because the goal is only to absorb local click bursts, not to enforce
	// stable rate limiting semantics across process restarts or deployments.
	for key, expiresAt := range s.downloadDedupe {
		if !expiresAt.After(now) {
			delete(s.downloadDedupe, key)
		}
	}
	if len(s.downloadDedupe) > 10000 {
		s.downloadDedupe = make(map[string]time.Time)
	}

	recordKey := sourceKey + ":" + strconv.FormatInt(gameID, 10) + ":" + strconv.FormatInt(fileID, 10)
	if expiresAt, exists := s.downloadDedupe[recordKey]; exists && expiresAt.After(now) {
		return false
	}

	s.downloadDedupe[recordKey] = now.Add(downloadRecordWindow)
	return true
}

func (s *GameFilesService) RecordDownload(gameID, fileID int64, includeAll bool) error {
	game, err := s.gamesRepo.GetByID(gameID)
	if err != nil {
		return normalizeRepoError(err)
	}
	if !includeAll && game.Visibility == domain.GameVisibilityPrivate {
		return domain.ErrNotFound
	}

	if _, err := s.gameFilesRepo.GetByID(gameID, fileID); err != nil {
		return normalizeRepoError(err)
	}

	return s.gamesRepo.IncrementDownloads(gameID)
}

func validateGameFileInput(input domain.GameFileWriteInput) error {
	if strings.TrimSpace(input.FilePath) == "" {
		return domain.ErrValidation
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
		return domain.ErrForbiddenPath
	case errors.Is(err, files.ErrFileNotFound):
		return domain.ErrMissingFile
	case errors.Is(err, files.ErrNotAFile):
		return domain.ErrInvalidFile
	case errors.Is(err, files.ErrNoPrimaryRoot):
		return domain.ErrMissingConfig
	default:
		return err
	}
}
