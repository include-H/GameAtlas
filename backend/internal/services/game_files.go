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

var ErrForbiddenPath = errors.New("file path is outside primary ROM root")
var ErrMissingFile = errors.New("registered file is unavailable")
var ErrInvalidFile = errors.New("registered path is not a file")
var ErrMissingConfig = errors.New("PRIMARY_ROM_ROOT is not configured")
var ErrInvalidLaunchFile = errors.New("launch script only supports VHD or VHDX files")
var ErrMissingSMBConfig = errors.New("SMB launch script configuration is incomplete")

type GameFilesService struct {
	gamesRepo     *repositories.GamesRepository
	gameFilesRepo *repositories.GameFilesRepository
	fileGuard     *files.Guard
	cfg           config.Config
}

type DownloadFile struct {
	GameID       int64
	FileID       int64
	ResolvedPath string
	SizeBytes    int64
	ModTime      int64
}

func NewGameFilesService(cfg config.Config, gamesRepo *repositories.GamesRepository, gameFilesRepo *repositories.GameFilesRepository) *GameFilesService {
	return &GameFilesService{
		gamesRepo:     gamesRepo,
		gameFilesRepo: gameFilesRepo,
		fileGuard:     files.NewGuard(cfg.PrimaryROMRoot),
		cfg:           cfg,
	}
}

func (s *GameFilesService) List(gameID int64, includeAll bool) ([]domain.GameFile, error) {
	game, err := s.gamesRepo.GetByID(gameID)
	if err != nil {
		return nil, normalizeRepoError(err)
	}
	if !includeAll && game.Visibility == domain.GameVisibilityPrivate {
		return nil, ErrNotFound
	}
	files, err := s.gameFilesRepo.ListByGameID(gameID)
	if err != nil {
		return nil, err
	}
	if files == nil {
		return []domain.GameFile{}, nil
	}
	return files, nil
}

func (s *GameFilesService) Create(gameID int64, input domain.GameFileWriteInput) (*domain.GameFile, error) {
	if _, err := s.gamesRepo.GetByID(gameID); err != nil {
		return nil, normalizeRepoError(err)
	}
	if err := validateGameFileInput(input); err != nil {
		return nil, err
	}
	input = trimGameFileInput(input)
	resolved, err := s.fileGuard.ValidateFile(input.FilePath)
	if err != nil {
		return nil, normalizeFileError(err)
	}
	input.FilePath = resolved.ResolvedPath

	file, err := s.gameFilesRepo.Create(gameID, input)
	if err != nil {
		return nil, err
	}

	file.SizeBytes = &resolved.SizeBytes
	if err := s.gameFilesRepo.UpdateSizeBytes(gameID, file.ID, resolved.SizeBytes); err != nil {
		return nil, err
	}

	return file, nil
}

func (s *GameFilesService) Update(gameID, fileID int64, input domain.GameFileWriteInput) (*domain.GameFile, error) {
	if _, err := s.gamesRepo.GetByID(gameID); err != nil {
		return nil, normalizeRepoError(err)
	}
	if err := validateGameFileInput(input); err != nil {
		return nil, err
	}
	input = trimGameFileInput(input)
	resolved, err := s.fileGuard.ValidateFile(input.FilePath)
	if err != nil {
		return nil, normalizeFileError(err)
	}
	input.FilePath = resolved.ResolvedPath

	file, err := s.gameFilesRepo.Update(gameID, fileID, input)
	if err != nil {
		return nil, normalizeRepoError(err)
	}

	file.SizeBytes = &resolved.SizeBytes
	if err := s.gameFilesRepo.UpdateSizeBytes(gameID, fileID, resolved.SizeBytes); err != nil {
		return nil, err
	}

	return file, nil
}

func (s *GameFilesService) Delete(gameID, fileID int64) error {
	if _, err := s.gamesRepo.GetByID(gameID); err != nil {
		return normalizeRepoError(err)
	}
	deleted, err := s.gameFilesRepo.Delete(gameID, fileID)
	if err != nil {
		return err
	}
	if !deleted {
		return ErrNotFound
	}
	return nil
}

