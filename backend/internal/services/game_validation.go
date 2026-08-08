package services

import (
	"strings"

	"github.com/hao/game/internal/domain"
)

// normalizeGameCoreInput trims aggregate-edit core fields before validation.
// Aggregate updates are full replacements, so callers must provide visibility explicitly.
func normalizeGameCoreInput(input domain.GameCoreInput) domain.GameCoreInput {
	input.Title = strings.TrimSpace(input.Title)
	input.TitleAlt = trimStringPtr(input.TitleAlt)
	input.Visibility = strings.TrimSpace(input.Visibility)
	input.Summary = trimStringPtr(input.Summary)
	input.ReleaseDate = trimStringPtr(input.ReleaseDate)
	input.SavePathTemplate = trimStringPtr(input.SavePathTemplate)
	return input
}

// validateGameCoreInput enforces the repository-facing invariants for the
// always-written core columns in games.update/create flows.
func validateGameCoreInput(input domain.GameCoreInput) error {
	if input.Title == "" {
		return domain.ErrValidation
	}
	if input.Visibility != domain.GameVisibilityPublic &&
		input.Visibility != domain.GameVisibilityPrivate {
		return domain.ErrValidation
	}
	return nil
}

func validateAndTrimGameCreateInput(input domain.GameCreateInput) (domain.GameCreateInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Visibility = strings.TrimSpace(input.Visibility)
	if input.Visibility == "" {
		input.Visibility = domain.GameVisibilityPublic
	}
	if input.Title == "" {
		return domain.GameCreateInput{}, domain.ErrValidation
	}
	if input.Visibility != domain.GameVisibilityPublic && input.Visibility != domain.GameVisibilityPrivate {
		return domain.GameCreateInput{}, domain.ErrValidation
	}
	return input, nil
}

func validateAndTrimGameAggregateCoreUpdateInput(input domain.GameAggregateCoreUpdateInput) (domain.GameAggregateCoreUpdateInput, error) {
	input.GameCoreInput = normalizeGameCoreInput(input.GameCoreInput)
	if err := validateGameCoreInput(input.GameCoreInput); err != nil {
		return domain.GameAggregateCoreUpdateInput{}, err
	}
	input.DeveloperIDs = uniqueIDs(input.DeveloperIDs)
	input.PublisherIDs = uniqueIDs(input.PublisherIDs)

	return input, nil
}
