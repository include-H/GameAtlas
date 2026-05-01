package services

import (
	"fmt"
	"strings"
	"time"

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
		return ErrValidation
	}

	sort := strings.TrimSpace(params.Sort)
	if !domain.IsAllowedGamesListSort(sort) {
		return ErrValidation
	}

	order := strings.TrimSpace(params.Order)
	if !domain.IsAllowedGamesListOrder(order) {
		return ErrValidation
	}

	if sort == "random" && params.SortSeed <= 0 {
		return ErrValidation
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

	if strings.TrimSpace(params.FromDate) == "" || strings.TrimSpace(params.ToDate) == "" {
		return ErrValidation
	}

	fromDate, fromTime, err := parseTimelineDate(params.FromDate)
	if err != nil {
		return ErrValidation
	}
	toDate, toTime, err := parseTimelineDate(params.ToDate)
	if err != nil {
		return ErrValidation
	}
	if fromTime.After(toTime) {
		return ErrValidation
	}
	params.FromDate = fromDate
	params.ToDate = toDate

	if params.CursorReleaseDate != "" {
		cursorDate, _, err := parseTimelineDate(params.CursorReleaseDate)
		if err != nil {
			return ErrValidation
		}
		if params.CursorID <= 0 {
			return ErrValidation
		}
		params.CursorReleaseDate = cursorDate
		if params.CursorReleaseDate < params.FromDate || params.CursorReleaseDate > params.ToDate {
			return fmt.Errorf("%w: cursor date out of range", ErrValidation)
		}
	}

	if !params.IncludeAll && strings.TrimSpace(params.Visibility) == "" {
		params.Visibility = domain.GameVisibilityPublic
	}

	return nil
}

func normalizeTimelineDate(value string) (string, error) {
	normalized, _, err := parseTimelineDate(value)
	return normalized, err
}

func parseTimelineDate(value string) (string, time.Time, error) {
	trimmed := strings.TrimSpace(value)
	layouts := []string{
		"2006-01-02",
		"2006-1-2",
		"2006-01",
		"2006-1",
		"2006",
	}

	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, trimmed); err == nil {
			normalized := parsed.Format("2006-01-02")
			return normalized, parsed, nil
		}
	}

	return "", time.Time{}, ErrValidation
}
