package services

import (
	"strings"
	"unicode"

	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/repositories"
)

type TagsService struct {
	repo *repositories.TagsRepository
}

func NewTagsService(repo *repositories.TagsRepository) *TagsService {
	return &TagsService{repo: repo}
}

func (s *TagsService) ListGroups() ([]domain.TagGroup, error) {
	items, err := s.repo.ListGroups()
	if err != nil {
		return nil, err
	}
	if items == nil {
		return []domain.TagGroup{}, nil
	}
	return items, nil
}

func (s *TagsService) CreateGroup(input domain.TagGroupWriteInput) (*domain.TagGroup, error) {
	key := slugifyTagValue(input.Key)
	name := strings.TrimSpace(input.Name)
	if key == "" || name == "" {
		return nil, ErrValidation
	}

	sortOrder := 0
	if input.SortOrder != nil {
		sortOrder = *input.SortOrder
	}
	allowMultiple := true
	if input.AllowMultiple != nil {
		allowMultiple = *input.AllowMultiple
	}
	isFilterable := true
	if input.IsFilterable != nil {
		isFilterable = *input.IsFilterable
	}

	return s.repo.CreateGroup(domain.TagGroupWriteInput{
		Key:           key,
		Name:          name,
		Description:   trimStringPtr(input.Description),
		SortOrder:     &sortOrder,
		AllowMultiple: &allowMultiple,
		IsFilterable:  &isFilterable,
	}, sortOrder, allowMultiple, isFilterable)
}

func (s *TagsService) ListTags(params domain.TagsListParams) ([]domain.Tag, error) {
	items, err := s.repo.ListTags(params)
	if err != nil {
		return nil, err
	}
	if items == nil {
		return []domain.Tag{}, nil
	}
	return items, nil
}

func (s *TagsService) CreateTag(input domain.TagWriteInput) (*domain.Tag, error) {
	name := strings.TrimSpace(input.Name)
	if input.GroupID <= 0 || name == "" {
		return nil, ErrValidation
	}

	slugValue := ""
	if input.Slug != nil {
		slugValue = slugifyTagValue(*input.Slug)
	}
	if slugValue == "" {
		slugValue = slugifyTagValue(name)
	}
	if slugValue == "" {
		return nil, ErrValidation
	}

	sortOrder := 0
	if input.SortOrder != nil {
		sortOrder = *input.SortOrder
	}
	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	return s.repo.CreateTag(domain.TagWriteInput{
		GroupID:   input.GroupID,
		Name:      name,
		Slug:      &slugValue,
		ParentID:  input.ParentID,
		SortOrder: &sortOrder,
		IsActive:  &isActive,
	}, slugValue, sortOrder, isActive)
}

func slugifyTagValue(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastDash := false

	for _, r := range value {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(r)
			lastDash = false
		case r == '-' || r == '_' || unicode.IsSpace(r):
			if builder.Len() > 0 && !lastDash {
				builder.WriteRune('-')
				lastDash = true
			}
		}
	}

	return strings.Trim(builder.String(), "-")
}
