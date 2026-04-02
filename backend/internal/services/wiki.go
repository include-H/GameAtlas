package services

import (
	"strings"

	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/repositories"
)

type WikiService struct {
	gamesRepo        *repositories.GamesRepository
	wikiRepo         *repositories.WikiRepository
	wikiHistoryLimit int
}

type WikiDocument struct {
	GameID       int64   `json:"game_id"`
	Title        string  `json:"title"`
	Content      *string `json:"content"`
	UpdatedAt    string  `json:"updated_at"`
	HistoryCount int     `json:"history_count,omitempty"`
}

func NewWikiService(gamesRepo *repositories.GamesRepository, wikiRepo *repositories.WikiRepository, wikiHistoryLimit int) *WikiService {
	return &WikiService{
		gamesRepo:        gamesRepo,
		wikiRepo:         wikiRepo,
		wikiHistoryLimit: wikiHistoryLimit,
	}
}

func (s *WikiService) ResolveGameID(publicID string) (int64, error) {
	id, err := s.gamesRepo.ResolveIDByPublicID(publicID)
	if err != nil {
		return 0, normalizeRepoError(err)
	}
	return id, nil
}

func (s *WikiService) Get(gameID int64, includeAll bool) (*WikiDocument, error) {
	game, err := s.gamesRepo.GetByID(gameID)
	if err != nil {
		return nil, normalizeRepoError(err)
	}
	if !includeAll && game.Visibility == domain.GameVisibilityPrivate {
		return nil, ErrNotFound
	}

	return &WikiDocument{
		GameID:    game.ID,
		Title:     game.Title,
		Content:   game.WikiContent,
		UpdatedAt: game.UpdatedAt,
	}, nil
}

func (s *WikiService) Update(gameID int64, input domain.WikiWriteInput) (*WikiDocument, error) {
	if _, err := s.gamesRepo.GetByID(gameID); err != nil {
		return nil, normalizeRepoError(err)
	}

	content := strings.TrimSpace(input.Content)
	changeSummary := trimStringPtr(input.ChangeSummary)
	game, err := s.wikiRepo.Update(gameID, content, changeSummary, s.wikiHistoryLimit)
	if err != nil {
		return nil, normalizeRepoError(err)
	}

	return &WikiDocument{
		GameID:    game.ID,
		Title:     game.Title,
		Content:   game.WikiContent,
		UpdatedAt: game.UpdatedAt,
	}, nil
}

func (s *WikiService) History(gameID int64, includeAll bool) ([]domain.WikiHistoryEntry, error) {
	game, err := s.gamesRepo.GetByID(gameID)
	if err != nil {
		return nil, normalizeRepoError(err)
	}
	if !includeAll && game.Visibility == domain.GameVisibilityPrivate {
		return nil, ErrNotFound
	}
	items, err := s.wikiRepo.ListHistory(gameID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		return []domain.WikiHistoryEntry{}, nil
	}
	return items, nil
}
