package services

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

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

type namedError struct {
	name string
	err  error
}

func wrapSteamUpstreamError(message string, errs ...namedError) error {
	details := make([]string, 0, len(errs))
	for _, item := range errs {
		if item.err == nil {
			continue
		}
		details = append(details, fmt.Sprintf("%s: %v", item.name, item.err))
	}
	if len(details) == 0 {
		return fmt.Errorf("%w: %s", ErrUpstream, message)
	}
	return fmt.Errorf("%w: %s (%s)", ErrUpstream, message, strings.Join(details, "; "))
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

			text := stripSteamHTML(match[1])
			if text != "" {
				return text
			}
		}
	}

	return ""
}

func stripSteamHTML(raw string) string {
	value := html.UnescapeString(raw)
	value = regexp.MustCompile(`(?is)<br\s*/?>`).ReplaceAllString(value, " ")
	value = regexp.MustCompile(`(?is)<[^>]+>`).ReplaceAllString(value, " ")
	value = strings.ReplaceAll(value, "\u00a0", " ")
	value = strings.Join(strings.Fields(value), " ")
	return strings.TrimSpace(value)
}
