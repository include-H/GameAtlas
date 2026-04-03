package config

import (
	"bufio"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv            string
	Host              string
	Port              int
	DBPath            string
	StaticDir         string
	AssetsDir         string
	PrimaryROMRoot    string
	Proxy             string
	SMBShareRoot      string
	SMBPathMappings   string
	SMBUsername       string
	SMBPassword       string
	VHDDiffRoot       string
	WikiHistoryLimit  int
	AdminDisplayName  string
	AdminPassword     string
	AuthMaxFails      int
	AuthCooldown      time.Duration
	AuthFailWindow    time.Duration
	AuthStateTTL      time.Duration
	AuthTrackBy       string
	ReadHeaderTimeout time.Duration
	ShutdownTimeout   time.Duration
}

func Load() (Config, error) {
	runtimeBaseDir, dotEnvPath, err := detectRuntimeBaseDir()
	if err != nil {
		return Config{}, err
	}

	if dotEnvPath != "" {
		if err := loadDotEnv(dotEnvPath); err != nil {
			return Config{}, err
		}
	}

	proxy := getEnv("PROXY", "")
	primaryROMRoot := resolveRuntimePath(runtimeBaseDir, getEnv("PRIMARY_ROM_ROOT", "ROM"))

	cfg := Config{
		AppEnv:           getEnv("APP_ENV", "development"),
		Host:             getEnv("HOST", "0.0.0.0"),
		DBPath:           resolveRuntimePath(runtimeBaseDir, getEnv("DB_PATH", "data/db.db")),
		StaticDir:        resolveRuntimePath(runtimeBaseDir, getEnv("STATIC_DIR", "../frontend/dist")),
		AssetsDir:        resolveRuntimePath(runtimeBaseDir, getEnv("ASSETS_DIR", "data/gamelist")),
		PrimaryROMRoot:   primaryROMRoot,
		Proxy:            proxy,
		SMBShareRoot:     getEnv("SMB_SHARE_ROOT", ""),
		SMBPathMappings:  getEnv("SMB_PATH_MAPPINGS", ""),
		SMBUsername:      getEnv("SMB_USERNAME", ""),
		SMBPassword:      getEnv("SMB_PASSWORD", ""),
		VHDDiffRoot:      getEnv("VHD_DIFF_ROOT", `C:`),
		AdminDisplayName: getEnv("ADMIN_DISPLAY_NAME", "Admin"),
		AdminPassword:    getEnv("ADMIN_PASSWORD", ""),
		AuthTrackBy:      getEnv("AUTH_TRACK_BY", "ip"),
	}

	var errs []error

	cfg.Port, errs = appendParsedInt(errs, "PORT", 3000, &cfg.Port)
	cfg.WikiHistoryLimit, errs = appendParsedInt(errs, "WIKI_HISTORY_LIMIT", 100, &cfg.WikiHistoryLimit)
	cfg.AuthMaxFails, errs = appendParsedInt(errs, "AUTH_MAX_FAILS", 5, &cfg.AuthMaxFails)
	cfg.AuthCooldown, errs = appendParsedDuration(errs, "AUTH_COOLDOWN", 10*time.Minute, &cfg.AuthCooldown)
	cfg.AuthFailWindow, errs = appendParsedDuration(errs, "AUTH_FAIL_WINDOW", 30*time.Minute, &cfg.AuthFailWindow)
	cfg.AuthStateTTL, errs = appendParsedDuration(errs, "AUTH_STATE_TTL", 24*time.Hour, &cfg.AuthStateTTL)
	cfg.ReadHeaderTimeout, errs = appendParsedDuration(errs, "READ_HEADER_TIMEOUT", 5*time.Second, &cfg.ReadHeaderTimeout)
	cfg.ShutdownTimeout, errs = appendParsedDuration(errs, "SHUTDOWN_TIMEOUT", 10*time.Second, &cfg.ShutdownTimeout)

	if err := errors.Join(errs...); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func detectRuntimeBaseDir() (string, string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", fmt.Errorf("determine current working directory: %w", err)
	}

	executablePath, err := os.Executable()
	executableDir := ""
	if err == nil {
		executableDir = filepath.Dir(executablePath)
	}

	baseDir, dotEnvPath := chooseRuntimeBaseDir(cwd, executableDir, pathExists)
	if baseDir == "" {
		baseDir = cwd
	}

	return baseDir, dotEnvPath, nil
}

