package services

import (
	"bytes"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

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
	script := s.renderLaunchScript(game.ID, file.ID, title, shareRoot, baseVHDPath, diffFileName)
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

const (
	launchBannerWidth = 52 // 横幅总宽度（显示列）
	launchBannerText  = launchBannerWidth - 6
)

func (s *WindowsLaunchService) renderLaunchScript(gameID, fileID int64, gameTitle, shareRoot, baseVHDPath, diffFileName string) string {
	// 游戏标题会出现在 set / title / 彩色输出行中，双引号会破坏批处理引号配对，统一替换为全角引号。
	gameTitle = strings.ReplaceAll(gameTitle, `"`, "＂")
	shareRoot = normalizeUNCPath(shareRoot)
	shareHost := extractSMBHost(shareRoot)

	var script bytes.Buffer
	script.WriteString("@echo off\r\n")
	script.WriteString("chcp 936 >nul\r\n")
	script.WriteString("setlocal\r\n")
	script.WriteString("\r\n")
	script.WriteString(":: 检查管理员权限\r\n")
	script.WriteString("fltmc >nul 2>&1\r\n")
	script.WriteString("if errorlevel 1 (\r\n")
	script.WriteString("  echo 正在请求管理员权限...\r\n")
	script.WriteString("  powershell -NoProfile -ExecutionPolicy Bypass -Command \"Start-Process -FilePath '%~f0' -Verb RunAs\"\r\n")
	script.WriteString("  exit /b\r\n")
	script.WriteString(")\r\n")
	script.WriteString("\r\n")
	script.WriteString(":: 初始化颜色\r\n")
	script.WriteString("for /f %%a in ('echo prompt $E ^| cmd') do set \"ESC=%%a\"\r\n")
	script.WriteString("set \"COLOR_TITLE=%ESC%[96m\"\r\n")
	script.WriteString("set \"COLOR_INFO=%ESC%[96m\"\r\n")
	script.WriteString("set \"COLOR_WARN=%ESC%[93m\"\r\n")
	script.WriteString("set \"COLOR_ERROR=%ESC%[91m\"\r\n")
	script.WriteString("set \"COLOR_SUCCESS=%ESC%[92m\"\r\n")
	script.WriteString("set \"COLOR_DIM=%ESC%[90m\"\r\n")
	script.WriteString("set \"COLOR_RESET=%ESC%[0m\"\r\n")
	script.WriteString("\r\n")
	// The script carries SMB credentials because the current implementation is optimized for
	// personal/trusted environments where the share account is read-only. Treat this as an explicit
	// deployment constraint, not a generic multi-user safe default.
	script.WriteString(":: SMB 参数\r\n")
	script.WriteString("set \"GAME_TITLE=" + escapeBatchValue(gameTitle) + "\"\r\n")
	script.WriteString("set \"SMB_HOST=" + escapeBatchValue(shareHost) + "\"\r\n")
	script.WriteString("set \"SMB_SHARE=" + escapeBatchValue(shareRoot) + "\"\r\n")
	script.WriteString("set \"SMB_USER=" + escapeBatchValue(s.cfg.SMBUsername) + "\"\r\n")
	script.WriteString("set \"SMB_PASS=" + escapeBatchValue(s.cfg.SMBPassword) + "\"\r\n")
	script.WriteString("set \"BASE_VHD=" + escapeBatchValue(baseVHDPath) + "\"\r\n")
	script.WriteString("set \"DIFF_VHD=" + escapeBatchValue(buildDiffVHDPath(s.cfg.VHDDiffRoot, diffFileName)) + "\"\r\n")
	script.WriteString("set \"DIFF_NAME=" + escapeBatchValue(filepath.Base(diffFileName)) + "\"\r\n")
	script.WriteString("title GameAtlas · %GAME_TITLE%\r\n")
	script.WriteString("\r\n")
	script.WriteString(":: 标题横幅\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_TITLE%\" \"" + "┌" + strings.Repeat("─", launchBannerWidth-2) + "┐" + "\"\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_TITLE%\" \"" + bannerContentLine("GameAtlas · VHD 远程启动") + "\"\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_TITLE%\" \"" + bannerContentLine("") + "\"\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_TITLE%\" \"" + bannerContentLine("《"+clampDisplayText(gameTitle, launchBannerText-4)+"》") + "\"\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_TITLE%\" \"" + "└" + strings.Repeat("─", launchBannerWidth-2) + "┘" + "\"\r\n")
	script.WriteString("\r\n")
	script.WriteString(":: 当前配置\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_DIM%\" \"" + menuSeparator() + "\"\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_INFO%\" \"  当前配置\"\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_DIM%\" \"  SMB 主机　　：%SMB_HOST%\"\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_DIM%\" \"  SMB 共享　　：%SMB_SHARE%\"\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_DIM%\" \"  基础 VHD　　：%BASE_VHD%\"\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_DIM%\" \"  差分 VHD　　：%DIFF_VHD%\"\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_DIM%\" \"" + menuSeparator() + "\"\r\n")
	script.WriteString("\r\n")
	script.WriteString(":MENU\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_INFO%\" \"  请选择操作：\"\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_INFO%\" \"    [1] 挂载游戏并游玩（结束后自动卸载并清理凭据）\"\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_INFO%\" \"    [2] 仅挂载（保留连接与凭据）\"\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_INFO%\" \"    [3] 清理 SMB 凭据并断开共享\"\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_DIM%\" \"" + menuSeparator() + "\"\r\n")
	script.WriteString("choice /c 123 /n /m \"  请输入选项 [1/2/3]： \"\r\n")
	script.WriteString("if errorlevel 3 goto REMOVE_SMB_CREDENTIAL\r\n")
	script.WriteString("if errorlevel 2 goto MOUNT_ONLY\r\n")
	script.WriteString("goto MOUNT_PLAY\r\n")
	script.WriteString("\r\n")
	script.WriteString(":MOUNT_PLAY\r\n")
	script.WriteString("set \"AUTO_CLEANUP=1\"\r\n")
	script.WriteString("goto DO_MOUNT\r\n")
	script.WriteString("\r\n")
	script.WriteString(":MOUNT_ONLY\r\n")
	script.WriteString("set \"AUTO_CLEANUP=0\"\r\n")
	script.WriteString("goto DO_MOUNT\r\n")
	script.WriteString("\r\n")
	script.WriteString(":DO_MOUNT\r\n")
	script.WriteString(":: 连接 SMB 共享\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_WARN%\" \"  [1/4] 添加 SMB 凭据...\"\r\n")
	script.WriteString("cmdkey /add:%SMB_HOST% /user:%SMB_USER% /pass:%SMB_PASS% >nul 2>&1\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_WARN%\" \"  [2/4] 连接 SMB 共享 %SMB_SHARE%...\"\r\n")
	script.WriteString("net use %SMB_SHARE% /user:%SMB_USER% %SMB_PASS% /persistent:no >nul\r\n")
	script.WriteString("if errorlevel 1 (\r\n")
	script.WriteString("  call :PRINT_COLOR \"%COLOR_ERROR%\" \"  [×] 错误：SMB 共享连接失败。\"\r\n")
	script.WriteString("  call :PRINT_COLOR \"%COLOR_ERROR%\" \"  凭据可能已残留，可再次运行本脚本选择 [3] 清理。\"\r\n")
	script.WriteString("  pause\r\n")
	script.WriteString("  exit /b 1\r\n")
	script.WriteString(")\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_SUCCESS%\" \"  [√] SMB 共享连接成功。\"\r\n")
	script.WriteString("\r\n")
	script.WriteString(":: 生成 DiskPart 脚本\r\n")
	script.WriteString("set \"DISKPART_SCRIPT=%TEMP%\\mount-game-" + strconv.FormatInt(gameID, 10) + "-" + strconv.FormatInt(fileID, 10) + ".txt\"\r\n")
	script.WriteString("if not exist \"%DIFF_VHD%\" (\r\n")
	script.WriteString("  call :PRINT_COLOR \"%COLOR_WARN%\" \"  [3/4] 创建差分 VHD：%DIFF_VHD%\"\r\n")
	script.WriteString("  >\"%DISKPART_SCRIPT%\" echo create vdisk file=\"%DIFF_VHD%\" parent=\"%BASE_VHD%\"\r\n")
	script.WriteString("  >>\"%DISKPART_SCRIPT%\" echo select vdisk file=\"%DIFF_VHD%\"\r\n")
	script.WriteString("  >>\"%DISKPART_SCRIPT%\" echo attach vdisk\r\n")
	script.WriteString(") else (\r\n")
	script.WriteString("  call :PRINT_COLOR \"%COLOR_WARN%\" \"  [3/4] 差分 VHD 已存在，准备挂载：%DIFF_VHD%\"\r\n")
	script.WriteString("  >\"%DISKPART_SCRIPT%\" echo select vdisk file=\"%DIFF_VHD%\"\r\n")
	script.WriteString("  >>\"%DISKPART_SCRIPT%\" echo attach vdisk\r\n")
	script.WriteString(")\r\n")
	script.WriteString("\r\n")
	script.WriteString(":: 执行挂载\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_WARN%\" \"  [4/4] 正在挂载 VHD...\"\r\n")
	script.WriteString("diskpart /s \"%DISKPART_SCRIPT%\"\r\n")
	script.WriteString("set \"ERR=%ERRORLEVEL%\"\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_DIM%\" \"  DiskPart 执行完毕，错误码：%ERR%\"\r\n")
	script.WriteString("del /q \"%DISKPART_SCRIPT%\" >nul 2>&1\r\n")
	script.WriteString("if not \"%ERR%\"==\"0\" (\r\n")
	script.WriteString("  call :PRINT_COLOR \"%COLOR_ERROR%\" \"  [×] 错误：VHD 挂载失败，错误码 %ERR%。\"\r\n")
	script.WriteString("  call :PRINT_COLOR \"%COLOR_ERROR%\" \"  凭据可能已残留，可再次运行本脚本选择 [3] 清理。\"\r\n")
	script.WriteString("  pause\r\n")
	script.WriteString("  exit /b %ERR%\r\n")
	script.WriteString(")\r\n")
	script.WriteString("\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_SUCCESS%\" \"  [√] 差分 VHD 已挂载成功。\"\r\n")
	script.WriteString("\r\n")
	script.WriteString(":: 查找游戏盘符（尽力而为，失败不影响游玩）\r\n")
	script.WriteString("set \"GAME_DRIVE=\"\r\n")
	script.WriteString(`for /f "delims=" %%d in ('powershell -NoProfile -Command "(Get-Disk ^| Where-Object { $_.Path ^| Select-String -SimpleMatch $env:DIFF_NAME } ^| Get-Partition ^| Where-Object { $_.DriveLetter } ^| Select-Object -ExpandProperty DriveLetter)"') do set "GAME_DRIVE=%%d"` + "\r\n")
	script.WriteString("if defined GAME_DRIVE (\r\n")
	script.WriteString("  call :PRINT_COLOR \"%COLOR_INFO%\" \"  游戏盘符：%GAME_DRIVE%:\"\r\n")
	script.WriteString(") else (\r\n")
	script.WriteString("  call :PRINT_COLOR \"%COLOR_INFO%\" \"  请打开“此电脑”找到新出现的盘符进入游戏。\"\r\n")
	script.WriteString(")\r\n")
	script.WriteString("if \"%AUTO_CLEANUP%\"==\"1\" goto WAIT_PLAY\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_INFO%\" \"  已保持挂载，SMB 凭据与共享连接保留。\"\r\n")
	script.WriteString("goto END\r\n")
	script.WriteString("\r\n")
	script.WriteString(":WAIT_PLAY\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_WARN%\" \"  开始游玩吧！\"\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_WARN%\" \"  游玩结束后回到本窗口按任意键。\"\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_WARN%\" \"  脚本将自动卸载 VHD 并清理 SMB 凭据。\"\r\n")
	script.WriteString("pause >nul\r\n")
	script.WriteString("\r\n")
	script.WriteString(":: 卸载 VHD\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_WARN%\" \"  [1/3] 正在卸载 VHD...\"\r\n")
	script.WriteString(">\"%DISKPART_SCRIPT%\" echo select vdisk file=\"%DIFF_VHD%\"\r\n")
	script.WriteString(">>\"%DISKPART_SCRIPT%\" echo detach vdisk\r\n")
	script.WriteString("diskpart /s \"%DISKPART_SCRIPT%\" >nul\r\n")
	script.WriteString("set \"ERR=%ERRORLEVEL%\"\r\n")
	script.WriteString("del /q \"%DISKPART_SCRIPT%\" >nul 2>&1\r\n")
	script.WriteString("if not \"%ERR%\"==\"0\" (\r\n")
	script.WriteString("  call :PRINT_COLOR \"%COLOR_ERROR%\" \"  [×] 错误：VHD 卸载失败（错误码 %ERR%）。\"\r\n")
	script.WriteString("  call :PRINT_COLOR \"%COLOR_ERROR%\" \"  请手动卸载 %DIFF_VHD% 后，再运行本脚本清理凭据。\"\r\n")
	script.WriteString(") else (\r\n")
	script.WriteString("  call :PRINT_COLOR \"%COLOR_SUCCESS%\" \"  [√] VHD 已卸载，差分盘已保留（下次启动直接复用）。\"\r\n")
	script.WriteString(")\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_WARN%\" \"  [2/3] 正在断开 SMB 共享...\"\r\n")
	script.WriteString("net use %SMB_SHARE% /delete /y >nul 2>&1\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_WARN%\" \"  [3/3] 正在删除 SMB 凭据...\"\r\n")
	script.WriteString("cmdkey /delete:%SMB_HOST% >nul 2>&1\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_SUCCESS%\" \"  [√] 清理完成，可以安全关闭窗口。\"\r\n")
	script.WriteString("goto END\r\n")
	script.WriteString("\r\n")
	script.WriteString(":REMOVE_SMB_CREDENTIAL\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_WARN%\" \"  [1/2] 正在断开 SMB 共享 %SMB_SHARE%...\"\r\n")
	script.WriteString("net use %SMB_SHARE% /delete /y >nul 2>&1\r\n")
	script.WriteString("if errorlevel 1 (\r\n")
	script.WriteString("  call :PRINT_COLOR \"%COLOR_WARN%\" \"  提示：当前没有活动的 SMB 共享连接，或断开失败。\"\r\n")
	script.WriteString(") else (\r\n")
	script.WriteString("  call :PRINT_COLOR \"%COLOR_SUCCESS%\" \"  [√] SMB 共享连接已断开。\"\r\n")
	script.WriteString(")\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_WARN%\" \"  [2/2] 正在删除 %SMB_HOST% 的已保存凭据...\"\r\n")
	script.WriteString("cmdkey /delete:%SMB_HOST% >nul 2>&1\r\n")
	script.WriteString("if errorlevel 1 (\r\n")
	script.WriteString("  call :PRINT_COLOR \"%COLOR_WARN%\" \"  提示：没有找到 %SMB_HOST% 的已保存凭据，或删除失败。\"\r\n")
	script.WriteString(") else (\r\n")
	script.WriteString("  call :PRINT_COLOR \"%COLOR_SUCCESS%\" \"  [√] 已删除 %SMB_HOST% 的已保存凭据。\"\r\n")
	script.WriteString(")\r\n")
	script.WriteString("goto END\r\n")
	script.WriteString("\r\n")
	script.WriteString(":PRINT_COLOR\r\n")
	script.WriteString("echo %~1%~2%COLOR_RESET%\r\n")
	script.WriteString("exit /b\r\n")
	script.WriteString("\r\n")
	script.WriteString(":END\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_DIM%\" \"" + menuSeparator() + "\"\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_SUCCESS%\" \"  操作完成，感谢使用 GameAtlas！\"\r\n")
	script.WriteString("pause\r\n")
	script.WriteString("endlocal\r\n")

	return script.String()
}

func bannerContentLine(text string) string {
	text = clampDisplayText(text, launchBannerText)
	width := displayWidth(text)
	left := (launchBannerText - width) / 2
	right := launchBannerText - width - left
	return "│  " + strings.Repeat(" ", left) + text + strings.Repeat(" ", right) + "  │"
}

func menuSeparator() string {
	return "  " + strings.Repeat("─", 38)
}

func clampDisplayText(text string, maxWidth int) string {
	if displayWidth(text) <= maxWidth {
		return text
	}
	width := 0
	var b strings.Builder
	for _, r := range text {
		w := runeDisplayWidth(r)
		if width+w > maxWidth-2 {
			break
		}
		b.WriteRune(r)
		width += w
	}
	return b.String() + "…"
}

func displayWidth(text string) int {
	width := 0
	for _, r := range text {
		width += runeDisplayWidth(r)
	}
	return width
}

func runeDisplayWidth(r rune) int {
	switch {
	case r == 0x00B7 || r == 0x2014 || r == 0x2018 || r == 0x2019 || r == 0x201C || r == 0x201D || r == 0x2026:
		return 2
	case r >= 0x1100 && r <= 0x115F:
		return 2
	case r >= 0x2E80 && r <= 0xA4CF && r != 0x303F:
		return 2
	case r >= 0xAC00 && r <= 0xD7A3:
		return 2
	case r >= 0xF900 && r <= 0xFAFF:
		return 2
	case r >= 0xFE30 && r <= 0xFE4F:
		return 2
	case r >= 0xFF00 && r <= 0xFF60:
		return 2
	case r >= 0xFFE0 && r <= 0xFFE6:
		return 2
	default:
		return 1
	}
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

func extractSMBHost(shareRoot string) string {
	trimmed := strings.TrimPrefix(normalizeUNCPath(shareRoot), `\\`)
	parts := strings.Split(trimmed, `\`)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

var invalidBatchFileNameChars = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F]+`)

func sanitizeBatchFileName(value string) string {
	name := strings.TrimSpace(value)
	name = invalidBatchFileNameChars.ReplaceAllString(name, "-")
	name = strings.Join(strings.Fields(name), "-")
	name = strings.Trim(name, ".- ")
	return name
}

func escapeBatchValue(value string) string {
	replacer := strings.NewReplacer("^", "^^", "&", "^&", "|", "^|", "<", "^<", ">", "^>", "%", "%%")
	return replacer.Replace(value)
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
