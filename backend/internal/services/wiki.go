package services

import (
	"strings"

	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/markdown"
	"github.com/hao/game/internal/repositories"
)

type WikiService struct {
	gamesRepo *repositories.GamesRepository
	wikiRepo  *repositories.WikiRepository
	renderer  *markdown.Renderer
}

type WikiDocument struct {
	GameID       int64   `json:"game_id"`
	Title        string  `json:"title"`
	Content      *string `json:"content"`
	ContentHTML  *string `json:"content_html"`
	UpdatedAt    string  `json:"updated_at"`
	HistoryCount int     `json:"history_count,omitempty"`
}

func NewWikiService(gamesRepo *repositories.GamesRepository, wikiRepo *repositories.WikiRepository, renderer *markdown.Renderer) *WikiService {
	return &WikiService{
		gamesRepo: gamesRepo,
		wikiRepo:  wikiRepo,
		renderer:  renderer,
	}
}

func (s *WikiService) Get(gameID int64) (*WikiDocument, error) {
	game, err := s.wikiRepo.Get(gameID)
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
	game, err := s.wikiRepo.Update(gameID, content, html, changeSummary)
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

func (s *WikiService) History(gameID int64) ([]domain.WikiHistoryEntry, error) {
	if _, err := s.gamesRepo.GetByID(gameID); err != nil {
		return nil, normalizeRepoError(err)
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
