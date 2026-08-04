package services

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/repositories"
)

type MetadataService struct {
	repo      *repositories.MetadataRepository
	listCache sync.Map // table -> cachedMetadataList
}

type cachedMetadataList struct {
	items    []domain.MetadataItem
	cachedAt time.Time
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
}

type SeriesDetail struct {
	Series *domain.MetadataItem
	Games  []domain.MetadataGameSummary
}

type PublisherDetail struct {
	Publisher *domain.MetadataItem
	Games     []domain.MetadataGameSummary
}

func NewMetadataService(repo *repositories.MetadataRepository) *MetadataService {
	return &MetadataService{
		repo: repo,
	}
}

func (s *MetadataService) invalidateListCache(types ...domain.MetadataType) {
	for _, typ := range types {
		s.listCache.Delete(typ)
	}
}

func (s *MetadataService) List(resource MetadataResource, includeAll bool, options MetadataListOptions) ([]domain.MetadataItem, error) {
	// Check cache first
	var items []domain.MetadataItem
	if cached, ok := s.listCache.Load(resource.Type); ok {
		entry := cached.(cachedMetadataList)
		if time.Since(entry.cachedAt) < 60*time.Second {
			items = make([]domain.MetadataItem, len(entry.items))
			copy(items, entry.items)
		}
	}

	if items == nil {
		var err error
		items, err = s.repo.List(resource.Type)
		if err != nil {
			return nil, err
		}
		s.listCache.Store(resource.Type, cachedMetadataList{items: items, cachedAt: time.Now()})
	}
	if supportsMetadataGameGrouping(resource.Type) {
		if err := s.enrichMetadataItems(items, resource.Type, includeAll); err != nil {
			return nil, err
		}
		filtered := make([]domain.MetadataItem, 0, len(items))
		for index := range items {
			if includeAll || items[index].GameCount > 0 {
				filtered = append(filtered, items[index])
			}
		}
		items = filtered
	}
	items = filterMetadataItems(items, options)
	if items == nil {
		return []domain.MetadataItem{}, nil
	}
	return items, nil
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
		s.invalidateListCache(resource.Type)
		return result, nil
	default:
		return nil, fmt.Errorf("unsupported metadata resource type: %d", resource.Type)
	}
}

func (s *MetadataService) GetSeriesDetail(id int64, includeAll bool) (*SeriesDetail, error) {
	detail, err := s.getMetadataDetail(domain.MetadataSeries, id, includeAll)
	if err != nil {
		return nil, err
	}

	return &SeriesDetail{
		Series: detail.Item,
		Games:  detail.Games,
	}, nil
}

func (s *MetadataService) GetPublisherDetail(id int64, includeAll bool) (*PublisherDetail, error) {
	detail, err := s.getMetadataDetail(domain.MetadataPublishers, id, includeAll)
	if err != nil {
		return nil, err
	}

	return &PublisherDetail{
		Publisher: detail.Item,
		Games:     detail.Games,
	}, nil
}

type metadataDetail struct {
	Item  *domain.MetadataItem
	Games []domain.MetadataGameSummary
}

func supportsMetadataGameGrouping(typ domain.MetadataType) bool {
	return typ == domain.MetadataSeries || typ == domain.MetadataPublishers
}

func (s *MetadataService) getMetadataDetail(typ domain.MetadataType, id int64, includeAll bool) (*metadataDetail, error) {
	if !supportsMetadataGameGrouping(typ) {
		return nil, fmt.Errorf("unsupported metadata detail type: %d", typ)
	}

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

	games, err := s.repo.ListMetadataGames(typ, id, includeAll)
	if err != nil {
		return nil, err
	}

	return &metadataDetail{Item: item, Games: games}, nil
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

func filterMetadataItems(items []domain.MetadataItem, options MetadataListOptions) []domain.MetadataItem {
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
