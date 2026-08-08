package services

import (
	"encoding/base64"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/hao/game/internal/config"
	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/files"
	"github.com/hao/game/internal/repositories"
)

type WindowsLaunchService struct {
	gamesRepo     gameDetailReadRepository
	gameFilesRepo *repositories.GameFilesRepository
	fileGuard     *files.Guard
	cfg           config.Config
}

func NewWindowsLaunchService(cfg config.Config, gamesRepo gameDetailReadRepository, gameFilesRepo *repositories.GameFilesRepository) *WindowsLaunchService {
	return &WindowsLaunchService{
		gamesRepo:     gamesRepo,
		gameFilesRepo: gameFilesRepo,
		fileGuard:     files.NewGuard(cfg.PrimaryROMRoot),
		cfg:           cfg,
	}
}

func (s *WindowsLaunchService) BuildLaunchScript(gameID, fileID int64, includeAll bool) (string, string, error) {
	if strings.TrimSpace(s.cfg.SMBUsername) == "" ||
		strings.TrimSpace(s.cfg.SMBPassword) == "" {
		return "", "", domain.ErrMissingSMBConfig
	}

	game, err := s.gamesRepo.GetByID(gameID)
	if err != nil {
		return "", "", normalizeRepoError(err)
	}
	if !includeAll && game.Visibility == domain.GameVisibilityPrivate {
		return "", "", domain.ErrNotFound
	}

	file, err := s.gameFilesRepo.GetByID(gameID, fileID)
	if err != nil {
		return "", "", normalizeRepoError(err)
	}

	resolved, err := s.fileGuard.ValidateFile(file.FilePath)
	if err != nil {
		return "", "", normalizeFileError(err)
	}

	ext := strings.ToLower(filepath.Ext(resolved.ResolvedPath))
	if ext != ".vhd" && ext != ".vhdx" {
		return "", "", domain.ErrInvalidLaunchFile
	}

	baseVHDPath, shareRoot, err := s.buildSMBMountedPath(resolved.ResolvedPath)
	if err != nil {
		return "", "", err
	}

	diffFileName := filepath.Base(resolved.ResolvedPath)
	title := strings.TrimSpace(game.Title)
	if title == "" {
		title = "未命名游戏"
	}
	script := s.renderBatLauncher(game.ID, file.ID, title, shareRoot, baseVHDPath, diffFileName, game.SavePathTemplate)
	filename := sanitizeBatchFileName(game.Title)
	if filename == "" {
		filename = "launch-game"
	}

	return script, filename + ".bat", nil
}

func (s *WindowsLaunchService) buildSMBMountedPath(resolvedPath string) (string, string, error) {
	mappings, err := s.cfg.ParseSMBPathMappings()
	if err != nil {
		return "", "", err
	}
	if len(mappings) == 0 {
		return "", "", domain.ErrMissingSMBConfig
	}

	base, shareRoot, mappingErr := s.buildMappedSMBPath(resolvedPath, mappings)
	if mappingErr != nil {
		return "", "", mappingErr
	}

	return base, shareRoot, nil
}

