package services

import (
	"database/sql"
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
	// These resources share the same CRUD shape at the HTTP/service boundary,
	// but they are still "game-attached metadata" in product semantics rather
	// than permanently curated master data. They can be pre-created for form
	// input convenience and are expected to be auto-pruned once no games refer
	// to them anymore.
	Type domain.MetadataType
}

type MetadataListOptions struct {
	Search string
	Limit  int
	Sort   string
	Page   int
}

type MetadataListResult struct {
	Items      []domain.MetadataItem
	Page       int
	Limit      int
	Total      int
	TotalPages int
}

type SeriesDetail struct {
	Series     *domain.MetadataItem
	Games      []domain.MetadataGameSummary
	Page       int
	Limit      int
	Total      int
	TotalPages int
}

type PublisherDetail struct {
	Publisher  *domain.MetadataItem
	Games      []domain.MetadataGameSummary
	Page       int
	Limit      int
	Total      int
	TotalPages int
}

type MetadataDetailOptions struct {
	Page  int
	Limit int
}

func NewMetadataService(repo *repositories.MetadataRepository) *MetadataService {
	return &MetadataService{
		repo: repo,
	}
}

func (s *MetadataService) ListPage(resource MetadataResource, includeAll bool, options MetadataListOptions) (*MetadataListResult, error) {
	page, limit := normalizeMetadataPagination(options.Page, options.Limit)
	search := strings.TrimSpace(options.Search)

	total, err := s.repo.CountMetadata(resource.Type, search, includeAll)
	if err != nil {
		return nil, err
	}

	offset := (page - 1) * limit
	items, err := s.repo.ListMetadataPage(resource.Type, search, options.Sort, limit, offset, includeAll)
	if err != nil {
		return nil, err
	}
	if supportsMetadataGameGrouping(resource.Type) && len(items) > 0 {
		if err := s.enrichMetadataItems(items, resource.Type, includeAll); err != nil {
			return nil, err
		}
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + limit - 1) / limit
	}

	return &MetadataListResult{
		Items:      items,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

func (s *MetadataService) Create(resource MetadataResource, input domain.MetadataWriteInput) (*domain.MetadataItem, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, domain.ErrValidation
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
		return nil, domain.ErrValidation
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

	switch resource.Type {
	case domain.MetadataSeries, domain.MetadataDevelopers, domain.MetadataPublishers:
		result, err := s.repo.CreateOrGet(resource.Type, cleanInput, slugValue, sortOrder)
		if err != nil {
			return nil, err
		}
		return result, nil
	default:
		return nil, fmt.Errorf("unsupported metadata resource type: %d", resource.Type)
	}
}

func (s *MetadataService) GetSeriesDetail(id int64, includeAll bool, options MetadataDetailOptions) (*SeriesDetail, error) {
	detail, err := s.getMetadataDetail(domain.MetadataSeries, id, includeAll, options)
	if err != nil {
		return nil, err
	}

	return &SeriesDetail{
		Series:     detail.Item,
		Games:      detail.Games,
		Page:       detail.Page,
		Limit:      detail.Limit,
		Total:      detail.Total,
		TotalPages: detail.TotalPages,
	}, nil
}

func (s *MetadataService) GetPublisherDetail(id int64, includeAll bool, options MetadataDetailOptions) (*PublisherDetail, error) {
	detail, err := s.getMetadataDetail(domain.MetadataPublishers, id, includeAll, options)
	if err != nil {
		return nil, err
	}

	return &PublisherDetail{
		Publisher:  detail.Item,
		Games:      detail.Games,
		Page:       detail.Page,
		Limit:      detail.Limit,
		Total:      detail.Total,
		TotalPages: detail.TotalPages,
	}, nil
}

type metadataDetail struct {
	Item       *domain.MetadataItem
	Games      []domain.MetadataGameSummary
	Page       int
	Limit      int
	Total      int
	TotalPages int
}

func supportsMetadataGameGrouping(typ domain.MetadataType) bool {
	return typ == domain.MetadataSeries || typ == domain.MetadataPublishers
}

func (s *MetadataService) getMetadataDetail(typ domain.MetadataType, id int64, includeAll bool, options MetadataDetailOptions) (*metadataDetail, error) {
	if !supportsMetadataGameGrouping(typ) {
		return nil, fmt.Errorf("unsupported metadata detail type: %d", typ)
	}

	page, limit := normalizeMetadataPagination(options.Page, options.Limit)
	item, err := s.repo.Get(typ, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	if err := s.enrichMetadataItem(item, typ, includeAll); err != nil {
		return nil, err
	}
	if !includeAll && item.GameCount == 0 {
		return nil, domain.ErrNotFound
	}

	total, err := s.repo.CountMetadataGames(typ, id, includeAll)
	if err != nil {
		return nil, err
	}

	offset := (page - 1) * limit
	games, err := s.repo.ListMetadataGamesPage(typ, id, includeAll, limit, offset)
	if err != nil {
		return nil, err
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + limit - 1) / limit
	}

	return &metadataDetail{
		Item:       item,
		Games:      games,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

func normalizeMetadataPagination(page int, limit int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 24
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastDash := false

	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
			lastDash = false
		} else if builder.Len() > 0 && !lastDash {
			builder.WriteRune('-')
			lastDash = true
		}
	}

	return strings.Trim(builder.String(), "-")
}

func (s *MetadataService) enrichMetadataItem(item *domain.MetadataItem, typ domain.MetadataType, includeAll bool) error {
	games, err := s.repo.ListMetadataGames(typ, item.ID, includeAll)
	if err != nil {
		return err
	}

	applyMetadataItemGames(item, games)
	return nil
}

func (s *MetadataService) enrichMetadataItems(items []domain.MetadataItem, typ domain.MetadataType, includeAll bool) error {
	if len(items) == 0 {
		return nil
	}

	metadataIDs := make([]int64, 0, len(items))
	for _, item := range items {
		metadataIDs = append(metadataIDs, item.ID)
	}

	gamesByMetadataID, err := s.repo.ListMetadataGamesByIDs(typ, metadataIDs, includeAll)
	if err != nil {
		return err
	}

	for index := range items {
		applyMetadataItemGames(&items[index], gamesByMetadataID[items[index].ID])
	}

	return nil
}

func applyMetadataItemGames(item *domain.MetadataItem, games []domain.MetadataGameSummary) {
	item.GameCount = len(games)
	item.LatestUpdatedAt = nil
	item.CoverCandidates = nil
	item.CoverImage = nil
	item.BackgroundCandidates = nil
	if len(games) == 0 {
		return
	}

	item.LatestUpdatedAt = &games[0].UpdatedAt
	coverCandidates := make([]string, 0, 4)
	backgroundCandidates := make([]string, 0, 4)
	seen := make(map[string]struct{}, 8)
	bgSeen := make(map[string]struct{}, 8)
	for _, game := range games {
		path := pickMetadataCoverSource(game)
		if path != "" {
			if _, exists := seen[path]; !exists {
				seen[path] = struct{}{}
				coverCandidates = append(coverCandidates, path)
			}
		}

		// Collect landscape images (banner + screenshot) for ambient background
		for _, bg := range pickMetadataBackgroundSources(game) {
			if _, exists := bgSeen[bg]; !exists {
				bgSeen[bg] = struct{}{}
				backgroundCandidates = append(backgroundCandidates, bg)
			}
		}

		if len(coverCandidates) >= 4 && len(backgroundCandidates) >= 4 {
			break
		}
	}

	if len(coverCandidates) > 0 {
		item.CoverCandidates = coverCandidates[:min(len(coverCandidates), 4)]
		item.CoverImage = &coverCandidates[0]
	}
	if len(backgroundCandidates) > 0 {
		item.BackgroundCandidates = backgroundCandidates[:min(len(backgroundCandidates), 4)]
	}
}

func pickMetadataBackgroundSources(game domain.MetadataGameSummary) []string {
	var sources []string
	if game.BannerImage != nil && strings.TrimSpace(*game.BannerImage) != "" {
		sources = append(sources, strings.TrimSpace(*game.BannerImage))
	}
	if game.PrimaryScreenshot != nil && strings.TrimSpace(*game.PrimaryScreenshot) != "" {
		sources = append(sources, strings.TrimSpace(*game.PrimaryScreenshot))
	}
	return sources
}

func pickMetadataCoverSource(game domain.MetadataGameSummary) string {
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
