package config

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type SettingDefinition struct {
	Key       string
	Label     string
	Group     string
	Sensitive bool
}

var runtimeSettingDefinitions = []SettingDefinition{
	{Key: "APP_ENV", Label: "运行环境", Group: "runtime"},
	{Key: "HOST", Label: "监听地址", Group: "runtime"},
	{Key: "PORT", Label: "监听端口", Group: "runtime"},
	{Key: "STATIC_DIR", Label: "前端资源目录", Group: "paths"},
	{Key: "ASSETS_DIR", Label: "游戏素材目录", Group: "paths"},
	{Key: "PRIMARY_ROM_ROOT", Label: "ROM 根目录", Group: "paths"},
	{Key: "DB_BACKUP_ENABLED", Label: "启用数据库备份", Group: "backup"},
	{Key: "DB_BACKUP_DIR", Label: "数据库备份目录", Group: "backup"},
	{Key: "DB_BACKUP_INTERVAL", Label: "数据库备份间隔", Group: "backup"},
	{Key: "DB_BACKUP_RETENTION_COUNT", Label: "数据库备份保留份数", Group: "backup"},
	{Key: "ADMIN_PASSWORD", Label: "管理员密码", Group: "auth", Sensitive: true},
	{Key: "ADMIN_DISPLAY_NAME", Label: "管理员显示名称", Group: "auth"},
	{Key: "AUTH_MAX_FAILS", Label: "登录失败次数限制", Group: "auth"},
	{Key: "AUTH_COOLDOWN", Label: "限制冷却时间", Group: "auth"},
	{Key: "AUTH_FAIL_WINDOW", Label: "失败计数时间窗口", Group: "auth"},
	{Key: "AUTH_STATE_TTL", Label: "锁定状态保留时间", Group: "auth"},
	{Key: "AUTH_TRACK_BY", Label: "失败追踪方式", Group: "auth"},
	{Key: "WIKI_HISTORY_LIMIT", Label: "Wiki 历史记录上限", Group: "general"},
	{Key: "SMB_PATH_MAPPINGS", Label: "SMB 路径映射", Group: "smb"},
	{Key: "SMB_USERNAME", Label: "SMB 用户名", Group: "smb"},
	{Key: "SMB_PASSWORD", Label: "SMB 密码", Group: "smb", Sensitive: true},
	{Key: "VHD_DIFF_ROOT", Label: "VHD 差分盘根路径", Group: "smb"},
	{Key: "PROXY", Label: "出站代理", Group: "network"},
	{Key: "STEAMGRIDDB_API_KEY", Label: "SteamGridDB API Key", Group: "network", Sensitive: true},
	{Key: "READ_HEADER_TIMEOUT", Label: "HTTP 请求头超时", Group: "runtime"},
	{Key: "SHUTDOWN_TIMEOUT", Label: "服务关闭超时", Group: "runtime"},
	{Key: "STREAM_ENABLED", Label: "启用浏览器串流", Group: "stream"},
	{Key: "STREAM_HOST", Label: "串流监听地址", Group: "stream"},
	{Key: "STREAM_PORT", Label: "串流监听端口", Group: "stream"},
	{Key: "STREAM_DATA_DIR", Label: "串流数据目录", Group: "stream"},
	{Key: "STREAM_WWW_ROOT", Label: "串流前端目录", Group: "stream"},
}

func RuntimeSettingDefinitions() []SettingDefinition {
	definitions := make([]SettingDefinition, len(runtimeSettingDefinitions))
	copy(definitions, runtimeSettingDefinitions)
	return definitions
}

func RuntimeSettingKeys() map[string]SettingDefinition {
	keys := make(map[string]SettingDefinition, len(runtimeSettingDefinitions))
	for _, definition := range runtimeSettingDefinitions {
		keys[definition.Key] = definition
	}
	return keys
}

