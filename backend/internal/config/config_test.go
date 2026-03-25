package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	loadDotEnv(path)

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
