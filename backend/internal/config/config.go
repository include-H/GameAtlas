package config

import (
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
	AppEnv                 string
	Host                   string
	Port                   int
	DBPath                 string
	DBBackupEnabled        bool
	DBBackupDir            string
	DBBackupInterval       time.Duration
	DBBackupRetentionCount int
	StaticDir              string
	AssetsDir              string
	PrimaryROMRoot         string
	Proxy                  string
	SMBPathMappings        string
	SMBUsername            string
	SMBPassword            string
	VHDDiffRoot            string
	WikiHistoryLimit       int
	AdminDisplayName       string
	AdminPassword          string
	AuthMaxFails           int
	AuthCooldown           time.Duration
	AuthFailWindow         time.Duration
	AuthStateTTL           time.Duration
	AuthTrackBy            string
	SteamGridDBAPIKey      string
	ReadHeaderTimeout      time.Duration
	ShutdownTimeout        time.Duration
	StreamEnabled          bool
	StreamHost             string
	StreamPort             int
	StreamDataDir          string
	StreamWWWRoot          string
	runtimeBaseDir         string
	pathSettings           map[string]string
}

func Load() (Config, error) {
	runtimeBaseDir, err := detectRuntimeBaseDir()
	if err != nil {
		return Config{}, err
	}

	pathBaseDir := runtimeBaseDir

	proxy := getEnv("PROXY", "")
	staticDirSetting := getEnv("STATIC_DIR", "../frontend/dist")
	assetsDirSetting := getEnv("ASSETS_DIR", "data/gamelist")
	primaryROMRootSetting := getEnv("PRIMARY_ROM_ROOT", "/mnt")
	backupDirSetting := getEnv("DB_BACKUP_DIR", "data/backups")
	streamDataDirSetting := getEnv("STREAM_DATA_DIR", "data/streaming")
	streamWWWRootSetting := getEnv("STREAM_WWW_ROOT", "")

	cfg := Config{
		AppEnv:           getEnv("APP_ENV", "production"),
		Host:             getEnv("HOST", "0.0.0.0"),
		DBPath:           resolveRuntimePath(pathBaseDir, getEnv("DB_PATH", "data/db.db")),
		DBBackupDir:      resolveRuntimePath(pathBaseDir, backupDirSetting),
		StaticDir:        resolveRuntimePath(pathBaseDir, staticDirSetting),
		AssetsDir:        resolveRuntimePath(pathBaseDir, assetsDirSetting),
		PrimaryROMRoot:   resolveRuntimePath(pathBaseDir, primaryROMRootSetting),
		Proxy:            proxy,
		SMBPathMappings:  getEnv("SMB_PATH_MAPPINGS", ""),
		SMBUsername:      getEnv("SMB_USERNAME", ""),
		SMBPassword:      getEnv("SMB_PASSWORD", ""),
		VHDDiffRoot:      getEnv("VHD_DIFF_ROOT", `C:`),
		AdminDisplayName: getEnv("ADMIN_DISPLAY_NAME", "Admin"),
		// 保留默认密码 1234：ADMIN_PASSWORD 为空时 Validate() 会直接拒绝启动，
		// 未显式配置的管理员将无法登录，连创建游戏都做不到，开箱即用必须依赖默认值。
		// 该默认值只适用于家庭 / 内网可信环境；暴露到公网前必须通过设置页或
		// ADMIN_PASSWORD 环境变量改掉默认密码，否则管理接口可被直接接管。
		AdminPassword:     getEnv("ADMIN_PASSWORD", "1234"),
		SteamGridDBAPIKey: getEnv("STEAMGRIDDB_API_KEY", ""),
		AuthTrackBy:       getEnv("AUTH_TRACK_BY", "ip"),
		StreamHost:        getEnv("STREAM_HOST", "0.0.0.0"),
		StreamDataDir:     resolveRuntimePath(pathBaseDir, streamDataDirSetting),
		StreamWWWRoot:     streamWWWRootSetting,
		runtimeBaseDir:    pathBaseDir,
		pathSettings: map[string]string{
			"STATIC_DIR":       staticDirSetting,
			"ASSETS_DIR":       assetsDirSetting,
			"PRIMARY_ROM_ROOT": primaryROMRootSetting,
			"DB_BACKUP_DIR":    backupDirSetting,
			"STREAM_DATA_DIR":  streamDataDirSetting,
		},
	}

	var errs []error

	cfg.Port, errs = appendParsedInt(errs, "PORT", 3000, &cfg.Port)
	cfg.DBBackupEnabled, errs = appendParsedBool(errs, "DB_BACKUP_ENABLED", true, &cfg.DBBackupEnabled)
	cfg.DBBackupInterval, errs = appendParsedDuration(errs, "DB_BACKUP_INTERVAL", 24*time.Hour, &cfg.DBBackupInterval)
	cfg.DBBackupRetentionCount, errs = appendParsedInt(errs, "DB_BACKUP_RETENTION_COUNT", 5, &cfg.DBBackupRetentionCount)
	cfg.WikiHistoryLimit, errs = appendParsedInt(errs, "WIKI_HISTORY_LIMIT", 100, &cfg.WikiHistoryLimit)
	cfg.AuthMaxFails, errs = appendParsedInt(errs, "AUTH_MAX_FAILS", 5, &cfg.AuthMaxFails)
	cfg.AuthCooldown, errs = appendParsedDuration(errs, "AUTH_COOLDOWN", 10*time.Minute, &cfg.AuthCooldown)
	cfg.AuthFailWindow, errs = appendParsedDuration(errs, "AUTH_FAIL_WINDOW", 30*time.Minute, &cfg.AuthFailWindow)
	cfg.AuthStateTTL, errs = appendParsedDuration(errs, "AUTH_STATE_TTL", 24*time.Hour, &cfg.AuthStateTTL)
	cfg.ReadHeaderTimeout, errs = appendParsedDuration(errs, "READ_HEADER_TIMEOUT", 5*time.Second, &cfg.ReadHeaderTimeout)
	cfg.ShutdownTimeout, errs = appendParsedDuration(errs, "SHUTDOWN_TIMEOUT", 10*time.Second, &cfg.ShutdownTimeout)
	cfg.StreamEnabled, errs = appendParsedBool(errs, "STREAM_ENABLED", true, &cfg.StreamEnabled)
	cfg.StreamPort, errs = appendParsedInt(errs, "STREAM_PORT", 47999, &cfg.StreamPort)

	if err := errors.Join(errs...); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func detectRuntimeBaseDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("determine current working directory: %w", err)
	}

	executable, err := os.Executable()
	if err != nil {
		executable = ""
	}
	baseDir := runtimeBaseDirForExecutable(cwd, executable)
	if baseDir == "" {
		baseDir = cwd
	}

	return baseDir, nil
}

