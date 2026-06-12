package services

import (
	"strings"

	"github.com/hao/game/internal/domain"
)

func normalizeListParams(params *domain.GamesListParams) error {
	if !params.IncludeAll && strings.TrimSpace(params.Visibility) == "" {
		params.Visibility = domain.GameVisibilityPublic
	}
	if params.PendingRecentDays < 0 {
		params.PendingRecentDays = 0
	}
	if params.PendingRecentDays > 365 {
		params.PendingRecentDays = 365
	}
	return nil
}

func validateListParamsContract(params domain.GamesListParams) error {
	// 2026-05-01: internal callers may bypass the HTTP decoder, but they still must satisfy
	// the same list contract. Reject incomplete/invalid sort+order here instead of allowing
	// repository code to fall through into implicit defaults or malformed SQL behavior.
	if params.Page <= 0 || params.Limit <= 0 {
		return domain.ErrValidation
	}

	sort := strings.TrimSpace(params.Sort)
	if !domain.IsAllowedGamesListSort(sort) {
		return domain.ErrValidation
	}

	order := strings.TrimSpace(params.Order)
	if !domain.IsAllowedGamesListOrder(order) {
		return domain.ErrValidation
	}

	if sort == "random" && params.SortSeed <= 0 {
		return domain.ErrValidation
	}

	return nil
}

func normalizeTimelineParams(params *domain.GamesTimelineParams) error {
	if params.Limit <= 0 {
		params.Limit = 60
	}
	if params.Limit > 100 {
		params.Limit = 100
	}

	if !params.IncludeAll && strings.TrimSpace(params.Visibility) == "" {
		params.Visibility = domain.GameVisibilityPublic
	}

	return nil
}
