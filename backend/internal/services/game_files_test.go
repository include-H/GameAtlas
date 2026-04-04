package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hao/game/internal/config"
	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/repositories"
)

func TestNormalizeUNCPath(t *testing.T) {
	cases := map[string]string{
		`\\NAS\Share\Game`:    `\\NAS\Share\Game`,
		` //NAS/Share//Game `: `\\NAS\Share\Game`,
		"":                    `\\`,
	}
	for input, want := range cases {
		if got := normalizeUNCPath(input); got != want {
			t.Fatalf("normalizeUNCPath(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestExtractSMBHost(t *testing.T) {
	if got := extractSMBHost(`\\NAS\Share\Games`); got != "NAS" {
		t.Fatalf("extractSMBHost() = %q, want NAS", got)
	}
}

func TestSanitizeBatchFileName(t *testing.T) {
	if got := sanitizeBatchFileName(`  bad<name>: demo?.bat  `); got != "bad-name--demo-.bat" {
		t.Fatalf("sanitizeBatchFileName() = %q", got)
	}
}

func TestEscapeBatchValue(t *testing.T) {
	if got := escapeBatchValue(`100%^&|<>`); got != `100%%^^^&^|^<^>` {
		t.Fatalf("escapeBatchValue() = %q", got)
	}
}

func TestBuildDiffVHDPathAndNormalizeDriveRoot(t *testing.T) {
	if got := normalizeDriveRoot(" d:\\games "); got != "D:" {
		t.Fatalf("normalizeDriveRoot() = %q, want D:", got)
	}
	if got := buildDiffVHDPath("d:", `\diffs\game.vhdx`); got != `D:\diffs\game.vhdx` {
		t.Fatalf("buildDiffVHDPath() = %q", got)
	}
}

func TestWindowsLaunchServiceBuildLaunchScriptUsesMappedSMBPath(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	root := t.TempDir()
	romPath := filepath.Join(root, "nested", "game.vhdx")
	if err := os.MkdirAll(filepath.Dir(romPath), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(romPath, []byte("vhdx"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	gameID := insertServicesTestGame(t, db, "launch-script-game", "Launch Script Game", domain.GameVisibilityPublic)
	fileID := insertServicesGameFile(t, db, gameID, romPath, 0)
	service := NewWindowsLaunchService(
		config.Config{
			PrimaryROMRoot:  root,
			SMBPathMappings: root + "=//NAS/Share/Games",
			SMBUsername:     "demo-user",
			SMBPassword:     "demo-pass",
			VHDDiffRoot:     "d:",
		},
		repositories.NewGamesRepository(db),
		repositories.NewGameFilesRepository(db),
	)

	script, filename, err := service.BuildLaunchScript(gameID, fileID, false)
	if err != nil {
		t.Fatalf("BuildLaunchScript returned error: %v", err)
	}

	if filename != "Launch-Script-Game.bat" {
		t.Fatalf("filename = %q, want Launch-Script-Game.bat", filename)
	}
	if !strings.Contains(script, `set "SMB_SHARE=\\NAS\Share\Games"`) {
		t.Fatalf("script missing SMB_SHARE mapping: %s", script)
	}
	if !strings.Contains(script, `set "BASE_VHD=\\NAS\Share\Games\nested\game.vhdx"`) {
		t.Fatalf("script missing BASE_VHD mapping: %s", script)
	}
	if !strings.Contains(script, `set "DIFF_VHD=D:\game.vhdx"`) {
		t.Fatalf("script missing DIFF_VHD path: %s", script)
	}
	if !strings.Contains(script, `set "COLOR_INFO=%ESC%[96m"`) {
		t.Fatalf("script missing color init: %s", script)
	}
	if !strings.Contains(script, `call :PRINT_COLOR "%COLOR_INFO%" "  1. 挂载 SMB 并挂载游戏"`) {
		t.Fatalf("script missing mount menu option: %s", script)
	}
	if !strings.Contains(script, `call :PRINT_COLOR "%COLOR_INFO%" "  2. 删除 Windows 中刚刚添加的 SMB 凭据"`) {
		t.Fatalf("script missing credential removal menu option: %s", script)
	}
	if !strings.Contains(script, `goto REMOVE_SMB_CREDENTIAL`) {
		t.Fatalf("script missing credential removal branch: %s", script)
	}
	if !strings.Contains(script, `cmdkey /delete:%SMB_HOST% >nul 2>&1`) {
		t.Fatalf("script missing credential delete command: %s", script)
	}
	if !strings.Contains(script, `net use %SMB_SHARE% /delete /y >nul 2>&1`) {
		t.Fatalf("script missing SMB disconnect command: %s", script)
	}
	if !strings.Contains(script, `:PRINT_COLOR`) {
		t.Fatalf("script missing color print helper: %s", script)
	}
}
