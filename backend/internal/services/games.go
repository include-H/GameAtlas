package services

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/hao/game/internal/config"
	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/files"
	"github.com/hao/game/internal/repositories"
)

var ErrNotFound = errors.New("resource not found")
var ErrValidation = errors.New("validation error")

type GamesService struct {
	gamesRepo                *repositories.GamesRepository
	gameFilesRepo            *repositories.GameFilesRepository
	metadataRepo             *repositories.MetadataRepository
	tagsRepo                 *repositories.TagsRepository
	reviewIssueOverridesRepo *repositories.ReviewIssueOverrideRepository
	fileGuard                *files.Guard
	assetStore               *files.AssetStore
}

type GamesListResult struct {
	Games              []domain.Game
	Page               int
	Limit              int
	Total              int
	TotalPages         int
	PendingIssueCounts *domain.PendingIssueCountSummary
}

type TimelineCursor struct {
	ReleaseDate string
	ID          int64
}

type GamesTimelineResult struct {
	Games      []domain.TimelineGame
	Limit      int
	FromDate   string
	ToDate     string
	HasMore    bool
	NextCursor *TimelineCursor
}

type GameDetail struct {
	Game          *domain.Game
	PendingIssues *domain.PendingIssueEvaluation
	PreviewVideo  *domain.GameAsset
	PreviewVideos []domain.GameAsset
	Screenshots   []domain.GameAsset
	Series        *domain.MetadataItem
	Platforms     []domain.MetadataItem
	Developers    []domain.MetadataItem
	Publishers    []domain.MetadataItem
	Tags          []domain.Tag
	TagGroups     []domain.GameTagGroup
	Files         []domain.GameFile
}

func NewGamesService(cfg config.Config, gamesRepo *repositories.GamesRepository, gameFilesRepo *repositories.GameFilesRepository, metadataRepo *repositories.MetadataRepository, tagsRepo *repositories.TagsRepository, reviewIssueOverridesRepo ...*repositories.ReviewIssueOverrideRepository) *GamesService {
	var overridesRepo *repositories.ReviewIssueOverrideRepository
	if len(reviewIssueOverridesRepo) > 0 {
		overridesRepo = reviewIssueOverridesRepo[0]
	}

	return &GamesService{
		gamesRepo:                gamesRepo,
		gameFilesRepo:            gameFilesRepo,
		metadataRepo:             metadataRepo,
		tagsRepo:                 tagsRepo,
		reviewIssueOverridesRepo: overridesRepo,
		fileGuard:                files.NewGuard(cfg.PrimaryROMRoot),
		assetStore:               files.NewAssetStore(cfg.AssetsDir, cfg.Proxy, 30*time.Second),
	}
}

func (s *GamesService) List(params domain.GamesListParams) (*GamesListResult, error) {
	if err := normalizeListParams(&params); err != nil {
		return nil, err
	}
	games, total, err := s.gamesRepo.List(params)
	if err != nil {
		return nil, err
	}

	var pendingIssueCounts *domain.PendingIssueCountSummary
	if params.PendingOnly {
		counts, err := s.gamesRepo.CountPendingGroups(params)
		if err != nil {
			return nil, err
		}
		pendingIssueCounts = &domain.PendingIssueCountSummary{
			Groups: map[domain.PendingIssueKey]int{
				domain.PendingIssueMissingAssets:   counts.MissingAssets,
				domain.PendingIssueMissingWiki:     counts.MissingWiki,
				domain.PendingIssueMissingFiles:    counts.MissingFiles,
				domain.PendingIssueMissingMetadata: counts.MissingMetadata,
			},
			IgnoredTotal: counts.IgnoredTotal,
		}
	}

	if err := s.attachPendingIssues(games); err != nil {
		return nil, err
	}

	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(params.Limit)))
	}

	return &GamesListResult{
		Games:              games,
		Page:               params.Page,
		Limit:              params.Limit,
		Total:              total,
		TotalPages:         totalPages,
		PendingIssueCounts: pendingIssueCounts,
	}, nil
}

func (s *GamesService) Stats(params domain.GamesListParams) (*domain.GameStats, error) {
	if err := normalizeListParams(&params); err != nil {
		return nil, err
	}
	return s.gamesRepo.Stats(params)
}

