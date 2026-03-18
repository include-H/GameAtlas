package services

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	htmlpkg "html"
	"io"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/hao/game/internal/domain"
)

// steam_video_module: plugin-style Steam trailer discovery.
// It focuses on extracting direct MP4/WEBM URLs first and reporting DASH presence for diagnostics.

func appendUniqueURLs(target []string, values ...string) []string {
	if len(values) == 0 {
		return target
	}
	seen := make(map[string]struct{}, len(target))
	for _, value := range target {
		seen[value] = struct{}{}
	}
	for _, value := range values {
		url := strings.TrimSpace(value)
		if url == "" {
			continue
		}
		if _, exists := seen[url]; exists {
			continue
		}
		seen[url] = struct{}{}
		target = append(target, url)
	}
	return target
}

func collectTrailerURLsFromMovies(movies []steamMovieEntry) []string {
	urls := make([]string, 0, len(movies)*8)
	trailerPathPattern := regexp.MustCompile(`store_trailers[\\\/]+(\d+)[\\\/]+(\d+)[\\\/]+([a-f0-9]+)[\\\/]+(\d+)`)
	for _, movie := range movies {
		movieURLs := collectTrailerURLsFromAny(movie)
		urls = appendUniqueURLs(urls, movieURLs...)
		raw, err := json.Marshal(movie)
		if err != nil {
			continue
		}
		for _, match := range trailerPathPattern.FindAllStringSubmatch(string(raw), -1) {
			if len(match) != 5 {
				continue
			}
			appID := match[1]
			movieID := match[2]
			hash := match[3]
			version := match[4]
			urls = appendUniqueURLs(urls,
				fmt.Sprintf("https://video.fastly.steamstatic.com/store_trailers/%s/%s/%s/%s/movie_max.mp4", appID, movieID, hash, version),
				fmt.Sprintf("https://video.fastly.steamstatic.com/store_trailers/%s/%s/%s/%s/movie480.mp4", appID, movieID, hash, version),
				fmt.Sprintf("https://video.fastly.steamstatic.com/store_trailers/%s/%s/%s/%s/dash_h264.mpd", appID, movieID, hash, version),
				fmt.Sprintf("https://video.fastly.steamstatic.com/store_trailers/%s/%s/%s/%s/dash_av1.mpd", appID, movieID, hash, version),
			)
		}
	}
	return urls
}

func collectTrailerURLsFromAny(value any) []string {
	results := []string{}
	var walk func(node any)
	walk = func(node any) {
		switch v := node.(type) {
		case map[string]any:
			for _, item := range v {
				walk(item)
			}
		case []any:
			for _, item := range v {
				walk(item)
			}
		case string:
			candidate := normalizeSteamEscapedURL(v)
			if candidate == "" {
				return
			}
			lowered := strings.ToLower(candidate)
			if !strings.HasPrefix(lowered, "http") {
				return
			}
			if strings.Contains(lowered, ".mp4") || strings.Contains(lowered, ".webm") || strings.Contains(lowered, ".mpd") || strings.Contains(lowered, ".m3u8") {
				results = appendUniqueURLs(results, candidate)
			}
		}
	}
	walk(value)
	return results
}

func choosePreferredTrailerURL(candidates []string) *string {
	if len(candidates) == 0 {
		return nil
	}
	ranked := make([]string, 0, len(candidates))
	ranked = append(ranked, candidates...)
	score := func(url string) int {
		lowered := strings.ToLower(url)
		value := 0
		if strings.Contains(lowered, ".mp4") {
			value += 100
		}
		if strings.Contains(lowered, "/max") || strings.Contains(lowered, "max.") {
			value += 40
		}
		if strings.Contains(lowered, "1080") {
			value += 30
		}
		if strings.Contains(lowered, "720") {
			value += 20
		}
		if strings.Contains(lowered, ".webm") {
			value += 10
		}
		if strings.Contains(lowered, ".mpd") || strings.Contains(lowered, ".m3u8") {
			value -= 100
		}
		return value
	}

	best := strings.TrimSpace(ranked[0])
	bestScore := score(best)
	for _, item := range ranked[1:] {
		url := strings.TrimSpace(item)
		if url == "" {
			continue
		}
		value := score(url)
		if value > bestScore {
			best = url
			bestScore = value
		}
	}
	if best == "" {
		return nil
	}
	return &best
}

