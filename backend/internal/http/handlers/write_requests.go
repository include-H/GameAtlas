package handlers

import "github.com/hao/game/internal/domain"

type metadataWriteRequest struct {
	Name      string  `json:"name"`
	Slug      *string `json:"slug"`
	SortOrder *int    `json:"sort_order"`
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

func (request wikiWriteRequest) toInput() domain.WikiWriteInput {
	return domain.WikiWriteInput{
		Content:       request.Content,
		ChangeSummary: request.ChangeSummary,
	}
}
