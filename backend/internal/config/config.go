package config

import (
	"bufio"
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
	SessionSecret     string
	AuthMaxFails      int
	AuthCooldown      time.Duration
	AuthFailWindow    time.Duration
	AuthStateTTL      time.Duration
	AuthTrackBy       string
	LogLevel          string
	ReadHeaderTimeout time.Duration
	ShutdownTimeout   time.Duration
}

func Load() Config {
	loadDotEnv(".env")
	proxy := getEnv("PROXY", "")
	primaryROMRoot := filepath.Clean(getEnv("PRIMARY_ROM_ROOT", "ROM"))

	return Config{
		AppEnv:            getEnv("APP_ENV", "development"),
		Host:              getEnv("HOST", "0.0.0.0"),
		Port:              getEnvAsInt("PORT", 3000),
		DBPath:            filepath.Clean(getEnv("DB_PATH", "data/db.db")),
		StaticDir:         filepath.Clean(getEnv("STATIC_DIR", "../frontend/dist")),
		AssetsDir:         filepath.Clean(getEnv("ASSETS_DIR", "data/gamelist")),
		PrimaryROMRoot:    primaryROMRoot,
		Proxy:             proxy,
		SMBShareRoot:      getEnv("SMB_SHARE_ROOT", ""),
		SMBPathMappings:   getEnv("SMB_PATH_MAPPINGS", ""),
		SMBUsername:       getEnv("SMB_USERNAME", ""),
		SMBPassword:       getEnv("SMB_PASSWORD", ""),
		VHDDiffRoot:       getEnv("VHD_DIFF_ROOT", `C:`),
		WikiHistoryLimit:  getEnvAsInt("WIKI_HISTORY_LIMIT", 100),
		AdminDisplayName:  getEnv("ADMIN_DISPLAY_NAME", "Admin"),
		AdminPassword:     getEnv("ADMIN_PASSWORD", ""),
		SessionSecret:     getEnv("SESSION_SECRET", "change-me"),
		AuthMaxFails:      getEnvAsInt("AUTH_MAX_FAILS", 5),
		AuthCooldown:      getEnvAsDuration("AUTH_COOLDOWN", 10*time.Minute),
		AuthFailWindow:    getEnvAsDuration("AUTH_FAIL_WINDOW", 30*time.Minute),
		AuthStateTTL:      getEnvAsDuration("AUTH_STATE_TTL", 24*time.Hour),
		AuthTrackBy:       getEnv("AUTH_TRACK_BY", "ip"),
		LogLevel:          getEnv("LOG_LEVEL", "info"),
		ReadHeaderTimeout: getEnvAsDuration("READ_HEADER_TIMEOUT", 5*time.Second),
		ShutdownTimeout:   getEnvAsDuration("SHUTDOWN_TIMEOUT", 10*time.Second),
	}
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.AdminPassword) == "" {
		return fmt.Errorf("ADMIN_PASSWORD must be configured")
	}
	if strings.TrimSpace(c.SessionSecret) == "" || strings.TrimSpace(c.SessionSecret) == "change-me" {
		return fmt.Errorf("SESSION_SECRET must be configured with a non-default value")
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

func loadDotEnv(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}

		value = normalizeEnvValue(strings.TrimSpace(value))
		_ = os.Setenv(key, value)
	}
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

func getEnvAsInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}

	return value
}

func getEnvAsDuration(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}

	value, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}

	return value
}
