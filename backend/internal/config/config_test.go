package config

import (
	"os"
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

func TestNormalizeEnvValueStripsQuotes(t *testing.T) {
	cases := map[string]string{
		`"hello"`: "hello",
		`'world'`: "world",
		"plain":   "plain",
	}

	for input, want := range cases {
		if got := normalizeEnvValue(input); got != want {
			t.Fatalf("normalizeEnvValue(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestLoadDotEnvDoesNotOverrideExistingVariables(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := "EXISTING=from_file\nQUOTED=\"quoted value\"\nSINGLE='single value'\nINVALID_LINE\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	t.Setenv("EXISTING", "from_env")
	if err := loadDotEnv(path); err == nil {
		t.Fatalf("expected loadDotEnv to reject invalid line")
	}

	if got := os.Getenv("EXISTING"); got != "from_env" {
		t.Fatalf("EXISTING = %q, want from_env", got)
	}
	if got := os.Getenv("QUOTED"); got != "quoted value" {
		t.Fatalf("QUOTED = %q, want quoted value", got)
	}
	if got := os.Getenv("SINGLE"); got != "single value" {
		t.Fatalf("SINGLE = %q, want single value", got)
	}
}

func TestLoadDotEnvRejectsMalformedLineWithLocation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := "GOOD=value\nBROKEN_LINE\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	err := loadDotEnv(path)
	if err == nil {
		t.Fatalf("expected loadDotEnv to return error")
	}
	if !strings.Contains(err.Error(), ".env:2") {
		t.Fatalf("loadDotEnv error = %v, want line number", err)
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

func TestChooseRuntimeBaseDirPrefersDotEnvDirectory(t *testing.T) {
	cwd := filepath.Join(string(filepath.Separator), "workspace", "backend")
	executableDir := filepath.Join(string(filepath.Separator), "release")
	exists := func(path string) bool {
		return path == filepath.Join(executableDir, ".env")
	}

	baseDir, dotEnvPath := chooseRuntimeBaseDir(cwd, executableDir, exists)
	if baseDir != executableDir {
		t.Fatalf("baseDir = %q, want %q", baseDir, executableDir)
	}
	if dotEnvPath != filepath.Join(executableDir, ".env") {
		t.Fatalf("dotEnvPath = %q, want executable .env path", dotEnvPath)
	}
}

func TestChooseRuntimeBaseDirFallsBackToGoModuleDirectory(t *testing.T) {
	cwd := filepath.Join(string(filepath.Separator), "workspace", "backend")
	executableDir := filepath.Join(string(filepath.Separator), "tmp", "go-build")
	exists := func(path string) bool {
		return path == filepath.Join(cwd, "go.mod")
	}

	baseDir, dotEnvPath := chooseRuntimeBaseDir(cwd, executableDir, exists)
	if baseDir != cwd {
		t.Fatalf("baseDir = %q, want %q", baseDir, cwd)
	}
	if dotEnvPath != "" {
		t.Fatalf("dotEnvPath = %q, want empty string", dotEnvPath)
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
