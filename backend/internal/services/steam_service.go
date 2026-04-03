package services

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hao/game/internal/config"
	"github.com/hao/game/internal/domain"
)

var ErrUpstream = errors.New("upstream request failed")

type SteamService struct {
	client *http.Client
	proxy  string
	assets *AssetsService
}

type steamStoreSearchResponse struct {
	Items []struct {
		ID          int64  `json:"id"`
		Name        string `json:"name"`
		TinyImage   string `json:"tiny_image"`
		ReleaseDate *struct {
			Date string `json:"date"`
		} `json:"release_date"`
	} `json:"items"`
}

type steamMovieEntry map[string]any

type steamAppDetailsResponse map[string]struct {
	Success bool `json:"success"`
	Data    *struct {
		Name                string   `json:"name"`
		HeaderImage         string   `json:"header_image"`
		Background          string   `json:"background"`
		DetailedDescription string   `json:"detailed_description"`
		AboutTheGame        string   `json:"about_the_game"`
		ShortDescription    string   `json:"short_description"`
		Developers          []string `json:"developers"`
		Publishers          []string `json:"publishers"`
		ReleaseDate         *struct {
			ComingSoon bool   `json:"coming_soon"`
			Date       string `json:"date"`
		} `json:"release_date"`
		Movies      []steamMovieEntry `json:"movies"`
		Screenshots []struct {
			PathFull string `json:"path_full"`
		} `json:"screenshots"`
	} `json:"data"`
}

// NewSteamService keeps Steam integration behind one service entrypoint for now.
// 2026-04-03 review: file-level splitting is enough to stop it from turning into
// another giant module, while further splitting into injected sub-services would
// add wiring and test overhead without enough current payoff.
func NewSteamService(cfg config.Config, assetsService *AssetsService) *SteamService {
	proxy := cfg.Proxy

	transport := &http.Transport{Proxy: http.ProxyFromEnvironment}
	if proxy != "" {
		if parsed, err := url.Parse(proxy); err == nil {
			transport.Proxy = http.ProxyURL(parsed)
		}
	}

	return &SteamService{
		client: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
		proxy:  proxy,
		assets: assetsService,
	}
}

func (s *SteamService) Search(query string, proxyOverride string) ([]domain.SteamSearchResult, error) {
	if query == "" {
		return []domain.SteamSearchResult{}, nil
	}

	payloads := make([]steamStoreSearchResponse, 0, 2)
	for _, locale := range []struct {
		lang string
		cc   string
	}{
		{lang: "schinese", cc: "CN"},
		{lang: "english", cc: "US"},
	} {
		endpoint := fmt.Sprintf(
			"https://store.steampowered.com/api/storesearch/?term=%s&l=%s&cc=%s",
			url.QueryEscape(query),
			locale.lang,
			locale.cc,
		)
		var payload steamStoreSearchResponse
		if err := s.fetchJSON(endpoint, &payload, proxyOverride); err == nil {
			payloads = append(payloads, payload)
		}
	}

	if len(payloads) == 0 {
		return nil, fmt.Errorf("steam search failed")
	}

	seen := make(map[int64]struct{})
	results := make([]domain.SteamSearchResult, 0)
	for _, payload := range payloads {
		for _, item := range payload.Items {
			if _, exists := seen[item.ID]; exists {
				continue
			}
			seen[item.ID] = struct{}{}

			var releaseDate *string
			if item.ReleaseDate != nil && item.ReleaseDate.Date != "" {
				releaseDate = &item.ReleaseDate.Date
			}
			var tinyImage *string
			if item.TinyImage != "" {
				tinyImage = &item.TinyImage
			}
			results = append(results, domain.SteamSearchResult{
				AppID:       item.ID,
				Name:        item.Name,
				ReleaseDate: releaseDate,
				TinyImage:   tinyImage,
			})
		}
	}
	return results, nil
}

