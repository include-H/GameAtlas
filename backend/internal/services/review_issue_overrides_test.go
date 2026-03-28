package services

import (
	"errors"
	"testing"

	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/repositories"
)

func TestReviewIssueOverrideServiceIgnoreNormalizesReasonAndDeleteRemovesOverride(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertServicesTestGame(t, db, "review-game", "Review Game", domain.GameVisibilityPublic)
	service := NewReviewIssueOverrideService(
		repositories.NewGamesRepository(db),
		repositories.NewReviewIssueOverrideRepository(db),
	)

	reason := "  accepted gap  "
	item, err := service.Ignore(gameID, "missing-cover", &reason)
	if err != nil {
		t.Fatalf("Ignore returned error: %v", err)
	}
	if item.Status != "ignored" {
		t.Fatalf("Status = %q, want ignored", item.Status)
	}
	if item.Reason == nil || *item.Reason != "accepted gap" {
		t.Fatalf("Reason = %v, want trimmed reason", item.Reason)
	}

	items, err := service.List([]int64{gameID})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}

	if err := service.Delete(gameID, "missing-cover"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	items, err = service.List([]int64{gameID})
	if err != nil {
		t.Fatalf("List after delete returned error: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("len(items) after delete = %d, want 0", len(items))
	}
}

func TestReviewIssueOverrideServiceRejectsUnknownIssueKeyAndBlankReasonBecomesNil(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertServicesTestGame(t, db, "review-invalid", "Review Invalid", domain.GameVisibilityPublic)
	service := NewReviewIssueOverrideService(
		repositories.NewGamesRepository(db),
		repositories.NewReviewIssueOverrideRepository(db),
	)

	blankReason := "   "
	item, err := service.Ignore(gameID, "missing-summary", &blankReason)
	if err != nil {
		t.Fatalf("Ignore returned error: %v", err)
	}
	if item.Reason != nil {
		t.Fatalf("Reason = %v, want nil for blank reason", item.Reason)
	}

	_, err = service.Ignore(gameID, "not-a-real-issue", nil)
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Ignore error = %v, want ErrValidation", err)
	}
}