func (s *GamesService) ListTimeline(params domain.GamesTimelineParams) (*GamesTimelineResult, error) {
	if err := normalizeTimelineParams(&params); err != nil {
		return nil, err
	}

	games, hasMoreInWindow, err := s.gamesRepo.ListTimeline(params)
	if err != nil {
		return nil, err
	}

	hasOlderWindow := false
	if !hasMoreInWindow && params.CursorReleaseDate == "" && len(games) > 0 {
		last := games[len(games)-1]
		if last.ReleaseDate != nil {
			exists, err := s.gamesRepo.HasOlderTimelineGame(params, *last.ReleaseDate, last.ID)
			if err != nil {
				return nil, err
			}
			hasOlderWindow = exists
		}
	}

	var nextCursor *TimelineCursor
	if hasMoreInWindow && len(games) > 0 {
		last := games[len(games)-1]
		if last.ReleaseDate != nil {
			nextCursor = &TimelineCursor{
				ReleaseDate: *last.ReleaseDate,
				ID:          last.ID,
			}
		}
	}

	return &GamesTimelineResult{
		Games:      games,
		Limit:      params.Limit,
		FromDate:   params.FromDate,
		ToDate:     params.ToDate,
		HasMore:    hasMoreInWindow || hasOlderWindow,
		NextCursor: nextCursor,
	}, nil
}

func (s *GamesService) LatestTimelineReleaseDate(includeAll bool) (string, bool, error) {
	releaseDate, err := s.gamesRepo.LatestTimelineReleaseDate(includeAll, domain.GameVisibilityPublic)
	if err != nil {
		return "", false, err
	}
	if releaseDate == nil {
		return "", false, nil
	}

	normalized, err := normalizeTimelineDate(*releaseDate)
	if err != nil {
		return "", false, nil
	}

	return normalized, true, nil
}

func (s *GamesService) ResolveGameID(publicID string) (int64, error) {
	id, err := s.gamesRepo.ResolveIDByPublicID(publicID)
	if err != nil {
		return 0, normalizeRepoError(err)
	}
	return id, nil
}

func (s *GamesService) GetDetail(id int64, includeAll bool) (*GameDetail, error) {
	game, err := s.gamesRepo.GetByID(id)
	if err != nil {
		return nil, normalizeRepoError(err)
	}
	if !includeAll && game.Visibility == domain.GameVisibilityPrivate {
		return nil, ErrNotFound
	}

	screenshots, err := s.gamesRepo.ListScreenshots(id)
	if err != nil {
		return nil, err
	}
	videos, err := s.gamesRepo.ListVideos(id)
	if err != nil {
		return nil, err
	}
	primarySeries, err := s.gamesRepo.GetSeriesMetadata(id)
	if err != nil {
		return nil, err
	}
	platforms, err := s.gamesRepo.ListMetadata("platforms", "game_platforms", "platform_id", id)
	if err != nil {
		return nil, err
	}
	developers, err := s.gamesRepo.ListMetadata("developers", "game_developers", "developer_id", id)
	if err != nil {
		return nil, err
	}
	publishers, err := s.gamesRepo.ListMetadata("publishers", "game_publishers", "publisher_id", id)
	if err != nil {
		return nil, err
	}
	tags, err := s.tagsRepo.ListByGameID(id)
	if err != nil {
		return nil, err
	}
	files, err := s.gameFilesRepo.ListByGameID(id)
	if err != nil {
		return nil, err
	}
	pendingIssues, err := s.getPendingIssueEvaluation(*game)
	if err != nil {
		return nil, err
	}

	var previewVideo *domain.GameAsset
	if len(videos) > 0 {
		// Preview video is derived from the first sorted video only.
		previewVideo = &videos[0]
	}

	return &GameDetail{
		Game:          game,
		PendingIssues: pendingIssues,
		PreviewVideo:  previewVideo,
		PreviewVideos: videos,
		Screenshots:   screenshots,
		Series:        primarySeries,
		Platforms:     emptyMetadata(platforms),
		Developers:    emptyMetadata(developers),
		Publishers:    emptyMetadata(publishers),
		Tags:          emptyTags(tags),
		TagGroups:     groupGameTags(tags),
		Files:         emptyFiles(files),
	}, nil
}

func (s *GamesService) Create(input domain.GameWriteInput) (*domain.Game, error) {
	trimmedInput, err := s.validateAndTrimGameInput(input)
	if err != nil {
		return nil, err
	}
	game, err := s.gamesRepo.Create(trimmedInput)
	if err != nil {
		return nil, err
	}
	return game, nil
}

