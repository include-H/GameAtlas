package services

import (
	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/repositories"
)

type gameFavoriteLookupRepository interface {
	ResolveIDByPublicID(publicID string) (int64, error)
	GetByID(id int64) (*domain.Game, error)
}

type GameFavoriteService struct {
	gamesRepo     gameFavoriteLookupRepository
	favoritesRepo *repositories.FavoriteGamesRepository
}

func NewGameFavoriteService(
	gamesRepo gameFavoriteLookupRepository,
	favoritesRepo *repositories.FavoriteGamesRepository,
) *GameFavoriteService {
	return &GameFavoriteService{
		gamesRepo:     gamesRepo,
		favoritesRepo: favoritesRepo,
	}
}

func (s *GameFavoriteService) ResolveGameID(publicID string) (int64, error) {
	id, err := s.gamesRepo.ResolveIDByPublicID(publicID)
	if err != nil {
		return 0, normalizeRepoError(err)
	}
	return id, nil
}

// Set 保持"匿名可改全局收藏"的产品约定，但对非管理员（includeAll=false）在写操作前
// 检查游戏可见性：私有游戏返回 ErrNotFound，与"不存在"统一为 404，避免通过收藏接口
// 探测私有游戏是否存在。管理员不受限，可收藏任意游戏。
func (s *GameFavoriteService) Set(gameID int64, isFavorite bool, includeAll bool) (bool, error) {
	if !includeAll {
		game, err := s.gamesRepo.GetByID(gameID)
		if err != nil {
			return false, normalizeRepoError(err)
		}
		if game.Visibility != domain.GameVisibilityPublic {
			return false, domain.ErrNotFound
		}
	}

	if err := s.favoritesRepo.Set(gameID, isFavorite); err != nil {
		return false, err
	}
	return s.favoritesRepo.IsFavorite(gameID)
}