func runtimeBaseDirForExecutable(cwd, executable string) string {
	cwd = cleanOptionalPath(cwd)
	executable = cleanOptionalPath(executable)
	if executable == "" || isGoRunExecutable(executable) {
		return cwd
	}
	return filepath.Dir(executable)
}

func isGoRunExecutable(executable string) bool {
	return strings.Contains(filepath.ToSlash(executable), "/go-build")
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

func (c Config) Validate() error {
	if strings.TrimSpace(c.AdminPassword) == "" {
		return fmt.Errorf("ADMIN_PASSWORD must be configured")
	}
	if c.Port < 0 || c.Port > 65535 {
		return fmt.Errorf("PORT must be between 0 and 65535")
	}
	if c.DBBackupInterval < 0 {
		return fmt.Errorf("DB_BACKUP_INTERVAL must be zero or positive")
	}
	if c.DBBackupRetentionCount < 0 {
		return fmt.Errorf("DB_BACKUP_RETENTION_COUNT must be zero or positive")
	}
	if c.WikiHistoryLimit < 0 {
		return fmt.Errorf("WIKI_HISTORY_LIMIT must be zero or positive")
	}
	if c.AuthMaxFails < 0 {
		return fmt.Errorf("AUTH_MAX_FAILS must be zero or positive")
	}
	if c.AuthCooldown < 0 {
		return fmt.Errorf("AUTH_COOLDOWN must be zero or positive")
	}
	if c.AuthFailWindow < 0 {
		return fmt.Errorf("AUTH_FAIL_WINDOW must be zero or positive")
	}
	if c.AuthStateTTL < 0 {
		return fmt.Errorf("AUTH_STATE_TTL must be zero or positive")
	}
	if c.ReadHeaderTimeout <= 0 {
		return fmt.Errorf("READ_HEADER_TIMEOUT must be positive")
	}
	if c.ShutdownTimeout <= 0 {
		return fmt.Errorf("SHUTDOWN_TIMEOUT must be positive")
	}
	if c.StreamPort < 0 || c.StreamPort > 65535 {
		return fmt.Errorf("STREAM_PORT must be between 0 and 65535")
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

func getEnvAsBool(key string, fallback bool) (bool, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}

	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("invalid %s=%q: expected boolean", key, raw)
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

func appendParsedBool(errs []error, key string, fallback bool, target *bool) (bool, []error) {
	value, err := getEnvAsBool(key, fallback)
	if err != nil {
		errs = append(errs, err)
		return false, errs
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