func (c Config) RuntimeSettings() map[string]string {
	return map[string]string{
		"APP_ENV":                   c.AppEnv,
		"HOST":                      c.Host,
		"PORT":                      strconv.Itoa(c.Port),
		"STATIC_DIR":                c.runtimePathSetting("STATIC_DIR", c.StaticDir),
		"ASSETS_DIR":                c.runtimePathSetting("ASSETS_DIR", c.AssetsDir),
		"PRIMARY_ROM_ROOT":          c.runtimePathSetting("PRIMARY_ROM_ROOT", c.PrimaryROMRoot),
		"DB_BACKUP_ENABLED":         strconv.FormatBool(c.DBBackupEnabled),
		"DB_BACKUP_DIR":             c.runtimePathSetting("DB_BACKUP_DIR", c.DBBackupDir),
		"DB_BACKUP_INTERVAL":        c.DBBackupInterval.String(),
		"DB_BACKUP_RETENTION_COUNT": strconv.Itoa(c.DBBackupRetentionCount),
		"ADMIN_PASSWORD":            c.AdminPassword,
		"ADMIN_DISPLAY_NAME":        c.AdminDisplayName,
		"AUTH_MAX_FAILS":            strconv.Itoa(c.AuthMaxFails),
		"AUTH_COOLDOWN":             c.AuthCooldown.String(),
		"AUTH_FAIL_WINDOW":          c.AuthFailWindow.String(),
		"AUTH_STATE_TTL":            c.AuthStateTTL.String(),
		"AUTH_TRACK_BY":             c.AuthTrackBy,
		"WIKI_HISTORY_LIMIT":        strconv.Itoa(c.WikiHistoryLimit),
		"SMB_PATH_MAPPINGS":         c.SMBPathMappings,
		"SMB_USERNAME":              c.SMBUsername,
		"SMB_PASSWORD":              c.SMBPassword,
		"VHD_DIFF_ROOT":             c.VHDDiffRoot,
		"PROXY":                     c.Proxy,
		"STEAMGRIDDB_API_KEY":       c.SteamGridDBAPIKey,
		"READ_HEADER_TIMEOUT":       c.ReadHeaderTimeout.String(),
		"SHUTDOWN_TIMEOUT":          c.ShutdownTimeout.String(),
		"STREAM_ENABLED":            strconv.FormatBool(c.StreamEnabled),
		"STREAM_HOST":               c.StreamHost,
		"STREAM_PORT":               strconv.Itoa(c.StreamPort),
		"STREAM_DATA_DIR":           c.runtimePathSetting("STREAM_DATA_DIR", c.StreamDataDir),
		"STREAM_WWW_ROOT":           c.StreamWWWRoot,
	}
}