func trailerDisplayName(rawURL *string) *string {
	if rawURL == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*rawURL)
	if trimmed == "" {
		return nil
	}

	parsed, err := url.Parse(trimmed)
	if err == nil {
		name := strings.TrimSpace(pathBase(parsed.Path))
		if name != "" {
			return &name
		}
	}

	name := strings.TrimSpace(pathBase(trimmed))
	if name == "" {
		return nil
	}
	return &name
}

func buildSteamVideoCandidates(values []string) []domain.SteamVideoCandidate {
	if len(values) == 0 {
		return []domain.SteamVideoCandidate{}
	}
	grouped := make(map[string][]string)
	order := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		groupKey := trailerGroupKey(trimmed)
		if _, exists := grouped[groupKey]; !exists {
			order = append(order, groupKey)
		}
		grouped[groupKey] = appendUniqueURLs(grouped[groupKey], trimmed)
	}

	results := make([]domain.SteamVideoCandidate, 0, len(grouped))
	for index, groupKey := range order {
		candidates := grouped[groupKey]
		selected := choosePreferredTrailerCandidate(candidates)
		if selected == nil || strings.TrimSpace(*selected) == "" {
			continue
		}
		lowered := strings.ToLower(*selected)
		results = append(results, domain.SteamVideoCandidate{
			URL:    *selected,
			Name:   fmt.Sprintf("Trailer %d", index+1),
			IsDash: strings.Contains(lowered, ".mpd") || strings.Contains(lowered, ".m3u8"),
		})
	}
	return results
}

func choosePreferredTrailerCandidate(candidates []string) *string {
	if len(candidates) == 0 {
		return nil
	}

	dashCandidates := make([]string, 0, len(candidates))
	directCandidates := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		lowered := strings.ToLower(strings.TrimSpace(candidate))
		if lowered == "" {
			continue
		}
		if strings.Contains(lowered, ".mpd") || strings.Contains(lowered, ".m3u8") {
			dashCandidates = append(dashCandidates, candidate)
			continue
		}
		directCandidates = append(directCandidates, candidate)
	}

	if selected := choosePreferredDashURL(dashCandidates); selected != nil {
		return selected
	}
	return choosePreferredTrailerURL(directCandidates)
}

func choosePreferredDashURL(candidates []string) *string {
	if len(candidates) == 0 {
		return nil
	}

	best := ""
	bestScore := -1
	for _, candidate := range candidates {
		trimmed := strings.TrimSpace(candidate)
		if trimmed == "" {
			continue
		}
		lowered := strings.ToLower(trimmed)
		score := 0
		if strings.Contains(lowered, "dash_h264") {
			score += 200
		}
		if strings.Contains(lowered, "dash_av1") {
			score += 120
		}
		if strings.Contains(lowered, ".mpd") {
			score += 40
		}
		if strings.Contains(lowered, ".m3u8") {
			score += 20
		}
		if score > bestScore {
			best = trimmed
			bestScore = score
		}
	}
	if best == "" {
		return nil
	}
	return &best
}

func trailerGroupKey(rawURL string) string {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return ""
	}

	parsed, err := url.Parse(trimmed)
	if err == nil {
		if key := trailerGroupKeyFromPath(parsed.Path); key != "" {
			return key
		}
	}

	if key := trailerGroupKeyFromPath(trimmed); key != "" {
		return key
	}
	return strings.ToLower(trimmed)
}

func trailerGroupKeyFromPath(path string) string {
	storeTrailerPattern := regexp.MustCompile(`(?i)(store_trailers/\d+/\d+/[a-f0-9]+/\d+)`)
	if match := storeTrailerPattern.FindStringSubmatch(path); len(match) == 2 {
		return strings.ToLower(match[1])
	}

	lowered := strings.ToLower(path)
	lowered = strings.TrimSuffix(lowered, ".mp4")
	lowered = strings.TrimSuffix(lowered, ".webm")
	lowered = strings.TrimSuffix(lowered, ".mpd")
	lowered = strings.TrimSuffix(lowered, ".m3u8")
	replacements := []string{
		"dash_h264",
		"dash_av1",
		"movie_max",
		"movie480",
		"movie_480",
		"trailer_1080p",
		"trailer_720p",
	}
	for _, replacement := range replacements {
		lowered = strings.ReplaceAll(lowered, replacement, "trailer")
	}
	return lowered
}

