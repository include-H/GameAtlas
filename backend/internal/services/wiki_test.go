package services

import (
	"errors"
	"testing"

	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/repositories"
)

func TestWikiServiceGetRejectsPrivateGameForPublicRequest(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertServicesTestGame(t, db, "private-game", "Private Game", domain.GameVisibilityPrivate)
	service := NewWikiService(
		repositories.NewGamesRepository(db),
		repositories.NewWikiRepository(db),
		2,
	)

	_, err := service.Get(gameID, false)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get error = %v, want ErrNotFound", err)
	}
}

func TestWikiServiceUpdateRendersMarkdownAndPrunesHistory(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertServicesTestGame(t, db, "wiki-game", "Wiki Game", domain.GameVisibilityPublic)
	service := NewWikiService(
		repositories.NewGamesRepository(db),
		repositories.NewWikiRepository(db),
		2,
	)

	firstSummary := "  first pass  "
	secondSummary := " second pass "
	thirdSummary := "  third pass "

	if _, err := service.Update(gameID, domain.WikiWriteInput{
		Content:       "  first body  ",
		ChangeSummary: &firstSummary,
	}); err != nil {
		t.Fatalf("first Update returned error: %v", err)
	}
	if _, err := service.Update(gameID, domain.WikiWriteInput{
		Content:       "## Second",
		ChangeSummary: &secondSummary,
	}); err != nil {
		t.Fatalf("second Update returned error: %v", err)
	}

	document, err := service.Update(gameID, domain.WikiWriteInput{
		Content:       "  # Third Title  ",
		ChangeSummary: &thirdSummary,
	})
	if err != nil {
		t.Fatalf("third Update returned error: %v", err)
	}

	if document.Content == nil || *document.Content != "# Third Title" {
		t.Fatalf("document.Content = %v, want trimmed markdown", document.Content)
	}
	history, err := service.History(gameID, true)
	if err != nil {
		t.Fatalf("History returned error: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("len(history) = %d, want 2", len(history))
	}
	if history[0].Content != "# Third Title" {
		t.Fatalf("history[0].Content = %q, want latest trimmed content", history[0].Content)
	}
	if history[0].ChangeSummary == nil || *history[0].ChangeSummary != "third pass" {
		t.Fatalf("history[0].ChangeSummary = %v, want trimmed latest summary", history[0].ChangeSummary)
	}
	if history[1].Content != "## Second" {
		t.Fatalf("history[1].Content = %q, want second update", history[1].Content)
	}
	if history[1].ChangeSummary == nil || *history[1].ChangeSummary != "second pass" {
		t.Fatalf("history[1].ChangeSummary = %v, want trimmed second summary", history[1].ChangeSummary)
	}
}

func TestWikiServiceHistoryReturnsEmptySliceWhenNoEntriesExist(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertServicesTestGame(t, db, "history-empty", "History Empty", domain.GameVisibilityPublic)
	service := NewWikiService(
		repositories.NewGamesRepository(db),
		repositories.NewWikiRepository(db),
		2,
	)

	history, err := service.History(gameID, true)
	if err != nil {
		t.Fatalf("History returned error: %v", err)
	}
	if history == nil {
		t.Fatalf("expected History to return empty slice, got nil")
	}
	if len(history) != 0 {
		t.Fatalf("len(history) = %d, want 0", len(history))
	}
}