func (s *GamesService) UpdateAggregate(id int64, input domain.GameAggregateUpdateInput) (*domain.Game, []string, error) {
	trimmedInput, err := s.validateAndTrimGameAggregatePatchInput(input.Game)
	if err != nil {
		return nil, nil, err
	}

	normalizedFiles := make([]domain.GameFileUpsertInput, 0, len(input.Assets.Files))
	for index, item := range input.Assets.Files {
		fileInput := domain.GameFileWriteInput{
			FilePath:  item.FilePath,
			Label:     item.Label,
			Notes:     item.Notes,
			SortOrder: item.SortOrder,
		}
		if err := validateGameFileInput(fileInput); err != nil {
			return nil, nil, err
		}

		trimmedFileInput := trimGameFileInput(fileInput)
		resolved, err := s.fileGuard.ValidateFile(trimmedFileInput.FilePath)
		if err != nil {
			return nil, nil, normalizeFileError(err)
		}
		normalizedFiles = append(normalizedFiles, domain.GameFileUpsertInput{
			ID:        item.ID,
			FilePath:  resolved.ResolvedPath,
			Label:     trimmedFileInput.Label,
			Notes:     trimmedFileInput.Notes,
			SortOrder: index,
		})
	}

	for _, item := range input.Assets.DeleteAssets {
		assetType := strings.TrimSpace(item.AssetType)
		switch assetType {
		case "cover", "banner", "screenshot", "video":
		default:
			return nil, nil, ErrValidation
		}
	}

	deletedAssetPaths, err := s.gamesRepo.UpdateAggregate(id, domain.GameAggregateUpdateInput{
		Game: trimmedInput,
		Assets: domain.GameAggregateAssetsInput{
			Files:                    normalizedFiles,
			DeleteAssets:             input.Assets.DeleteAssets,
			ScreenshotOrderAssetUIDs: input.Assets.ScreenshotOrderAssetUIDs,
			VideoOrderAssetUIDs:      input.Assets.VideoOrderAssetUIDs,
		},
	})
	if err != nil {
		return nil, nil, normalizeRepoError(err)
	}

	if err := s.cleanupUnusedMetadata(); err != nil {
		return nil, nil, err
	}

	assetDeleteWarnings := make([]string, 0)
	for _, path := range deletedAssetPaths {
		if err := s.assetStore.DeleteAsset(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Printf("aggregate update: failed to delete asset file %s: %v", path, err)
			assetDeleteWarnings = append(assetDeleteWarnings, path)
		}
	}

	game, err := s.gamesRepo.GetByID(id)
	if err != nil {
		return nil, nil, normalizeRepoError(err)
	}

	return game, assetDeleteWarnings, nil
}

func (s *GamesService) Delete(id int64) error {
	deleted, err := s.gamesRepo.Delete(id)
	if err != nil {
		return err
	}
	if !deleted {
		return ErrNotFound
	}
	if err := s.cleanupUnusedMetadata(); err != nil {
		return err
	}
	return nil
}

func (s *GamesService) cleanupUnusedMetadata() error {
	if err := s.metadataRepo.DeleteUnusedSeries(); err != nil {
		return err
	}

	targets := []struct {
		table      string
		joinTable  string
		joinColumn string
	}{
		{table: "platforms", joinTable: "game_platforms", joinColumn: "platform_id"},
		{table: "developers", joinTable: "game_developers", joinColumn: "developer_id"},
		{table: "publishers", joinTable: "game_publishers", joinColumn: "publisher_id"},
	}

	for _, target := range targets {
		if err := s.metadataRepo.DeleteUnused(target.table, target.joinTable, target.joinColumn); err != nil {
			return err
		}
	}

	return nil
}

