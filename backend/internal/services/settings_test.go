package services

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hao/game/internal/config"
	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/repositories"
)

func TestSettingsServiceReadsAndUpdatesDatabaseBackedConfig(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	assetsDir := filepath.Join(t.TempDir(), "assets")
	cfg := config.Config{
		AppEnv:                 "development",
		Host:                   "0.0.0.0",
		Port:                   3000,
		StaticDir:              filepath.Join(t.TempDir(), "dist"),
		AssetsDir:              assetsDir,
		PrimaryROMRoot:         filepath.Join(t.TempDir(), "ROM"),
		DBBackupEnabled:        true,
		DBBackupDir:            filepath.Join(t.TempDir(), "backups"),
		DBBackupInterval:       24 * time.Hour,
		DBBackupRetentionCount: 5,
		AdminPassword:          "secret",
		AdminDisplayName:       "Admin",
		AuthMaxFails:           5,
		AuthCooldown:           10 * time.Minute,
		AuthFailWindow:         30 * time.Minute,
		AuthStateTTL:           24 * time.Hour,
		AuthTrackBy:            "ip_ua",
		WikiHistoryLimit:       100,
		VHDDiffRoot:            "C:",
		ReadHeaderTimeout:      5 * time.Second,
		ShutdownTimeout:        10 * time.Second,
	}

	repo := repositories.NewAppSettingsRepository(db)
	if err := repo.EnsureDefaults(cfg.RuntimeSettings()); err != nil {
		t.Fatalf("EnsureDefaults returned error: %v", err)
	}

	service := NewSettingsService(cfg, repo)
	entries, err := service.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig returned error: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("entries is empty")
	}
	requiredEntries := map[string]bool{
		"DB_BACKUP_ENABLED":         false,
		"DB_BACKUP_RETENTION_COUNT": false,
	}
	for _, entry := range entries {
		if entry.Key == "ADMIN_PASSWORD" && entry.Value != "****" {
			t.Fatalf("ADMIN_PASSWORD entry = %q, want masked value", entry.Value)
		}
		if _, ok := requiredEntries[entry.Key]; ok {
			requiredEntries[entry.Key] = true
		}
	}
	for key, found := range requiredEntries {
		if !found {
			t.Fatalf("missing %s from settings entries", key)
		}
	}

	if err := service.UpdateConfig(map[string]string{
		"ADMIN_DISPLAY_NAME": "Owner",
		"PORT":               "3001",
		"ADMIN_PASSWORD":     "****",
	}); err != nil {
		t.Fatalf("UpdateConfig returned error: %v", err)
	}

	values, err := repo.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if values["ADMIN_DISPLAY_NAME"] != "Owner" {
		t.Fatalf("ADMIN_DISPLAY_NAME = %q, want Owner", values["ADMIN_DISPLAY_NAME"])
	}
	if values["PORT"] != "3001" {
		t.Fatalf("PORT = %q, want 3001", values["PORT"])
	}
	if values["ADMIN_PASSWORD"] != "secret" {
		t.Fatalf("ADMIN_PASSWORD changed to %q, want preserved secret", values["ADMIN_PASSWORD"])
	}
}

func TestSettingsServiceRejectsInvalidDatabaseBackedConfig(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	tempDir := t.TempDir()
	assetsDir := filepath.Join(tempDir, "assets")
	cfg := config.Config{
		AppEnv:                 "development",
		Host:                   "0.0.0.0",
		Port:                   3000,
		StaticDir:              filepath.Join(tempDir, "dist"),
		AssetsDir:              assetsDir,
		PrimaryROMRoot:         filepath.Join(tempDir, "ROM"),
		DBBackupEnabled:        true,
		DBBackupDir:            filepath.Join(tempDir, "backups"),
		DBBackupInterval:       24 * time.Hour,
		DBBackupRetentionCount: 5,
		AdminPassword:          "secret",
		AdminDisplayName:       "Admin",
		AuthMaxFails:           5,
		AuthCooldown:           10 * time.Minute,
		AuthFailWindow:         30 * time.Minute,
		AuthStateTTL:           24 * time.Hour,
		AuthTrackBy:            "ip_ua",
		WikiHistoryLimit:       100,
		VHDDiffRoot:            "C:",
		ReadHeaderTimeout:      5 * time.Second,
		ShutdownTimeout:        10 * time.Second,
	}

	repo := repositories.NewAppSettingsRepository(db)
	if err := repo.EnsureDefaults(cfg.RuntimeSettings()); err != nil {
		t.Fatalf("EnsureDefaults returned error: %v", err)
	}
	service := NewSettingsService(cfg, repo)

	if err := service.UpdateConfig(map[string]string{"PORT": "abc"}); err == nil || !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("invalid PORT error = %v, want domain.ErrValidation", err)
	}
	if err := service.UpdateConfig(map[string]string{"NOT_A_SETTING": "value"}); err == nil || !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("unsupported setting error = %v, want domain.ErrValidation", err)
	}
}

func TestSettingsServiceRejectsUnsupportedBackgroundImageFormat(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	service := NewSettingsService(config.Config{}, repositories.NewAppSettingsRepository(db))
	err := service.SaveBackgroundImage(nil, &multipart.FileHeader{Filename: "background.svg"})
	if err == nil || !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("unsupported background format error = %v, want domain.ErrValidation", err)
	}
}

func TestSettingsServiceRejectsInvalidBackgroundImageWithoutReplacingExistingFile(t *testing.T) {
	dataDir := t.TempDir()
	assetsDir := filepath.Join(dataDir, "assets")
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()
	service := NewSettingsService(config.Config{AssetsDir: assetsDir}, repositories.NewAppSettingsRepository(db))
	backgroundPath := filepath.Join(dataDir, "bg.jpg")
	if err := os.WriteFile(backgroundPath, []byte("existing-background"), 0o644); err != nil {
		t.Fatalf("write existing background: %v", err)
	}

	invalidPath := filepath.Join(t.TempDir(), "invalid.png")
	if err := os.WriteFile(invalidPath, []byte("not-an-image"), 0o644); err != nil {
		t.Fatalf("write invalid image: %v", err)
	}
	file, err := os.Open(invalidPath)
	if err != nil {
		t.Fatalf("open invalid image: %v", err)
	}
	defer file.Close()
	header := &multipart.FileHeader{
		Filename: "invalid.png",
		Size:     int64(len("not-an-image")),
		Header:   textproto.MIMEHeader{"Content-Type": []string{"image/png"}},
	}

	err = service.SaveBackgroundImage(file, header)
	if err == nil || !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("SaveBackgroundImage error = %v, want domain.ErrValidation", err)
	}
	content, readErr := os.ReadFile(backgroundPath)
	if readErr != nil {
		t.Fatalf("read existing background: %v", readErr)
	}
	if !bytes.Equal(content, []byte("existing-background")) {
		t.Fatalf("existing background changed to %q after rejected upload", content)
	}
}
