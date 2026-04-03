package handlers

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/services"
)

// 2026-04-03: response mappers and game-specific transport helpers live here
// after splitting the old games.go file by transport concern.
func toGameListItemResponse(game domain.GameListItem) gameListItemResponse {
	return gameListItemResponse{
		ID:                game.ID,
		PublicID:          game.PublicID,
		Title:             game.Title,
		TitleAlt:          game.TitleAlt,
		Visibility:        game.Visibility,
		Summary:           game.Summary,
		ReleaseDate:       game.ReleaseDate,
		Engine:            game.Engine,
		CoverImage:        game.CoverImage,
		BannerImage:       game.BannerImage,
		WikiContent:       game.WikiContent,
		Downloads:         game.Downloads,
		PrimaryScreenshot: game.PrimaryScreenshot,
		ScreenshotCount:   game.ScreenshotCount,
		FileCount:         game.FileCount,
		DeveloperCount:    game.DeveloperCount,
		PublisherCount:    game.PublisherCount,
		PlatformCount:     game.PlatformCount,
		IsFavorite:        game.IsFavorite,
		PendingIssues:     game.PendingIssues,
		CreatedAt:         game.CreatedAt,
		UpdatedAt:         game.UpdatedAt,
	}
}

func toTimelineGameItemResponse(game domain.TimelineGame) timelineGameItemResponse {
	return timelineGameItemResponse{
		ID:          game.ID,
		PublicID:    game.PublicID,
		Title:       game.Title,
		ReleaseDate: game.ReleaseDate,
		CoverImage:  game.CoverImage,
		BannerImage: game.BannerImage,
	}
}

func toGameListItemResponses(games []domain.GameListItem) []gameListItemResponse {
	result := make([]gameListItemResponse, 0, len(games))
	for _, game := range games {
		result = append(result, toGameListItemResponse(game))
	}
	return result
}

func toGameSummaryResponse(game domain.Game) gameListItemResponse {
	return gameListItemResponse{
		ID:          game.ID,
		PublicID:    game.PublicID,
		Title:       game.Title,
		TitleAlt:    game.TitleAlt,
		Visibility:  game.Visibility,
		Summary:     game.Summary,
		ReleaseDate: game.ReleaseDate,
		Engine:      game.Engine,
		CoverImage:  game.CoverImage,
		BannerImage: game.BannerImage,
		WikiContent: game.WikiContent,
		Downloads:   game.Downloads,
		IsFavorite:  game.IsFavorite,
		CreatedAt:   game.CreatedAt,
		UpdatedAt:   game.UpdatedAt,
	}
}

func toSeriesGameSummaryResponses(games []domain.SeriesGameSummary) []gameListItemResponse {
	result := make([]gameListItemResponse, 0, len(games))
	for _, game := range games {
		result = append(result, gameListItemResponse{
			ID:                game.ID,
			PublicID:          game.PublicID,
			Title:             game.Title,
			TitleAlt:          game.TitleAlt,
			Visibility:        game.Visibility,
			Summary:           game.Summary,
			ReleaseDate:       game.ReleaseDate,
			Engine:            game.Engine,
			CoverImage:        game.CoverImage,
			BannerImage:       game.BannerImage,
			WikiContent:       game.WikiContent,
			Downloads:         game.Downloads,
			PrimaryScreenshot: game.PrimaryScreenshot,
			IsFavorite:        game.IsFavorite,
			CreatedAt:         game.CreatedAt,
			UpdatedAt:         game.UpdatedAt,
		})
	}
	return result
}

func parseTimelineCursor(raw string) (string, int64, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", 0, true
	}

	parts := strings.Split(trimmed, "|")
	if len(parts) != 2 {
		return "", 0, false
	}

	releaseDate := strings.TrimSpace(parts[0])
	id, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
	if err != nil || id <= 0 {
		return "", 0, false
	}

	return releaseDate, id, true
}

func formatTimelineCursor(releaseDate string, id int64) string {
	return releaseDate + "|" + strconv.FormatInt(id, 10)
}

