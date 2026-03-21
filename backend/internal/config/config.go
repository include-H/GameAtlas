package config

import (
	"bufio"
	"fmt"
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
	MigrationsDir     string
	AllowedRoots      []string
	PrimaryROMRoot    string
	Proxy             string
	HTTPProxy         string
	HTTPSProxy        string
	SteamProxy        string
	SMBShareRoot      string
	SMBUsername       string
	SMBPassword       string
	VHDDiffRoot       string
	WikiHistoryLimit  int
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

	return Config{
		AppEnv:            getEnv("APP_ENV", "development"),
		Host:              getEnv("HOST", "0.0.0.0"),
		Port:              getEnvAsInt("PORT", 3000),
		DBPath:            filepath.Clean(getEnv("DB_PATH", "data/db.db")),
		StaticDir:         filepath.Clean(getEnv("STATIC_DIR", "../frontend/dist")),
		AssetsDir:         filepath.Clean(getEnv("ASSETS_DIR", "data/gamelist")),
		MigrationsDir:     filepath.Clean(getEnv("MIGRATIONS_DIR", "migrations")),
		AllowedRoots:      getEnvAsList("ALLOWED_LIBRARY_ROOTS", []string{"ROM"}),
		PrimaryROMRoot:    filepath.Clean(getEnv("PRIMARY_ROM_ROOT", "ROM")),
		Proxy:             proxy,
		HTTPProxy:         getEnv("HTTP_PROXY", proxy),
		HTTPSProxy:        getEnv("HTTPS_PROXY", proxy),
		SteamProxy:        getEnv("STEAM_PROXY", proxy),
		SMBShareRoot:      getEnv("SMB_SHARE_ROOT", ""),
		SMBUsername:       getEnv("SMB_USERNAME", ""),
		SMBPassword:       getEnv("SMB_PASSWORD", ""),
		VHDDiffRoot:       getEnv("VHD_DIFF_ROOT", `C:`),
		WikiHistoryLimit:  getEnvAsInt("WIKI_HISTORY_LIMIT", 100),
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
	return nil
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

func getEnvAsList(key string, fallback []string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}

	parts := strings.Split(raw, ",")
	items := make([]string, 0, len(parts))

	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		items = append(items, filepath.Clean(item))
	}

	if len(items) == 0 {
		return fallback
	}

	return items
}
