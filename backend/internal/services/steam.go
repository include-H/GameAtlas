package services

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
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
	primaryPayload, _ := s.fetchAppDetails(appID, "schinese", proxyOverride)
	fallbackPayload, _ := s.fetchAppDetails(appID, "english", proxyOverride)

	primaryDetails, primaryOK := primaryPayload[appKey]
	fallbackDetails, fallbackOK := fallbackPayload[appKey]
	name := fmt.Sprintf("Steam App %d", appID)
	description := ""
	releaseDate := ""
	developers := []string{}
	publishers := []string{}
	videoDebug := []string{}
	trailerCandidates := []string{}
	screenshotURLs := []string{}
	description = s.fetchDescriptionFromStorePage(appID, proxyOverride)
	if primaryOK && primaryDetails.Success && primaryDetails.Data != nil {
		videoDebug = append(videoDebug, fmt.Sprintf("appdetails(schinese).movies=%d", len(primaryDetails.Data.Movies)))
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
		trailerCandidates = appendUniqueURLs(trailerCandidates, collectTrailerURLsFromMovies(primaryDetails.Data.Movies)...)
		videoDebug = append(videoDebug, fmt.Sprintf("movies-candidates(schinese)=%d", len(trailerCandidates)))
		screenshotURLs = make([]string, 0, len(primaryDetails.Data.Screenshots))
		for _, screenshot := range primaryDetails.Data.Screenshots {
			if screenshot.PathFull != "" {
				screenshotURLs = append(screenshotURLs, screenshot.PathFull)
			}
		}
	}
	if !primaryOK || (primaryOK && !primaryDetails.Success) {
		videoDebug = append(videoDebug, "appdetails(schinese) unavailable")
	}
	if fallbackOK && fallbackDetails.Success && fallbackDetails.Data != nil {
		videoDebug = append(videoDebug, fmt.Sprintf("appdetails(english).movies=%d", len(fallbackDetails.Data.Movies)))
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
		if len(trailerCandidates) == 0 {
			trailerCandidates = appendUniqueURLs(trailerCandidates, collectTrailerURLsFromMovies(fallbackDetails.Data.Movies)...)
			videoDebug = append(videoDebug, fmt.Sprintf("movies-candidates(english)=%d", len(trailerCandidates)))
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
	if !fallbackOK || (fallbackOK && !fallbackDetails.Success) {
		videoDebug = append(videoDebug, "appdetails(english) unavailable")
	}
	if len(screenshotURLs) == 0 {
		screenshotURLs = s.fetchScreenshotURLsFromStorePage(appID, proxyOverride)
	}
	if len(trailerCandidates) == 0 {
		videoDebug = append(videoDebug, "movies-candidates=0, fallback to store page extraction")
		storePageCandidates, storePageDebug := s.fetchVideoURLsFromStorePage(appID, proxyOverride)
		trailerCandidates = storePageCandidates
		videoDebug = append(videoDebug, storePageDebug...)
		videoDebug = append(videoDebug, fmt.Sprintf("store-page-candidates=%d", len(trailerCandidates)))
	}
	previewVideos := buildSteamVideoCandidates(trailerCandidates)
	previewVideoURL := choosePreferredTrailerCandidate(candidateURLs(previewVideos))
	previewVideoName := trailerDisplayName(previewVideoURL)
	if previewVideoURL == nil {
		videoDebug = append(videoDebug, "no downloadable trailer source found")
	} else {
		if strings.Contains(strings.ToLower(*previewVideoURL), ".mpd") || strings.Contains(strings.ToLower(*previewVideoURL), ".m3u8") {
			videoDebug = append(videoDebug, "selected=dash-manifest")
		}
		videoDebug = append(videoDebug, "selected="+truncateDebugURL(*previewVideoURL))
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
		AppID:             appID,
		Name:              name,
		Description:       description,
		ReleaseDate:       releaseDate,
		Developers:        developers,
		Publishers:        publishers,
		PreviewVideos:     previewVideos,
		PreviewVideoURL:   previewVideoURL,
		PreviewVideoName:  previewVideoName,
		PreviewVideoDebug: videoDebug,
		CoverURL:          coverURL,
		BannerURL:         bannerURL,
		ScreenshotURLs:    screenshotURLs,
	}, nil
}

func (s *SteamService) ApplyAssets(appID int64, input domain.SteamApplyAssetsInput) (*domain.SteamAssetsPreview, error) {
	if input.GameID <= 0 {
		return nil, ErrValidation
	}

	sortOrder := 0
	var appliedCover *string
	var appliedBanner *string
	var appliedPreviewVideo *string
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
	if input.PreviewVideoURL != nil && *input.PreviewVideoURL != "" {
		videoURL := strings.TrimSpace(*input.PreviewVideoURL)
		lowered := strings.ToLower(videoURL)
		if strings.Contains(lowered, ".mpd") || strings.Contains(lowered, ".m3u8") {
			progressiveURL, _ := s.resolveProgressiveTrailerFromManifest(videoURL, "")
			if progressiveURL != "" {
				path, err := s.assets.ApplyRemoteAsset(input.GameID, "video", progressiveURL, 0)
				if err != nil {
					return nil, err
				}
				appliedPreviewVideo = &path
			} else {
				videoData, _, dashErr := s.downloadDashVideoTrack(videoURL, "")
				if dashErr != nil {
					return nil, dashErr
				}
				path, err := s.assets.ApplyRawAsset(input.GameID, "video", videoData, "video/mp4", 0)
				if err != nil {
					return nil, err
				}
				appliedPreviewVideo = &path
			}
		} else {
			path, err := s.assets.ApplyRemoteAsset(input.GameID, "video", videoURL, 0)
			if err != nil {
				return nil, err
			}
			appliedPreviewVideo = &path
		}
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
		AppID:             appID,
		ReleaseDate:       "",
		Developers:        []string{},
		Publishers:        []string{},
		PreviewVideos:     []domain.SteamVideoCandidate{},
		PreviewVideoURL:   appliedPreviewVideo,
		PreviewVideoName:  trailerDisplayName(appliedPreviewVideo),
		PreviewVideoDebug: []string{},
		CoverURL:          appliedCover,
		BannerURL:         appliedBanner,
		ScreenshotURLs:    appliedScreenshots,
	}, nil
}

func truncateDebugURL(value string) string {
	const limit = 140
	trimmed := strings.TrimSpace(value)
	if len(trimmed) <= limit {
		return trimmed
	}
	return trimmed[:limit] + "..."
}

func cleanSteamNames(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	results := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		results = append(results, trimmed)
	}
	return results
}