func (s *SteamService) PreviewAssets(appID int64, proxyOverride string) (*domain.SteamAssetsPreview, error) {
	appKey := fmt.Sprintf("%d", appID)
	primaryPayload, primaryErr := s.fetchAppDetails(appID, "schinese", proxyOverride)
	fallbackPayload, fallbackErr := s.fetchAppDetails(appID, "english", proxyOverride)

	primaryDetails, primaryOK := primaryPayload[appKey]
	fallbackDetails, fallbackOK := fallbackPayload[appKey]
	primaryUsable := primaryOK && primaryDetails.Success && primaryDetails.Data != nil
	fallbackUsable := fallbackOK && fallbackDetails.Success && fallbackDetails.Data != nil
	if !primaryUsable && !fallbackUsable {
		if primaryErr != nil || fallbackErr != nil {
			return nil, wrapSteamUpstreamError(
				"steam preview appdetails failed",
				namedError{name: "schinese appdetails", err: primaryErr},
				namedError{name: "english appdetails", err: fallbackErr},
			)
		}
		return nil, ErrNotFound
	}

	name := fmt.Sprintf("Steam App %d", appID)
	description := ""
	releaseDate := ""
	developers := []string{}
	publishers := []string{}
	screenshotURLs := []string{}
	description = s.fetchDescriptionFromStorePage(appID, proxyOverride)
	if primaryUsable {
		if primaryDetails.Data.Name != "" {
			name = primaryDetails.Data.Name
		}
		if strings.TrimSpace(description) == "" {
			description = firstNonEmpty(
				primaryDetails.Data.ShortDescription,
				primaryDetails.Data.AboutTheGame,
				primaryDetails.Data.DetailedDescription,
			)
		}
		releaseDate = normalizeSteamReleaseDate(primaryDetails.Data.ReleaseDate)
		developers = cleanSteamNames(primaryDetails.Data.Developers)
		publishers = cleanSteamNames(primaryDetails.Data.Publishers)
		screenshotURLs = make([]string, 0, len(primaryDetails.Data.Screenshots))
		for _, screenshot := range primaryDetails.Data.Screenshots {
			if screenshot.PathFull != "" {
				screenshotURLs = append(screenshotURLs, screenshot.PathFull)
			}
		}
	}
	if fallbackUsable {
		if name == fmt.Sprintf("Steam App %d", appID) && fallbackDetails.Data.Name != "" {
			name = fallbackDetails.Data.Name
		}
		if strings.TrimSpace(description) == "" {
			description = firstNonEmpty(
				fallbackDetails.Data.ShortDescription,
				fallbackDetails.Data.AboutTheGame,
				fallbackDetails.Data.DetailedDescription,
			)
		}
		if releaseDate == "" {
			releaseDate = normalizeSteamReleaseDate(fallbackDetails.Data.ReleaseDate)
		}
		if len(developers) == 0 {
			developers = cleanSteamNames(fallbackDetails.Data.Developers)
		}
		if len(publishers) == 0 {
			publishers = cleanSteamNames(fallbackDetails.Data.Publishers)
		}
		if len(screenshotURLs) == 0 {
			screenshotURLs = make([]string, 0, len(fallbackDetails.Data.Screenshots))
			for _, screenshot := range fallbackDetails.Data.Screenshots {
				if screenshot.PathFull != "" {
					screenshotURLs = append(screenshotURLs, screenshot.PathFull)
				}
			}
		}
	}
	if len(screenshotURLs) == 0 {
		screenshotURLs = s.fetchScreenshotURLsFromStorePage(appID, proxyOverride)
	}

	coverURL := s.resolveSteamAssetURL(appID, proxyOverride,
		"https://steamcdn-a.akamaihd.net/steam/apps/%d/library_600x900_2x.jpg",
		"https://steamcdn-a.akamaihd.net/steam/apps/%d/library_600x900.jpg",
	)
	bannerURL := s.resolveSteamAssetURL(appID, proxyOverride,
		"https://steamcdn-a.akamaihd.net/steam/apps/%d/library_hero_2x.jpg",
		"https://steamcdn-a.akamaihd.net/steam/apps/%d/library_hero.jpg",
	)

	return &domain.SteamAssetsPreview{
		AppID:          appID,
		Name:           name,
		Description:    description,
		ReleaseDate:    releaseDate,
		Developers:     developers,
		Publishers:     publishers,
		CoverURL:       coverURL,
		BannerURL:      bannerURL,
		ScreenshotURLs: screenshotURLs,
	}, nil
}

func (s *SteamService) ApplyAssets(appID int64, input domain.SteamApplyAssetsInput) (*domain.SteamAssetsPreview, error) {
	if input.GameID <= 0 {
		return nil, ErrValidation
	}

	sortOrder := 0
	var appliedCover *string
	var appliedBanner *string
	appliedScreenshots := make([]string, 0, len(input.ScreenshotURLs))

	if input.CoverURL != nil && *input.CoverURL != "" {
		path, err := s.assets.ApplyRemoteAsset(input.GameID, "cover", *input.CoverURL, 0)
		if err != nil {
			return nil, err
		}
		appliedCover = &path
	}
	if input.BannerURL != nil && *input.BannerURL != "" {
		path, err := s.assets.ApplyRemoteAsset(input.GameID, "banner", *input.BannerURL, 0)
		if err != nil {
			return nil, err
		}
		appliedBanner = &path
	}
	for _, rawURL := range input.ScreenshotURLs {
		if rawURL == "" {
			continue
		}
		path, err := s.assets.ApplyRemoteAsset(input.GameID, "screenshot", rawURL, sortOrder)
		if err != nil {
			return nil, err
		}
		appliedScreenshots = append(appliedScreenshots, path)
		sortOrder++
	}

	return &domain.SteamAssetsPreview{
		AppID:          appID,
		ReleaseDate:    "",
		Developers:     []string{},
		Publishers:     []string{},
		CoverURL:       appliedCover,
		BannerURL:      appliedBanner,
		ScreenshotURLs: appliedScreenshots,
	}, nil
}