func candidateURLs(values []domain.SteamVideoCandidate) []string {
	if len(values) == 0 {
		return []string{}
	}
	results := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value.URL) == "" {
			continue
		}
		results = append(results, value.URL)
	}
	return results
}

func (s *SteamService) fetchVideoURLsFromStorePage(appID int64, proxyOverride string) ([]string, []string) {
	html, err := s.fetchText(fmt.Sprintf("https://store.steampowered.com/app/%d/?l=english", appID), proxyOverride)
	if err != nil || html == "" {
		return []string{}, []string{"store-page-fetch=failed"}
	}

	debug := make([]string, 0, 6)
	results := make([]string, 0, 8)
	dashMatches := 0

	appendFromRaw := func(raw string) {
		url := normalizeSteamEscapedURL(raw)
		if url == "" || !strings.HasPrefix(url, "http") {
			return
		}
		lowered := strings.ToLower(url)
		if strings.Contains(lowered, ".mpd") || strings.Contains(lowered, ".m3u8") {
			dashMatches++
			results = appendUniqueURLs(results, url)
			return
		}
		if !strings.Contains(lowered, ".mp4") && !strings.Contains(lowered, ".webm") {
			return
		}
		results = appendUniqueURLs(results, url)
	}

	if match := regexp.MustCompile(`rgMovieFlashvars\s*=\s*(\{.*?\});`).FindStringSubmatch(html); len(match) == 2 {
		type movieFlashvars struct {
			WEBMSource string `json:"WEBM_SOURCE"`
			MP4Source  string `json:"MP4_SOURCE"`
			DashAV1    string `json:"DASH_AV1_SOURCE"`
			DashH264   string `json:"DASH_H264_SOURCE"`
		}
		flashvars := map[string]movieFlashvars{}
		if err := json.Unmarshal([]byte(match[1]), &flashvars); err == nil {
			debug = append(debug, fmt.Sprintf("rgMovieFlashvars.entries=%d", len(flashvars)))
			for _, entry := range flashvars {
				appendFromRaw(entry.WEBMSource)
				appendFromRaw(entry.MP4Source)
				if strings.TrimSpace(entry.DashAV1) != "" || strings.TrimSpace(entry.DashH264) != "" {
					dashMatches++
				}
			}
		} else {
			debug = append(debug, "rgMovieFlashvars.parse=failed")
		}
	} else {
		debug = append(debug, "rgMovieFlashvars=not-found")
	}

	patterns := []*regexp.Regexp{
		regexp.MustCompile(`"(?:WEBM_SOURCE|MP4_SOURCE)"\s*:\s*"([^"]+)"`),
		regexp.MustCompile(`https?:\\?\/\\?\/[^"'\\s]*(?:steamstatic|akamai)[^"'\\s]*\.(?:mp4|webm|mpd)`),
		regexp.MustCompile(`https://[^"'\\s]*(?:steamstatic|akamai)[^"'\\s]*\.(?:mp4|webm|mpd)`),
		regexp.MustCompile(`https:\\\\/\\\\/[^"'\\s]*(?:steamstatic|akamai)[^"'\\s]*\.(?:mp4|webm|mpd)`),
	}
	for _, pattern := range patterns {
		matches := pattern.FindAllStringSubmatch(html, -1)
		for _, match := range matches {
			if len(match) > 1 && match[1] != "" {
				appendFromRaw(match[1])
				continue
			}
			appendFromRaw(match[0])
		}
	}

	// Derive DASH URLs from store_trailers fragments (plugin method fallback).
	trailerPathPattern := regexp.MustCompile(`store_trailers[\\\/]+(\d+)[\\\/]+(\d+)[\\\/]+([a-f0-9]+)[\\\/]+(\d+)`)
	for _, match := range trailerPathPattern.FindAllStringSubmatch(html, -1) {
		if len(match) != 5 {
			continue
		}
		app := match[1]
		movieID := match[2]
		hash := match[3]
		version := match[4]
		appendFromRaw(fmt.Sprintf("https://video.fastly.steamstatic.com/store_trailers/%s/%s/%s/%s/dash_h264.mpd", app, movieID, hash, version))
		appendFromRaw(fmt.Sprintf("https://video.fastly.steamstatic.com/store_trailers/%s/%s/%s/%s/dash_av1.mpd", app, movieID, hash, version))
	}

	// Parse data-appassets for extras direct videos (plugin findExtras).
	if match := regexp.MustCompile(`data-appassets="([^"]+)"`).FindStringSubmatch(html); len(match) == 2 {
		raw := htmlpkg.UnescapeString(match[1])
		raw = strings.ReplaceAll(raw, "&quot;", `"`)
		type assetVariant struct {
			Extension string `json:"extension"`
			URLPart   string `json:"urlPart"`
		}
		assets := map[string][]assetVariant{}
		if err := json.Unmarshal([]byte(raw), &assets); err == nil {
			debug = append(debug, fmt.Sprintf("data-appassets.entries=%d", len(assets)))
			for _, variants := range assets {
				for _, variant := range variants {
					ext := strings.ToLower(strings.TrimSpace(variant.Extension))
					if ext != "mp4" && ext != "webm" {
						continue
					}
					urlPart := strings.TrimSpace(variant.URLPart)
					if urlPart == "" {
						continue
					}
					appendFromRaw(fmt.Sprintf("https://shared.fastly.steamstatic.com/store_item_assets/steam/apps/%d/%s", appID, strings.TrimPrefix(urlPart, "/")))
				}
			}
		} else {
			debug = append(debug, "data-appassets.parse=failed")
		}
	} else {
		debug = append(debug, "data-appassets=not-found")
	}

	debug = append(debug, fmt.Sprintf("store-page-direct-candidates=%d", len(results)))
	if dashMatches > 0 {
		debug = append(debug, fmt.Sprintf("store-page-dash-detected=%d", dashMatches))
	}
	return results, debug
}

