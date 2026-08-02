package services

import (
	"bufio"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hao/game/internal/config"
)

type SettingsService struct {
	cfg     config.Config
	dataDir string
	envPath string
}

func NewSettingsService(cfg config.Config) *SettingsService {
	dataDir := filepath.Dir(cfg.AssetsDir)

	envPath := filepath.Join(dataDir, ".env")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		envPath = ".env"
	}

	return &SettingsService{
		cfg:     cfg,
		dataDir: dataDir,
		envPath: envPath,
	}
}

type EnvEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Label string `json:"label"`
	Group string `json:"group"`
}

var editableEnvKeys = map[string]string{
	"ADMIN_PASSWORD":      "管理员密码",
	"ADMIN_DISPLAY_NAME":  "管理员显示名称",
	"PRIMARY_ROM_ROOT":    "ROM 根目录",
	"VHD_DIFF_ROOT":       "VHD 差分盘根路径",
	"PROXY":               "出站代理",
	"STEAMGRIDDB_API_KEY": "SteamGridDB API Key",
	"WIKI_HISTORY_LIMIT":  "Wiki 历史记录上限",
	"AUTH_MAX_FAILS":      "登录失败次数限制",
	"AUTH_COOLDOWN":       "限制冷却时间",
	"AUTH_FAIL_WINDOW":    "失败计数时间窗口",
	"AUTH_STATE_TTL":      "登录会话有效期",
	"AUTH_TRACK_BY":       "失败追踪方式",
}

var smbEnvKeys = map[string]string{
	"SMB_PATH_MAPPINGS": "SMB 路径映射",
	"SMB_USERNAME":      "SMB 用户名",
	"SMB_PASSWORD":      "SMB 密码",
}

var allEditableKeys = buildEditableKeysMap()

func buildEditableKeysMap() map[string]struct{} {
	m := make(map[string]struct{}, len(editableEnvKeys)+len(smbEnvKeys))
	for k := range editableEnvKeys {
		m[k] = struct{}{}
	}
	for k := range smbEnvKeys {
		m[k] = struct{}{}
	}
	return m
}

func envLabelAndGroup(key string) (label, group string) {
	if l, ok := editableEnvKeys[key]; ok {
		return l, "general"
	}
	if l, ok := smbEnvKeys[key]; ok {
		return l, "smb"
	}
	return key, "general"
}

func (s *SettingsService) GetConfig() ([]EnvEntry, error) {
	file, err := os.Open(s.envPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []EnvEntry{}, nil
		}
		return nil, fmt.Errorf("打开配置文件失败: %w", err)
	}
	defer file.Close()

	type entry struct {
		key       string
		value     string
		lineIndex int
	}

	var entries []entry
	scanner := bufio.NewScanner(file)
	lineIndex := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		lineIndex++

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])

		if _, ok := allEditableKeys[key]; !ok {
			continue
		}

		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, `"'`)

		entries = append(entries, entry{key: key, value: value, lineIndex: lineIndex})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].lineIndex < entries[j].lineIndex
	})

	result := make([]EnvEntry, 0, len(entries))
	for _, e := range entries {
		label, group := envLabelAndGroup(e.key)
		displayValue := e.value
		if e.key == "ADMIN_PASSWORD" {
			displayValue = "****"
		}
		result = append(result, EnvEntry{
			Key:   e.key,
			Value: displayValue,
			Label: label,
			Group: group,
		})
	}

	return result, nil
}

func (s *SettingsService) UpdateConfig(updates map[string]string) error {
	existing := make(map[string]string)
	for k := range allEditableKeys {
		existing[k] = ""
	}

	file, err := os.Open(s.envPath)
	if err != nil {
		if os.IsNotExist(err) {
			for _, k := range sortedKeys(updates) {
				existing[k] = updates[k]
			}
			return s.writeEnvFile(existing)
		}
		return fmt.Errorf("打开配置文件失败: %w", err)
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		if _, ok := allEditableKeys[key]; ok {
			value := strings.TrimSpace(parts[1])
			value = strings.Trim(value, `"'`)
			existing[key] = value
		}
	}
	file.Close()

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	for k, v := range updates {
		if k == "ADMIN_PASSWORD" && (v == "****" || strings.TrimSpace(v) == "") {
			continue
		}
		existing[k] = v
	}

	return s.writeEnvFile(existing)
}

func (s *SettingsService) writeEnvFile(values map[string]string) error {
	content, err := os.ReadFile(s.envPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	updatedKeys := make(map[string]bool)

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		parts := strings.SplitN(trimmed, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		if val, ok := values[key]; ok {
			lines[i] = fmt.Sprintf("%s=%s", key, val)
			updatedKeys[key] = true
		}
	}

	for k, v := range values {
		if !updatedKeys[k] {
			lines = append(lines, fmt.Sprintf("%s=%s", k, v))
		}
	}

	tmpPath := s.envPath + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(strings.Join(lines, "\n")+"\n"), 0600); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	if err := os.Rename(tmpPath, s.envPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("保存配置文件失败: %w", err)
	}

	return nil
}

func (s *SettingsService) SaveBackgroundImage(file multipart.File, header *multipart.FileHeader) error {
	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowed := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true}
	if !allowed[ext] {
		return fmt.Errorf("不支持的图片格式: %s", ext)
	}

	dstPath := filepath.Join(s.dataDir, "bg.jpg")
	fmt.Printf("[settings] saving background to: %s\n", dstPath)
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	dst, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer dst.Close()

	written, err := io.Copy(dst, file)
	if err != nil {
		return fmt.Errorf("保存文件失败: %w", err)
	}
	fmt.Printf("[settings] background saved: %s (%d bytes)\n", dstPath, written)

	return nil
}

func (s *SettingsService) RemoveBackgroundImage() error {
	dstPath := filepath.Join(s.dataDir, "bg.jpg")
	if err := os.Remove(dstPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除背景图片失败: %w", err)
	}
	return nil
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
