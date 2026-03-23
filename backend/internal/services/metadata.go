package services

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/hao/game/internal/config"
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

type MetadataListOptions struct {
	Search string
	Limit  int
	Sort   string
}

type SeriesDetail struct {
	Series *domain.MetadataItem
	Games  []domain.Game
}

func NewMetadataService(cfg config.Config, repo *repositories.MetadataRepository) *MetadataService {
	return &MetadataService{
		repo: repo,
	}
}

func (s *MetadataService) List(resource MetadataResource, includeAll bool, options MetadataListOptions) ([]domain.MetadataItem, error) {
	items, err := s.repo.List(resource.Table)
	if err != nil {
		return nil, err
	}
	if resource.Table == "series" {
		filtered := make([]domain.MetadataItem, 0, len(items))
		for index := range items {
			s.enrichSeriesItem(&items[index], includeAll)
			if includeAll || items[index].GameCount > 0 {
				filtered = append(filtered, items[index])
			}
		}
		filtered = filterSeriesItems(filtered, options)
		items = filtered
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
		existing, err := s.repo.FindSimpleByName(resource.Table, name)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return existing, nil
		}
		return s.repo.CreateSimple(resource.Table, cleanInput, slugValue, sortOrder)
	default:
		return nil, fmt.Errorf("unsupported metadata resource: %s", resource.Table)
	}
}

func (s *MetadataService) GetSeriesDetail(id int64, includeAll bool) (*SeriesDetail, error) {
	item, err := s.repo.Get("series", id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	s.enrichSeriesItem(item, includeAll)
	if !includeAll && item.GameCount == 0 {
		return nil, ErrNotFound
	}

	games, err := s.repo.ListSeriesGames(id, includeAll)
	if err != nil {
		return nil, err
	}

	return &SeriesDetail{
		Series: item,
		Games:  games,
	}, nil
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

func filterSeriesItems(items []domain.MetadataItem, options MetadataListOptions) []domain.MetadataItem {
	search := strings.ToLower(strings.TrimSpace(options.Search))
	if search != "" {
		filtered := make([]domain.MetadataItem, 0, len(items))
		for _, item := range items {
			if strings.Contains(strings.ToLower(item.Name), search) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}

	sortKey := strings.TrimSpace(strings.ToLower(options.Sort))
	switch sortKey {
	case "popular":
		sort.SliceStable(items, func(i, j int) bool {
			if items[i].GameCount != items[j].GameCount {
				return items[i].GameCount > items[j].GameCount
			}
			return strings.Compare(items[i].Name, items[j].Name) < 0
		})
	default:
		sort.SliceStable(items, func(i, j int) bool {
			return strings.Compare(items[i].Name, items[j].Name) < 0
		})
	}

	if options.Limit > 0 && len(items) > options.Limit {
		items = items[:options.Limit]
	}

	return items
}

func (s *MetadataService) enrichSeriesItem(item *domain.MetadataItem, includeAll bool) {
	games, err := s.repo.ListSeriesGames(item.ID, includeAll)
	if err != nil {
		return
	}

	item.GameCount = len(games)
	if len(games) == 0 {
		return
	}

	item.LatestUpdatedAt = &games[0].UpdatedAt
	coverCandidates := make([]string, 0, 4)
	seen := make(map[string]struct{}, 4)
	for _, game := range games {
		path := pickSeriesCoverSource(game)
		if path == "" {
			continue
		}
		if _, exists := seen[path]; exists {
			continue
		}
		seen[path] = struct{}{}
		coverCandidates = append(coverCandidates, path)
		if len(coverCandidates) == 4 {
			break
		}
	}

	if len(coverCandidates) > 0 {
		item.CoverCandidates = coverCandidates
		item.CoverImage = &coverCandidates[0]
	}
}

func pickSeriesCoverSource(game domain.Game) string {
	if game.CoverImage != nil && strings.TrimSpace(*game.CoverImage) != "" {
		return strings.TrimSpace(*game.CoverImage)
	}
	if game.BannerImage != nil && strings.TrimSpace(*game.BannerImage) != "" {
		return strings.TrimSpace(*game.BannerImage)
	}
	if game.PrimaryScreenshot != nil && strings.TrimSpace(*game.PrimaryScreenshot) != "" {
		return strings.TrimSpace(*game.PrimaryScreenshot)
	}
	return ""
}