func (s *GameFilesService) GetDownloadFile(gameID, fileID int64, includeAll bool) (*DownloadFile, error) {
	game, err := s.gamesRepo.GetByID(gameID)
	if err != nil {
		return nil, normalizeRepoError(err)
	}
	if !includeAll && game.Visibility == domain.GameVisibilityPrivate {
		return nil, ErrNotFound
	}

	file, err := s.gameFilesRepo.GetByID(gameID, fileID)
	if err != nil {
		return nil, normalizeRepoError(err)
	}

	resolved, err := s.fileGuard.ValidateFile(file.FilePath)
	if err != nil {
		return nil, normalizeFileError(err)
	}

	return &DownloadFile{
		GameID:       gameID,
		FileID:       fileID,
		ResolvedPath: resolved.ResolvedPath,
		SizeBytes:    resolved.SizeBytes,
		ModTime:      resolved.ModTime,
	}, nil
}

func (s *GameFilesService) RecordDownload(gameID, fileID int64, includeAll bool) error {
	game, err := s.gamesRepo.GetByID(gameID)
	if err != nil {
		return normalizeRepoError(err)
	}
	if !includeAll && game.Visibility == domain.GameVisibilityPrivate {
		return ErrNotFound
	}

	if _, err := s.gameFilesRepo.GetByID(gameID, fileID); err != nil {
		return normalizeRepoError(err)
	}

	return s.gamesRepo.IncrementDownloads(gameID)
}

func (s *GameFilesService) BuildLaunchScript(gameID, fileID int64, includeAll bool) (string, string, error) {
	if strings.TrimSpace(s.cfg.SMBShareRoot) == "" ||
		strings.TrimSpace(s.cfg.SMBUsername) == "" ||
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

	baseVHDPath, err := s.buildSMBMountedPath(resolved.ResolvedPath)
	if err != nil {
		return "", "", err
	}

	diffFileName := filepath.Base(resolved.ResolvedPath)
	script := s.renderLaunchScript(game.ID, file.ID, baseVHDPath, diffFileName)
	filename := sanitizeBatchFileName(game.Title)
	if filename == "" {
		filename = "launch-game"
	}

	return script, filename + ".bat", nil
}

func validateGameFileInput(input domain.GameFileWriteInput) error {
	if strings.TrimSpace(input.FilePath) == "" {
		return ErrValidation
	}
	return nil
}

func trimGameFileInput(input domain.GameFileWriteInput) domain.GameFileWriteInput {
	input.FilePath = strings.TrimSpace(input.FilePath)
	input.Label = trimStringPtr(input.Label)
	input.Notes = trimStringPtr(input.Notes)
	return input
}

func normalizeFileError(err error) error {
	switch {
	case errors.Is(err, files.ErrPathOutsideRoot):
		return ErrForbiddenPath
	case errors.Is(err, files.ErrFileNotFound):
		return ErrMissingFile
	case errors.Is(err, files.ErrNotAFile):
		return ErrInvalidFile
	case errors.Is(err, files.ErrNoPrimaryRoot):
		return ErrMissingConfig
	default:
		return err
	}
}

