package services

import "github.com/hao/game/internal/domain"

type gameDetailReadRepository interface {
	ResolveIDByPublicID(publicID string) (int64, error)
	GetByID(id int64) (*domain.Game, error)
}
