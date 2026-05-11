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
	CoverImage   *string `json:"cover_image"`
	BannerImage  *string `json:"banner_image"`
	LogoVisible  *bool   `json:"logo_visible"`
	SeriesID     *int64  `json:"series_id"`
	DeveloperIDs []int64 `json:"developer_ids"`
	PublisherIDs []int64 `json:"publisher_ids"`
}

type gameAggregateAssetsRequest struct {
	Files                    []gameAggregateFileRequest        `json:"files"`
	ScreenshotOrderAssetUIDs []string                          `json:"screenshot_order_asset_uids"`
	VideoOrderAssetUIDs      []string                          `json:"video_order_asset_uids"`
	CoverOrderAssetUIDs      []string                          `json:"cover_order_asset_uids"`
	LogoOrderAssetUIDs       []string                          `json:"logo_order_asset_uids"`
	BannerOrderAssetUIDs     []string                          `json:"banner_order_asset_uids"`
	LogoPositions            []logoPositionRequest             `json:"logo_positions"`
	NewAssets                []newAssetEntryRequest            `json:"new_assets"`
}

type newAssetEntryRequest struct {
	AssetUID  string `json:"asset_uid"`
	AssetType string `json:"asset_type"`
	Path      string `json:"path"`
}

type logoPositionRequest struct {
	AssetUID  string   `json:"asset_uid"`
	PositionX *float64 `json:"position_x"`
	PositionY *float64 `json:"position_y"`
	WidthPct  *float64 `json:"width_pct"`
}

type gameAggregateFileRequest struct {
	ID       *int64  `json:"id"`
	FilePath string  `json:"file_path"`
	Label    *string `json:"label"`
	Notes    *string `json:"notes"`
}

func (request gameCreateRequest) toInput() domain.GameCreateInput {
	return domain.GameCreateInput{
		Title:      request.Title,
		Visibility: request.Visibility,
	}
}

func (request gameAggregateUpdateRequest) toInput() domain.GameAggregateUpdateInput {
	// 2026-05-01: aggregate update is a full replacement write, not a sparse patch.
	// Impact: omitted relation/order arrays are intentionally treated the same as explicit
	// empty arrays at the handler DTO boundary. This is domain write semantics for the
	// aggregate endpoint itself, not frontend-form compatibility glue.
	return domain.GameAggregateUpdateInput{
		Game: domain.GameAggregateCoreUpdateInput{
			GameCoreInput: request.Game.toDomain(),
			SeriesID:      request.Game.SeriesID,
			DeveloperIDs:  emptyInt64Slice(request.Game.DeveloperIDs),
			PublisherIDs:  emptyInt64Slice(request.Game.PublisherIDs),
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
		CoverImage:  request.CoverImage,
		BannerImage: request.BannerImage,
		LogoVisible: request.LogoVisible,
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

	logoPositions := make([]domain.LogoPositionInput, 0, len(request.LogoPositions))
	for _, lp := range request.LogoPositions {
		logoPositions = append(logoPositions, domain.LogoPositionInput{
			AssetUID:  lp.AssetUID,
			PositionX: lp.PositionX,
			PositionY: lp.PositionY,
			WidthPct:  lp.WidthPct,
		})
	}

	newAssets := make([]domain.NewAssetEntry, 0, len(request.NewAssets))
	for _, item := range request.NewAssets {
		newAssets = append(newAssets, domain.NewAssetEntry{
			AssetUID:  item.AssetUID,
			AssetType: item.AssetType,
			Path:      item.Path,
		})
	}

	return domain.GameAggregateAssetsInput{
		Files:                    files,
		ScreenshotOrderAssetUIDs: emptyStringSlice(request.ScreenshotOrderAssetUIDs),
		VideoOrderAssetUIDs:      emptyStringSlice(request.VideoOrderAssetUIDs),
		CoverOrderAssetUIDs:      emptyStringSlice(request.CoverOrderAssetUIDs),
		LogoOrderAssetUIDs:       emptyStringSlice(request.LogoOrderAssetUIDs),
		BannerOrderAssetUIDs:     emptyStringSlice(request.BannerOrderAssetUIDs),
		LogoPositions:            logoPositions,
		NewAssets:                newAssets,
	}
}

func emptyInt64Slice(values []int64) []int64 {
	// 2026-05-01: normalize omitted JSON arrays to empty slices for aggregate replace semantics.
	// Missing developer_ids/publisher_ids means "clear this collection" on this endpoint,
	// because aggregate updates rewrite the full editable relationship set in one request.
	if values == nil {
		return []int64{}
	}
	return values
}

func emptyStringSlice(values []string) []string {
	// 2026-05-01: reorder arrays follow the same full-replacement contract as relation arrays.
	// Missing screenshot/video order fields mean "no items remain in this ordered collection",
	// not "leave the existing order untouched".
	if values == nil {
		return []string{}
	}
	return values
}
