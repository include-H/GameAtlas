package handlers

import "github.com/hao/game/internal/services"

// 2026-05-23: response DTOs for SteamGridDB endpoints.
// Decouple handler JSON shapes from service-layer structs so that
// external API deserialization tags and frontend response tags evolve independently.

type steamGridDBGameResponse struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	ReleaseDate int      `json:"release_date"`
	Types       []string `json:"types"`
	Verified    bool     `json:"verified"`
}

type steamGridDBImageResponse struct {
	ID       int    `json:"id"`
	Score    int    `json:"score"`
	Style    string `json:"style,omitempty"`
	Notes    string `json:"notes"`
	Language string `json:"language"`
	URL      string `json:"url"`
	Thumb    string `json:"thumb"`
}

func toSteamGridDBGameResponse(g services.SteamGridDBGame) steamGridDBGameResponse {
	return steamGridDBGameResponse{
		ID:          g.ID,
		Name:        g.Name,
		ReleaseDate: g.ReleaseDate,
		Types:       g.Types,
		Verified:    g.Verified,
	}
}

func toSteamGridDBGameResponses(games []services.SteamGridDBGame) []steamGridDBGameResponse {
	result := make([]steamGridDBGameResponse, 0, len(games))
	for _, g := range games {
		result = append(result, toSteamGridDBGameResponse(g))
	}
	return result
}

func toSteamGridDBImageResponse(img services.SteamGridDBImage) steamGridDBImageResponse {
	return steamGridDBImageResponse{
		ID:       img.ID,
		Score:    img.Score,
		Style:    img.Style,
		Notes:    img.Notes,
		Language: img.Language,
		URL:      img.URL,
		Thumb:    img.Thumb,
	}
}

func toSteamGridDBImageResponses(images []services.SteamGridDBImage) []steamGridDBImageResponse {
	result := make([]steamGridDBImageResponse, 0, len(images))
	for _, img := range images {
		result = append(result, toSteamGridDBImageResponse(img))
	}
	return result
}
