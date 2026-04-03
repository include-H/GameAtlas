package services

import (
	"bytes"
	"errors"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/hao/game/internal/config"
	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/files"
	"github.com/hao/game/internal/repositories"
)

var ErrInvalidLaunchFile = errors.New("launch script only supports VHD or VHDX files")
var ErrMissingSMBConfig = errors.New("SMB launch script configuration is incomplete")

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
		return "", "", ErrMissingSMBConfig
	}

	game, err := s.gamesRepo.GetByID(gameID)
	if err != nil {
		return "", "", normalizeRepoError(err)
	}
	if !includeAll && game.Visibility == domain.GameVisibilityPrivate {
		return "", "", ErrNotFound
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
		return "", "", ErrInvalidLaunchFile
	}

	baseVHDPath, shareRoot, err := s.buildSMBMountedPath(resolved.ResolvedPath)
	if err != nil {
		return "", "", err
	}

	diffFileName := filepath.Base(resolved.ResolvedPath)
	script := s.renderLaunchScript(game.ID, file.ID, shareRoot, baseVHDPath, diffFileName)
	filename := sanitizeBatchFileName(game.Title)
	if filename == "" {
		filename = "launch-game"
	}

	return script, filename + ".bat", nil
}

func (s *WindowsLaunchService) buildSMBMountedPath(resolvedPath string) (string, string, error) {
	if mappings, err := s.cfg.ParseSMBPathMappings(); err != nil {
		return "", "", err
	} else if len(mappings) > 0 {
		base, shareRoot, mappingErr := s.buildMappedSMBPath(resolvedPath, mappings)
		if mappingErr == nil {
			return base, shareRoot, nil
		}
		if !errors.Is(mappingErr, ErrForbiddenPath) {
			return "", "", mappingErr
		}
	}

	if strings.TrimSpace(s.cfg.SMBShareRoot) == "" {
		return "", "", ErrMissingSMBConfig
	}

	root := filepath.Clean(strings.TrimSpace(s.cfg.PrimaryROMRoot))
	if root == "" {
		root = "ROM"
	}

	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", "", normalizeFileError(err)
	}

	relative, err := filepath.Rel(resolvedRoot, resolvedPath)
	if err != nil {
		return "", "", ErrForbiddenPath
	}
	if relative == "." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || relative == ".." {
		return "", "", ErrForbiddenPath
	}

	relativeWindows := strings.ReplaceAll(filepath.ToSlash(relative), "/", `\`)
	if relativeWindows == "." {
		relativeWindows = ""
	}

	base := normalizeUNCPath(s.cfg.SMBShareRoot)
	if relativeWindows != "" {
		base += `\` + relativeWindows
	}

	return base, normalizeUNCPath(s.cfg.SMBShareRoot), nil
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
		return "", "", ErrForbiddenPath
	}

	return selectedBase, selectedShareRoot, nil
}

func (s *WindowsLaunchService) renderLaunchScript(gameID, fileID int64, shareRoot string, baseVHDPath string, diffFileName string) string {
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
	script.WriteString("set \"COLOR_INFO=%ESC%[96m\"\r\n")
	script.WriteString("set \"COLOR_WARN=%ESC%[93m\"\r\n")
	script.WriteString("set \"COLOR_ERROR=%ESC%[91m\"\r\n")
	script.WriteString("set \"COLOR_SUCCESS=%ESC%[92m\"\r\n")
	script.WriteString("set \"COLOR_RESET=%ESC%[0m\"\r\n")
	script.WriteString("\r\n")
	// The script carries SMB credentials because the current implementation is optimized for
	// personal/trusted environments where the share account is read-only. Treat this as an explicit
	// deployment constraint, not a generic multi-user safe default.
	script.WriteString(":: SMB 参数\r\n")
	script.WriteString("set \"SMB_HOST=" + escapeBatchValue(shareHost) + "\"\r\n")
	script.WriteString("set \"SMB_SHARE=" + escapeBatchValue(shareRoot) + "\"\r\n")
	script.WriteString("set \"SMB_USER=" + escapeBatchValue(s.cfg.SMBUsername) + "\"\r\n")
	script.WriteString("set \"SMB_PASS=" + escapeBatchValue(s.cfg.SMBPassword) + "\"\r\n")
	script.WriteString("set \"BASE_VHD=" + escapeBatchValue(baseVHDPath) + "\"\r\n")
	script.WriteString("set \"DIFF_VHD=" + escapeBatchValue(buildDiffVHDPath(s.cfg.VHDDiffRoot, diffFileName)) + "\"\r\n")
	script.WriteString("\r\n")
	script.WriteString(":: 当前配置\r\n")
	script.WriteString("echo SMB 主机: %SMB_HOST%\r\n")
	script.WriteString("echo SMB 共享路径: %SMB_SHARE%\r\n")
	script.WriteString("echo 基础 VHD 路径: %BASE_VHD%\r\n")
	script.WriteString("echo 差分 VHD 路径: %DIFF_VHD%\r\n")
	script.WriteString("\r\n")
	script.WriteString(":MENU\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_INFO%\" \"请选择操作:\"\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_INFO%\" \"  1. 挂载 SMB 并挂载游戏\"\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_INFO%\" \"  2. 删除 Windows 中刚刚添加的 SMB 凭据\"\r\n")
	script.WriteString("set /p \"ACTION=请输入选项 (1/2): \"\r\n")
	script.WriteString("if \"%ACTION%\"==\"1\" goto MOUNT_GAME\r\n")
	script.WriteString("if \"%ACTION%\"==\"2\" goto REMOVE_SMB_CREDENTIAL\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_ERROR%\" \"错误: 请输入 1 或 2。\"\r\n")
	script.WriteString("echo.\r\n")
	script.WriteString("goto MENU\r\n")
	script.WriteString("\r\n")
	script.WriteString(":MOUNT_GAME\r\n")
	script.WriteString(":: 连接 SMB 共享\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_WARN%\" \"正在为 %SMB_HOST% 添加凭据...\"\r\n")
	script.WriteString("cmdkey /add:%SMB_HOST% /user:%SMB_USER% /pass:%SMB_PASS% >nul 2>&1\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_WARN%\" \"正在连接 SMB 共享 %SMB_SHARE%...\"\r\n")
	script.WriteString("net use %SMB_SHARE% /user:%SMB_USER% %SMB_PASS% /persistent:no >nul\r\n")
	script.WriteString("if errorlevel 1 (\r\n")
	script.WriteString("  call :PRINT_COLOR \"%COLOR_ERROR%\" \"错误: SMB 共享连接失败。\"\r\n")
	script.WriteString("  pause\r\n")
	script.WriteString("  exit /b 1\r\n")
	script.WriteString(")\r\n")
	script.WriteString("\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_SUCCESS%\" \"SMB 共享连接成功。\"\r\n")
	script.WriteString("\r\n")
	script.WriteString(":: 生成 DiskPart 脚本\r\n")
	script.WriteString("set \"DISKPART_SCRIPT=%TEMP%\\mount-game-" + strconv.FormatInt(gameID, 10) + "-" + strconv.FormatInt(fileID, 10) + ".txt\"\r\n")
	script.WriteString("if not exist \"%DIFF_VHD%\" (\r\n")
	script.WriteString("  call :PRINT_COLOR \"%COLOR_WARN%\" \"正在创建差分 VHD: %DIFF_VHD%\"\r\n")
	script.WriteString("  >\"%DISKPART_SCRIPT%\" echo create vdisk file=\"%DIFF_VHD%\" parent=\"%BASE_VHD%\"\r\n")
	script.WriteString("  >>\"%DISKPART_SCRIPT%\" echo select vdisk file=\"%DIFF_VHD%\"\r\n")
	script.WriteString("  >>\"%DISKPART_SCRIPT%\" echo attach vdisk\r\n")
	script.WriteString(") else (\r\n")
	script.WriteString("  call :PRINT_COLOR \"%COLOR_WARN%\" \"差分 VHD 已存在，准备挂载: %DIFF_VHD%\"\r\n")
	script.WriteString("  >\"%DISKPART_SCRIPT%\" echo select vdisk file=\"%DIFF_VHD%\"\r\n")
	script.WriteString("  >>\"%DISKPART_SCRIPT%\" echo attach vdisk\r\n")
	script.WriteString(")\r\n")
	script.WriteString("\r\n")
	script.WriteString(":: 执行挂载\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_WARN%\" \"正在挂载 VHD...\"\r\n")
	script.WriteString("diskpart /s \"%DISKPART_SCRIPT%\"\r\n")
	script.WriteString("set \"ERR=%ERRORLEVEL%\"\r\n")
	script.WriteString("\r\n")
	script.WriteString(":: 输出结果\r\n")
	script.WriteString("echo DiskPart 执行完毕，错误码: %ERR%\r\n")
	script.WriteString("del /q \"%DISKPART_SCRIPT%\" >nul 2>&1\r\n")
	script.WriteString("if not \"%ERR%\"==\"0\" (\r\n")
	script.WriteString("  call :PRINT_COLOR \"%COLOR_ERROR%\" \"错误: VHD 挂载失败，错误码 %ERR%。\"\r\n")
	script.WriteString("  pause\r\n")
	script.WriteString("  exit /b %ERR%\r\n")
	script.WriteString(")\r\n")
	script.WriteString("\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_SUCCESS%\" \"差分 VHD 已挂载成功。\"\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_SUCCESS%\" \"请打开此电脑找到盘符进行游玩。\"\r\n")
	script.WriteString("goto END\r\n")
	script.WriteString("\r\n")
	script.WriteString(":REMOVE_SMB_CREDENTIAL\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_WARN%\" \"正在断开 SMB 共享 %SMB_SHARE%...\"\r\n")
	script.WriteString("net use %SMB_SHARE% /delete /y >nul 2>&1\r\n")
	script.WriteString("if errorlevel 1 (\r\n")
	script.WriteString("  call :PRINT_COLOR \"%COLOR_WARN%\" \"提示: 当前没有活动的 SMB 共享连接，或断开失败。\"\r\n")
	script.WriteString(") else (\r\n")
	script.WriteString("  call :PRINT_COLOR \"%COLOR_SUCCESS%\" \"SMB 共享连接已断开。\"\r\n")
	script.WriteString(")\r\n")
	script.WriteString("call :PRINT_COLOR \"%COLOR_WARN%\" \"正在删除 %SMB_HOST% 的已保存凭据...\"\r\n")
	script.WriteString("cmdkey /delete:%SMB_HOST% >nul 2>&1\r\n")
	script.WriteString("if errorlevel 1 (\r\n")
	script.WriteString("  call :PRINT_COLOR \"%COLOR_WARN%\" \"提示: 没有找到 %SMB_HOST% 的已保存凭据，或删除失败。\"\r\n")
	script.WriteString(") else (\r\n")
	script.WriteString("  call :PRINT_COLOR \"%COLOR_SUCCESS%\" \"已删除 %SMB_HOST% 的已保存凭据。\"\r\n")
	script.WriteString(")\r\n")
	script.WriteString("goto END\r\n")
	script.WriteString("\r\n")
	script.WriteString(":PRINT_COLOR\r\n")
	script.WriteString("echo %~1%~2%COLOR_RESET%\r\n")
	script.WriteString("exit /b\r\n")
	script.WriteString("\r\n")
	script.WriteString(":END\r\n")
	script.WriteString("pause\r\n")
	script.WriteString("endlocal\r\n")

	return script.String()
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
