package files

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var allowedImageContentTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/jpg":  ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
	"image/gif":  ".gif",
}

var ErrInvalidImageType = errors.New("invalid image type")
var ErrInvalidRemoteURL = errors.New("invalid remote image url")
var ErrBlockedRemoteURL = errors.New("blocked remote image url")

type AssetStore struct {
	baseDir string
	client  *http.Client
}

func NewAssetStore(baseDir string, proxyURL string, timeout time.Duration) *AssetStore {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}

	if proxyURL != "" {
		if parsed, err := url.Parse(proxyURL); err == nil {
			transport.Proxy = http.ProxyURL(parsed)
		}
	}

	return &AssetStore{
		baseDir: baseDir,
		client: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
	}
}

func (s *AssetStore) SaveUploadedAsset(gameID int64, assetType string, file io.Reader, contentType string, index int) (string, error) {
	extension, err := validateImageContentType(contentType)
	if err != nil {
		return "", err
	}

	dir, filename := assetTarget(gameID, assetType, extension, index)
	targetDir := filepath.Join(s.baseDir, dir)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", fmt.Errorf("create asset directory: %w", err)
	}

	targetPath := filepath.Join(targetDir, filename)
	output, err := os.Create(targetPath)
	if err != nil {
		return "", fmt.Errorf("create asset file: %w", err)
	}
	defer output.Close()

	if _, err := io.Copy(output, file); err != nil {
		return "", fmt.Errorf("write asset file: %w", err)
	}

	return "/" + filepath.ToSlash(filepath.Join("assets", dir, filename)), nil
}

func (s *AssetStore) DownloadRemoteAsset(gameID int64, assetType string, remoteURL string, index int) (string, error) {
	parsed, err := validateRemoteImageURL(remoteURL)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodGet, parsed.String(), nil)
	if err != nil {
		return "", fmt.Errorf("build remote asset request: %w", err)
	}
	req.Header.Set("User-Agent", "NAS-Game-Library-Manager/1.0")
	req.Header.Set("Accept", "image/webp,image/apng,image/*,*/*;q=0.8")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download remote asset: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download remote asset: unexpected status %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if idx := strings.Index(contentType, ";"); idx >= 0 {
		contentType = contentType[:idx]
	}

	return s.SaveUploadedAsset(gameID, assetType, resp.Body, contentType, index)
}

func (s *AssetStore) DeleteAsset(assetPath string) error {
	cleaned := strings.TrimSpace(assetPath)
	cleaned = strings.TrimPrefix(cleaned, "/")
	if cleaned == "" || !strings.HasPrefix(cleaned, "assets/") {
		return os.ErrNotExist
	}

	relativeAssetPath := strings.TrimPrefix(cleaned, "assets/")
	targetPath := filepath.Join(s.baseDir, filepath.FromSlash(relativeAssetPath))
	relative, err := filepath.Rel(s.baseDir, targetPath)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return ErrInvalidRemoteURL
	}
	if err := os.Remove(targetPath); err != nil {
		return err
	}
	return nil
}

func validateImageContentType(contentType string) (string, error) {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	if extension, ok := allowedImageContentTypes[contentType]; ok {
		return extension, nil
	}

	if parsed, _, err := mime.ParseMediaType(contentType); err == nil {
		if extension, ok := allowedImageContentTypes[parsed]; ok {
			return extension, nil
		}
	}

	return "", ErrInvalidImageType
}

func assetTarget(gameID int64, assetType string, extension string, index int) (string, string) {
	dir := fmt.Sprintf("%d", gameID)
	switch assetType {
	case "cover":
		return dir, "cover" + extension
	case "banner":
		return dir, "banner" + extension
	default:
		return dir, fmt.Sprintf("%d%s", index+1, extension)
	}
}

func validateRemoteImageURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed == nil || parsed.Host == "" {
		return nil, ErrInvalidRemoteURL
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, ErrInvalidRemoteURL
	}
	host := parsed.Hostname()
	if isBlockedHost(host) {
		return nil, ErrBlockedRemoteURL
	}
	return parsed, nil
}

func isBlockedHost(host string) bool {
	lower := strings.ToLower(strings.TrimSpace(host))
	if lower == "localhost" || strings.HasSuffix(lower, ".localhost") || strings.HasSuffix(lower, ".local") {
		return true
	}

	ip := net.ParseIP(lower)
	if ip != nil {
		return isPrivateIP(ip)
	}

	addrs, err := net.LookupIP(lower)
	if err != nil {
		return false
	}
	for _, addr := range addrs {
		if isPrivateIP(addr) {
			return true
		}
	}
	return false
}

func isPrivateIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()
}
