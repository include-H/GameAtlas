package services

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"mime/multipart"
	"os"
	"strings"
	"time"

	"github.com/hao/game/internal/config"
	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/files"
	"github.com/hao/game/internal/repositories"
)

type AssetsService struct {
	gamesRepo  *repositories.GamesRepository
	assetsRepo *repositories.AssetsRepository
	store      *files.AssetStore
}

type UploadResult struct {
	Path     string
	AssetID  *int64
	AssetUID string
}

func NewAssetsService(cfg config.Config, gamesRepo *repositories.GamesRepository, assetsRepo *repositories.AssetsRepository) *AssetsService {
	proxy := cfg.HTTPProxy
	if proxy == "" {
		proxy = cfg.Proxy
	}

	return &AssetsService{
		gamesRepo:  gamesRepo,
		assetsRepo: assetsRepo,
		store:      files.NewAssetStore(cfg.AssetsDir, proxy, 30*time.Second),
	}
}

func (s *AssetsService) Upload(gameID int64, assetType string, header *multipart.FileHeader, sortOrder int) (*UploadResult, error) {
	if _, err := s.gamesRepo.GetByID(gameID); err != nil {
		return nil, normalizeRepoError(err)
	}

	src, err := header.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	contentType := header.Header.Get("Content-Type")
	assetUID, assetName := allocateAssetIdentity(gameID, assetType)
	path, err := s.store.SaveUploadedAsset(gameID, assetType, assetName, src, contentType)
	if err != nil {
		return nil, normalizeAssetError(err)
	}

	asset, err := s.persistAssetPath(gameID, assetType, assetUID, path, sortOrder)
	if err != nil {
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

	if err := s.store.DeleteAsset(assetPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (s *AssetsService) ApplyRemoteAsset(gameID int64, assetType string, remoteURL string, sortOrder int) (string, error) {
	if _, err := s.gamesRepo.GetByID(gameID); err != nil {
		return "", normalizeRepoError(err)
	}
	assetUID, assetName := allocateAssetIdentity(gameID, assetType)
	path, err := s.store.DownloadRemoteAsset(gameID, assetType, assetName, remoteURL)
	if err != nil {
		return "", normalizeAssetError(err)
	}
	if _, err := s.persistAssetPath(gameID, assetType, assetUID, path, sortOrder); err != nil {
		return "", err
	}
	return path, nil
}

func (s *AssetsService) ApplyRawAsset(gameID int64, assetType string, content []byte, contentType string, sortOrder int) (string, error) {
	if _, err := s.gamesRepo.GetByID(gameID); err != nil {
		return "", normalizeRepoError(err)
	}
	assetUID, assetName := allocateAssetIdentity(gameID, assetType)
	path, err := s.store.SaveUploadedAsset(gameID, assetType, assetName, bytes.NewReader(content), contentType)
	if err != nil {
		return "", normalizeAssetError(err)
	}
	if _, err := s.persistAssetPath(gameID, assetType, assetUID, path, sortOrder); err != nil {
		return "", err
	}
	return path, nil
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

func newAssetUID(gameID int64) string {
	buf := make([]byte, 3)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d-%06x", gameID, time.Now().UnixNano()&0xffffff)
	}
	return fmt.Sprintf("%d-%s", gameID, hex.EncodeToString(buf))
}

func newAssetToken(gameID int64, assetType string) string {
	switch assetType {
	case "cover", "banner":
		return fmt.Sprintf("%s-%s", assetType, newAssetUID(gameID))
	default:
		return newAssetUID(gameID)
	}
}

func allocateAssetIdentity(gameID int64, assetType string) (string, string) {
	switch assetType {
	case "screenshot":
		uid := newAssetUID(gameID)
		return uid, uid
	case "video":
		uid := newAssetUID(gameID)
		return uid, uid
	case "cover", "banner":
		return "", newAssetToken(gameID, assetType)
	default:
		return "", newAssetToken(gameID, assetType)
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

func normalizeAssetError(err error) error {
	switch {
	case errors.Is(err, files.ErrInvalidImageType):
		return ErrValidation
	case errors.Is(err, files.ErrInvalidRemoteURL), errors.Is(err, files.ErrBlockedRemoteURL):
		return ErrValidation
	default:
		return err
	}
}