func (s *GamesService) attachPendingIssues(games []domain.Game) error {
	if len(games) == 0 {
		return nil
	}
	if s.reviewIssueOverridesRepo == nil {
		for index := range games {
			evaluation := domain.EvaluatePendingIssues(games[index], nil)
			games[index].PendingIssues = &evaluation
		}
		return nil
	}

	gameIDs := make([]int64, 0, len(games))
	for _, game := range games {
		gameIDs = append(gameIDs, game.ID)
	}

	overrides, err := s.reviewIssueOverridesRepo.List(gameIDs)
	if err != nil {
		return err
	}

	ignoredReasonsByGameID := make(map[int64]map[domain.PendingIssueDetailKey]*string, len(games))
	for _, item := range overrides {
		if item.Status != "ignored" || !domain.IsAllowedPendingIssueDetail(item.IssueKey) {
			continue
		}
		if ignoredReasonsByGameID[item.GameID] == nil {
			ignoredReasonsByGameID[item.GameID] = make(map[domain.PendingIssueDetailKey]*string)
		}
		ignoredReasonsByGameID[item.GameID][domain.PendingIssueDetailKey(item.IssueKey)] = item.Reason
	}

	for index := range games {
		evaluation := domain.EvaluatePendingIssues(games[index], ignoredReasonsByGameID[games[index].ID])
		games[index].PendingIssues = &evaluation
	}

	return nil
}

func (s *GamesService) getPendingIssueEvaluation(game domain.Game) (*domain.PendingIssueEvaluation, error) {
	if s.reviewIssueOverridesRepo == nil {
		evaluation := domain.EvaluatePendingIssues(game, nil)
		return &evaluation, nil
	}

	overrides, err := s.reviewIssueOverridesRepo.List([]int64{game.ID})
	if err != nil {
		return nil, err
	}

	ignoredReasons := make(map[domain.PendingIssueDetailKey]*string)
	for _, item := range overrides {
		if item.Status != "ignored" || !domain.IsAllowedPendingIssueDetail(item.IssueKey) {
			continue
		}
		ignoredReasons[domain.PendingIssueDetailKey(item.IssueKey)] = item.Reason
	}

	evaluation := domain.EvaluatePendingIssues(game, ignoredReasons)
	return &evaluation, nil
}

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

func (s *GamesService) validateAndTrimGameInput(input domain.GameWriteInput) (domain.GameWriteInput, error) {
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

	tagIDs, err := s.tagsRepo.ValidateTagSelection(input.TagIDs)
	if err != nil {
		return domain.GameWriteInput{}, ErrValidation
	}
	input.TagIDs = tagIDs

	return input, nil
}

func (s *GamesService) validateAndTrimGameAggregatePatchInput(input domain.GameAggregatePatchInput) (domain.GameAggregatePatchInput, error) {
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
		tagIDs, err := s.tagsRepo.ValidateTagSelection(input.TagIDs.Values)
		if err != nil {
			return domain.GameAggregatePatchInput{}, ErrValidation
		}
		input.TagIDs.Values = tagIDs
	}

	return input, nil
}

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

func normalizeRepoError(err error) error {
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, sqlxErrNotFound()) {
		return ErrNotFound
	}
	return err
}

func emptyMetadata(items []domain.MetadataItem) []domain.MetadataItem {
	if items == nil {
		return []domain.MetadataItem{}
	}
	return items
}

func emptyTags(items []domain.Tag) []domain.Tag {
	if items == nil {
		return []domain.Tag{}
	}
	return items
}

func groupGameTags(tags []domain.Tag) []domain.GameTagGroup {
	if len(tags) == 0 {
		return []domain.GameTagGroup{}
	}

	groups := make([]domain.GameTagGroup, 0)
	indexByGroupID := map[int64]int{}

	for _, tag := range tags {
		index, ok := indexByGroupID[tag.GroupID]
		if !ok {
			groups = append(groups, domain.GameTagGroup{
				ID:            tag.GroupID,
				Key:           tag.GroupKey,
				Name:          tag.GroupName,
				AllowMultiple: tag.GroupAllowMultiple,
				IsFilterable:  tag.GroupIsFilterable,
				Tags:          []domain.Tag{},
			})
			index = len(groups) - 1
			indexByGroupID[tag.GroupID] = index
		}
		groups[index].Tags = append(groups[index].Tags, tag)
	}

	return groups
}

func uniqueIDs(ids []int64) []int64 {
	if len(ids) == 0 {
		return []int64{}
	}

	seen := make(map[int64]struct{}, len(ids))
	result := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}

	return result
}

func emptyFiles(items []domain.GameFile) []domain.GameFile {
	if items == nil {
		return []domain.GameFile{}
	}
	return items
}

func trimStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func sqlxErrNotFound() error {
	return sql.ErrNoRows
}
