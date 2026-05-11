package services

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"mime/multipart"
	"time"

	"github.com/hao/game/internal/config"
	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/files"
	"github.com/hao/game/internal/repositories"
	"github.com/jmoiron/sqlx"
)

type assetGameRepository interface {
	GetByID(id int64) (*domain.Game, error)
	DB() *sqlx.DB
}

type AssetsService struct {
	gamesRepo             assetGameRepository
	assetsRepo            *repositories.AssetsRepository
	assetCleanupTasksRepo *repositories.AssetCleanupTasksRepository
	store                 *files.AssetStore
}

type UploadResult struct {
	Path     string
	AssetUID string
}

func NewAssetsService(cfg config.Config, gamesRepo assetGameRepository, assetsRepo *repositories.AssetsRepository) *AssetsService {
	return &AssetsService{
		gamesRepo:             gamesRepo,
		assetsRepo:            assetsRepo,
		assetCleanupTasksRepo: repositories.NewAssetCleanupTasksRepository(gamesRepo.DB()),
		store:                 files.NewAssetStore(cfg.AssetsDir, cfg.Proxy, 30*time.Second),
	}
}

// 2026-05-09: Upload accepts *multipart.FileHeader rather than io.Reader because
// the caller (the single HTTP handler) always has one, and the method needs both
// the stream and the content-type from the header. For a single-process app with
// one upload entrypoint, extracting an io.Reader interface adds indirection
// without practical testability gain.
func (s *AssetsService) Upload(gameID int64, assetType string, header *multipart.FileHeader, sortOrder int) (*UploadResult, error) {
	game, err := s.gamesRepo.GetByID(gameID)
	if err != nil {
		return nil, normalizeRepoError(err)
	}

	src, err := header.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	contentType := header.Header.Get("Content-Type")
	assetUID, assetName := allocateAssetIdentity(assetType)
	path, err := s.store.SaveToStaging(game.PublicID, assetType, assetName, src, contentType)
	if err != nil {
		return nil, normalizeAssetError(err)
	}

	return &UploadResult{Path: path, AssetUID: assetUID}, nil
}

func (s *AssetsService) ApplyRemoteAsset(gameID int64, assetType string, remoteURL string, sortOrder int) (string, error) {
	game, err := s.gamesRepo.GetByID(gameID)
	if err != nil {
		return "", normalizeRepoError(err)
	}
	assetUID, assetName := allocateAssetIdentity(assetType)
	path, err := s.store.DownloadRemoteToStaging(game.PublicID, assetType, assetName, remoteURL)
	if err != nil {
		return "", normalizeAssetError(err)
	}
	_ = assetUID
	return path, nil
}

func (s *AssetsService) ApplyRawAsset(gameID int64, assetType string, content []byte, contentType string, sortOrder int) (string, error) {
	game, err := s.gamesRepo.GetByID(gameID)
	if err != nil {
		return "", normalizeRepoError(err)
	}
	assetUID, assetName := allocateAssetIdentity(assetType)
	path, err := s.store.SaveToStaging(game.PublicID, assetType, assetName, bytes.NewReader(content), contentType)
	if err != nil {
		return "", normalizeAssetError(err)
	}
	_ = assetUID
	return path, nil
}

// MoveStagingToPermanent moves an asset file from the staging directory to
// the permanent game-specific directory. If the file is already in the permanent
// location, it returns the path as-is.
func (s *AssetsService) MoveStagingToPermanent(gamePublicID string, assetPath string) (string, error) {
	return s.store.MoveToPermanent(assetPath, gamePublicID)
}

func newAssetUID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		now := time.Now().UnixNano()
		return fmt.Sprintf(
			"a%07x-%04x-4%03x-a%03x-%012x",
			now&0x0fffffff,
			now&0xffff,
			now&0x0fff,
			now&0x0fff,
			now&0x0fffffffffff,
		)
	}

	// UUIDv4 bits.
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80
	hexText := hex.EncodeToString(buf)
	return fmt.Sprintf(
		"%s-%s-%s-%s-%s",
		hexText[0:8],
		hexText[8:12],
		hexText[12:16],
		hexText[16:20],
		hexText[20:32],
	)
}

func allocateAssetIdentity(assetType string) (string, string) {
	switch assetType {
	case "screenshot", "video", "cover", "logo":
		uid := newAssetUID()
		return uid, uid
	case "banner":
		uid := newAssetUID()
		return uid, uid
	default:
		return "", newAssetUID()
	}
}

func normalizeAssetError(err error) error {
	switch {
	case errors.Is(err, files.ErrInvalidImageType):
		return ErrValidation
	case errors.Is(err, files.ErrInvalidAssetName):
		return ErrValidation
	case errors.Is(err, files.ErrInvalidRemoteURL), errors.Is(err, files.ErrBlockedRemoteURL):
		return ErrValidation
	default:
		return err
	}
}
