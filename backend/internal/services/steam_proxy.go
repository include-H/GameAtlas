package services

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

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
	if !isAllowedSteamAssetHost(parsed.Hostname()) {
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

func sanitizeRawURLForLog(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed == nil {
		if strings.TrimSpace(raw) == "" {
			return "direct"
		}
		return strings.TrimSpace(raw)
	}
	return sanitizeURLForLog(parsed)
}

func sanitizeURLForLog(value *url.URL) string {
	if value == nil {
		return "direct"
	}

	sanitized := *value
	if sanitized.User != nil {
		username := sanitized.User.Username()
		if _, hasPassword := sanitized.User.Password(); hasPassword {
			sanitized.User = url.UserPassword(username, "REDACTED")
		} else if username != "" {
			sanitized.User = url.User(username)
		} else {
			sanitized.User = nil
		}
	}

	return sanitized.String()
}

func isAllowedSteamAssetHost(host string) bool {
	host = strings.Trim(strings.ToLower(strings.TrimSpace(host)), ".")
	if host == "" {
		return false
	}

	if host == "steamcdn-a.akamaihd.net" {
		return true
	}

	return strings.HasSuffix(host, ".steamstatic.com") || strings.HasSuffix(host, ".steampowered.com")
}
