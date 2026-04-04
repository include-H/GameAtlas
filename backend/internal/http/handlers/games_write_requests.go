package handlers

import "github.com/hao/game/internal/domain"

// Aggregate updates have two explicit responsibilities only:
// core game writes plus relation replacement, and asset operations.
type gameAggregateUpdateRequest struct {
	Game   gameAggregateCoreUpdateRequest `json:"game"`
	Assets gameAggregateAssetsRequest     `json:"assets"`
}

type gameCreateRequest struct {
	Title      string `json:"title"`
	Visibility string `json:"visibility"`
}

type gameAggregateCoreUpdateRequest struct {
	Title        string  `json:"title"`
	TitleAlt     *string `json:"title_alt"`
	Visibility   string  `json:"visibility"`
	Summary      *string `json:"summary"`
	ReleaseDate  *string `json:"release_date"`
	Engine       *string `json:"engine"`
	CoverImage   *string `json:"cover_image"`
	BannerImage  *string `json:"banner_image"`
	SeriesID     *int64  `json:"series_id"`
	PlatformIDs  []int64 `json:"platform_ids"`
	DeveloperIDs []int64 `json:"developer_ids"`
	PublisherIDs []int64 `json:"publisher_ids"`
	TagIDs       []int64 `json:"tag_ids"`
}

type gameAggregateAssetsRequest struct {
	Files                    []gameAggregateFileRequest        `json:"files"`
	DeleteAssets             []gameAggregateDeleteAssetRequest `json:"delete_assets"`
	ScreenshotOrderAssetUIDs []string                          `json:"screenshot_order_asset_uids"`
	VideoOrderAssetUIDs      []string                          `json:"video_order_asset_uids"`
}

type gameAggregateFileRequest struct {
	ID       *int64  `json:"id"`
	FilePath string  `json:"file_path"`
	Label    *string `json:"label"`
	Notes    *string `json:"notes"`
}

type gameAggregateDeleteAssetRequest struct {
	AssetType string `json:"asset_type"`
	Path      string `json:"path"`
	AssetID   *int64 `json:"asset_id"`
	AssetUID  string `json:"asset_uid"`
}

func (request gameCreateRequest) toInput() domain.GameCreateInput {
	return domain.GameCreateInput{
		Title:      request.Title,
		Visibility: request.Visibility,
	}
}

func (request gameAggregateUpdateRequest) toInput() domain.GameAggregateUpdateInput {
	return domain.GameAggregateUpdateInput{
		Game: domain.GameAggregateCoreUpdateInput{
			GameCoreInput: request.Game.toDomain(),
			SeriesID:      request.Game.SeriesID,
			PlatformIDs:   request.Game.PlatformIDs,
			DeveloperIDs:  request.Game.DeveloperIDs,
			PublisherIDs:  request.Game.PublisherIDs,
			TagIDs:        request.Game.TagIDs,
		},
		Assets: request.Assets.toDomain(),
	}
}

func (request gameAggregateCoreUpdateRequest) toDomain() domain.GameCoreInput {
	return domain.GameCoreInput{
		Title:       request.Title,
		TitleAlt:    request.TitleAlt,
		Visibility:  request.Visibility,
		Summary:     request.Summary,
		ReleaseDate: request.ReleaseDate,
		Engine:      request.Engine,
		CoverImage:  request.CoverImage,
		BannerImage: request.BannerImage,
	}
}

func (request gameAggregateAssetsRequest) toDomain() domain.GameAggregateAssetsInput {
	files := make([]domain.GameFileUpsertInput, 0, len(request.Files))
	for _, item := range request.Files {
		files = append(files, domain.GameFileUpsertInput{
			ID:       item.ID,
			FilePath: item.FilePath,
			Label:    item.Label,
			Notes:    item.Notes,
		})
	}

	deleteAssets := make([]domain.GameAssetDeleteInput, 0, len(request.DeleteAssets))
	for _, item := range request.DeleteAssets {
		deleteAssets = append(deleteAssets, domain.GameAssetDeleteInput{
			AssetType: item.AssetType,
			Path:      item.Path,
			AssetID:   item.AssetID,
			AssetUID:  item.AssetUID,
		})
	}

	return domain.GameAggregateAssetsInput{
		Files:                    files,
		DeleteAssets:             deleteAssets,
		ScreenshotOrderAssetUIDs: request.ScreenshotOrderAssetUIDs,
		VideoOrderAssetUIDs:      request.VideoOrderAssetUIDs,
	}
}