func normalizeSteamEscapedURL(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	value = strings.Trim(value, `"'`)
	value = strings.ReplaceAll(value, `\\u002F`, `/`)
	value = strings.ReplaceAll(value, `\u002F`, `/`)
	value = strings.ReplaceAll(value, `\\u0026`, "&")
	value = strings.ReplaceAll(value, `\u0026`, "&")
	value = strings.ReplaceAll(value, `\\\/`, `/`)
	value = strings.ReplaceAll(value, `\/`, `/`)
	value = strings.ReplaceAll(value, `\\`, "")
	return strings.TrimSpace(value)
}

func (s *SteamService) resolveProgressiveTrailerFromManifest(manifestURL string, proxyOverride string) (string, []string) {
	debug := []string{}
	trimmed := strings.TrimSpace(manifestURL)
	if trimmed == "" {
		return "", []string{"manifest-url=empty"}
	}
	lowered := strings.ToLower(trimmed)
	if !strings.Contains(lowered, ".mpd") && !strings.Contains(lowered, ".m3u8") {
		return "", []string{"manifest-url=not-dash"}
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", []string{"manifest-url=parse-failed"}
	}
	base := trimmed
	if idx := strings.LastIndex(base, "/"); idx > 0 {
		base = base[:idx+1]
	}

	candidates := []string{
		base + "movie_max.mp4",
		base + "movie480.mp4",
		base + "movie_480.mp4",
		base + "movie_max.webm",
		base + "movie480.webm",
		base + "movie_480.webm",
	}

	// If path hints a quality token, try same family with mp4/webm swap.
	fileName := pathBase(parsed.Path)
	if fileName != "" {
		withoutExt := strings.TrimSuffix(fileName, filepathExt(fileName))
		candidates = append(candidates,
			base+withoutExt+".mp4",
			base+withoutExt+".webm",
		)
	}

	candidates = uniqueNonEmpty(candidates)
	debug = append(debug, fmt.Sprintf("progressive-probe-candidates=%d", len(candidates)))
	for _, candidate := range candidates {
		ok, kind := s.checkVideoURLExists(candidate, proxyOverride)
		if !ok {
			continue
		}
		debug = append(debug, "progressive-selected="+truncateDebugURL(candidate)+" ("+kind+")")
		return candidate, debug
	}
	debug = append(debug, "progressive-probe=not-found")
	return "", debug
}

