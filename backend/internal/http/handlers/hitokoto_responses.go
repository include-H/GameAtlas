package handlers

import "github.com/hao/game/internal/services"

// 2026-05-23: response DTO for hitokoto endpoint.
// Decouple handler JSON shape from service-layer struct so that
// embedded-data deserialization tags and frontend response tags evolve independently.

type hitokotoSentenceResponse struct {
	ID         int64   `json:"id"`
	UUID       string  `json:"uuid"`
	Hitokoto   string  `json:"hitokoto"`
	Type       string  `json:"type"`
	From       string  `json:"from"`
	FromWho    *string `json:"from_who"`
	Creator    string  `json:"creator"`
	CreatorUID int64   `json:"creator_uid"`
	Reviewer   int64   `json:"reviewer"`
	CommitFrom string  `json:"commit_from"`
	CreatedAt  string  `json:"created_at"`
	Length     int     `json:"length"`
}

func toHitokotoSentenceResponse(s *services.HitokotoSentence) hitokotoSentenceResponse {
	return hitokotoSentenceResponse{
		ID:         s.ID,
		UUID:       s.UUID,
		Hitokoto:   s.Hitokoto,
		Type:       s.Type,
		From:       s.From,
		FromWho:    s.FromWho,
		Creator:    s.Creator,
		CreatorUID: s.CreatorUID,
		Reviewer:   s.Reviewer,
		CommitFrom: s.CommitFrom,
		CreatedAt:  s.CreatedAt,
		Length:     s.Length,
	}
}