func (s *GameFilesService) buildSMBMountedPath(resolvedPath string) (string, error) {
	root := filepath.Clean(strings.TrimSpace(s.cfg.PrimaryROMRoot))
	if root == "" {
		root = "ROM"
	}

	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", normalizeFileError(err)
	}

	relative, err := filepath.Rel(resolvedRoot, resolvedPath)
	if err != nil {
		return "", ErrForbiddenPath
	}
	if relative == "." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || relative == ".." {
		return "", ErrForbiddenPath
	}

	relativeWindows := strings.ReplaceAll(filepath.ToSlash(relative), "/", `\`)
	if relativeWindows == "." {
		relativeWindows = ""
	}

	base := normalizeUNCPath(s.cfg.SMBShareRoot)
	if relativeWindows != "" {
		base += `\` + relativeWindows
	}

	return base, nil
}

func (s *GameFilesService) renderLaunchScript(gameID, fileID int64, baseVHDPath string, diffFileName string) string {
	shareRoot := normalizeUNCPath(s.cfg.SMBShareRoot)
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
	script.WriteString(":: 连接 SMB 共享\r\n")
	script.WriteString("echo 正在为 %SMB_HOST% 添加凭据...\r\n")
	script.WriteString("cmdkey /add:%SMB_HOST% /user:%SMB_USER% /pass:%SMB_PASS% >nul 2>&1\r\n")
	script.WriteString("echo 正在连接 SMB 共享 %SMB_SHARE%...\r\n")
	script.WriteString("net use %SMB_SHARE% /user:%SMB_USER% %SMB_PASS% /persistent:no >nul\r\n")
	script.WriteString("if errorlevel 1 (\r\n")
	script.WriteString("  echo 错误: SMB 共享连接失败。\r\n")
	script.WriteString("  pause\r\n")
	script.WriteString("  exit /b 1\r\n")
	script.WriteString(")\r\n")
	script.WriteString("\r\n")
	script.WriteString("echo SMB 共享连接成功。\r\n")
	script.WriteString("\r\n")
	script.WriteString(":: 生成 DiskPart 脚本\r\n")
	script.WriteString("set \"DISKPART_SCRIPT=%TEMP%\\mount-game-" + strconv.FormatInt(gameID, 10) + "-" + strconv.FormatInt(fileID, 10) + ".txt\"\r\n")
	script.WriteString("if not exist \"%DIFF_VHD%\" (\r\n")
	script.WriteString("  echo 正在创建差分 VHD: %DIFF_VHD%\r\n")
	script.WriteString("  >\"%DISKPART_SCRIPT%\" echo create vdisk file=\"%DIFF_VHD%\" parent=\"%BASE_VHD%\"\r\n")
	script.WriteString("  >>\"%DISKPART_SCRIPT%\" echo select vdisk file=\"%DIFF_VHD%\"\r\n")
	script.WriteString("  >>\"%DISKPART_SCRIPT%\" echo attach vdisk\r\n")
	script.WriteString(") else (\r\n")
	script.WriteString("  echo 差分 VHD 已存在，准备挂载: %DIFF_VHD%\r\n")
	script.WriteString("  >\"%DISKPART_SCRIPT%\" echo select vdisk file=\"%DIFF_VHD%\"\r\n")
	script.WriteString("  >>\"%DISKPART_SCRIPT%\" echo attach vdisk\r\n")
	script.WriteString(")\r\n")
	script.WriteString("\r\n")
	script.WriteString(":: 显示 DiskPart 脚本\r\n")
	script.WriteString("echo 将执行以下 DiskPart 脚本:\r\n")
	script.WriteString("type \"%DISKPART_SCRIPT%\"\r\n")
	script.WriteString("\r\n")
	script.WriteString(":: 执行挂载\r\n")
	script.WriteString("echo 正在挂载 VHD...\r\n")
	script.WriteString("diskpart /s \"%DISKPART_SCRIPT%\"\r\n")
	script.WriteString("set \"ERR=%ERRORLEVEL%\"\r\n")
	script.WriteString("\r\n")
	script.WriteString(":: 输出结果\r\n")
	script.WriteString("echo DiskPart 执行完毕，错误码: %ERR%\r\n")
	script.WriteString("del /q \"%DISKPART_SCRIPT%\" >nul 2>&1\r\n")
	script.WriteString("if not \"%ERR%\"==\"0\" (\r\n")
	script.WriteString("  echo 错误: VHD 挂载失败，错误码 %ERR%。\r\n")
	script.WriteString("  pause\r\n")
	script.WriteString("  exit /b %ERR%\r\n")
	script.WriteString(")\r\n")
	script.WriteString("\r\n")
	script.WriteString("echo 差分 VHD 已挂载成功。\r\n")
	script.WriteString("echo 请打开此电脑找到盘符进行游玩。\r\n")
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
