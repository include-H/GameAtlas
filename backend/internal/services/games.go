package services

import (
	"database/sql"
	"errors"
	"math"
	"strings"

	"github.com/hao/game/internal/config"
	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/files"
	"github.com/hao/game/internal/repositories"
)

var ErrNotFound = errors.New("resource not found")
var ErrValidation = errors.New("validation error")

type GamesService struct {
	gamesRepo     *repositories.GamesRepository
	gameFilesRepo *repositories.GameFilesRepository
	metadataRepo  *repositories.MetadataRepository
	tagsRepo      *repositories.TagsRepository
	fileGuard     *files.Guard
}

type GamesListResult struct {
	Games      []domain.Game
	Page       int
	Limit      int
	Total      int
	TotalPages int
}

type GameDetail struct {
	Game          *domain.Game
	PreviewVideo  *domain.GameAsset
	PreviewVideos []domain.GameAsset
	Screenshots   []domain.GameAsset
	Series        []domain.MetadataItem
	Platforms     []domain.MetadataItem
	Developers    []domain.MetadataItem
	Publishers    []domain.MetadataItem
	Tags          []domain.Tag
	TagGroups     []domain.GameTagGroup
	Files         []domain.GameFile
}

func NewGamesService(cfg config.Config, gamesRepo *repositories.GamesRepository, gameFilesRepo *repositories.GameFilesRepository, metadataRepo *repositories.MetadataRepository, tagsRepo *repositories.TagsRepository) *GamesService {
	roots := cfg.AllowedRoots
	if len(roots) == 0 && strings.TrimSpace(cfg.PrimaryROMRoot) != "" {
		roots = append(roots, cfg.PrimaryROMRoot)
	}

	return &GamesService{
		gamesRepo:     gamesRepo,
		gameFilesRepo: gameFilesRepo,
		metadataRepo:  metadataRepo,
		tagsRepo:      tagsRepo,
		fileGuard:     files.NewGuard(roots),
	}
}

func (s *GamesService) List(params domain.GamesListParams) (*GamesListResult, error) {
	normalizeListParams(&params)
	games, total, err := s.gamesRepo.List(params)
	if err != nil {
		return nil, err
	}

	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(params.Limit)))
	}

	return &GamesListResult{
		Games:      games,
		Page:       params.Page,
		Limit:      params.Limit,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

func (s *GamesService) Stats(params domain.GamesListParams) (*domain.GameStats, error) {
	normalizeListParams(&params)
	return s.gamesRepo.Stats(params)
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
	series, err := s.gamesRepo.ListMetadata("series", "game_series", "series_id", id)
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
	for index := range files {
		_ = s.refreshFileSize(id, &files[index])
	}

	var previewVideo *domain.GameAsset
	if len(videos) > 0 {
		if game.PreviewVideoAssetUID != nil && strings.TrimSpace(*game.PreviewVideoAssetUID) != "" {
			for index := range videos {
				if videos[index].AssetUID == strings.TrimSpace(*game.PreviewVideoAssetUID) {
					previewVideo = &videos[index]
					break
				}
			}
		}
		if previewVideo == nil {
			previewVideo = &videos[0]
		}
	}

	return &GameDetail{
		Game:          game,
		PreviewVideo:  previewVideo,
		PreviewVideos: videos,
		Screenshots:   screenshots,
		Series:        emptyMetadata(series),
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

func (s *GamesService) Update(id int64, input domain.GameWriteInput) (*domain.Game, error) {
	trimmedInput, err := s.validateAndTrimGameInput(input)
	if err != nil {
		return nil, err
	}
	if trimmedInput.PreviewVideoAssetUID != nil && strings.TrimSpace(*trimmedInput.PreviewVideoAssetUID) != "" {
		videos, err := s.gamesRepo.ListVideos(id)
		if err != nil {
			return nil, err
		}
		targetUID := strings.TrimSpace(*trimmedInput.PreviewVideoAssetUID)
		found := false
		for _, video := range videos {
			if video.AssetUID == targetUID {
				found = true
				break
			}
		}
		if !found {
			return nil, ErrValidation
		}
	}
	game, err := s.gamesRepo.Update(id, trimmedInput)
	if err != nil {
		return nil, normalizeRepoError(err)
	}
	if err := s.cleanupUnusedMetadata(); err != nil {
		return nil, err
	}
	return game, nil
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
	targets := []struct {
		table      string
		joinTable  string
		joinColumn string
	}{
		{table: "series", joinTable: "game_series", joinColumn: "series_id"},
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

func validateGameInput(input domain.GameWriteInput) error {
	if strings.TrimSpace(input.Title) == "" {
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
	if err := validateGameInput(input); err != nil {
		return domain.GameWriteInput{}, err
	}

	trimmed := trimGameInput(input)
	if trimmed.TagIDs == nil {
		trimmed.TagIDs = []int64{}
	}

	tagIDs, err := s.tagsRepo.ValidateTagSelection(trimmed.TagIDs)
	if err != nil {
		return domain.GameWriteInput{}, ErrValidation
	}
	trimmed.TagIDs = tagIDs

	return trimmed, nil
}

func trimGameInput(input domain.GameWriteInput) domain.GameWriteInput {
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
	input.PreviewVideoAssetUID = trimStringPtr(input.PreviewVideoAssetUID)
	input.SeriesIDs = uniqueIDs(input.SeriesIDs)
	input.PlatformIDs = uniqueIDs(input.PlatformIDs)
	input.DeveloperIDs = uniqueIDs(input.DeveloperIDs)
	input.PublisherIDs = uniqueIDs(input.PublisherIDs)
	input.TagIDs = uniqueIDs(input.TagIDs)
	return input
}

func normalizeListParams(params *domain.GamesListParams) {
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
	if !params.IncludeAll && strings.TrimSpace(params.Visibility) == "" {
		params.Visibility = domain.GameVisibilityPublic
	}
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
				AllowMultiple: true,
				IsFilterable:  true,
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

func (s *GamesService) refreshFileSize(gameID int64, file *domain.GameFile) error {
	resolved, err := s.fileGuard.ValidateFile(file.FilePath)
	if err != nil {
		return nil
	}

	if file.SizeBytes != nil && *file.SizeBytes == resolved.SizeBytes {
		return nil
	}

	file.SizeBytes = &resolved.SizeBytes
	return s.gameFilesRepo.UpdateSizeBytes(gameID, file.ID, resolved.SizeBytes)
}
