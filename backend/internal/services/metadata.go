package services

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/repositories"
)

type MetadataService struct {
	repo *repositories.MetadataRepository
}

type MetadataResource struct {
	Table        string
	ResourceName string
}

func NewMetadataService(repo *repositories.MetadataRepository) *MetadataService {
	return &MetadataService{repo: repo}
}

func (s *MetadataService) List(resource MetadataResource) ([]domain.MetadataItem, error) {
	items, err := s.repo.List(resource.Table)
	if err != nil {
		return nil, err
	}
	if items == nil {
		return []domain.MetadataItem{}, nil
	}
	return items, nil
}

func (s *MetadataService) Create(resource MetadataResource, input domain.MetadataWriteInput) (*domain.MetadataItem, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ErrValidation
	}

	slug := trimStringPtr(input.Slug)
	slugValue := ""
	if slug != nil {
		slugValue = slugify(*slug)
	}
	if slugValue == "" {
		slugValue = slugify(name)
	}
	if slugValue == "" {
		return nil, ErrValidation
	}

	sortOrder := 0
	if input.SortOrder != nil {
		sortOrder = *input.SortOrder
	}

	cleanInput := domain.MetadataWriteInput{
		Name:      name,
		Slug:      &slugValue,
		SortOrder: &sortOrder,
	}

	switch resource.Table {
	case "series":
		return s.repo.CreateSeries(cleanInput, slugValue, sortOrder)
	case "platforms", "developers", "publishers":
		return s.repo.CreateSimple(resource.Table, cleanInput, slugValue, sortOrder)
	default:
		return nil, fmt.Errorf("unsupported metadata resource: %s", resource.Table)
	}
}

func slugify(value string) string {
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

	result := strings.Trim(builder.String(), "-")
	return result
}
