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

func validateAndTrimGameInput(input domain.GameWriteInput, tagsRepo *repositories.TagsRepository) (domain.GameWriteInput, error) {
	input.GameCoreInput = normalizeGameCoreInput(input.GameCoreInput)
	if err := validateGameCoreInput(input.GameCoreInput); err != nil {
		return domain.GameWriteInput{}, err
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
		return domain.GameWriteInput{}, ErrValidation
	}
	input.TagIDs = tagIDs

	return input, nil
}

func validateAndTrimGameAggregatePatchInput(input domain.GameAggregatePatchInput, tagsRepo *repositories.TagsRepository) (domain.GameAggregatePatchInput, error) {
	input.GameCoreInput = normalizeGameCoreInput(input.GameCoreInput)
	if err := validateGameCoreInput(input.GameCoreInput); err != nil {
		return domain.GameAggregatePatchInput{}, err
	}
	if input.PlatformIDs.Present {
		input.PlatformIDs.Values = uniqueIDs(input.PlatformIDs.Values)
	}
	if input.DeveloperIDs.Present {
		input.DeveloperIDs.Values = uniqueIDs(input.DeveloperIDs.Values)
	}
	if input.PublisherIDs.Present {
		input.PublisherIDs.Values = uniqueIDs(input.PublisherIDs.Values)
	}
	if input.TagIDs.Present {
		input.TagIDs.Values = uniqueIDs(input.TagIDs.Values)
	}
	if input.TagIDs.Present && input.TagIDs.Values == nil {
		input.TagIDs.Values = []int64{}
	}

	if input.TagIDs.Present {
		tagIDs, err := tagsRepo.ValidateTagSelection(input.TagIDs.Values)
		if err != nil {
			return domain.GameAggregatePatchInput{}, ErrValidation
		}
		input.TagIDs.Values = tagIDs
	}

	return input, nil
}
