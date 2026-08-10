package services

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"mime/multipart"
	"time"

	"github.com/hao/game/internal/config"
	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/files"
)

type assetGameRepository interface {
	GetByID(id int64) (*domain.Game, error)
}

type AssetsService struct {
	gamesRepo assetGameRepository
	store     *files.AssetStore
}

type UploadResult struct {
	Path     string
	AssetUID string
}

const (
	maxImageUploadBytes = files.MaxImageUploadBytes
	maxVideoUploadBytes = files.MaxVideoUploadBytes
)

var errAssetTooLarge = errors.New("asset too large")

func NewAssetsService(cfg config.Config, gamesRepo assetGameRepository) *AssetsService {
	return &AssetsService{
		gamesRepo: gamesRepo,
		store:     files.NewAssetStore(cfg.AssetsDir),
	}
}

// 2026-05-09: Upload accepts *multipart.FileHeader rather than io.Reader because
// the caller (the single HTTP handler) always has one, and the method needs both
// the stream and the content-type from the header. For a single-process app with
// one upload entrypoint, extracting an io.Reader interface adds indirection
// without practical testability gain.
func (s *AssetsService) Upload(gameID int64, assetType string, header *multipart.FileHeader) (*UploadResult, error) {
	if header == nil {
		return nil, domain.ErrValidation
	}
	if assetType != "cover" && assetType != "banner" && assetType != "screenshot" &&
		assetType != "logo" && assetType != "video" && assetType != "poster" {
		return nil, domain.ErrValidation
	}
	limit := maxImageUploadBytes
	if assetType == "video" {
		limit = maxVideoUploadBytes
	}
	if header.Size > limit {
		return nil, normalizeAssetError(fmt.Errorf(
			"%w: %d bytes exceeds limit %d",
			errAssetTooLarge,
			header.Size,
			limit,
		))
	}

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
	assetUID := newAssetUID()
	assetName := assetUID
	path, err := s.store.SaveToStaging(game.PublicID, assetType, assetName, src, contentType)
	if err != nil {
		return nil, normalizeAssetError(err)
	}

	return &UploadResult{Path: path, AssetUID: assetUID}, nil
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

func normalizeAssetError(err error) error {
	switch {
	case errors.Is(err, files.ErrInvalidImageType):
		return domain.ErrValidation
	case errors.Is(err, files.ErrInvalidAssetName):
		return domain.ErrValidation
	case errors.Is(err, errAssetTooLarge):
		return domain.ErrValidation
	case errors.Is(err, files.ErrUploadTooLarge), errors.Is(err, files.ErrInvalidImageContent):
		return domain.ErrValidation
	case errors.Is(err, files.ErrInvalidAssetPath):
		return domain.ErrValidation
	default:
		return err
	}
}
