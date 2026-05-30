package services

import (
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func (s *SteamService) doRequest(req *http.Request, proxyOverride string) (*http.Response, error) {
	log.Printf(
		"steam outbound request: method=%s url=%s proxy=%s",
		req.Method,
		sanitizeURLForLog(req.URL),
		s.proxyLogValue(proxyOverride),
	)
	return s.clientForProxy(proxyOverride).Do(req)
}

func (s *SteamService) clientForProxy(proxyOverride string) *http.Client {
	proxyOverride = strings.TrimSpace(proxyOverride)
	if proxyOverride == "" || proxyOverride == s.proxy {
		return s.client
	}

	if cached, ok := s.proxyClients.Load(proxyOverride); ok {
		return cached.(*http.Client)
	}

	transport := &http.Transport{Proxy: http.ProxyFromEnvironment}
	if parsed, err := url.Parse(proxyOverride); err == nil {
		transport.Proxy = http.ProxyURL(parsed)
	}

	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}
	s.proxyClients.Store(proxyOverride, client)
	return client
}

func (s *SteamService) proxyLogValue(proxyOverride string) string {
	proxyOverride = strings.TrimSpace(proxyOverride)
	if proxyOverride == "" {
		if strings.TrimSpace(s.proxy) == "" {
			return "direct"
		}
		return sanitizeRawURLForLog(s.proxy)
	}
	return sanitizeRawURLForLog(proxyOverride)
}
