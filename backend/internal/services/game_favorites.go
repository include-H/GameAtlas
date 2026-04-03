package services

import "github.com/hao/game/internal/repositories"

type gameFavoriteLookupRepository interface {
	ResolveIDByPublicID(publicID string) (int64, error)
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

func (s *GameFavoriteService) Set(gameID int64, isFavorite bool) (bool, error) {
	if err := s.favoritesRepo.Set(gameID, isFavorite); err != nil {
		return false, err
	}
	return s.favoritesRepo.IsFavorite(gameID)
}
