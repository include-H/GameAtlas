package handlers

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/hao/game/internal/domain"
)

// 2026-04-03: keep aggregate patch decoding in a dedicated transport file so
// handler actions can stay small and response mapping can evolve separately.
var errInvalidAggregatePatch = errors.New("invalid aggregate patch")

// Aggregate updates have two explicit responsibilities only:
// game patch fields and asset operations.
type gameAggregateUpdateRequest struct {
	Game   gameAggregatePatchRequest  `json:"game"`
	Assets gameAggregateAssetsRequest `json:"assets"`
}

type gameCreateRequest struct {
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

type gameFileWriteRequest struct {
	FilePath  string  `json:"file_path"`
	Label     *string `json:"label"`
	Notes     *string `json:"notes"`
	SortOrder int     `json:"sort_order"`
}

type gameAggregatePatchRequest struct {
	Title        string          `json:"title"`
	TitleAlt     *string         `json:"title_alt"`
	Visibility   string          `json:"visibility"`
	Summary      *string         `json:"summary"`
	ReleaseDate  *string         `json:"release_date"`
	Engine       *string         `json:"engine"`
	CoverImage   *string         `json:"cover_image"`
	BannerImage  *string         `json:"banner_image"`
	SeriesID     json.RawMessage `json:"series_id"`
	PlatformIDs  json.RawMessage `json:"platform_ids"`
	DeveloperIDs json.RawMessage `json:"developer_ids"`
	PublisherIDs json.RawMessage `json:"publisher_ids"`
	TagIDs       json.RawMessage `json:"tag_ids"`
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

func (request gameCreateRequest) toInput() domain.GameWriteInput {
	return domain.GameWriteInput{
		GameCoreInput: domain.GameCoreInput{
			Title:       request.Title,
			TitleAlt:    request.TitleAlt,
			Visibility:  request.Visibility,
			Summary:     request.Summary,
			ReleaseDate: request.ReleaseDate,
			Engine:      request.Engine,
			CoverImage:  request.CoverImage,
			BannerImage: request.BannerImage,
		},
		SeriesID:     request.SeriesID,
		PlatformIDs:  request.PlatformIDs,
		DeveloperIDs: request.DeveloperIDs,
		PublisherIDs: request.PublisherIDs,
		TagIDs:       request.TagIDs,
	}
}

func (request gameFileWriteRequest) toInput() domain.GameFileWriteInput {
	return domain.GameFileWriteInput{
		FilePath:  request.FilePath,
		Label:     request.Label,
		Notes:     request.Notes,
		SortOrder: request.SortOrder,
	}
}

func (request gameAggregateUpdateRequest) toInput() (domain.GameAggregateUpdateInput, error) {
	seriesIDPatch, err := decodeOptionalInt64Patch(request.Game.SeriesID)
	if err != nil {
		return domain.GameAggregateUpdateInput{}, err
	}
	platformIDsPatch, err := decodeInt64SlicePatch(request.Game.PlatformIDs)
	if err != nil {
		return domain.GameAggregateUpdateInput{}, err
	}
	developerIDsPatch, err := decodeInt64SlicePatch(request.Game.DeveloperIDs)
	if err != nil {
		return domain.GameAggregateUpdateInput{}, err
	}
	publisherIDsPatch, err := decodeInt64SlicePatch(request.Game.PublisherIDs)
	if err != nil {
		return domain.GameAggregateUpdateInput{}, err
	}
	tagIDsPatch, err := decodeInt64SlicePatch(request.Game.TagIDs)
	if err != nil {
		return domain.GameAggregateUpdateInput{}, err
	}

	return domain.GameAggregateUpdateInput{
		Game: domain.GameAggregatePatchInput{
			GameCoreInput: request.Game.toDomain(),
			SeriesID:      seriesIDPatch,
			PlatformIDs:   platformIDsPatch,
			DeveloperIDs:  developerIDsPatch,
			PublisherIDs:  publisherIDsPatch,
			TagIDs:        tagIDsPatch,
		},
		Assets: request.Assets.toDomain(),
	}, nil
}

func (request gameAggregatePatchRequest) toDomain() domain.GameCoreInput {
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

func decodeOptionalInt64Patch(raw json.RawMessage) (domain.OptionalInt64Patch, error) {
	if raw == nil {
		return domain.OptionalInt64Patch{}, nil
	}
	if string(raw) == "null" {
		return domain.OptionalInt64Patch{Present: true, Value: nil}, nil
	}

	var value int64
	if err := json.Unmarshal(raw, &value); err != nil {
		return domain.OptionalInt64Patch{}, err
	}
	return domain.OptionalInt64Patch{Present: true, Value: &value}, nil
}

func decodeInt64SlicePatch(raw json.RawMessage) (domain.Int64SlicePatch, error) {
	if raw == nil {
		return domain.Int64SlicePatch{}, nil
	}
	if string(raw) == "null" {
		return domain.Int64SlicePatch{}, errInvalidAggregatePatch
	}

	var values []int64
	if err := json.Unmarshal(raw, &values); err != nil {
		return domain.Int64SlicePatch{}, err
	}
	return domain.Int64SlicePatch{Present: true, Values: values}, nil
}

func decodeGamesListParams(c *gin.Context) domain.GamesListParams {
	params := domain.GamesListParams{
		Page:              parseQueryInt(c, "page", 1),
		Limit:             parseQueryInt(c, "limit", 20),
		Search:            c.Query("search"),
		SeriesID:          parseQueryInt64(c, "series", 0),
		PlatformID:        parseQueryInt64(c, "platform", 0),
		TagIDs:            parseQueryInt64List(c, "tag"),
		PendingIssue:      strings.TrimSpace(c.Query("pending_issue")),
		PendingRecentDays: parseQueryInt(c, "pending_recent_days", 0),
		Sort:              c.Query("sort"),
		Order:             c.Query("order"),
		SortSeed:          parseQueryInt64(c, "seed", 0),
		IncludeAll:        isAdminRequest(c),
	}

	if raw := c.Query("pending"); raw != "" {
		if value, err := strconv.ParseBool(raw); err == nil {
			params.PendingOnly = value
		}
	}
	if raw := c.Query("pending_include_ignored"); raw != "" {
		if value, err := strconv.ParseBool(raw); err == nil {
			params.PendingIncludeIgnored = value
		}
	}
	if raw := c.Query("pending_severe"); raw != "" {
		if value, err := strconv.ParseBool(raw); err == nil {
			params.PendingSevereOnly = value
		}
	}
	if raw := c.Query("favorite"); raw != "" {
		if value, err := strconv.ParseBool(raw); err == nil {
			params.FavoriteOnly = value
		}
	}

	return params
}

func decodeGamesTimelineRequest(c *gin.Context) (int, string, string, string, int64, bool) {
	years := parseQueryInt(c, "years", 2)
	if years <= 0 {
		years = 2
	}
	if years > 10 {
		years = 10
	}

	cursorReleaseDate, cursorID, ok := parseTimelineCursor(c.Query("cursor"))
	if !ok {
		return 0, "", "", "", 0, false
	}

	return years, c.Query("from"), c.Query("to"), cursorReleaseDate, cursorID, true
}

func parseQueryInt64List(c *gin.Context, key string) []int64 {
	values := c.QueryArray(key)
	if len(values) == 0 {
		return []int64{}
	}

	result := make([]int64, 0, len(values))
	for _, raw := range values {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value <= 0 {
			continue
		}
		result = append(result, value)
	}
	return result
}
