package services

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/hao/game/internal/config"
	"github.com/hao/game/internal/domain"
)

type cachedPreview struct {
	preview  *domain.SteamAssetsPreview
	cachedAt time.Time
}

type SteamService struct {
	client       *http.Client
	proxy        string
	assets       *AssetsService
	previewCache sync.Map // int64 -> cachedPreview
	proxyClients sync.Map // string -> *http.Client, keyed by proxy URL
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
		client: newSteamHTTPClient(transport, 30*time.Second),
		proxy:  proxy,
		assets: assetsService,
	}
}

type steamSearchLocale struct {
	CC   string
	Lang string
}

var steamSearchLocales = []steamSearchLocale{
	{CC: "CN", Lang: "schinese"},
	{CC: "US", Lang: "english"},
}

func (s *SteamService) Search(query string, proxyOverride string) ([]domain.SteamSearchResult, error) {
	payloads := make([]steamStoreSearchResponse, 0, len(steamSearchLocales))
	for _, loc := range steamSearchLocales {
		endpoint := fmt.Sprintf(
			"https://store.steampowered.com/api/storesearch/?term=%s&l=%s&cc=%s",
			url.QueryEscape(query),
			loc.Lang,
			loc.CC,
		)
		var payload steamStoreSearchResponse
		if err := s.fetchJSON(endpoint, &payload, proxyOverride); err == nil {
			payloads = append(payloads, payload)
		}
	}

	if len(payloads) == 0 {
		return nil, fmt.Errorf("steam search failed: %w", domain.ErrUpstream)
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
	if v, ok := s.previewCache.Load(appID); ok {
		entry := v.(cachedPreview)
		if time.Since(entry.cachedAt) < 24*time.Hour {
			return entry.preview, nil
		}
		s.previewCache.Delete(appID)
	}

	appKey := fmt.Sprintf("%d", appID)

	// Fetch both locales concurrently
	var wg sync.WaitGroup
	var mu sync.Mutex
	var primaryPayload, fallbackPayload steamAppDetailsResponse
	var primaryErr, fallbackErr error

	wg.Add(2)
	go func() {
		defer wg.Done()
		payload, err := s.fetchAppDetails(appID, "schinese", proxyOverride)
		mu.Lock()
		primaryPayload = payload
		primaryErr = err
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		payload, err := s.fetchAppDetails(appID, "english", proxyOverride)
		mu.Lock()
		fallbackPayload = payload
		fallbackErr = err
		mu.Unlock()
	}()
	wg.Wait()

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
		// Both locales returned success:false — the app exists on Steam but has no
		// appdetails data (known Steam behavior for some store-only pages like 696360).
		// Fall back to scraping the store page directly.
		return s.previewAssetsFromStorePage(appID, proxyOverride)
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

	// Resolve cover/banner concurrently
	var coverURL, bannerURL *string
	var wg2 sync.WaitGroup
	wg2.Add(2)
	go func() {
		defer wg2.Done()
		coverURL = s.resolveSteamAssetURL(appID, proxyOverride,
			"https://steamcdn-a.akamaihd.net/steam/apps/%d/library_600x900_2x.jpg",
			"https://steamcdn-a.akamaihd.net/steam/apps/%d/library_600x900.jpg",
		)
	}()
	go func() {
		defer wg2.Done()
		bannerURL = s.resolveSteamAssetURL(appID, proxyOverride,
			"https://steamcdn-a.akamaihd.net/steam/apps/%d/library_hero_2x.jpg",
			"https://steamcdn-a.akamaihd.net/steam/apps/%d/library_hero.jpg",
		)
	}()
	wg2.Wait()

	// Use API-provided header_image as fallback — it uses hashed CDN paths that
	// the old-style URL guessing cannot discover.
	apiHeaderImage := ""
	if primaryUsable && primaryDetails.Data.HeaderImage != "" {
		apiHeaderImage = primaryDetails.Data.HeaderImage
	} else if fallbackUsable && fallbackDetails.Data.HeaderImage != "" {
		apiHeaderImage = fallbackDetails.Data.HeaderImage
	}

	if bannerURL == nil && apiHeaderImage != "" {
		bannerURL = &apiHeaderImage
	}
	if coverURL == nil {
		coverURL = s.resolveSteamAssetURL(appID, proxyOverride,
			"https://steamcdn-a.akamaihd.net/steam/apps/%d/capsule_616x353.jpg",
		)
	}
	if coverURL == nil && apiHeaderImage != "" {
		// Last resort: use header_image as cover (not ideal aspect ratio but better than nothing).
		coverURL = &apiHeaderImage
	}

	result := &domain.SteamAssetsPreview{
		AppID:          appID,
		Name:           name,
		Description:    description,
		ReleaseDate:    releaseDate,
		Developers:     developers,
		Publishers:     publishers,
		CoverURL:       coverURL,
		BannerURL:      bannerURL,
		ScreenshotURLs: screenshotURLs,
	}
	s.previewCache.Store(appID, cachedPreview{preview: result, cachedAt: time.Now()})
	return result, nil
}

// previewAssetsFromStorePage is the fallback when appdetails returns success:false
// for all locales but the store page itself still exists (known Steam behavior for
// some store-only listings). It scrapes the store page directly for whatever data
// is available.
func (s *SteamService) previewAssetsFromStorePage(appID int64, proxyOverride string) (*domain.SteamAssetsPreview, error) {
	storePageURL := fmt.Sprintf("https://store.steampowered.com/app/%d/?l=schinese", appID)
	pageHTML, err := s.fetchText(storePageURL, proxyOverride)
	if err != nil || pageHTML == "" {
		// Store page doesn't exist either — truly not found.
		return nil, domain.ErrNotFound
	}

	name := extractStorePageName(pageHTML, appID)
	description := s.fetchDescriptionFromStorePage(appID, proxyOverride)
	screenshotURLs := s.fetchScreenshotURLsFromStorePage(appID, proxyOverride)

	// Try CDN asset URLs — these exist independently of appdetails.
	coverURL := s.resolveSteamAssetURL(appID, proxyOverride,
		"https://steamcdn-a.akamaihd.net/steam/apps/%d/library_600x900_2x.jpg",
		"https://steamcdn-a.akamaihd.net/steam/apps/%d/library_600x900.jpg",
	)
	// Some newer games only have store capsule art, not the tall library cover.
	if coverURL == nil {
		coverURL = s.resolveSteamAssetURL(appID, proxyOverride,
			"https://steamcdn-a.akamaihd.net/steam/apps/%d/capsule_616x353.jpg",
		)
	}
	bannerURL := s.resolveSteamAssetURL(appID, proxyOverride,
		"https://steamcdn-a.akamaihd.net/steam/apps/%d/library_hero_2x.jpg",
		"https://steamcdn-a.akamaihd.net/steam/apps/%d/library_hero.jpg",
	)

	// If no library-style banner, try the header image that the store page uses.
	if bannerURL == nil {
		bannerURL = s.resolveSteamAssetURL(appID, proxyOverride,
			"https://shared.akamai.steamstatic.com/store_item_assets/steam/apps/%d/header_schinese.jpg",
			"https://shared.akamai.steamstatic.com/store_item_assets/steam/apps/%d/header.jpg",
		)
	}

	result := &domain.SteamAssetsPreview{
		AppID:          appID,
		Name:           name,
		Description:    description,
		ReleaseDate:    "",
		Developers:     []string{},
		Publishers:     []string{},
		CoverURL:       coverURL,
		BannerURL:      bannerURL,
		ScreenshotURLs: screenshotURLs,
	}
	s.previewCache.Store(appID, cachedPreview{preview: result, cachedAt: time.Now()})
	return result, nil
}

var storePageTitlePattern = regexp.MustCompile(`(?i)<title>(.*?)</title>`)

func extractStorePageName(pageHTML string, appID int64) string {
	match := storePageTitlePattern.FindStringSubmatch(pageHTML)
	if len(match) >= 2 {
		raw := strings.TrimSpace(match[1])
		// Steam titles end with " on Steam" or " on Steam" variants.
		raw = strings.TrimSuffix(raw, " on Steam")
		raw = strings.TrimSuffix(raw, " on steam")
		raw = strings.TrimSpace(raw)
		if raw != "" {
			return raw
		}
	}
	return fmt.Sprintf("Steam App %d", appID)
}
