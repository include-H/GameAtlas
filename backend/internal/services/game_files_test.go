package services

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"

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

func TestSanitizeBatchFileName(t *testing.T) {
	if got := sanitizeBatchFileName(`  bad<name>: demo?.bat  `); got != "bad-name--demo-.bat" {
		t.Fatalf("sanitizeBatchFileName() = %q", got)
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

func TestPsQuoted(t *testing.T) {
	if got := psQuoted(`it's "quoted"`); got != `'it''s "quoted"'` {
		t.Fatalf("psQuoted() = %q", got)
	}
}

func TestEncodeUTF16LEWithBOM(t *testing.T) {
	got := encodeUTF16LEWithBOM("中文A")
	want := []byte{0xFF, 0xFE, 0x2D, 0x4E, 0x87, 0x65, 0x41, 0x00}
	if string(got) != string(want) {
		t.Fatalf("encodeUTF16LEWithBOM() = %v, want %v", got, want)
	}
}

// decodePayload 提取 bat 壳最后一行 base64，解码回 UTF-8 的 PS 主脚本。
func decodePayload(t *testing.T, script string) string {
	t.Helper()
	lines := strings.Split(strings.TrimRight(script, "\r\n"), "\r\n")
	payload := strings.TrimSpace(lines[len(lines)-1])
	raw, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		t.Fatalf("decode base64 payload: %v", err)
	}
	if len(raw) < 2 || raw[0] != 0xFF || raw[1] != 0xFE {
		t.Fatalf("payload missing UTF-16LE BOM: %v", raw[:8])
	}
	units := make([]uint16, (len(raw)-2)/2)
	for i := range units {
		units[i] = uint16(raw[2+i*2]) | uint16(raw[2+i*2+1])<<8
	}
	return string(utf16.Decode(units))
}

// newLaunchServiceForTest 构造带临时 ROM 根目录的启动服务。
func newLaunchServiceForTest(t *testing.T, saveTemplate string) (int64, int64, *WindowsLaunchService) {
	t.Helper()
	db := openServicesTestDB(t)
	t.Cleanup(func() { _ = db.Close() })

	root := t.TempDir()
	romPath := filepath.Join(root, "nested", "game.vhdx")
	if err := os.MkdirAll(filepath.Dir(romPath), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(romPath, []byte("vhdx"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	gameID := insertServicesTestGame(t, db, "launch-script-game", "Launch Script Game", domain.GameVisibilityPublic)
	if saveTemplate != "" {
		if _, err := db.Exec("UPDATE games SET save_path_template = ? WHERE id = ?", saveTemplate, gameID); err != nil {
			t.Fatalf("set save_path_template: %v", err)
		}
	}
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
	return gameID, fileID, service
}

func TestBuildLaunchScriptReturnsBatShellWithBase64Payload(t *testing.T) {
	gameID, fileID, service := newLaunchServiceForTest(t, "")

	script, filename, err := service.BuildLaunchScript(gameID, fileID, false)
	if err != nil {
		t.Fatalf("BuildLaunchScript returned error: %v", err)
	}

	if filename != "Launch-Script-Game.bat" {
		t.Fatalf("filename = %q, want Launch-Script-Game.bat", filename)
	}
	if !strings.HasPrefix(script, "@echo off\r\n") {
		t.Fatalf("script missing bat shell header: %s", script[:60])
	}
	if !strings.Contains(script, "Start-Process -FilePath '%~f0' -Verb RunAs") {
		t.Fatalf("script missing elevation bootstrap: %s", script)
	}
	if !strings.Contains(script, "[IO.File]::WriteAllBytes") {
		t.Fatalf("script missing payload decode step: %s", script)
	}

	ps := decodePayload(t, script)
	if !strings.Contains(ps, "$GameTitle   = 'Launch Script Game'") {
		t.Fatalf("ps missing injected title: %s", ps[:300])
	}
	if !strings.Contains(ps, "$SMBHost     = 'NAS'") {
		t.Fatalf("ps missing injected smb host: %s", ps[:300])
	}
	if !strings.Contains(ps, "$SMBShare    = '\\\\NAS\\Share\\Games'") {
		t.Fatalf("ps missing injected smb share: %s", ps[:300])
	}
	if !strings.Contains(ps, "$BaseVHD     = '\\\\NAS\\Share\\Games\\nested\\game.vhdx'") {
		t.Fatalf("ps missing injected base vhd: %s", ps[:400])
	}
	if !strings.Contains(ps, "$DiffVHD     = 'D:\\game.vhdx'") {
		t.Fatalf("ps missing injected diff vhd: %s", ps[:400])
	}
}

func TestPowerShellMainSMBReuseDetection(t *testing.T) {
	gameID, fileID, service := newLaunchServiceForTest(t, "")
	script, _, err := service.BuildLaunchScript(gameID, fileID, false)
	if err != nil {
		t.Fatalf("BuildLaunchScript returned error: %v", err)
	}
	ps := decodePayload(t, script)

	if !strings.Contains(ps, "net use 2>$null") {
		t.Fatalf("ps missing smb session list: %s", ps)
	}
	if !strings.Contains(ps, "Select-String -SimpleMatch $SMBShare -Quiet") {
		t.Fatalf("ps missing smb reuse check: %s", ps)
	}
	if !strings.Contains(ps, "直接复用现有连接") {
		t.Fatalf("ps missing reuse message: %s", ps)
	}
}

func TestPowerShellMainDriveDiffDetection(t *testing.T) {
	gameID, fileID, service := newLaunchServiceForTest(t, "")
	script, _, err := service.BuildLaunchScript(gameID, fileID, false)
	if err != nil {
		t.Fatalf("BuildLaunchScript returned error: %v", err)
	}
	ps := decodePayload(t, script)

	if !strings.Contains(ps, "$beforeDrives = @(Get-Partition") {
		t.Fatalf("ps missing pre-mount drive snapshot: %s", ps)
	}
	if !strings.Contains(ps, "$beforeDrives -notcontains $_") {
		t.Fatalf("ps missing drive diff: %s", ps)
	}
	if !strings.Contains(ps, "游戏盘符") {
		t.Fatalf("ps missing drive display: %s", ps)
	}
}

func TestPowerShellMainExeScan(t *testing.T) {
	gameID, fileID, service := newLaunchServiceForTest(t, "")
	script, _, err := service.BuildLaunchScript(gameID, fileID, false)
	if err != nil {
		t.Fatalf("BuildLaunchScript returned error: %v", err)
	}
	ps := decodePayload(t, script)

	if !strings.Contains(ps, "-Recurse -Depth 3 -Filter *.exe -File") {
		t.Fatalf("ps missing exe scan: %s", ps)
	}
	if !strings.Contains(ps, "-notmatch '^(unins|uninstall|vcredist|dxsetup|redist|setup|dotnet|vulkaninfo|crash)'") {
		t.Fatalf("ps missing exe exclusion filter: %s", ps)
	}
	if !strings.Contains(ps, "Start-Process -FilePath $selected") {
		t.Fatalf("ps missing exe launch: %s", ps)
	}
	// 转义地狱已消除：PS 内不再出现 cmd 的 ^| 转义。
	if strings.Contains(ps, "^|") {
		t.Fatalf("ps must not contain cmd caret escapes: %s", ps)
	}
}

func TestPowerShellMainSaveTemplateRendersMenuAndAction(t *testing.T) {
	gameID, fileID, service := newLaunchServiceForTest(t, `%USERPROFILE%\Documents\My Games\Save Template Game\SaveGame`)
	script, _, err := service.BuildLaunchScript(gameID, fileID, false)
	if err != nil {
		t.Fatalf("BuildLaunchScript returned error: %v", err)
	}
	ps := decodePayload(t, script)

	if !strings.Contains(ps, "挂载并打开存档目录") {
		t.Fatalf("ps missing save-dir menu option: %s", ps)
	}
	if !strings.Contains(ps, "[Environment]::ExpandEnvironmentVariables($SaveTemplate)") {
		t.Fatalf("ps missing template expansion: %s", ps)
	}
	if !strings.Contains(ps, "Start-Process explorer.exe -ArgumentList $saveDir") {
		t.Fatalf("ps missing explorer open: %s", ps)
	}
	// 模板保留 %VAR%：不能被转义。
	if !strings.Contains(ps, `$SaveTemplate = '%USERPROFILE%\Documents\My Games\Save Template Game\SaveGame'`) {
		t.Fatalf("ps missing raw save template: %s", ps)
	}
}

func TestPowerShellMainWithoutSaveTemplateOmitsMenuOption(t *testing.T) {
	gameID, fileID, service := newLaunchServiceForTest(t, "")
	script, _, err := service.BuildLaunchScript(gameID, fileID, false)
	if err != nil {
		t.Fatalf("BuildLaunchScript returned error: %v", err)
	}
	ps := decodePayload(t, script)

	if strings.Contains(ps, "挂载并打开存档目录") {
		t.Fatalf("ps should not render save-dir option when template empty: %s", ps)
	}
	if strings.Contains(ps, "$hasSave") {
		t.Fatalf("ps should not contain hasSave guard when template empty: %s", ps)
	}
}

// TestPowerShellMainDiskpartScriptEncoding 锁定 diskpart 脚本生成方式：
//   - 逐行 Add 到 List[string]（PS 5.1 的 @(...) 多行数组+字符串拼接会被解析成单元素，
//     真机事故：三命令变一行导致 diskpart 无法执行）；
//   - WriteAllLines 写 GBK(936)：diskpart /s 只认 ANSI 脚本（真机验证 UTF-16 返回 0 无回显，
//     原始 BAT 用 GBK 能执行到挂载），与中文系统代码页匹配；
//   - 必须带命令数日志便于排错。
func TestPowerShellMainDiskpartScriptEncoding(t *testing.T) {
	gameID, fileID, service := newLaunchServiceForTest(t, "")
	script, _, err := service.BuildLaunchScript(gameID, fileID, false)
	if err != nil {
		t.Fatalf("BuildLaunchScript returned error: %v", err)
	}
	ps := decodePayload(t, script)

	if !strings.Contains(ps, "New-Object System.Collections.Generic.List[string]") {
		t.Fatalf("ps must build diskpart script via List[string] (not @() multi-line): %s", ps)
	}
	if !strings.Contains(ps, "[System.IO.File]::WriteAllLines($dp, $scriptLines, [System.Text.Encoding]::GetEncoding(936))") {
		t.Fatalf("ps must write diskpart script via WriteAllLines GBK(936): %s", ps)
	}
	if strings.Contains(ps, "[System.Text.Encoding]::Unicode") {
		t.Fatalf("ps must NOT use Unicode encoding for diskpart script (diskpart ignores UTF-16): %s", ps)
	}
	if strings.Contains(ps, "$lines = @(") {
		t.Fatalf("ps must NOT use multi-line @() array for diskpart commands: %s", ps)
	}
	// diskpart 输出必须保留（去掉 2>$null），排错时能看到具体错误。
	if !strings.Contains(ps, "diskpart /s $dp\r\n") && !strings.Contains(ps, "diskpart /s $dp\n") {
		t.Fatalf("ps diskpart invocation must not swallow output: %s", ps)
	}
	if !strings.Contains(ps, "[脚本命令数]") {
		t.Fatalf("ps missing command count diagnostic: %s", ps)
	}
}

// TestPowerShellMainChinesePathsSurviveRoundTrip 验证中文路径经 UTF-16LE base64
// 编码往返后完整保留（bat 壳解码到 .ps1 时中文不被破坏）。
func TestPowerShellMainChinesePathsSurviveRoundTrip(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	root := t.TempDir()
	romPath := filepath.Join(root, "中文游戏", "nested", "游戏.vhdx")
	if err := os.MkdirAll(filepath.Dir(romPath), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(romPath, []byte("vhdx"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	gameID := insertServicesTestGame(t, db, "chinese-game", "中文游戏标题", domain.GameVisibilityPublic)
	fileID := insertServicesGameFile(t, db, gameID, romPath, 0)
	service := NewWindowsLaunchService(
		config.Config{
			PrimaryROMRoot:  root,
			SMBPathMappings: root + "=//NAS/Share/游戏库",
			SMBUsername:     "demo-user",
			SMBPassword:     "demo-pass",
			VHDDiffRoot:     "d:",
		},
		repositories.NewGamesRepository(db),
		repositories.NewGameFilesRepository(db),
	)

	script, _, err := service.BuildLaunchScript(gameID, fileID, false)
	if err != nil {
		t.Fatalf("BuildLaunchScript returned error: %v", err)
	}
	ps := decodePayload(t, script)

	if !strings.Contains(ps, "$GameTitle   = '中文游戏标题'") {
		t.Fatalf("ps missing Chinese title: %s", ps[:500])
	}
	if !strings.Contains(ps, `$BaseVHD     = '\\NAS\Share\游戏库\中文游戏\nested\游戏.vhdx'`) {
		t.Fatalf("ps missing Chinese base vhd path: %s", ps[:500])
	}
}
