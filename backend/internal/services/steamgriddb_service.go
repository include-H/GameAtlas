package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type SteamGridDBService struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

type SteamGridDBImage struct {
	ID       int    `json:"id"`
	Score    int    `json:"score"`
	Style    string `json:"style,omitempty"`
	Notes    string `json:"notes"`
	Language string `json:"language"`
	URL      string `json:"url"`
	Thumb    string `json:"thumb"`
}

type SteamGridDBGame struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	ReleaseDate int    `json:"release_date"`
	Types       []string `json:"types"`
	Verified    bool   `json:"verified"`
}

type steamGridDBImageResponse struct {
	Success bool               `json:"success"`
	Data    []SteamGridDBImage `json:"data"`
}

type steamGridDBSearchResponse struct {
	Success bool               `json:"success"`
	Data    []SteamGridDBGame  `json:"data"`
}

func NewSteamGridDBService(apiKey string) *SteamGridDBService {
	return &SteamGridDBService{
		apiKey:  apiKey,
		baseURL: "https://www.steamgriddb.com/api/v2",
		client:  &http.Client{Timeout: 15 * time.Second},
	}
}

func (s *SteamGridDBService) Available() bool {
	return strings.TrimSpace(s.apiKey) != ""
}

func (s *SteamGridDBService) Search(query string) ([]SteamGridDBGame, error) {
	endpoint := fmt.Sprintf("search/autocomplete/%s", url.PathEscape(query))
	fullURL := fmt.Sprintf("%s/%s", s.baseURL, endpoint)

	req, err := http.NewRequest(http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request search: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("steamgriddb search: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result steamGridDBSearchResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("steamgriddb search: api returned success=false")
	}

	return result.Data, nil
}

func (s *SteamGridDBService) GetGridsBySteamAppID(steamAppID int64) ([]SteamGridDBImage, error) {
	return s.fetchImages(fmt.Sprintf("grids/steam/%d", steamAppID), url.Values{"dimensions": {"600x900"}})
}

func (s *SteamGridDBService) GetHeroesBySteamAppID(steamAppID int64) ([]SteamGridDBImage, error) {
	return s.fetchImages(fmt.Sprintf("heroes/steam/%d", steamAppID), nil)
}

func (s *SteamGridDBService) GetLogosBySteamAppID(steamAppID int64) ([]SteamGridDBImage, error) {
	return s.fetchImages(fmt.Sprintf("logos/steam/%d", steamAppID), nil)
}

func (s *SteamGridDBService) GetIconsBySteamAppID(steamAppID int64) ([]SteamGridDBImage, error) {
	return s.fetchImages(fmt.Sprintf("icons/steam/%d", steamAppID), nil)
}

func (s *SteamGridDBService) GetGridsByGameID(gameID int) ([]SteamGridDBImage, error) {
	return s.fetchImages(fmt.Sprintf("grids/game/%d", gameID), url.Values{"dimensions": {"600x900"}})
}

func (s *SteamGridDBService) GetHeroesByGameID(gameID int) ([]SteamGridDBImage, error) {
	return s.fetchImages(fmt.Sprintf("heroes/game/%d", gameID), nil)
}

func (s *SteamGridDBService) GetLogosByGameID(gameID int) ([]SteamGridDBImage, error) {
	return s.fetchImages(fmt.Sprintf("logos/game/%d", gameID), nil)
}

func (s *SteamGridDBService) fetchImages(endpoint string, extra url.Values) ([]SteamGridDBImage, error) {
	fullURL := fmt.Sprintf("%s/%s", s.baseURL, endpoint)

	req, err := http.NewRequest(http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Accept", "application/json")

	q := url.Values{}
	q.Set("nsfw", "false")
	q.Set("humor", "false")
	for k, vs := range extra {
		for _, v := range vs {
			q.Set(k, v)
		}
	}
	req.URL.RawQuery = q.Encode()

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("steamgriddb %s: HTTP %d: %s", endpoint, resp.StatusCode, string(body))
	}

	var result steamGridDBImageResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("steamgriddb %s: api returned success=false", endpoint)
	}

	return result.Data, nil
}
