package services

import (
	"strings"

	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/repositories"
)

var allowedReviewIssueKeys = map[string]struct{}{
	"missing-cover":        {},
	"missing-banner":       {},
	"missing-screenshots":  {},
	"missing-wiki-content": {},
	"missing-files-list":   {},
	"missing-developer":    {},
	"missing-publisher":    {},
	"missing-platform":     {},
	"missing-summary":      {},
}

type ReviewIssueOverrideService struct {
	gamesRepo     *repositories.GamesRepository
	overridesRepo *repositories.ReviewIssueOverrideRepository
}

func NewReviewIssueOverrideService(
	gamesRepo *repositories.GamesRepository,
	overridesRepo *repositories.ReviewIssueOverrideRepository,
) *ReviewIssueOverrideService {
	return &ReviewIssueOverrideService{
		gamesRepo:     gamesRepo,
		overridesRepo: overridesRepo,
	}
}

func (s *ReviewIssueOverrideService) List(gameIDs []int64) ([]domain.ReviewIssueOverride, error) {
	return s.overridesRepo.List(gameIDs)
}

func (s *ReviewIssueOverrideService) Ignore(gameID int64, issueKey string, reason *string) (*domain.ReviewIssueOverride, error) {
	if _, err := s.gamesRepo.GetByID(gameID); err != nil {
		return nil, normalizeRepoError(err)
	}

	normalizedIssueKey, normalizedReason, err := normalizeReviewOverrideInput(issueKey, reason)
	if err != nil {
		return nil, err
	}

	return s.overridesRepo.Upsert(gameID, normalizedIssueKey, "ignored", normalizedReason)
}

func (s *ReviewIssueOverrideService) Delete(gameID int64, issueKey string) error {
	if _, err := s.gamesRepo.GetByID(gameID); err != nil {
		return normalizeRepoError(err)
	}

	normalizedIssueKey, _, err := normalizeReviewOverrideInput(issueKey, nil)
	if err != nil {
		return err
	}

	return s.overridesRepo.Delete(gameID, normalizedIssueKey)
}

func normalizeReviewOverrideInput(issueKey string, reason *string) (string, *string, error) {
	normalizedIssueKey := strings.TrimSpace(issueKey)
	if _, ok := allowedReviewIssueKeys[normalizedIssueKey]; !ok {
		return "", nil, ErrValidation
	}

	if reason == nil {
		return normalizedIssueKey, nil, nil
	}

	trimmed := strings.TrimSpace(*reason)
	if trimmed == "" {
		return normalizedIssueKey, nil, nil
	}

	return normalizedIssueKey, &trimmed, nil
}