func normalizeSteamReleaseDate(release *struct {
	ComingSoon bool   `json:"coming_soon"`
	Date       string `json:"date"`
}) string {
	if release == nil || release.ComingSoon {
		return ""
	}
	raw := strings.TrimSpace(release.Date)
	if raw == "" {
		return ""
	}

	lowered := strings.ToLower(raw)
	if strings.Contains(lowered, "coming soon") || strings.Contains(lowered, "to be announced") {
		return ""
	}

	if year, month, day, ok := parseSteamChineseDate(raw); ok {
		return fmt.Sprintf("%04d-%02d-%02d", year, month, day)
	}

	for _, layout := range []string{
		"2 Jan, 2006",
		"Jan 2, 2006",
		"January 2, 2006",
		"2 January, 2006",
		"2 Jan 2006",
		"Jan 2 2006",
		"2006-01-02",
		"02 Jan, 2006",
		"Jan 02, 2006",
	} {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return parsed.Format("2006-01-02")
		}
	}

	return ""
}

func parseSteamChineseDate(value string) (int, int, int, bool) {
	pattern := regexp.MustCompile(`(\d{4})\s*年\s*(\d{1,2})\s*月\s*(\d{1,2})\s*日`)
	match := pattern.FindStringSubmatch(value)
	if len(match) != 4 {
		return 0, 0, 0, false
	}

	year, err1 := strconv.Atoi(match[1])
	month, err2 := strconv.Atoi(match[2])
	day, err3 := strconv.Atoi(match[3])
	if err1 != nil || err2 != nil || err3 != nil {
		return 0, 0, 0, false
	}
	return year, month, day, true
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

	resp, err := s.doRequest(req, proxyOverride)
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

	resp, err := s.doRequest(req, proxyOverride)
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

func (s *SteamService) doRequest(req *http.Request, proxyOverride string) (*http.Response, error) {
	log.Printf("steam outbound request: method=%s url=%s proxy=%s", req.Method, req.URL.String(), s.proxyLogValue(proxyOverride))
	return s.clientForProxy(proxyOverride).Do(req)
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

func (s *SteamService) proxyLogValue(proxyOverride string) string {
	proxyOverride = strings.TrimSpace(proxyOverride)
	if proxyOverride == "" {
		if strings.TrimSpace(s.proxy) == "" {
			return "direct"
		}
		return s.proxy
	}
	return proxyOverride
}

func (s *SteamService) ProxyAsset(assetURL string, proxyOverride string) (string, []byte, error) {
	parsed, err := url.Parse(strings.TrimSpace(assetURL))
	if err != nil {
		return "", nil, ErrValidation
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", nil, ErrValidation
	}

	if parsed.Hostname() == "" {
		return "", nil, ErrValidation
	}

	req, err := http.NewRequest(http.MethodGet, parsed.String(), nil)
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", "https://store.steampowered.com/")

	resp, err := s.doRequest(req, proxyOverride)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return "", nil, fmt.Errorf("steam request failed with status %d", resp.StatusCode)
	}

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, err
	}
	return strings.TrimSpace(resp.Header.Get("Content-Type")), payload, nil
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
