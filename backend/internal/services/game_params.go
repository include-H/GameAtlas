package services

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/hao/game/internal/domain"
)

func normalizeListParams(params *domain.GamesListParams) error {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.Limit <= 0 {
		params.Limit = 20
	}
	if params.Limit > 100 {
		params.Limit = 100
	}
	if params.Sort == "" {
		params.Sort = "updated_at"
	}
	if params.Order == "" {
		params.Order = "desc"
	}
	if params.Sort == "random" && params.SortSeed == 0 {
		params.SortSeed = rand.New(rand.NewSource(time.Now().UnixNano())).Int63n(2147483646) + 1
	}
	if !params.IncludeAll && strings.TrimSpace(params.Visibility) == "" {
		params.Visibility = domain.GameVisibilityPublic
	}
	params.PendingIssue = strings.TrimSpace(params.PendingIssue)
	if params.PendingIssue != "" {
		if !domain.IsAllowedPendingIssueFilter(params.PendingIssue) {
			return ErrValidation
		}
	}
	if params.PendingRecentDays < 0 {
		params.PendingRecentDays = 0
	}
	if params.PendingRecentDays > 365 {
		params.PendingRecentDays = 365
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
