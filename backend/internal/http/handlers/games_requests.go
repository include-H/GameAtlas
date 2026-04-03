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
	Game   gameAggregatePatchRequest       `json:"game"`
	Assets domain.GameAggregateAssetsInput `json:"assets"`
}

type gameAggregatePatchRequest struct {
	domain.GameCoreInput
	SeriesID     json.RawMessage `json:"series_id"`
	PlatformIDs  json.RawMessage `json:"platform_ids"`
	DeveloperIDs json.RawMessage `json:"developer_ids"`
	PublisherIDs json.RawMessage `json:"publisher_ids"`
	TagIDs       json.RawMessage `json:"tag_ids"`
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
			GameCoreInput: request.Game.GameCoreInput,
			SeriesID:      seriesIDPatch,
			PlatformIDs:   platformIDsPatch,
			DeveloperIDs:  developerIDsPatch,
			PublisherIDs:  publisherIDsPatch,
			TagIDs:        tagIDsPatch,
		},
		Assets: request.Assets,
	}, nil
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
