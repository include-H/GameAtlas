package handlers

import (
	"path/filepath"

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
		CoverImage:        game.CoverImage,
		BannerImage:       game.BannerImage,
		WikiContent:       game.WikiContent,
		Downloads:         game.Downloads,
		PrimaryScreenshot: game.PrimaryScreenshot,
		ScreenshotCount:   game.ScreenshotCount,
		LogoVisible:       game.LogoVisible,
		FileCount:         game.FileCount,
		DeveloperCount:    game.DeveloperCount,
		PublisherCount:    game.PublisherCount,
		IsFavorite:        game.IsFavorite,
		PendingIssues:     toPendingIssueEvaluationResponse(game.PendingIssues),
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

// 2026-05-09: toGameSummaryResponse intentionally reuses gameListItemResponse
// but only populates core fields. Aggregate counts (screenshot_count,
// file_count, etc.) are zero-valued because create/update responses return the
// base game row, not the full catalog read model. Callers must not rely on
// aggregate fields from this response.
func toGameSummaryResponse(game domain.Game) gameListItemResponse {
	return gameListItemResponse{
		ID:          game.ID,
		PublicID:    game.PublicID,
		Title:       game.Title,
		TitleAlt:    game.TitleAlt,
		Visibility:  game.Visibility,
		Summary:     game.Summary,
		ReleaseDate: game.ReleaseDate,
		CoverImage:  game.CoverImage,
		BannerImage: game.BannerImage,
		WikiContent: game.WikiContent,
		Downloads:   game.Downloads,
		IsFavorite:  game.IsFavorite,
		CreatedAt:   game.CreatedAt,
		UpdatedAt:   game.UpdatedAt,
	}
}

// 2026-05-09: toSeriesGameSummaryResponses maps series-scoped game summaries
// into the shared gameListItemResponse type. Aggregate count fields are omitted
// because series views do not need them and the SeriesGameSummary domain struct
// does not carry those values.
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

	covers := make([]gameAssetResponse, 0, len(detail.Covers))
	for _, asset := range detail.Covers {
		covers = append(covers, gameAssetResponse{
			ID:        asset.ID,
			AssetUID:  asset.AssetUID,
			Path:      asset.Path,
			SortOrder: asset.SortOrder,
		})
	}

	banners := make([]gameAssetResponse, 0, len(detail.Banners))
	for _, asset := range detail.Banners {
		banners = append(banners, gameAssetResponse{
			ID:        asset.ID,
			AssetUID:  asset.AssetUID,
			Path:      asset.Path,
			SortOrder: asset.SortOrder,
		})
	}

	logos := make([]gameAssetResponse, 0, len(detail.Logos))
	for _, asset := range detail.Logos {
		logos = append(logos, gameAssetResponse{
			ID:        asset.ID,
			AssetUID:  asset.AssetUID,
			Path:      asset.Path,
			SortOrder: asset.SortOrder,
			PositionX: asset.PositionX,
			PositionY: asset.PositionY,
			WidthPct:  asset.WidthPct,
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
		CoverImage:    detail.Game.CoverImage,
		BannerImage:   detail.Game.BannerImage,
		WikiContent:   detail.Game.WikiContent,
		Downloads:     detail.Game.Downloads,
		PreviewVideos: previewVideos,
		Screenshots:   screenshots,
		Covers:        covers,
		Banners:       banners,
		Logos:         logos,
		LogoVisible:   detail.Game.LogoVisible,
		Series:        series,
		Developers:    toMetadataResponses(detail.Developers),
		Publishers:    toMetadataResponses(detail.Publishers),
		Files:         toGameFileResponses(detail.Files, includePaths),
		IsFavorite:    detail.Game.IsFavorite,
		PendingIssues: toPendingIssueEvaluationResponse(detail.PendingIssues),
		CreatedAt:     detail.Game.CreatedAt,
		UpdatedAt:     detail.Game.UpdatedAt,
	}
}

func toMetadataResponses(items []domain.MetadataItem) []metadataItemResponse {
	result := make([]metadataItemResponse, 0, len(items))
	for _, item := range items {
		result = append(result, toMetadataResponse(item))
	}
	return result
}

func toMetadataResponse(item domain.MetadataItem) metadataItemResponse {
	return metadataItemResponse{
		ID:              item.ID,
		Name:            item.Name,
		Slug:            item.Slug,
		SortOrder:       item.SortOrder,
		CreatedAt:       item.CreatedAt,
		GameCount:       item.GameCount,
		CoverImage:      item.CoverImage,
		CoverCandidates: item.CoverCandidates,
		LatestUpdatedAt: item.LatestUpdatedAt,
	}
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

func toReviewIssueOverrideResponses(items []domain.ReviewIssueOverride) []reviewIssueOverrideResponse {
	result := make([]reviewIssueOverrideResponse, 0, len(items))
	for _, item := range items {
		result = append(result, toReviewIssueOverrideResponse(item))
	}
	return result
}

func toReviewIssueOverrideResponse(item domain.ReviewIssueOverride) reviewIssueOverrideResponse {
	return reviewIssueOverrideResponse{
		ID:        item.ID,
		GameID:    item.GameID,
		IssueKey:  item.IssueKey,
		Status:    item.Status,
		Reason:    item.Reason,
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
}

func toDirectoryListResponse(item *domain.DirectoryListResponse) directoryListResponse {
	items := make([]directoryItemResponse, 0, len(item.Items))
	for _, entry := range item.Items {
		items = append(items, directoryItemResponse{
			Name:        entry.Name,
			Path:        entry.Path,
			IsDirectory: entry.IsDirectory,
			SizeBytes:   entry.SizeBytes,
		})
	}
	return directoryListResponse{
		CurrentPath:  item.CurrentPath,
		ParentPath:   item.ParentPath,
		Items:        items,
		Incomplete:   item.Incomplete,
		SkippedCount: item.SkippedCount,
	}
}

func toSteamSearchResultResponses(items []domain.SteamSearchResult) []steamSearchResultResponse {
	result := make([]steamSearchResultResponse, 0, len(items))
	for _, item := range items {
		result = append(result, steamSearchResultResponse{
			AppID:       item.AppID,
			Name:        item.Name,
			ReleaseDate: item.ReleaseDate,
			TinyImage:   item.TinyImage,
		})
	}
	return result
}

func toSteamAssetsPreviewResponse(item *domain.SteamAssetsPreview) steamAssetsPreviewResponse {
	return steamAssetsPreviewResponse{
		AppID:          item.AppID,
		Name:           item.Name,
		Description:    item.Description,
		ReleaseDate:    item.ReleaseDate,
		Developers:     append([]string(nil), item.Developers...),
		Publishers:     append([]string(nil), item.Publishers...),
		CoverURL:       item.CoverURL,
		BannerURL:      item.BannerURL,
		ScreenshotURLs: append([]string(nil), item.ScreenshotURLs...),
	}
}

func toPendingIssueCatalogResponse(item domain.PendingIssueCatalog) pendingIssueCatalogResponse {
	groups := make([]pendingIssueDefinitionResponse, 0, len(item.Groups))
	for _, group := range item.Groups {
		groups = append(groups, pendingIssueDefinitionResponse{
			Key:         string(group.Key),
			Label:       group.Label,
			Description: group.Description,
		})
	}

	details := make([]pendingIssueDetailDefinitionResponse, 0, len(item.Details))
	for _, detail := range item.Details {
		details = append(details, pendingIssueDetailDefinitionResponse{
			Key:   string(detail.Key),
			Label: detail.Label,
			Group: string(detail.Group),
		})
	}

	return pendingIssueCatalogResponse{
		Groups:  groups,
		Details: details,
	}
}

func toPendingIssueEvaluationResponse(item *domain.PendingIssueEvaluation) *pendingIssueEvaluationResponse {
	if item == nil {
		return nil
	}

	groups := make([]string, 0, len(item.Groups))
	for _, group := range item.Groups {
		groups = append(groups, string(group))
	}

	details := make([]pendingIssueDetailStateResponse, 0, len(item.Details))
	for _, detail := range item.Details {
		details = append(details, pendingIssueDetailStateResponse{
			Key:     string(detail.Key),
			Group:   string(detail.Group),
			Ignored: detail.Ignored,
			Reason:  detail.Reason,
		})
	}

	return &pendingIssueEvaluationResponse{
		Groups:  groups,
		Details: details,
		Severe:  item.Severe,
	}
}

func toPendingIssueCountSummaryResponse(item *domain.PendingIssueCountSummary) *pendingIssueCountSummaryResponse {
	if item == nil {
		return nil
	}

	groups := make(map[string]int, len(item.Groups))
	for key, count := range item.Groups {
		groups[string(key)] = count
	}

	return &pendingIssueCountSummaryResponse{
		Groups:       groups,
		IgnoredTotal: item.IgnoredTotal,
	}
}

