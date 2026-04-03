package services

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"mime/multipart"
	"strings"
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
	AssetID  *int64
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
	path, err := s.store.SaveUploadedAsset(game.PublicID, assetType, assetName, src, contentType)
	if err != nil {
		return nil, normalizeAssetError(err)
	}

	asset, err := s.persistAssetPath(gameID, assetType, assetUID, path, sortOrder)
	if err != nil {
		if cleanupErr := s.cleanupPersistFailedAsset(path, "assets.upload"); cleanupErr != nil {
			return nil, cleanupErr
		}
		return nil, err
	}

	result := &UploadResult{Path: path}
	if asset != nil {
		result.AssetUID = asset.AssetUID
		result.AssetID = &asset.ID
	}
	return result, nil
}

func (s *AssetsService) Delete(input domain.DeleteAssetInput) error {
	if _, err := s.gamesRepo.GetByID(input.GameID); err != nil {
		return normalizeRepoError(err)
	}

	assetType := strings.TrimSpace(input.AssetType)
	assetPath := strings.TrimSpace(input.Path)
	if assetType == "" || assetPath == "" {
		return ErrValidation
	}

	switch assetType {
	case "cover":
		if err := s.assetsRepo.UpdateGameImage(input.GameID, "cover_image", nil); err != nil {
			return normalizeRepoError(err)
		}
	case "banner":
		if err := s.assetsRepo.UpdateGameImage(input.GameID, "banner_image", nil); err != nil {
			return normalizeRepoError(err)
		}
	case "screenshot":
		if strings.TrimSpace(input.AssetUID) != "" {
			asset, err := s.assetsRepo.DeleteScreenshotByUID(input.GameID, strings.TrimSpace(input.AssetUID))
			if err != nil {
				return normalizeRepoError(err)
			}
			assetPath = asset.Path
		} else if input.AssetID != nil && *input.AssetID > 0 {
			asset, err := s.assetsRepo.DeleteScreenshotByID(input.GameID, *input.AssetID)
			if err != nil {
				return normalizeRepoError(err)
			}
			assetPath = asset.Path
		} else {
			deleted, err := s.assetsRepo.DeleteScreenshot(input.GameID, assetPath)
			if err != nil {
				return err
			}
			if !deleted {
				return ErrNotFound
			}
		}
	case "video":
		if strings.TrimSpace(input.AssetUID) != "" {
			asset, err := s.assetsRepo.DeleteAssetByUID(input.GameID, strings.TrimSpace(input.AssetUID), "video")
			if err != nil {
				return normalizeRepoError(err)
			}
			assetPath = asset.Path
		} else {
			deleted, err := s.assetsRepo.DeleteAssetByPath(input.GameID, "video", assetPath)
			if err != nil {
				return err
			}
			if !deleted {
				return ErrNotFound
			}
		}
	default:
		return ErrValidation
	}

	if _, err := cleanupAssetPath(s.store, s.assetCleanupTasksRepo, assetPath, "assets.delete"); err != nil {
		return err
	}
	return nil
}

func (s *AssetsService) ApplyRemoteAsset(gameID int64, assetType string, remoteURL string, sortOrder int) (string, error) {
	game, err := s.gamesRepo.GetByID(gameID)
	if err != nil {
		return "", normalizeRepoError(err)
	}
	assetUID, assetName := allocateAssetIdentity(assetType)
	path, err := s.store.DownloadRemoteAsset(game.PublicID, assetType, assetName, remoteURL)
	if err != nil {
		return "", normalizeAssetError(err)
	}
	_, err = s.persistAssetPath(gameID, assetType, assetUID, path, sortOrder)
	if err != nil {
		if cleanupErr := s.cleanupPersistFailedAsset(path, "assets.apply_remote"); cleanupErr != nil {
			return "", cleanupErr
		}
		return "", err
	}
	return path, nil
}

func (s *AssetsService) ApplyRawAsset(gameID int64, assetType string, content []byte, contentType string, sortOrder int) (string, error) {
	game, err := s.gamesRepo.GetByID(gameID)
	if err != nil {
		return "", normalizeRepoError(err)
	}
	assetUID, assetName := allocateAssetIdentity(assetType)
	path, err := s.store.SaveUploadedAsset(game.PublicID, assetType, assetName, bytes.NewReader(content), contentType)
	if err != nil {
		return "", normalizeAssetError(err)
	}
	_, err = s.persistAssetPath(gameID, assetType, assetUID, path, sortOrder)
	if err != nil {
		if cleanupErr := s.cleanupPersistFailedAsset(path, "assets.apply_raw"); cleanupErr != nil {
			return "", cleanupErr
		}
		return "", err
	}
	return path, nil
}

func (s *AssetsService) cleanupPersistFailedAsset(path string, source string) error {
	_, err := cleanupAssetPath(s.store, s.assetCleanupTasksRepo, path, source)
	return err
}

func (s *AssetsService) persistAssetPath(gameID int64, assetType string, assetUID string, path string, sortOrder int) (*domain.GameAsset, error) {
	switch assetType {
	case "cover":
		return nil, s.assetsRepo.UpdateGameImage(gameID, "cover_image", &path)
	case "banner":
		return nil, s.assetsRepo.UpdateGameImage(gameID, "banner_image", &path)
	case "screenshot":
		asset, err := s.assetsRepo.AddScreenshot(gameID, assetUID, path, sortOrder)
		return asset, err
	case "video":
		asset, err := s.assetsRepo.AddVideo(gameID, assetUID, path, sortOrder)
		return asset, err
	default:
		return nil, ErrValidation
	}
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

func newAssetToken(assetType string) string {
	switch assetType {
	case "cover", "banner":
		return newAssetUID()
	default:
		return newAssetUID()
	}
}

func allocateAssetIdentity(assetType string) (string, string) {
	switch assetType {
	case "screenshot":
		uid := newAssetUID()
		return uid, uid
	case "video":
		uid := newAssetUID()
		return uid, uid
	case "cover", "banner":
		return "", newAssetToken(assetType)
	default:
		return "", newAssetToken(assetType)
	}
}

func (s *AssetsService) ReorderScreenshots(input domain.ScreenshotOrderUpdateInput) error {
	if _, err := s.gamesRepo.GetByID(input.GameID); err != nil {
		return normalizeRepoError(err)
	}
	if len(input.AssetUIDs) == 0 {
		return ErrValidation
	}
	return normalizeRepoError(s.assetsRepo.UpdateScreenshotSortOrders(input.GameID, input.AssetUIDs))
}

func (s *AssetsService) ReorderVideos(input domain.VideoOrderUpdateInput) error {
	if _, err := s.gamesRepo.GetByID(input.GameID); err != nil {
		return normalizeRepoError(err)
	}
	if len(input.AssetUIDs) == 0 {
		return ErrValidation
	}
	return normalizeRepoError(s.assetsRepo.UpdateVideoSortOrders(input.GameID, input.AssetUIDs))
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