func (s *WindowsLaunchService) buildMappedSMBPath(resolvedPath string, mappings []config.SMBPathMapping) (string, string, error) {
	longestPrefixLength := -1
	selectedBase := ""
	selectedShareRoot := ""

	for _, mapping := range mappings {
		resolvedRoot, err := filepath.EvalSymlinks(mapping.LocalRoot)
		if err != nil {
			return "", "", normalizeFileError(err)
		}

		relative, err := filepath.Rel(resolvedRoot, resolvedPath)
		if err != nil || relative == "." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || relative == ".." {
			continue
		}

		relativeWindows := strings.ReplaceAll(filepath.ToSlash(relative), "/", `\`)
		shareRoot := normalizeUNCPath(mapping.ShareRoot)
		base := shareRoot
		if relativeWindows != "" {
			base += `\` + relativeWindows
		}

		if len(resolvedRoot) > longestPrefixLength {
			longestPrefixLength = len(resolvedRoot)
			selectedBase = base
			selectedShareRoot = shareRoot
		}
	}

	if longestPrefixLength == -1 {
		return "", "", domain.ErrForbiddenPath
	}

	return selectedBase, selectedShareRoot, nil
}

// renderBatLauncher 生成 ASCII 引导壳：提权后解码文件末尾 base64（UTF-16LE+BOM 的
// PS 主脚本），写入临时 .ps1 执行。全 ASCII 因此不受 GBK/UTF-8 编码影响。
func (s *WindowsLaunchService) renderBatLauncher(gameID, fileID int64, gameTitle, shareRoot, baseVHDPath, diffFileName, savePathTemplate string) string {
	mainScript := s.renderPowerShellMain(gameID, fileID, gameTitle, shareRoot, baseVHDPath, diffFileName, savePathTemplate)
	// PS 5.1 对 LF 换行的多行结构（@() 数组等）解析异常（真机事故：数组变单元素），
	// 统一规范为 CRLF 后写入 payload。
	mainScript = strings.ReplaceAll(mainScript, "\r\n", "\n")
	mainScript = strings.ReplaceAll(mainScript, "\n", "\r\n")
	utf16Bytes := encodeUTF16LEWithBOM(mainScript)
	payload := base64.StdEncoding.EncodeToString(utf16Bytes)

	var b strings.Builder
	b.WriteString("@echo off\r\n")
	b.WriteString("setlocal\r\n")
	b.WriteString("rem GameAtlas VHD launcher (ASCII shell, PowerShell payload in last line)\r\n")
	b.WriteString("rem Elevate to admin then restart self\r\n")
	b.WriteString("fltmc >nul 2>&1\r\n")
	b.WriteString("if errorlevel 1 (\r\n")
	b.WriteString("  echo Requesting administrator privileges...\r\n")
	b.WriteString("  powershell -NoProfile -ExecutionPolicy Bypass -Command \"Start-Process -FilePath '%~f0' -Verb RunAs\"\r\n")
	b.WriteString("  exit /b\r\n")
	b.WriteString(")\r\n")
	b.WriteString("\r\n")
	b.WriteString("rem Decode base64 payload in last line into temp ps1 and run it\r\n")
	b.WriteString("powershell -NoProfile -ExecutionPolicy Bypass -Command \"$l = Get-Content -LiteralPath '%~f0'; $b64 = $l[$l.Count-1].Trim(); $p = Join-Path $env:TEMP ('ga-launch-" + strconv.FormatInt(gameID, 10) + "-" + strconv.FormatInt(fileID, 10) + "-' + [guid]::NewGuid().ToString('N') + '.ps1'); [IO.File]::WriteAllBytes($p, [Convert]::FromBase64String($b64)); & $p; Remove-Item $p -Force -ErrorAction SilentlyContinue\"\r\n")
	b.WriteString("pause\r\n")
	b.WriteString("endlocal\r\n")
	b.WriteString(payload)
	return b.String()
}

// renderPowerShellMain 生成 PS 主脚本（UTF-8 字符串，最终以 UTF-16LE+BOM 编码后
// base64 内嵌进 bat 壳）。占位符由值替换注入，PS 字符串用单引号包裹并转义 ”。
func (s *WindowsLaunchService) renderPowerShellMain(gameID, fileID int64, gameTitle, shareRoot, baseVHDPath, diffFileName, savePathTemplate string) string {
	tmpl := `# GameAtlas VHD remote launcher
# Generated by GameAtlas. Windows PowerShell 5.1+ required.

$ErrorActionPreference = 'Stop'

# ---- injected parameters ----
$GameTitle   = __GAME_TITLE__
$SMBHost     = __SMB_HOST__
$SMBShare    = __SMB_SHARE__
$SMBUser     = __SMB_USER__
$SMBPass     = __SMB_PASS__
$BaseVHD     = __BASE_VHD__
$DiffVHD     = __DIFF_VHD__
$SaveTemplate = __SAVE_TEMPLATE__

function Write-Step  { param([string]$m) Write-Host $m -ForegroundColor Yellow }
function Write-Ok    { param([string]$m) Write-Host $m -ForegroundColor Green }
function Write-Err   { param([string]$m) Write-Host $m -ForegroundColor Red }
function Write-Info  { param([string]$m) Write-Host $m -ForegroundColor Cyan }

function Wait-Key {
  Write-Host ''
  Read-Host '按任意键继续...' | Out-Null
}

# ---- banner ----
$sep = '─' * 50
Write-Host ('┌' + $sep + '┐') -ForegroundColor Cyan
Write-Host ('│' + (' ' * 12) + 'GameAtlas · VHD 远程启动' + (' ' * 12) + '│') -ForegroundColor Cyan
Write-Host ('│' + (' ' * 8) + ('《' + $GameTitle + '》').PadLeft(30) + (' ' * 8) + '│') -ForegroundColor Cyan
Write-Host ('└' + $sep + '┘') -ForegroundColor Cyan

Write-Host ''
Write-Info '当前配置'
Write-Host ('  SMB 主机 ：' + $SMBHost) -ForegroundColor DarkGray
Write-Host ('  SMB 共享 ：' + $SMBShare) -ForegroundColor DarkGray
Write-Host ('  基础 VHD ：' + $BaseVHD) -ForegroundColor DarkGray
Write-Host ('  差分 VHD ：' + $DiffVHD) -ForegroundColor DarkGray

# ---- menu ----
$mode = ''
while ($mode -notin @('1','2','3','4')) {
  Write-Host ''
  Write-Host '请选择操作：'
  Write-Host '  [1] 挂载并游玩（自动启动游戏，结束后自动卸载并清理凭据）'
  Write-Host '  [2] 仅挂载（保留连接与凭据）'
__MENU_OPTIONS__
  $mode = Read-Host '  请输入选项'
  $mode = $mode.Trim()
}
Clear-Host
Write-Info ('正在执行：' + $GameTitle)

__CLEANUP_COND__
  Write-Step '[1/2] 正在断开 SMB 共享...'
  net use $SMBShare /delete /y 2>$null | Out-Null
  Write-Step '[2/2] 正在删除已保存凭据...'
  cmdkey /delete:$SMBHost 2>$null | Out-Null
  Write-Ok '清理完成。'
  Wait-Key
  exit 0
}

$cleanupOnExit = ($mode -ne '2')

# ---- connect SMB (reuse existing session) ----
$existing = net use 2>$null
if ($existing | Select-String -SimpleMatch $SMBShare -Quiet) {
  Write-Ok 'SMB 共享已连接，直接复用现有连接。'
} else {
  Write-Step '添加 SMB 凭据...'
  cmdkey /add:$SMBHost /user:$SMBUser /pass:$SMBPass 2>$null | Out-Null
  Write-Step '连接 SMB 共享...'
  net use $SMBShare /user:$SMBUser $SMBPass /persistent:no 2>$null | Out-Null
  if ($LASTEXITCODE -ne 0) {
    Write-Err 'SMB 共享连接失败。凭据可能已残留，可再次运行本脚本选择清理选项。'
    Wait-Key
    exit 1
  }
  Write-Ok 'SMB 共享连接成功。'
}

# ---- snapshot drives before mount ----
$beforeDrives = @(Get-Partition -ErrorAction SilentlyContinue | Where-Object { $_.DriveLetter } | Select-Object -ExpandProperty DriveLetter)

# ---- mount VHD (create diff disk when missing) ----
$dp = Join-Path $env:TEMP ('ga-diskpart-' + [guid]::NewGuid().ToString('N') + '.txt')
$scriptLines = New-Object System.Collections.Generic.List[string]
if (Test-Path -LiteralPath $DiffVHD) {
  Write-Step '差分 VHD 已存在，准备挂载...'
  $scriptLines.Add('select vdisk file="' + $DiffVHD + '"')
  $scriptLines.Add('attach vdisk')
} else {
  Write-Step '创建差分 VHD...'
  $scriptLines.Add('create vdisk file="' + $DiffVHD + '" parent="' + $BaseVHD + '"')
  $scriptLines.Add('select vdisk file="' + $DiffVHD + '"')
  $scriptLines.Add('attach vdisk')
}
Write-Host '--- diskpart 脚本内容 ---' -ForegroundColor DarkGray
$scriptLines | ForEach-Object { Write-Host ('  ' + $_) -ForegroundColor DarkGray }
Write-Host ('  [脚本路径] ' + $dp) -ForegroundColor DarkGray
Write-Host ('  [脚本命令数] ' + $scriptLines.Count) -ForegroundColor DarkGray
# List 逐行 Add：PS 5.1 的 @(...) 多行数组 + 字符串拼接会被解析成单元素（真机事故）。
# GetEncoding(936)=GBK/ANSI：diskpart /s 只认 ANSI 脚本（真机验证 UTF-16 返回 0 无回显，
# 原始 BAT 用 GBK 能执行到挂载），与中文系统代码页匹配。
[System.IO.File]::WriteAllLines($dp, $scriptLines, [System.Text.Encoding]::GetEncoding(936))
Write-Host ('  [脚本文件字节数] ' + (Get-Item -LiteralPath $dp).Length) -ForegroundColor DarkGray
Write-Step '正在挂载 VHD...'
diskpart /s $dp
$err = $LASTEXITCODE
Write-Host ('  [diskpart 退出码] ' + $err) -ForegroundColor DarkGray
Remove-Item -LiteralPath $dp -Force -ErrorAction SilentlyContinue
if ($err -ne 0) {
  Write-Err ('VHD 挂载失败，错误码 ' + $err + '。凭据可能已残留，可再次运行本脚本选择清理选项。')
  Wait-Key
  exit $err
}
if (Test-Path -LiteralPath $DiffVHD) {
  Write-Ok '差分 VHD 已挂载成功。'
} else {
  Write-Err ('[检查] 差分 VHD 文件未生成：' + $DiffVHD)
}

# ---- find game drive by diff (new partition since mount) ----
$gameDrive = ''
$afterDrives = @(Get-Partition -ErrorAction SilentlyContinue | Where-Object { $_.DriveLetter } | Select-Object -ExpandProperty DriveLetter)
Write-Host ('  [挂载前盘符] ' + ($beforeDrives -join ',')) -ForegroundColor DarkGray
Write-Host ('  [挂载后盘符] ' + ($afterDrives -join ',')) -ForegroundColor DarkGray
$newDrives = @($afterDrives | Where-Object { $beforeDrives -notcontains $_ })
Write-Host ('  [新增盘符] ' + ($newDrives -join ',')) -ForegroundColor DarkGray
if ($newDrives.Count -gt 0) {
  $gameDrive = $newDrives[0]
  Write-Info ('游戏盘符：' + $gameDrive + ':')
} else {
  Write-Info '请打开“此电脑”找到新出现的盘符进入游戏。'
}

if (-not $cleanupOnExit) {
  Write-Ok '已保持挂载，SMB 凭据与共享连接保留。'
  Wait-Key
  exit 0
}

Clear-Host
Write-Info ('挂载完成，游戏盘符：' + $gameDrive + ':')

# ---- MODE=SAVE / MODE=PLAY dispatch ----
__MODE_DISPATCH__
  # ---- MODE=PLAY: scan and launch game executable ----
  Write-Step '正在扫描游戏主程序...'
  $exes = @()
  if ($gameDrive -ne '') {
    $exes = @(Get-ChildItem -LiteralPath ($gameDrive + ':\') -Recurse -Depth 3 -Filter *.exe -File -ErrorAction SilentlyContinue |
      Where-Object { $_.Name -notmatch '^(unins|uninstall|vcredist|dxsetup|redist|setup|dotnet|vulkaninfo|crash)' } |
      Sort-Object { $_.FullName.Length } |
      Select-Object -First 9)
  }
  $selected = ''
  if ($exes.Count -eq 0) {
    Write-Err '未找到可执行文件，请手动进入盘符打开游戏。'
  } elseif ($exes.Count -eq 1) {
    $selected = $exes[0].FullName
    Write-Ok ('正在启动：' + $selected)
    Start-Process -FilePath $selected
  } else {
    Write-Host '找到多个可执行文件，请选择要启动的程序：'
    for ($i = 0; $i -lt $exes.Count; $i++) {
      Write-Host ('  [' + ($i + 1) + '] ' + $exes[$i].FullName)
    }
    $n = Read-Host ('  请输入编号 [1-' + $exes.Count + ']')
    $n = $n.Trim()
    if ($n -match '^\d+$') {
      $idx = [int]$n - 1
      if ($idx -ge 0 -and $idx -lt $exes.Count) {
        $selected = $exes[$idx].FullName
        Write-Ok ('正在启动：' + $selected)
        Start-Process -FilePath $selected
      }
    }
    if ($selected -eq '') {
      Write-Err '无效编号，请手动进入盘符打开游戏。'
    }
  }
  Write-Host ''
  Write-Host '游玩结束后回到本窗口按任意键，脚本将自动卸载 VHD 并清理 SMB 凭据。'
  Read-Host '按任意键卸载...' | Out-Null
__MODE_DISPATCH_END__

Clear-Host
Write-Info '正在卸载并清理...'

# ---- unmount + cleanup ----
Write-Step '[1/3] 正在卸载 VHD...'
$dp = Join-Path $env:TEMP ('ga-diskpart-' + [guid]::NewGuid().ToString('N') + '.txt')
$scriptLines = New-Object System.Collections.Generic.List[string]
$scriptLines.Add('select vdisk file="' + $DiffVHD + '"')
$scriptLines.Add('detach vdisk')
[System.IO.File]::WriteAllLines($dp, $scriptLines, [System.Text.Encoding]::GetEncoding(936))
diskpart /s $dp
$err = $LASTEXITCODE
Write-Host ('  [卸载 diskpart 退出码] ' + $err) -ForegroundColor DarkGray
Remove-Item -LiteralPath $dp -Force -ErrorAction SilentlyContinue
if ($err -ne 0) {
  Write-Err ('VHD 卸载失败（错误码 ' + $err + '），请手动卸载后清理凭据。')
} else {
  Write-Ok 'VHD 已卸载，差分盘已保留（下次启动直接复用）。'
}
Write-Step '[2/3] 正在断开 SMB 共享...'
net use $SMBShare /delete /y 2>$null | Out-Null
Write-Step '[3/3] 正在删除 SMB 凭据...'
cmdkey /delete:$SMBHost 2>$null | Out-Null
Clear-Host
Write-Ok '操作完成。'
Write-Host '  VHD 已卸载，差分盘已保留（下次启动直接复用）。' -ForegroundColor DarkGray
Write-Host '  SMB 凭据与共享连接已清理。' -ForegroundColor DarkGray
Wait-Key
exit 0
`

	var menuOptions, cleanupCond, dispatch strings.Builder
	if savePathTemplate != "" {
		menuOptions.WriteString("  Write-Host '  [3] 挂载并打开存档目录（结束后自动卸载并清理凭据）'\n")
		menuOptions.WriteString("  Write-Host '  [4] 清理 SMB 凭据并断开共享'\n")
		cleanupCond.WriteString("if ($mode -eq '4') {\n")
		dispatch.WriteString("if ($mode -eq '3') {\n")
		dispatch.WriteString("  $saveDir = [Environment]::ExpandEnvironmentVariables($SaveTemplate)\n")
		dispatch.WriteString("  if ($gameDrive -ne '') { $saveDir = $saveDir.Replace('%GAME_DRIVE%', $gameDrive) }\n")
		dispatch.WriteString("  if (Test-Path -LiteralPath $saveDir) {\n")
		dispatch.WriteString("    Write-Ok ('存档目录：' + $saveDir)\n")
		dispatch.WriteString("    Start-Process explorer.exe -ArgumentList $saveDir\n")
		dispatch.WriteString("    Write-Host '存档目录已打开，可在此查看或备份存档。'\n")
		dispatch.WriteString("  } else {\n")
		dispatch.WriteString("    Write-Err ('存档目录不存在：' + $saveDir)\n")
		dispatch.WriteString("    Write-Err '可能尚未游玩过（存档目录在首次运行后创建），或模板配置有误。'\n")
		dispatch.WriteString("  }\n")
		dispatch.WriteString("  Read-Host '查看完毕后按任意键卸载...' | Out-Null\n")
		dispatch.WriteString("} else {\n")
	} else {
		menuOptions.WriteString("  Write-Host '  [3] 清理 SMB 凭据并断开共享'\n")
		cleanupCond.WriteString("if ($mode -eq '3') {\n")
	}
	dispatchEnd := ""
	if savePathTemplate != "" {
		dispatchEnd = "}"
	}

	replacer := strings.NewReplacer(
		"__GAME_TITLE__", psQuoted(gameTitle),
		"__SMB_HOST__", psQuoted(shareHostOf(shareRoot)),
		"__SMB_SHARE__", psQuoted(normalizeUNCPath(shareRoot)),
		"__SMB_USER__", psQuoted(s.cfg.SMBUsername),
		"__SMB_PASS__", psQuoted(s.cfg.SMBPassword),
		"__BASE_VHD__", psQuoted(baseVHDPath),
		"__DIFF_VHD__", psQuoted(buildDiffVHDPath(s.cfg.VHDDiffRoot, diffFileName)),
		"__SAVE_TEMPLATE__", psQuoted(savePathTemplate),
		"__MENU_OPTIONS__", menuOptions.String(),
		"__CLEANUP_COND__", cleanupCond.String(),
		"__MODE_DISPATCH__", dispatch.String(),
		"__MODE_DISPATCH_END__", dispatchEnd,
	)
	return replacer.Replace(tmpl)
}

func shareHostOf(shareRoot string) string {
	trimmed := strings.TrimPrefix(normalizeUNCPath(shareRoot), `\\`)
	parts := strings.Split(trimmed, `\`)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

// psQuoted 用 PS 单引号包裹并转义内嵌单引号（”）。
func psQuoted(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

// encodeUTF16LEWithBOM 将 UTF-8 字符串编码为带 BOM 的 UTF-16LE 字节，
// PS 5.1 读取此类 .ps1 时中文正常解码。
func encodeUTF16LEWithBOM(text string) []byte {
	units := utf16.Encode([]rune(text))
	buf := make([]byte, 2+len(units)*2)
	buf[0], buf[1] = 0xFF, 0xFE
	for i, u := range units {
		buf[2+i*2] = byte(u)
		buf[2+i*2+1] = byte(u >> 8)
	}
	return buf
}

func normalizeUNCPath(value string) string {
	trimmed := strings.TrimSpace(value)
	trimmed = strings.ReplaceAll(trimmed, "/", `\`)
	trimmed = strings.Trim(trimmed, `\`)
	if trimmed == "" {
		return `\\`
	}

	parts := strings.FieldsFunc(trimmed, func(r rune) bool {
		return r == '\\'
	})
	if len(parts) == 0 {
		return `\\`
	}

	return `\\` + strings.Join(parts, `\`)
}

var invalidBatchFileNameChars = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F]+`)

func sanitizeBatchFileName(value string) string {
	name := strings.TrimSpace(value)
	name = invalidBatchFileNameChars.ReplaceAllString(name, "-")
	name = strings.Join(strings.Fields(name), "-")
	name = strings.Trim(name, ".- ")
	return name
}

func buildDiffVHDPath(root string, fileName string) string {
	drive := normalizeDriveRoot(root)
	return drive + `\` + strings.TrimLeft(strings.TrimSpace(fileName), `\`)
}

func normalizeDriveRoot(root string) string {
	value := strings.TrimSpace(root)
	if len(value) >= 2 {
		letter := value[0]
		if ((letter >= 'A' && letter <= 'Z') || (letter >= 'a' && letter <= 'z')) && value[1] == ':' {
			return strings.ToUpper(string(letter)) + ":"
		}
	}
	return "C:"
}
