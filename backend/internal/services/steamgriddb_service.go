package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/hao/game/internal/domain"
)

type SteamGridDBService struct {
	apiKey      string
	baseURL     string
	client      *http.Client
	searchCache sync.Map // query -> cachedSearchResult
	imageCache  sync.Map // endpoint -> cachedImageResult
}

type cachedSearchResult struct {
	data     []SteamGridDBGame
	cachedAt time.Time
}

type cachedImageResult struct {
	data     []SteamGridDBImage
	cachedAt time.Time
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
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	ReleaseDate int      `json:"release_date"`
	Types       []string `json:"types"`
	Verified    bool     `json:"verified"`
}

type steamGridDBImageResponse struct {
	Success bool               `json:"success"`
	Data    []SteamGridDBImage `json:"data"`
}

type steamGridDBSearchResponse struct {
	Success bool              `json:"success"`
	Data    []SteamGridDBGame `json:"data"`
}

func NewSteamGridDBService(apiKey string, proxy string) *SteamGridDBService {
	transport := &http.Transport{Proxy: http.ProxyFromEnvironment}
	if strings.TrimSpace(proxy) != "" {
		if parsed, err := url.Parse(proxy); err == nil {
			transport.Proxy = http.ProxyURL(parsed)
		}
	}

	return &SteamGridDBService{
		apiKey:  apiKey,
		baseURL: "https://www.steamgriddb.com/api/v2",
		client:  &http.Client{Timeout: 15 * time.Second, Transport: transport},
	}
}

func (s *SteamGridDBService) Available() bool {
	return strings.TrimSpace(s.apiKey) != ""
}

func (s *SteamGridDBService) Search(query string) ([]SteamGridDBGame, error) {
	// Check cache first
	if cached, ok := s.searchCache.Load(query); ok {
		entry := cached.(cachedSearchResult)
		if time.Since(entry.cachedAt) < 1*time.Hour {
			return entry.data, nil
		}
		s.searchCache.Delete(query)
	}

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
		return nil, fmt.Errorf("%w: request search: %w", domain.ErrUpstream, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: read response: %w", domain.ErrUpstream, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: steamgriddb search: HTTP %d: %s", domain.ErrUpstream, resp.StatusCode, string(body))
	}

	var result steamGridDBSearchResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("%w: decode response: %w", domain.ErrUpstream, err)
	}

	if !result.Success {
		return nil, fmt.Errorf("%w: steamgriddb search: api returned success=false", domain.ErrUpstream)
	}

	// Cache the result
	s.searchCache.Store(query, cachedSearchResult{data: result.Data, cachedAt: time.Now()})

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
	// Build cache key
	cacheKey := endpoint
	if extra != nil {
		cacheKey += "?" + extra.Encode()
	}

	// Check cache first
	if cached, ok := s.imageCache.Load(cacheKey); ok {
		entry := cached.(cachedImageResult)
		if time.Since(entry.cachedAt) < 1*time.Hour {
			return entry.data, nil
		}
		s.imageCache.Delete(cacheKey)
	}

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
		return nil, fmt.Errorf("%w: request %s: %w", domain.ErrUpstream, endpoint, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: read response: %w", domain.ErrUpstream, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: steamgriddb %s: HTTP %d: %s", domain.ErrUpstream, endpoint, resp.StatusCode, string(body))
	}

	var result steamGridDBImageResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("%w: decode response: %w", domain.ErrUpstream, err)
	}

	if !result.Success {
		return nil, fmt.Errorf("%w: steamgriddb %s: api returned success=false", domain.ErrUpstream, endpoint)
	}

	// Cache the result
	s.imageCache.Store(cacheKey, cachedImageResult{data: result.Data, cachedAt: time.Now()})

	return result.Data, nil
}