func toGameDetailResponse(detail *services.GameDetail, includePaths bool) gameDetailResponse {
	screenshots := make([]gameAssetResponse, 0, len(detail.Screenshots))
	for _, asset := range detail.Screenshots {
		screenshots = append(screenshots, gameAssetResponse{
			ID:        asset.ID,
			AssetUID:  asset.AssetUID,
			Path:      asset.Path,
			SortOrder: asset.SortOrder,
		})
	}

	previewVideos := make([]gameAssetResponse, 0, len(detail.PreviewVideos))
	for _, asset := range detail.PreviewVideos {
		previewVideos = append(previewVideos, gameAssetResponse{
			ID:        asset.ID,
			AssetUID:  asset.AssetUID,
			Path:      asset.Path,
			SortOrder: asset.SortOrder,
		})
	}

	var series *metadataItemResponse
	if detail.Series != nil {
		series = &metadataItemResponse{
			ID:        detail.Series.ID,
			Name:      detail.Series.Name,
			Slug:      detail.Series.Slug,
			SortOrder: detail.Series.SortOrder,
			CreatedAt: detail.Series.CreatedAt,
		}
	}

	return gameDetailResponse{
		ID:            detail.Game.ID,
		PublicID:      detail.Game.PublicID,
		Title:         detail.Game.Title,
		TitleAlt:      detail.Game.TitleAlt,
		Visibility:    detail.Game.Visibility,
		Summary:       detail.Game.Summary,
		ReleaseDate:   detail.Game.ReleaseDate,
		Engine:        detail.Game.Engine,
		CoverImage:    detail.Game.CoverImage,
		BannerImage:   detail.Game.BannerImage,
		WikiContent:   detail.Game.WikiContent,
		Downloads:     detail.Game.Downloads,
		PreviewVideos: previewVideos,
		Screenshots:   screenshots,
		Series:        series,
		Platforms:     toMetadataResponses(detail.Platforms),
		Developers:    toMetadataResponses(detail.Developers),
		Publishers:    toMetadataResponses(detail.Publishers),
		Tags:          toTagResponses(detail.Tags),
		TagGroups:     toGameTagGroupResponses(detail.TagGroups),
		Files:         toGameFileResponses(detail.Files, includePaths),
		IsFavorite:    detail.Game.IsFavorite,
		PendingIssues: detail.PendingIssues,
		CreatedAt:     detail.Game.CreatedAt,
		UpdatedAt:     detail.Game.UpdatedAt,
	}
}

func toMetadataResponses(items []domain.MetadataItem) []metadataItemResponse {
	result := make([]metadataItemResponse, 0, len(items))
	for _, item := range items {
		result = append(result, metadataItemResponse{
			ID:        item.ID,
			Name:      item.Name,
			Slug:      item.Slug,
			SortOrder: item.SortOrder,
			CreatedAt: item.CreatedAt,
		})
	}
	return result
}

func toGameFileResponses(items []domain.GameFile, includePaths bool) []gameFileResponse {
	result := make([]gameFileResponse, 0, len(items))
	for _, item := range items {
		response := gameFileResponse{
			ID:              item.ID,
			GameID:          item.GameID,
			FileName:        filepath.Base(item.FilePath),
			Label:           item.Label,
			Notes:           item.Notes,
			SizeBytes:       item.SizeBytes,
			SortOrder:       item.SortOrder,
			CreatedAt:       item.CreatedAt,
			UpdatedAt:       item.UpdatedAt,
			SourceCreatedAt: item.SourceCreatedAt,
		}
		if includePaths {
			response.FilePath = item.FilePath
		}
		result = append(result, response)
	}
	return result
}

func toTagResponses(items []domain.Tag) []tagResponse {
	result := make([]tagResponse, 0, len(items))
	for _, item := range items {
		result = append(result, tagResponse{
			ID:        item.ID,
			GroupID:   item.GroupID,
			GroupKey:  item.GroupKey,
			GroupName: item.GroupName,
			Name:      item.Name,
			Slug:      item.Slug,
			ParentID:  item.ParentID,
			SortOrder: item.SortOrder,
			IsActive:  item.IsActive,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
		})
	}
	return result
}

func toGameTagGroupResponses(items []domain.GameTagGroup) []gameTagGroupResponse {
	result := make([]gameTagGroupResponse, 0, len(items))
	for _, item := range items {
		result = append(result, gameTagGroupResponse{
			ID:            item.ID,
			Key:           item.Key,
			Name:          item.Name,
			AllowMultiple: item.AllowMultiple,
			IsFilterable:  item.IsFilterable,
			Tags:          toTagResponses(item.Tags),
		})
	}
	return result
}
