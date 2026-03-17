package services

import (
	"errors"
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

func (s *AssetsService) Upload(gameID int64, assetType string, header *multipart.FileHeader, sortOrder int) (string, error) {
	if _, err := s.gamesRepo.GetByID(gameID); err != nil {
		return "", normalizeRepoError(err)
	}

	src, err := header.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	contentType := header.Header.Get("Content-Type")
	path, err := s.store.SaveUploadedAsset(gameID, assetType, src, contentType, sortOrder)
	if err != nil {
		return "", normalizeAssetError(err)
	}

	if err := s.persistAssetPath(gameID, assetType, path, sortOrder); err != nil {
		return "", err
	}

	return path, nil
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
		deleted, err := s.assetsRepo.DeleteScreenshot(input.GameID, assetPath)
		if err != nil {
			return err
		}
		if !deleted {
			return ErrNotFound
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
	path, err := s.store.DownloadRemoteAsset(gameID, assetType, remoteURL, sortOrder)
	if err != nil {
		return "", normalizeAssetError(err)
	}
	if err := s.persistAssetPath(gameID, assetType, path, sortOrder); err != nil {
		return "", err
	}
	return path, nil
}

func (s *AssetsService) persistAssetPath(gameID int64, assetType string, path string, sortOrder int) error {
	switch assetType {
	case "cover":
		return s.assetsRepo.UpdateGameImage(gameID, "cover_image", &path)
	case "banner":
		return s.assetsRepo.UpdateGameImage(gameID, "banner_image", &path)
	case "screenshot":
		_, err := s.assetsRepo.AddScreenshot(gameID, path, sortOrder)
		return err
	default:
		return ErrValidation
	}
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
