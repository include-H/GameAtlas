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
	repo := repositories.NewReviewIssueOverrideRepository(db)
	service := NewReviewIssueOverrideService(
		repositories.NewGamesRepository(db),
		repo,
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

	if err := service.Delete(gameID, "missing-cover"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	recreated, err := repo.Upsert(gameID, "missing-cover", "ignored", nil)
	if err != nil {
		t.Fatalf("Upsert after delete returned error: %v", err)
	}
	if recreated.ID == item.ID {
		t.Fatalf("recreated.ID = %d, want delete to remove the previous row", recreated.ID)
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
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("Ignore error = %v, want domain.ErrValidation", err)
	}
}
