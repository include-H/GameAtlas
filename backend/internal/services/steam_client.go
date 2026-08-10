package services

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hao/game/internal/domain"
)

func (s *SteamService) doRequest(req *http.Request, proxyOverride string) (*http.Response, error) {
	log.Printf(
		"steam outbound request: method=%s url=%s proxy=%s",
		req.Method,
		sanitizeURLForLog(req.URL),
		s.proxyLogValue(proxyOverride),
	)
	response, err := s.clientForProxy(proxyOverride).Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: steam outbound request: %v", domain.ErrUpstream, err)
	}
	return response, nil
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

	client := newSteamHTTPClient(transport, 30*time.Second)
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

func newSteamHTTPClient(transport http.RoundTripper, timeout time.Duration) *http.Client {
	return newRedirectCheckedHTTPClient(transport, timeout, isAllowedSteamAssetHost)
}

func newSteamGridDBHTTPClient(transport http.RoundTripper, timeout time.Duration) *http.Client {
	return newRedirectCheckedHTTPClient(transport, timeout, isAllowedSteamGridDBHost)
}

func newRedirectCheckedHTTPClient(
	transport http.RoundTripper,
	timeout time.Duration,
	allowedHost func(string) bool,
) *http.Client {
	if transport == nil {
		transport = http.DefaultTransport
	}
	return &http.Client{
		Timeout:       timeout,
		Transport:     transport,
		CheckRedirect: checkAllowedRedirect(allowedHost),
	}
}

func checkAllowedRedirect(allowedHost func(string) bool) func(*http.Request, []*http.Request) error {
	return func(request *http.Request, _ []*http.Request) error {
		if request == nil || request.URL == nil {
			return errors.New("redirect target is not allowed: missing URL")
		}
		scheme := strings.ToLower(strings.TrimSpace(request.URL.Scheme))
		host := request.URL.Hostname()
		if (scheme != "http" && scheme != "https") || !allowedHost(host) {
			return fmt.Errorf("redirect target is not allowed: %s", sanitizeURLForLog(request.URL))
		}
		return nil
	}
}