func chooseRuntimeBaseDir(cwd, executableDir string, exists func(string) bool) (string, string) {
	cwd = cleanOptionalPath(cwd)
	executableDir = cleanOptionalPath(executableDir)

	if cwd != "" {
		candidate := filepath.Join(cwd, ".env")
		if exists(candidate) {
			return cwd, candidate
		}
	}

	if executableDir != "" {
		candidate := filepath.Join(executableDir, ".env")
		if exists(candidate) {
			return executableDir, candidate
		}
	}

	if cwd != "" && exists(filepath.Join(cwd, "go.mod")) {
		return cwd, ""
	}

	if executableDir != "" {
		return executableDir, ""
	}

	return cwd, ""
}

func cleanOptionalPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	return filepath.Clean(path)
}

func resolveRuntimePath(baseDir, value string) string {
	cleaned := filepath.Clean(strings.TrimSpace(value))
	if filepath.IsAbs(cleaned) {
		return cleaned
	}
	if baseDir == "" {
		return cleaned
	}
	return filepath.Join(baseDir, cleaned)
}

func pathExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}

	_, err := os.Stat(path)
	return err == nil
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.AdminPassword) == "" {
		return fmt.Errorf("ADMIN_PASSWORD must be configured")
	}
	if _, err := parseProxyURL(c.Proxy); err != nil {
		return err
	}
	if _, err := c.ParseSMBPathMappings(); err != nil {
		return err
	}
	return nil
}

type SMBPathMapping struct {
	LocalRoot string
	ShareRoot string
}

func (c Config) ParseSMBPathMappings() ([]SMBPathMapping, error) {
	raw := strings.TrimSpace(c.SMBPathMappings)
	if raw == "" {
		return nil, nil
	}

	entries := strings.Split(raw, ";")
	mappings := make([]SMBPathMapping, 0, len(entries))
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		localRoot, shareRoot, ok := strings.Cut(entry, "=")
		if !ok {
			return nil, fmt.Errorf("invalid SMB_PATH_MAPPINGS entry %q: expected <local-path>=<unc-path>", entry)
		}

		localRoot = strings.TrimSpace(localRoot)
		shareRoot = strings.TrimSpace(shareRoot)
		if localRoot == "" || shareRoot == "" {
			return nil, fmt.Errorf("invalid SMB_PATH_MAPPINGS entry %q: local path and UNC path are required", entry)
		}

		mappings = append(mappings, SMBPathMapping{
			LocalRoot: filepath.Clean(localRoot),
			ShareRoot: shareRoot,
		})
	}

	return mappings, nil
}

func (c Config) ProxyLogValue() string {
	parsed, err := parseProxyURL(c.Proxy)
	if err != nil || parsed == nil {
		return "direct"
	}

	if parsed.User != nil {
		username := parsed.User.Username()
		if _, hasPassword := parsed.User.Password(); hasPassword {
			parsed.User = url.UserPassword(username, "******")
		} else if username != "" {
			parsed.User = url.User(username)
		}
	}

	return parsed.String()
}

func loadDotEnv(path string) error {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("invalid %s:%d: expected KEY=VALUE", path, lineNo)
		}

		key = strings.TrimSpace(key)
		if key == "" {
			return fmt.Errorf("invalid %s:%d: empty variable name", path, lineNo)
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}

		value = normalizeEnvValue(strings.TrimSpace(value))
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("set %s from %s:%d: %w", key, path, lineNo, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	return nil
}

func normalizeEnvValue(value string) string {
	if len(value) >= 2 {
		if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
			return strings.Trim(value, "\"")
		}
		if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
			return strings.Trim(value, "'")
		}
	}

	return value
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}

func parseProxyURL(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid PROXY %q: %w", raw, err)
	}
	if parsed.Scheme == "" {
		return nil, fmt.Errorf("invalid PROXY %q: missing scheme, expected http://, https://, or socks5://", raw)
	}
	switch parsed.Scheme {
	case "http", "https", "socks5":
	default:
		return nil, fmt.Errorf("invalid PROXY %q: unsupported scheme %q, expected http, https, or socks5", raw, parsed.Scheme)
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("invalid PROXY %q: missing host", raw)
	}

	return parsed, nil
}

func getEnvAsInt(key string, fallback int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid %s=%q: expected integer", key, raw)
	}

	return value, nil
}

func getEnvAsDuration(key string, fallback time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}

	value, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid %s=%q: expected Go duration such as 10m or 30s", key, raw)
	}

	return value, nil
}

func appendParsedInt(errs []error, key string, fallback int, target *int) (int, []error) {
	value, err := getEnvAsInt(key, fallback)
	if err != nil {
		errs = append(errs, err)
		return 0, errs
	}

	*target = value
	return value, errs
}

func appendParsedDuration(errs []error, key string, fallback time.Duration, target *time.Duration) (time.Duration, []error) {
	value, err := getEnvAsDuration(key, fallback)
	if err != nil {
		errs = append(errs, err)
		return 0, errs
	}

	*target = value
	return value, errs
}
