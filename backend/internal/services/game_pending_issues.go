package services

import (
	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/repositories"
)

// attachPendingIssues enriches catalog rows with review status without pushing that
// concern back into the catalog repository or domain list model.
func attachPendingIssues(games []domain.GameListItem, reviewIssueOverridesRepo *repositories.ReviewIssueOverrideRepository) error {
	if len(games) == 0 {
		return nil
	}
	if reviewIssueOverridesRepo == nil {
		for index := range games {
			evaluation := domain.EvaluatePendingIssuesForListItem(games[index], nil)
			games[index].PendingIssues = &evaluation
		}
		return nil
	}

	gameIDs := make([]int64, 0, len(games))
	for _, game := range games {
		gameIDs = append(gameIDs, game.ID)
	}

	overrides, err := reviewIssueOverridesRepo.List(gameIDs)
	if err != nil {
		return err
	}

	ignoredReasonsByGameID := make(map[int64]map[domain.PendingIssueDetailKey]*string, len(games))
	for _, item := range overrides {
		if item.Status != "ignored" || !domain.IsAllowedPendingIssueDetail(item.IssueKey) {
			continue
		}
		if ignoredReasonsByGameID[item.GameID] == nil {
			ignoredReasonsByGameID[item.GameID] = make(map[domain.PendingIssueDetailKey]*string)
		}
		ignoredReasonsByGameID[item.GameID][domain.PendingIssueDetailKey(item.IssueKey)] = item.Reason
	}

	for index := range games {
		evaluation := domain.EvaluatePendingIssuesForListItem(games[index], ignoredReasonsByGameID[games[index].ID])
		games[index].PendingIssues = &evaluation
	}

	return nil
}

func getPendingIssueEvaluation(game domain.Game, reviewIssueOverridesRepo *repositories.ReviewIssueOverrideRepository) (*domain.PendingIssueEvaluation, error) {
	if reviewIssueOverridesRepo == nil {
		evaluation := domain.EvaluatePendingIssues(game, nil)
		return &evaluation, nil
	}

	overrides, err := reviewIssueOverridesRepo.List([]int64{game.ID})
	if err != nil {
		return nil, err
	}

	ignoredReasons := make(map[domain.PendingIssueDetailKey]*string)
	for _, item := range overrides {
		if item.Status != "ignored" || !domain.IsAllowedPendingIssueDetail(item.IssueKey) {
			continue
		}
		ignoredReasons[domain.PendingIssueDetailKey(item.IssueKey)] = item.Reason
	}

	evaluation := domain.EvaluatePendingIssues(game, ignoredReasons)
	return &evaluation, nil
}
