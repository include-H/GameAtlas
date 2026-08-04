package services

import (
	"strings"

	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/repositories"
)

type StartScreenTilesService struct {
	tilesRepo *repositories.StartScreenTilesRepository
	gamesRepo *repositories.GamesRepository
}

func NewStartScreenTilesService(
	tilesRepo *repositories.StartScreenTilesRepository,
	gamesRepo *repositories.GamesRepository,
) *StartScreenTilesService {
	return &StartScreenTilesService{
		tilesRepo: tilesRepo,
		gamesRepo: gamesRepo,
	}
}

func (s *StartScreenTilesService) List() ([]domain.StartScreenTile, error) {
	tiles, err := s.tilesRepo.List()
	if err != nil {
		return nil, err
	}
	return tiles, nil
}

func (s *StartScreenTilesService) Update(tiles []domain.StartScreenTileWrite) ([]domain.StartScreenTile, error) {
	normalized, err := s.validateTiles(tiles)
	if err != nil {
		return nil, err
	}
	if err := s.tilesRepo.Replace(normalized); err != nil {
		return nil, err
	}
	return s.tilesRepo.List()
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
		normalized = append(normalized, domain.StartScreenTileWrite{
			GameID:   tile.GameID,
			TileSize: tileSize,
		})
	}
	return normalized, nil
}
