package services

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/hao/game/internal/config"
	"github.com/hao/game/internal/domain"
)

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

type steamAppDetailsResponse map[string]struct {
	Success bool `json:"success"`
	Data    *struct {
		Name                string `json:"name"`
		HeaderImage         string `json:"header_image"`
		Background          string `json:"background"`
		DetailedDescription string `json:"detailed_description"`
		AboutTheGame        string `json:"about_the_game"`
		ShortDescription    string `json:"short_description"`
		Screenshots         []struct {
			PathFull string `json:"path_full"`
		} `json:"screenshots"`
	} `json:"data"`
}

func NewSteamService(cfg config.Config, assetsService *AssetsService) *SteamService {
	proxy := cfg.SteamProxy
	if proxy == "" {
		proxy = cfg.Proxy
	}

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
	primaryPayload, _ := s.fetchAppDetails(appID, "schinese", proxyOverride)
	fallbackPayload, _ := s.fetchAppDetails(appID, "english", proxyOverride)

	primaryDetails, primaryOK := primaryPayload[appKey]
	fallbackDetails, fallbackOK := fallbackPayload[appKey]
	name := fmt.Sprintf("Steam App %d", appID)
	description := ""
	screenshotURLs := []string{}
	description = s.fetchDescriptionFromStorePage(appID, proxyOverride)
	if primaryOK && primaryDetails.Success && primaryDetails.Data != nil {
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
		screenshotURLs = make([]string, 0, len(primaryDetails.Data.Screenshots))
		for _, screenshot := range primaryDetails.Data.Screenshots {
			if screenshot.PathFull != "" {
				screenshotURLs = append(screenshotURLs, screenshot.PathFull)
			}
		}
	}
	if fallbackOK && fallbackDetails.Success && fallbackDetails.Data != nil {
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
		CoverURL:       appliedCover,
		BannerURL:      appliedBanner,
		ScreenshotURLs: appliedScreenshots,
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func (s *SteamService) fetchAppDetails(appID int64, language string, proxyOverride string) (steamAppDetailsResponse, error) {
	endpoint := fmt.Sprintf("https://store.steampowered.com/api/appdetails?appids=%d&l=%s", appID, language)
	var payload steamAppDetailsResponse
	if err := s.fetchJSON(endpoint, &payload, proxyOverride); err != nil {
		return steamAppDetailsResponse{}, err
	}
	return payload, nil
}

func (s *SteamService) fetchJSON(endpoint string, target any, proxyOverride string) error {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", "https://store.steampowered.com/")

	resp, err := s.clientForProxy(proxyOverride).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("steam request failed with status %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(target)
}

func (s *SteamService) fetchText(endpoint string, proxyOverride string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", "https://store.steampowered.com/")

	resp, err := s.clientForProxy(proxyOverride).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("steam request failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (s *SteamService) clientForProxy(proxyOverride string) *http.Client {
	proxyOverride = strings.TrimSpace(proxyOverride)
	if proxyOverride == "" || proxyOverride == s.proxy {
		return s.client
	}

	transport := &http.Transport{Proxy: http.ProxyFromEnvironment}
	if parsed, err := url.Parse(proxyOverride); err == nil {
		transport.Proxy = http.ProxyURL(parsed)
	}

	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}
}

func (s *SteamService) resolveSteamAssetURL(appID int64, proxyOverride string, candidates ...string) *string {
	client := s.clientForProxy(proxyOverride)
	for _, pattern := range candidates {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		value := fmt.Sprintf(pattern, appID)
		exists, err := s.checkAssetExists(client, value)
		if err == nil && exists {
			return &value
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	fallback := strings.TrimSpace(candidates[len(candidates)-1])
	if fallback == "" {
		return nil
	}
	value := fmt.Sprintf(fallback, appID)
	return &value
}

func (s *SteamService) checkAssetExists(client *http.Client, assetURL string) (bool, error) {
	req, err := http.NewRequest(http.MethodHead, assetURL, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "image/webp,image/apng,image/*,*/*;q=0.8")
	req.Header.Set("Referer", "https://store.steampowered.com/")

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

func (s *SteamService) fetchScreenshotURLsFromStorePage(appID int64, proxyOverride string) []string {
	html, err := s.fetchText(fmt.Sprintf("https://store.steampowered.com/app/%d", appID), proxyOverride)
	if err != nil || html == "" {
		return []string{}
	}

	patterns := []*regexp.Regexp{
		regexp.MustCompile(`"path_full":"(https:\\/\\/[^"]+)"`),
		regexp.MustCompile(`"path_full\\":\\"(https:\\\\/\\\\/[^"]+)\\"`),
	}

	seen := map[string]struct{}{}
	results := make([]string, 0, 8)

	for _, pattern := range patterns {
		matches := pattern.FindAllStringSubmatch(html, -1)
		for _, match := range matches {
			if len(match) < 2 || match[1] == "" {
				continue
			}
			value := strings.ReplaceAll(match[1], `\/`, `/`)
			value = strings.ReplaceAll(value, `\\`, `\`)
			value = strings.ReplaceAll(value, `\u0026`, "&")
			if _, exists := seen[value]; exists {
				continue
			}
			seen[value] = struct{}{}
			results = append(results, value)
		}
		if len(results) > 0 {
			break
		}
	}

	return results
}

func (s *SteamService) fetchDescriptionFromStorePage(appID int64, proxyOverride string) string {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?s)<div[^>]*class="[^"]*\bgame_description_snippet\b[^"]*"[^>]*>(.*?)</div>`),
		regexp.MustCompile(`(?s)<div[^>]*class='[^']*\bgame_description_snippet\b[^']*'[^>]*>(.*?)</div>`),
	}

	for _, endpoint := range []string{
		fmt.Sprintf("https://store.steampowered.com/app/%d/?l=schinese", appID),
		fmt.Sprintf("https://store.steampowered.com/app/%d/?l=english", appID),
		fmt.Sprintf("https://store.steampowered.com/app/%d", appID),
	} {
		pageHTML, err := s.fetchText(endpoint, proxyOverride)
		if err != nil || pageHTML == "" {
			continue
		}

		for _, pattern := range patterns {
			match := pattern.FindStringSubmatch(pageHTML)
			if len(match) < 2 {
				continue
			}

			text := stripHTML(match[1])
			text = html.UnescapeString(text)
			text = strings.ReplaceAll(text, "\u00a0", " ")
			text = regexp.MustCompile(`\s+`).ReplaceAllString(strings.TrimSpace(text), " ")
			if text != "" {
				return text
			}
		}
	}

	return ""
}

func stripHTML(value string) string {
	return regexp.MustCompile(`(?s)<[^>]+>`).ReplaceAllString(value, " ")
}
