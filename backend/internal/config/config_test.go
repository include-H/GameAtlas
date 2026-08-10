package config

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseSMBPathMappingsParsesAndCleansEntries(t *testing.T) {
	cfg := Config{
		SMBPathMappings: " ./ROM = \\\\NAS\\ROM ; ./ROM/PS2 = \\\\NAS\\PS2 ",
	}

	got, err := cfg.ParseSMBPathMappings()
	if err != nil {
		t.Fatalf("ParseSMBPathMappings returned error: %v", err)
	}

	want := []SMBPathMapping{
		{LocalRoot: filepath.Clean("./ROM"), ShareRoot: `\\NAS\ROM`},
		{LocalRoot: filepath.Clean("./ROM/PS2"), ShareRoot: `\\NAS\PS2`},
	}
	if len(got) != len(want) {
		t.Fatalf("len(mappings) = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("mapping[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestParseSMBPathMappingsRejectsInvalidEntry(t *testing.T) {
	cfg := Config{SMBPathMappings: "invalid-entry"}
	if _, err := cfg.ParseSMBPathMappings(); err == nil {
		t.Fatalf("expected ParseSMBPathMappings to return error")
	}
}

func TestProxyLogValueMasksPassword(t *testing.T) {
	cfg := Config{Proxy: "http://alice:secret@example.com:8080"}
	got := cfg.ProxyLogValue()
	if strings.Contains(got, "secret") {
		t.Fatalf("ProxyLogValue() leaked password: %q", got)
	}
	if !strings.Contains(got, "alice:") || !strings.Contains(got, "@example.com:8080") {
		t.Fatalf("ProxyLogValue() = %q, want masked userinfo and original host", got)
	}
}

func TestGetEnvAsIntRejectsInvalidConfiguredValue(t *testing.T) {
	t.Setenv("PORT", "abc")

	_, err := getEnvAsInt("PORT", 3000)
	if err == nil {
		t.Fatalf("expected getEnvAsInt to return error")
	}
	if !strings.Contains(err.Error(), `PORT="abc"`) {
		t.Fatalf("getEnvAsInt error = %v, want variable detail", err)
	}
}

func TestGetEnvAsBoolRejectsInvalidConfiguredValue(t *testing.T) {
	t.Setenv("DB_BACKUP_ENABLED", "maybe")

	_, err := getEnvAsBool("DB_BACKUP_ENABLED", true)
	if err == nil {
		t.Fatalf("expected getEnvAsBool to return error")
	}
	if !strings.Contains(err.Error(), `DB_BACKUP_ENABLED="maybe"`) {
		t.Fatalf("getEnvAsBool error = %v, want variable detail", err)
	}
}

func TestGetEnvAsDurationRejectsInvalidConfiguredValue(t *testing.T) {
	t.Setenv("AUTH_COOLDOWN", "10min")

	_, err := getEnvAsDuration("AUTH_COOLDOWN", 10*time.Minute)
	if err == nil {
		t.Fatalf("expected getEnvAsDuration to return error")
	}
	if !strings.Contains(err.Error(), `AUTH_COOLDOWN="10min"`) {
		t.Fatalf("getEnvAsDuration error = %v, want variable detail", err)
	}
}

func TestApplyRuntimeSettingsParsesDatabaseValues(t *testing.T) {
	cfg := Config{
		AdminPassword: "secret",
		AssetsDir:     filepath.Join(string(filepath.Separator), "data", "gamelist"),
	}

	got, err := cfg.ApplyRuntimeSettings(map[string]string{
		"PORT":                      "3001",
		"DB_BACKUP_ENABLED":         "false",
		"DB_BACKUP_INTERVAL":        "12h",
		"DB_BACKUP_RETENTION_COUNT": "5",
		"AUTH_MAX_FAILS":            "7",
	})
	if err != nil {
		t.Fatalf("ApplyRuntimeSettings returned error: %v", err)
	}
	if got.Port != 3001 {
		t.Fatalf("Port = %d, want 3001", got.Port)
	}
	if got.DBBackupEnabled {
		t.Fatalf("DBBackupEnabled = true, want false")
	}
	if got.DBBackupInterval != 12*time.Hour {
		t.Fatalf("DBBackupInterval = %s, want 12h", got.DBBackupInterval)
	}
	if got.DBBackupRetentionCount != 5 {
		t.Fatalf("DBBackupRetentionCount = %d, want 5", got.DBBackupRetentionCount)
	}
	if got.AuthMaxFails != 7 {
		t.Fatalf("AuthMaxFails = %d, want 7", got.AuthMaxFails)
	}
}

func TestConfigValidateRejectsInvalidOperationalValues(t *testing.T) {
	base := Config{
		AdminPassword:          "secret",
		Port:                   3000,
		DBBackupInterval:       time.Hour,
		ReadHeaderTimeout:      5 * time.Second,
		ShutdownTimeout:        10 * time.Second,
		AuthMaxFails:           5,
		AuthCooldown:           10 * time.Minute,
		AuthFailWindow:         30 * time.Minute,
		AuthStateTTL:           24 * time.Hour,
		WikiHistoryLimit:       100,
		DBBackupRetentionCount: 5,
	}

	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{name: "port below range", mutate: func(cfg *Config) { cfg.Port = -1 }},
		{name: "port above range", mutate: func(cfg *Config) { cfg.Port = 65536 }},
		{name: "negative wiki history", mutate: func(cfg *Config) { cfg.WikiHistoryLimit = -1 }},
		{name: "negative auth failures", mutate: func(cfg *Config) { cfg.AuthMaxFails = -1 }},
		{name: "negative auth cooldown", mutate: func(cfg *Config) { cfg.AuthCooldown = -time.Second }},
		{name: "negative auth window", mutate: func(cfg *Config) { cfg.AuthFailWindow = -time.Second }},
		{name: "negative auth ttl", mutate: func(cfg *Config) { cfg.AuthStateTTL = -time.Second }},
		{name: "zero read header timeout", mutate: func(cfg *Config) { cfg.ReadHeaderTimeout = 0 }},
		{name: "negative shutdown timeout", mutate: func(cfg *Config) { cfg.ShutdownTimeout = -time.Second }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := base
			tt.mutate(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatalf("Validate returned nil for invalid configuration")
			}
		})
	}
}

func TestRuntimeBaseDirForExecutableUsesBinaryDirOutsideGoRun(t *testing.T) {
	cwd := filepath.Join(string(filepath.Separator), "workspace", "backend")
	binaryPath := filepath.Join(string(filepath.Separator), "opt", "gameatlas", "game-server")
	if got := runtimeBaseDirForExecutable(cwd, binaryPath); got != filepath.Dir(binaryPath) {
		t.Fatalf("runtimeBaseDirForExecutable() = %q, want %q", got, filepath.Dir(binaryPath))
	}

	goRunPath := filepath.Join(string(filepath.Separator), "tmp", "go-build123", "b001", "exe", "server")
	if got := runtimeBaseDirForExecutable(cwd, goRunPath); got != cwd {
		t.Fatalf("runtimeBaseDirForExecutable(go run) = %q, want %q", got, cwd)
	}
}

func TestRuntimeSettingsKeepResourcePathsRelativeToRuntimeBase(t *testing.T) {
	baseDir := filepath.Join(string(filepath.Separator), "opt", "gameatlas")
	cfg := Config{
		runtimeBaseDir: baseDir,
		AssetsDir:      filepath.Join(baseDir, "data", "gamelist"),
		DBBackupDir:    filepath.Join(baseDir, "data", "backups"),
		pathSettings: map[string]string{
			"ASSETS_DIR":    "data/gamelist",
			"DB_BACKUP_DIR": "data/backups",
		},
	}

	settings := cfg.RuntimeSettings()
	if settings["DB_BACKUP_DIR"] != "data/backups" {
		t.Fatalf("DB_BACKUP_DIR = %q, want relative path", settings["DB_BACKUP_DIR"])
	}

	cfg.pathSettings["DB_BACKUP_DIR"] = cfg.DBBackupDir
	normalized := cfg.NormalizeStoredRuntimePaths()
	if normalized["DB_BACKUP_DIR"] != "data/backups" {
		t.Fatalf("normalized DB_BACKUP_DIR = %q, want relative path", normalized["DB_BACKUP_DIR"])
	}
}

func TestLoadAggregatesConfigurationErrors(t *testing.T) {
	t.Setenv("ADMIN_PASSWORD", "secret")
	t.Setenv("PORT", "abc")
	t.Setenv("AUTH_COOLDOWN", "10min")

	_, err := Load()
	if err == nil {
		t.Fatalf("expected Load to return error")
	}
	if !strings.Contains(err.Error(), `PORT="abc"`) {
		t.Fatalf("Load error = %v, want PORT parse failure", err)
	}
	if !strings.Contains(err.Error(), `AUTH_COOLDOWN="10min"`) {
		t.Fatalf("Load error = %v, want AUTH_COOLDOWN parse failure", err)
	}
}

func TestResolveRuntimePathResolvesRelativePathAgainstBaseDir(t *testing.T) {
	baseDir := filepath.Join(string(filepath.Separator), "workspace", "backend")
	got := resolveRuntimePath(baseDir, "data/app.db")
	want := filepath.Join(baseDir, "data", "app.db")
	if got != want {
		t.Fatalf("resolveRuntimePath() = %q, want %q", got, want)
	}
}

func TestResolveRuntimePathLeavesAbsolutePathUntouched(t *testing.T) {
	absolute := filepath.Join(string(filepath.Separator), "var", "lib", "game", "app.db")
	got := resolveRuntimePath(filepath.Join(string(filepath.Separator), "workspace", "backend"), absolute)
	if got != absolute {
		t.Fatalf("resolveRuntimePath() = %q, want %q", got, absolute)
	}
}