func (c Config) ApplyRuntimeSettings(values map[string]string) (Config, error) {
	keys := RuntimeSettingKeys()
	for _, key := range sortedSettingKeys(values) {
		if _, ok := keys[key]; !ok {
			continue
		}
		value := strings.TrimSpace(values[key])
		switch key {
		case "APP_ENV":
			c.AppEnv = value
		case "HOST":
			c.Host = value
		case "PORT":
			parsed, err := strconv.Atoi(value)
			if err != nil {
				return c, fmt.Errorf("invalid app setting PORT=%q: expected integer", value)
			}
			c.Port = parsed
		case "STATIC_DIR":
			c.StaticDir = c.applyRuntimePathSetting(key, value)
		case "ASSETS_DIR":
			c.AssetsDir = c.applyRuntimePathSetting(key, value)
		case "PRIMARY_ROM_ROOT":
			c.PrimaryROMRoot = c.applyRuntimePathSetting(key, value)
		case "DB_BACKUP_ENABLED":
			parsed, err := strconv.ParseBool(value)
			if err != nil {
				return c, fmt.Errorf("invalid app setting DB_BACKUP_ENABLED=%q: expected boolean", value)
			}
			c.DBBackupEnabled = parsed
		case "DB_BACKUP_DIR":
			c.DBBackupDir = c.applyRuntimePathSetting(key, value)
		case "DB_BACKUP_INTERVAL":
			parsed, err := time.ParseDuration(value)
			if err != nil {
				return c, fmt.Errorf("invalid app setting DB_BACKUP_INTERVAL=%q: expected duration", value)
			}
			c.DBBackupInterval = parsed
		case "DB_BACKUP_RETENTION_COUNT":
			parsed, err := strconv.Atoi(value)
			if err != nil {
				return c, fmt.Errorf("invalid app setting DB_BACKUP_RETENTION_COUNT=%q: expected integer", value)
			}
			c.DBBackupRetentionCount = parsed
		case "ADMIN_PASSWORD":
			c.AdminPassword = value
		case "ADMIN_DISPLAY_NAME":
			c.AdminDisplayName = value
		case "AUTH_MAX_FAILS":
			parsed, err := strconv.Atoi(value)
			if err != nil {
				return c, fmt.Errorf("invalid app setting AUTH_MAX_FAILS=%q: expected integer", value)
			}
			c.AuthMaxFails = parsed
		case "AUTH_COOLDOWN":
			parsed, err := time.ParseDuration(value)
			if err != nil {
				return c, fmt.Errorf("invalid app setting AUTH_COOLDOWN=%q: expected duration", value)
			}
			c.AuthCooldown = parsed
		case "AUTH_FAIL_WINDOW":
			parsed, err := time.ParseDuration(value)
			if err != nil {
				return c, fmt.Errorf("invalid app setting AUTH_FAIL_WINDOW=%q: expected duration", value)
			}
			c.AuthFailWindow = parsed
		case "AUTH_STATE_TTL":
			parsed, err := time.ParseDuration(value)
			if err != nil {
				return c, fmt.Errorf("invalid app setting AUTH_STATE_TTL=%q: expected duration", value)
			}
			c.AuthStateTTL = parsed
		case "AUTH_TRACK_BY":
			c.AuthTrackBy = value
		case "WIKI_HISTORY_LIMIT":
			parsed, err := strconv.Atoi(value)
			if err != nil {
				return c, fmt.Errorf("invalid app setting WIKI_HISTORY_LIMIT=%q: expected integer", value)
			}
			c.WikiHistoryLimit = parsed
		case "SMB_PATH_MAPPINGS":
			c.SMBPathMappings = value
		case "SMB_USERNAME":
			c.SMBUsername = value
		case "SMB_PASSWORD":
			c.SMBPassword = value
		case "VHD_DIFF_ROOT":
			c.VHDDiffRoot = value
		case "PROXY":
			c.Proxy = value
		case "STEAMGRIDDB_API_KEY":
			c.SteamGridDBAPIKey = value
		case "READ_HEADER_TIMEOUT":
			parsed, err := time.ParseDuration(value)
			if err != nil {
				return c, fmt.Errorf("invalid app setting READ_HEADER_TIMEOUT=%q: expected duration", value)
			}
			c.ReadHeaderTimeout = parsed
		case "SHUTDOWN_TIMEOUT":
			parsed, err := time.ParseDuration(value)
			if err != nil {
				return c, fmt.Errorf("invalid app setting SHUTDOWN_TIMEOUT=%q: expected duration", value)
			}
			c.ShutdownTimeout = parsed
		case "STREAM_ENABLED":
			parsed, err := strconv.ParseBool(value)
			if err != nil {
				return c, fmt.Errorf("invalid app setting STREAM_ENABLED=%q: expected boolean", value)
			}
			c.StreamEnabled = parsed
		case "STREAM_HOST":
			c.StreamHost = value
		case "STREAM_PORT":
			parsed, err := strconv.Atoi(value)
			if err != nil {
				return c, fmt.Errorf("invalid app setting STREAM_PORT=%q: expected integer", value)
			}
			c.StreamPort = parsed
		case "STREAM_DATA_DIR":
			c.StreamDataDir = c.applyRuntimePathSetting(key, value)
		case "STREAM_WWW_ROOT":
			c.StreamWWWRoot = value
		}
	}
	return c, nil
}

func (c Config) RuntimeRelativePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" || strings.TrimSpace(c.runtimeBaseDir) == "" || !filepath.IsAbs(path) {
		return path
	}

	relative, err := filepath.Rel(c.runtimeBaseDir, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return path
	}
	return relative
}

func (c *Config) NormalizeStoredRuntimePaths() map[string]string {
	pathValues := map[string]string{
		"ASSETS_DIR":      c.AssetsDir,
		"DB_BACKUP_DIR":   c.DBBackupDir,
		"STREAM_DATA_DIR": c.StreamDataDir,
	}
	normalized := make(map[string]string)
	for key, path := range pathValues {
		current := c.runtimePathSetting(key, path)
		if !filepath.IsAbs(current) {
			continue
		}
		relative := c.RuntimeRelativePath(current)
		if relative == current {
			continue
		}
		c.setRuntimePathSetting(key, relative)
		normalized[key] = relative
	}
	return normalized
}

func (c Config) runtimePathSetting(key, fallback string) string {
	if c.pathSettings != nil {
		if value, ok := c.pathSettings[key]; ok {
			return value
		}
	}
	return fallback
}

func (c *Config) applyRuntimePathSetting(key, value string) string {
	c.setRuntimePathSetting(key, value)
	return resolveRuntimePath(c.runtimeBaseDir, value)
}

func (c *Config) setRuntimePathSetting(key, value string) {
	if c.pathSettings == nil {
		c.pathSettings = make(map[string]string)
	}
	c.pathSettings[key] = value
}

func sortedSettingKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
