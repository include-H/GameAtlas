package handlers

import "github.com/hao/game/internal/domain"

type metadataWriteRequest struct {
	Name      string  `json:"name"`
	Slug      *string `json:"slug"`
	SortOrder *int    `json:"sort_order"`
}

type tagGroupWriteRequest struct {
	Key           string  `json:"key"`
	Name          string  `json:"name"`
	Description   *string `json:"description"`
	SortOrder     *int    `json:"sort_order"`
	AllowMultiple *bool   `json:"allow_multiple"`
	IsFilterable  *bool   `json:"is_filterable"`
}

type tagWriteRequest struct {
	GroupID   int64   `json:"group_id"`
	Name      string  `json:"name"`
	Slug      *string `json:"slug"`
	ParentID  *int64  `json:"parent_id"`
	SortOrder *int    `json:"sort_order"`
	IsActive  *bool   `json:"is_active"`
}

type steamApplyAssetsRequest struct {
	GameID         int64    `json:"game_id"`
	CoverURL       *string  `json:"cover_url"`
	BannerURL      *string  `json:"banner_url"`
	ScreenshotURLs []string `json:"screenshot_urls"`
}

type wikiWriteRequest struct {
	Content       string  `json:"content"`
	ChangeSummary *string `json:"change_summary"`
}

func (request metadataWriteRequest) toInput() domain.MetadataWriteInput {
	return domain.MetadataWriteInput{
		Name:      request.Name,
		Slug:      request.Slug,
		SortOrder: request.SortOrder,
	}
}

func (request tagGroupWriteRequest) toInput() domain.TagGroupWriteInput {
	return domain.TagGroupWriteInput{
		Key:           request.Key,
		Name:          request.Name,
		Description:   request.Description,
		SortOrder:     request.SortOrder,
		AllowMultiple: request.AllowMultiple,
		IsFilterable:  request.IsFilterable,
	}
}

func (request tagWriteRequest) toInput() domain.TagWriteInput {
	return domain.TagWriteInput{
		GroupID:   request.GroupID,
		Name:      request.Name,
		Slug:      request.Slug,
		ParentID:  request.ParentID,
		SortOrder: request.SortOrder,
		IsActive:  request.IsActive,
	}
}

func (request steamApplyAssetsRequest) toInput() domain.SteamApplyAssetsInput {
	return domain.SteamApplyAssetsInput{
		GameID:         request.GameID,
		CoverURL:       request.CoverURL,
		BannerURL:      request.BannerURL,
		ScreenshotURLs: normalizeStringSlice(request.ScreenshotURLs),
	}
}

func (request wikiWriteRequest) toInput() domain.WikiWriteInput {
	return domain.WikiWriteInput{
		Content:       request.Content,
		ChangeSummary: request.ChangeSummary,
	}
}

func normalizeStringSlice(items []string) []string {
	// 2026-04-04: keep this only for steam asset apply requests.
	// Impact: omitted screenshot_urls still mean "apply zero screenshots" before service validation.
	if items == nil {
		return []string{}
	}
	return items
}
