package services

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/hao/game/internal/config"
	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/repositories"
)

type SettingsService struct {
	cfg          config.Config
	dataDir      string
	settingsRepo *repositories.AppSettingsRepository
}

func NewSettingsService(cfg config.Config, settingsRepo *repositories.AppSettingsRepository) *SettingsService {
	dataDir := filepath.Dir(cfg.AssetsDir)

	return &SettingsService{
		cfg:          cfg,
		dataDir:      dataDir,
		settingsRepo: settingsRepo,
	}
}

type EnvEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Label string `json:"label"`
	Group string `json:"group"`
}

func (s *SettingsService) GetConfig() ([]EnvEntry, error) {
	values, err := s.settingsRepo.List()
	if err != nil {
		return nil, fmt.Errorf("读取配置失败: %w", err)
	}
	defaults := s.cfg.RuntimeSettings()

	definitions := config.RuntimeSettingDefinitions()
	result := make([]EnvEntry, 0, len(definitions))
	for _, definition := range definitions {
		displayValue, ok := values[definition.Key]
		if !ok {
			displayValue = defaults[definition.Key]
		}
		if definition.Sensitive && displayValue != "" {
			displayValue = "****"
		}
		result = append(result, EnvEntry{
			Key:   definition.Key,
			Value: displayValue,
			Label: definition.Label,
			Group: definition.Group,
		})
	}

	return result, nil
}

func (s *SettingsService) UpdateConfig(updates map[string]string) error {
	existing, err := s.settingsRepo.List()
	if err != nil {
		return fmt.Errorf("读取配置失败: %w", err)
	}

	definitions := config.RuntimeSettingKeys()
	toSave := make(map[string]string)
	for k, v := range updates {
		definition, ok := definitions[k]
		if !ok {
			return fmt.Errorf("%w: 不支持的配置项: %s", domain.ErrValidation, k)
		}
		if definition.Sensitive && (v == "****" || strings.TrimSpace(v) == "") {
			continue
		}
		toSave[k] = strings.TrimSpace(v)
	}

	for key, value := range s.cfg.RuntimeSettings() {
		if _, ok := existing[key]; !ok {
			toSave[key] = value
		}
	}

	candidate := make(map[string]string, len(existing)+len(toSave))
	for key, value := range existing {
		candidate[key] = value
	}
	for key, value := range toSave {
		candidate[key] = value
	}
	cfg, err := s.cfg.ApplyRuntimeSettings(candidate)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrValidation, err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrValidation, err)
	}

	return s.settingsRepo.UpsertMany(toSave)
}

func (s *SettingsService) SaveBackgroundImage(file multipart.File, header *multipart.FileHeader) error {
	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowed := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true}
	if !allowed[ext] {
		return fmt.Errorf("%w: 不支持的图片格式: %s", domain.ErrValidation, ext)
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