func (s *SteamService) checkVideoURLExists(assetURL string, proxyOverride string) (bool, string) {
	client := s.clientForProxy(proxyOverride)
	headReq, err := http.NewRequest(http.MethodHead, assetURL, nil)
	if err == nil {
		headReq.Header.Set("User-Agent", "Mozilla/5.0")
		headReq.Header.Set("Accept", "*/*")
		resp, doErr := client.Do(headReq)
		if doErr == nil {
			defer resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				ct := strings.ToLower(strings.TrimSpace(resp.Header.Get("Content-Type")))
				if strings.Contains(ct, "video/") || strings.Contains(strings.ToLower(assetURL), ".mp4") || strings.Contains(strings.ToLower(assetURL), ".webm") {
					return true, "head"
				}
			}
		}
	}

	// Some CDN edges reject HEAD; do a tiny range GET probe.
	getReq, err := http.NewRequest(http.MethodGet, assetURL, nil)
	if err != nil {
		return false, ""
	}
	getReq.Header.Set("User-Agent", "Mozilla/5.0")
	getReq.Header.Set("Accept", "*/*")
	getReq.Header.Set("Range", "bytes=0-1023")
	resp, doErr := client.Do(getReq)
	if doErr != nil {
		return false, ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return false, ""
	}
	ct := strings.ToLower(strings.TrimSpace(resp.Header.Get("Content-Type")))
	if strings.Contains(ct, "video/") || strings.Contains(strings.ToLower(assetURL), ".mp4") || strings.Contains(strings.ToLower(assetURL), ".webm") {
		return true, "range-get"
	}
	return false, ""
}

