package services

import (
	"strings"

	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/repositories"
)

// normalizeGameCoreInput keeps game core field cleanup in services, where this
// project already normalizes request inputs, without introducing another DTO
// that mirrors domain.GameCoreInput.
func normalizeGameCoreInput(input domain.GameCoreInput) domain.GameCoreInput {
	input.Title = strings.TrimSpace(input.Title)
	input.TitleAlt = trimStringPtr(input.TitleAlt)
	input.Visibility = strings.TrimSpace(input.Visibility)
	if input.Visibility == "" {
		input.Visibility = domain.GameVisibilityPublic
	}
	input.Summary = trimStringPtr(input.Summary)
	input.ReleaseDate = trimStringPtr(input.ReleaseDate)
	input.Engine = trimStringPtr(input.Engine)
	input.CoverImage = trimStringPtr(input.CoverImage)
	input.BannerImage = trimStringPtr(input.BannerImage)
	return input
}

// validateGameCoreInput enforces the repository-facing invariants for the
// always-written core columns in games.update/create flows.
func validateGameCoreInput(input domain.GameCoreInput) error {
	if input.Title == "" {
		return ErrValidation
	}
	if input.Visibility != "" &&
		input.Visibility != domain.GameVisibilityPublic &&
		input.Visibility != domain.GameVisibilityPrivate {
		return ErrValidation
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
		return domain.GameCreateInput{}, ErrValidation
	}
	if input.Visibility != domain.GameVisibilityPublic && input.Visibility != domain.GameVisibilityPrivate {
		return domain.GameCreateInput{}, ErrValidation
	}
	return input, nil
}

func validateAndTrimGameAggregateCoreUpdateInput(input domain.GameAggregateCoreUpdateInput, tagsRepo *repositories.TagsRepository) (domain.GameAggregateCoreUpdateInput, error) {
	input.GameCoreInput = normalizeGameCoreInput(input.GameCoreInput)
	if err := validateGameCoreInput(input.GameCoreInput); err != nil {
		return domain.GameAggregateCoreUpdateInput{}, err
	}
	input.PlatformIDs = uniqueIDs(input.PlatformIDs)
	input.DeveloperIDs = uniqueIDs(input.DeveloperIDs)
	input.PublisherIDs = uniqueIDs(input.PublisherIDs)
	input.TagIDs = uniqueIDs(input.TagIDs)
	if input.TagIDs == nil {
		input.TagIDs = []int64{}
	}

	tagIDs, err := tagsRepo.ValidateTagSelection(input.TagIDs)
	if err != nil {
		return domain.GameAggregateCoreUpdateInput{}, ErrValidation
	}
	input.TagIDs = tagIDs

	return input, nil
}
