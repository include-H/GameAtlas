package repositories

import (
	"github.com/jmoiron/sqlx"

	"github.com/hao/game/internal/domain"
)

type GameAggregateRepository struct {
	games *GamesRepository
}

func NewGameAggregateRepository(games *GamesRepository) *GameAggregateRepository {
	return &GameAggregateRepository{games: games}
}

func (r *GameAggregateRepository) DB() *sqlx.DB {
	return r.games.DB()
}

func (r *GameAggregateRepository) ResolveIDByPublicID(publicID string) (int64, error) {
	return r.games.ResolveIDByPublicID(publicID)
}

func (r *GameAggregateRepository) GetByID(id int64) (*domain.Game, error) {
	return r.games.GetByID(id)
}

func (r *GameAggregateRepository) Create(input domain.GameWriteInput) (*domain.Game, error) {
	return r.games.Create(input)
}

func (r *GameAggregateRepository) Update(id int64, input domain.GameAggregateUpdateInput) ([]string, error) {
	return r.games.UpdateAggregate(id, input)
}

func (r *GameAggregateRepository) Delete(id int64) ([]string, bool, error) {
	return r.games.Delete(id)
}
