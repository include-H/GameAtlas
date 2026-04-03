package repositories

import (
	"github.com/jmoiron/sqlx"

	"github.com/hao/game/internal/domain"
)

type GameDetailRepository struct {
	games *GamesRepository
}

func NewGameDetailRepository(games *GamesRepository) *GameDetailRepository {
	return &GameDetailRepository{games: games}
}

func (r *GameDetailRepository) ResolveIDByPublicID(publicID string) (int64, error) {
	return r.games.ResolveIDByPublicID(publicID)
}

func (r *GameDetailRepository) DB() *sqlx.DB {
	return r.games.DB()
}

func (r *GameDetailRepository) GetByID(id int64) (*domain.Game, error) {
	return r.games.GetByID(id)
}

func (r *GameDetailRepository) GetByPublicID(publicID string) (*domain.Game, error) {
	return r.games.GetByPublicID(publicID)
}

func (r *GameDetailRepository) IncrementDownloads(id int64) error {
	return r.games.IncrementDownloads(id)
}

func (r *GameDetailRepository) ListScreenshots(gameID int64) ([]domain.GameAsset, error) {
	return r.games.ListScreenshots(gameID)
}

func (r *GameDetailRepository) ListVideos(gameID int64) ([]domain.GameAsset, error) {
	return r.games.ListVideos(gameID)
}

func (r *GameDetailRepository) GetSeriesMetadata(gameID int64) (*domain.MetadataItem, error) {
	return r.games.GetSeriesMetadata(gameID)
}

func (r *GameDetailRepository) ListMetadata(table, joinTable, joinColumn string, gameID int64) ([]domain.MetadataItem, error) {
	return r.games.ListMetadata(table, joinTable, joinColumn, gameID)
}