func uniqueNonEmpty(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func pathBase(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func filepathExt(name string) string {
	idx := strings.LastIndex(name, ".")
	if idx < 0 {
		return ""
	}
	return name[idx:]
}

type dashMPD struct {
	MediaPresentationDuration string       `xml:"mediaPresentationDuration,attr"`
	Periods                   []dashPeriod `xml:"Period"`
}

type dashPeriod struct {
	AdaptationSets []dashAdaptationSet `xml:"AdaptationSet"`
}

type dashAdaptationSet struct {
	ContentType     string               `xml:"contentType,attr"`
	MimeType        string               `xml:"mimeType,attr"`
	Representations []dashRepresentation `xml:"Representation"`
	SegmentTemplate *dashSegmentTemplate `xml:"SegmentTemplate"`
	BaseURL         string               `xml:"BaseURL"`
}

type dashRepresentation struct {
	ID              string               `xml:"id,attr"`
	Bandwidth       int                  `xml:"bandwidth,attr"`
	Width           int                  `xml:"width,attr"`
	Height          int                  `xml:"height,attr"`
	MimeType        string               `xml:"mimeType,attr"`
	BaseURL         string               `xml:"BaseURL"`
	SegmentTemplate *dashSegmentTemplate `xml:"SegmentTemplate"`
}

type dashSegmentTemplate struct {
	Timescale       int                  `xml:"timescale,attr"`
	Duration        int                  `xml:"duration,attr"`
	StartNumber     int                  `xml:"startNumber,attr"`
	Media           string               `xml:"media,attr"`
	Initialization  string               `xml:"initialization,attr"`
	SegmentTimeline *dashSegmentTimeline `xml:"SegmentTimeline"`
}

type dashSegmentTimeline struct {
	Segments []dashTimelineSegment `xml:"S"`
}

type dashTimelineSegment struct {
	T int64 `xml:"t,attr"`
	D int64 `xml:"d,attr"`
	R int   `xml:"r,attr"`
}

func (s *SteamService) downloadDashVideoTrack(mpdURL string, proxyOverride string) ([]byte, []string, error) {
	debug := []string{fmt.Sprintf("dash-mpd=%s", truncateDebugURL(mpdURL))}
	mpdData, err := s.fetchBinary(mpdURL, proxyOverride)
	if err != nil {
		return nil, append(debug, "dash-fetch-mpd=failed"), err
	}

	var manifest dashMPD
	if err := xml.Unmarshal(mpdData, &manifest); err != nil {
		return nil, append(debug, "dash-parse-mpd=failed"), err
	}

	baseURL, err := url.Parse(mpdURL)
	if err != nil {
		return nil, append(debug, "dash-base-url=failed"), err
	}

	videoRep, repTemplate, repBase := selectBestVideoRepresentation(manifest)
	if videoRep == nil || repTemplate == nil {
		return nil, append(debug, "dash-video-representation=not-found"), fmt.Errorf("no video representation")
	}

	debug = append(debug, fmt.Sprintf("dash-video-selected=id:%s bw:%d res:%dx%d", videoRep.ID, videoRep.Bandwidth, videoRep.Width, videoRep.Height))

	initURL, err := buildDashSegmentURL(baseURL, repBase, repTemplate.Initialization, videoRep.ID, videoRep.Bandwidth, 0, 0)
	if err != nil {
		return nil, append(debug, "dash-init-url=failed"), err
	}
	initData, err := s.fetchBinary(initURL, proxyOverride)
	if err != nil {
		return nil, append(debug, "dash-init-download=failed"), err
	}

	segmentURLs := buildDashSegmentList(baseURL, repBase, repTemplate, videoRep.ID, videoRep.Bandwidth, manifest.MediaPresentationDuration)
	if len(segmentURLs) == 0 {
		return nil, append(debug, "dash-segments=0"), fmt.Errorf("no dash segments")
	}
	debug = append(debug, fmt.Sprintf("dash-segments=%d", len(segmentURLs)))

	payload := make([]byte, 0, len(initData)+len(segmentURLs)*400*1024)
	payload = append(payload, initData...)
	for i, segmentURL := range segmentURLs {
		segmentData, segmentErr := s.fetchBinary(segmentURL, proxyOverride)
		if segmentErr != nil {
			debug = append(debug, fmt.Sprintf("dash-segment-failed=%d", i))
			return nil, debug, segmentErr
		}
		payload = append(payload, segmentData...)
	}
	debug = append(debug, "dash-track-download=ok(video-only)")
	return payload, debug, nil
}

func selectBestVideoRepresentation(manifest dashMPD) (*dashRepresentation, *dashSegmentTemplate, string) {
	var best *dashRepresentation
	var bestTemplate *dashSegmentTemplate
	bestBase := ""
	score := -1

	for _, period := range manifest.Periods {
		for _, adaptation := range period.AdaptationSets {
			lowerType := strings.ToLower(adaptation.ContentType)
			lowerMime := strings.ToLower(adaptation.MimeType)
			isVideoSet := lowerType == "video" || strings.Contains(lowerMime, "video")
			if !isVideoSet {
				continue
			}

			for i := range adaptation.Representations {
				rep := &adaptation.Representations[i]
				repMime := strings.ToLower(rep.MimeType)
				if repMime != "" && !strings.Contains(repMime, "video") {
					continue
				}
				repScore := rep.Bandwidth + rep.Height*1000 + rep.Width*10
				if repScore <= score {
					continue
				}
				template := rep.SegmentTemplate
				if template == nil {
					template = adaptation.SegmentTemplate
				}
				if template == nil || strings.TrimSpace(template.Media) == "" || strings.TrimSpace(template.Initialization) == "" {
					continue
				}
				score = repScore
				best = rep
				bestTemplate = template
				if strings.TrimSpace(rep.BaseURL) != "" {
					bestBase = strings.TrimSpace(rep.BaseURL)
				} else {
					bestBase = strings.TrimSpace(adaptation.BaseURL)
				}
			}
		}
	}

	return best, bestTemplate, bestBase
}

func buildDashSegmentList(base *url.URL, repBase string, template *dashSegmentTemplate, representationID string, bandwidth int, mediaDuration string) []string {
	if template == nil {
		return []string{}
	}

	if template.StartNumber <= 0 {
		template.StartNumber = 1
	}
	if template.Timescale <= 0 {
		template.Timescale = 1
	}

	urls := make([]string, 0, 128)

	if template.SegmentTimeline != nil && len(template.SegmentTimeline.Segments) > 0 {
		current := int64(0)
		for _, segment := range template.SegmentTimeline.Segments {
			if segment.D <= 0 {
				continue
			}
			if segment.T > 0 {
				current = segment.T
			}
			repeat := segment.R
			if repeat < 0 {
				repeat = 0
			}
			for i := 0; i <= repeat; i++ {
				url, err := buildDashSegmentURL(base, repBase, template.Media, representationID, bandwidth, 0, current)
				if err == nil {
					urls = append(urls, url)
				}
				current += segment.D
			}
		}
		return urls
	}

	if template.Duration <= 0 {
		return urls
	}
	durationSeconds := parseISO8601DurationSeconds(mediaDuration)
	if durationSeconds <= 0 {
		// conservative fallback
		durationSeconds = 90
	}
	segmentSeconds := float64(template.Duration) / float64(template.Timescale)
	if segmentSeconds <= 0 {
		return urls
	}
	count := int(math.Ceil(durationSeconds / segmentSeconds))
	if count < 1 {
		count = 1
	}
	if count > 1200 {
		count = 1200
	}
	for i := 0; i < count; i++ {
		number := template.StartNumber + i
		url, err := buildDashSegmentURL(base, repBase, template.Media, representationID, bandwidth, number, 0)
		if err == nil {
			urls = append(urls, url)
		}
	}
	return urls
}

func buildDashSegmentURL(base *url.URL, repBase string, template string, representationID string, bandwidth int, number int, timeValue int64) (string, error) {
	path := template
	path = replaceDashIdentifier(path, "RepresentationID", representationID)
	path = replaceDashInteger(path, "Bandwidth", int64(bandwidth))
	if number > 0 {
		path = replaceDashInteger(path, "Number", int64(number))
	}
	if timeValue > 0 {
		path = replaceDashInteger(path, "Time", timeValue)
	}

	full := path
	if strings.TrimSpace(repBase) != "" && !strings.HasPrefix(path, "http://") && !strings.HasPrefix(path, "https://") {
		full = strings.TrimSuffix(repBase, "/") + "/" + strings.TrimPrefix(path, "/")
	}
	parsed, err := url.Parse(full)
	if err != nil {
		return "", err
	}
	return base.ResolveReference(parsed).String(), nil
}

func replaceDashIdentifier(template string, key string, value string) string {
	pattern := regexp.MustCompile(`\$` + key + `(?:%[^$]+)?\$`)
	return pattern.ReplaceAllString(template, value)
}

func replaceDashInteger(template string, key string, value int64) string {
	pattern := regexp.MustCompile(`\$` + key + `(?:%0?(\d+)d)?\$`)
	return pattern.ReplaceAllStringFunc(template, func(match string) string {
		formatMatch := pattern.FindStringSubmatch(match)
		if len(formatMatch) > 1 && formatMatch[1] != "" {
			if width, err := strconv.Atoi(formatMatch[1]); err == nil && width > 0 {
				return fmt.Sprintf("%0*d", width, value)
			}
		}
		return strconv.FormatInt(value, 10)
	})
}

func parseISO8601DurationSeconds(value string) float64 {
	if strings.TrimSpace(value) == "" {
		return 0
	}
	pattern := regexp.MustCompile(`^P(?:([0-9]+)D)?(?:T(?:([0-9]+)H)?(?:([0-9]+)M)?(?:([0-9]+(?:\.[0-9]+)?)S)?)?$`)
	match := pattern.FindStringSubmatch(strings.TrimSpace(value))
	if len(match) == 0 {
		return 0
	}
	total := 0.0
	if match[1] != "" {
		if days, err := strconv.Atoi(match[1]); err == nil {
			total += float64(days * 24 * 3600)
		}
	}
	if match[2] != "" {
		if hours, err := strconv.Atoi(match[2]); err == nil {
			total += float64(hours * 3600)
		}
	}
	if match[3] != "" {
		if minutes, err := strconv.Atoi(match[3]); err == nil {
			total += float64(minutes * 60)
		}
	}
	if match[4] != "" {
		if seconds, err := strconv.ParseFloat(match[4], 64); err == nil {
			total += seconds
		}
	}
	return total
}

func (s *SteamService) fetchBinary(endpoint string, proxyOverride string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Referer", "https://store.steampowered.com/")

	resp, err := s.clientForProxy(proxyOverride).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed: status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}
