package services

import (
	"mime/multipart"
	"strings"

	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/files"
	"github.com/hao/game/internal/repositories"
)

type StartScreenTilesService struct {
	tilesRepo   *repositories.StartScreenTilesRepository
	columnsRepo *repositories.StartScreenColumnsRepository
	gamesRepo   *repositories.GamesRepository
	store       *files.AssetStore
}

func NewStartScreenTilesService(
	tilesRepo *repositories.StartScreenTilesRepository,
	columnsRepo *repositories.StartScreenColumnsRepository,
	gamesRepo *repositories.GamesRepository,
	store *files.AssetStore,
) *StartScreenTilesService {
	return &StartScreenTilesService{
		tilesRepo:   tilesRepo,
		columnsRepo: columnsRepo,
		gamesRepo:   gamesRepo,
		store:       store,
	}
}

func (s *StartScreenTilesService) List(includeAll bool) (*domain.StartScreenLayout, error) {
	columns, err := s.columnsRepo.List()
	if err != nil {
		return nil, err
	}
	tiles, err := s.tilesRepo.List(includeAll)
	if err != nil {
		return nil, err
	}
	return &domain.StartScreenLayout{Columns: columns, Tiles: tiles}, nil
}

func (s *StartScreenTilesService) Update(
	columns []domain.StartScreenColumnWrite,
	tiles []domain.StartScreenTileWrite,
) (*domain.StartScreenLayout, error) {
	normalizedColumns, err := s.validateColumns(columns)
	if err != nil {
		return nil, err
	}
	normalized, err := s.validateTiles(tiles)
	if err != nil {
		return nil, err
	}
	if err := s.columnsRepo.Replace(normalizedColumns); err != nil {
		return nil, err
	}
	if err := s.tilesRepo.Replace(normalized); err != nil {
		return nil, err
	}
	return s.List(true)
}

// AddTile 从游戏库卡片入口追加一个磁贴到末尾；已在开始屏幕时幂等返回当前布局。
func (s *StartScreenTilesService) AddTile(tile domain.StartScreenTileWrite) (*domain.StartScreenLayout, error) {
	normalized, err := s.validateTiles([]domain.StartScreenTileWrite{tile})
	if err != nil {
		return nil, err
	}
	if len(normalized) == 0 {
		return nil, domain.ErrValidation
	}
	if _, err := s.tilesRepo.Append(normalized[0]); err != nil {
		return nil, err
	}
	return s.List(true)
}

// RemoveTile 从开始屏幕移除一个游戏的磁贴；不存在时幂等返回当前布局。
func (s *StartScreenTilesService) RemoveTile(gameID int64) (*domain.StartScreenLayout, error) {
	if gameID <= 0 {
		return nil, domain.ErrValidation
	}
	if _, err := s.tilesRepo.RemoveByGameID(gameID); err != nil {
		return nil, err
	}
	return s.List(true)
}

func (s *StartScreenTilesService) validateColumns(columns []domain.StartScreenColumnWrite) ([]domain.StartScreenColumnWrite, error) {
	if len(columns) > 100 {
		return nil, domain.ErrValidation
	}
	normalized := make([]domain.StartScreenColumnWrite, 0, len(columns))
	for _, column := range columns {
		name := strings.TrimSpace(column.Name)
		if len([]rune(name)) > 30 {
			return nil, domain.ErrValidation
		}
		normalized = append(normalized, domain.StartScreenColumnWrite{Name: name})
	}
	return normalized, nil
}

func (s *StartScreenTilesService) validateTiles(tiles []domain.StartScreenTileWrite) ([]domain.StartScreenTileWrite, error) {
	normalized := make([]domain.StartScreenTileWrite, 0, len(tiles))
	seen := make(map[int64]struct{}, len(tiles))
	for _, tile := range tiles {
		if tile.GameID <= 0 {
			return nil, domain.ErrValidation
		}
		tileSize := strings.TrimSpace(tile.TileSize)
		if !domain.IsAllowedStartScreenTileSize(tileSize) {
			return nil, domain.ErrValidation
		}
		if _, exists := seen[tile.GameID]; exists {
			return nil, domain.ErrValidation
		}
		seen[tile.GameID] = struct{}{}

		if _, err := s.gamesRepo.GetByID(tile.GameID); err != nil {
			return nil, normalizeRepoError(err)
		}

		imageSmallPath, err := s.validateTileImagePath(tile.ImageSmallPath)
		if err != nil {
			return nil, err
		}
		imageWidePath, err := s.validateTileImagePath(tile.ImageWidePath)
		if err != nil {
			return nil, err
		}
		imageLargePath, err := s.validateTileImagePath(tile.ImageLargePath)
		if err != nil {
			return nil, err
		}

		normalized = append(normalized, domain.StartScreenTileWrite{
			GameID:         tile.GameID,
			TileSize:       tileSize,
			ImageSmallPath: imageSmallPath,
			ImageWidePath:  imageWidePath,
			ImageLargePath: imageLargePath,
		})
	}
	return normalized, nil
}

func (s *StartScreenTilesService) validateTileImagePath(path *string) (*string, error) {
	if path == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*path)
	if trimmed == "" {
		return nil, nil
	}
	if !strings.HasPrefix(trimmed, "/assets/") {
		return nil, domain.ErrValidation
	}
	if s.store == nil || !s.store.AssetExists(trimmed) {
		return nil, domain.ErrValidation
	}
	return &trimmed, nil
}

// UploadTileImage 保存磁贴裁剪图到 assets/start-screen/，返回 /assets/start-screen/{uid}.{ext} 路径。
func (s *StartScreenTilesService) UploadTileImage(header *multipart.FileHeader) (string, error) {
	if s.store == nil {
		return "", domain.ErrMissingConfig
	}
	src, err := header.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	contentType := header.Header.Get("Content-Type")
	assetUID := newAssetUID()
	stagingPath, err := s.store.SaveToStaging("start-screen", "cover", assetUID, src, contentType)
	if err != nil {
		return "", normalizeAssetError(err)
	}
	return s.store.MoveToPermanent(stagingPath, "start-screen")
}
