package services

import (
	"strings"

	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/markdown"
	"github.com/hao/game/internal/repositories"
)

type WikiService struct {
	gamesRepo        *repositories.GamesRepository
	wikiRepo         *repositories.WikiRepository
	renderer         *markdown.Renderer
	wikiHistoryLimit int
}

type WikiDocument struct {
	GameID       int64   `json:"game_id"`
	Title        string  `json:"title"`
	Content      *string `json:"content"`
	ContentHTML  *string `json:"content_html"`
	UpdatedAt    string  `json:"updated_at"`
	HistoryCount int     `json:"history_count,omitempty"`
}

func NewWikiService(gamesRepo *repositories.GamesRepository, wikiRepo *repositories.WikiRepository, renderer *markdown.Renderer, wikiHistoryLimit int) *WikiService {
	return &WikiService{
		gamesRepo:        gamesRepo,
		wikiRepo:         wikiRepo,
		renderer:         renderer,
		wikiHistoryLimit: wikiHistoryLimit,
	}
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
		GameID:      game.ID,
		Title:       game.Title,
		Content:     game.WikiContent,
		ContentHTML: game.WikiContentHTML,
		UpdatedAt:   game.UpdatedAt,
	}, nil
}

func (s *WikiService) Update(gameID int64, input domain.WikiWriteInput) (*WikiDocument, error) {
	if _, err := s.gamesRepo.GetByID(gameID); err != nil {
		return nil, normalizeRepoError(err)
	}

	content := strings.TrimSpace(input.Content)
	html, err := s.renderer.Render(content)
	if err != nil {
		return nil, err
	}

	changeSummary := trimStringPtr(input.ChangeSummary)
	game, err := s.wikiRepo.Update(gameID, content, html, changeSummary, s.wikiHistoryLimit)
	if err != nil {
		return nil, normalizeRepoError(err)
	}

	return &WikiDocument{
		GameID:      game.ID,
		Title:       game.Title,
		Content:     game.WikiContent,
		ContentHTML: game.WikiContentHTML,
		UpdatedAt:   game.UpdatedAt,
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
