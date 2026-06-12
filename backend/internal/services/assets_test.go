package services

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hao/game/internal/domain"
)

func TestAssetsServiceApplyRemoteAssetRejectsBlockedOrInvalidURLs(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertServicesTestGame(t, db, "remote-asset", "Remote Asset", domain.GameVisibilityPublic)
	service := newServicesAssetsService(db, t.TempDir())

	_, err := service.ApplyRemoteAsset(gameID, "cover", "http://localhost/image.png", 0)
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("ApplyRemoteAsset localhost error = %v, want domain.ErrValidation", err)
	}

	_, err = service.ApplyRemoteAsset(gameID, "cover", "not-a-url", 0)
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("ApplyRemoteAsset invalid url error = %v, want domain.ErrValidation", err)
	}
}

func TestAssetsServiceApplyRawAssetRejectsInvalidContentType(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertServicesTestGame(t, db, "raw-asset", "Raw Asset", domain.GameVisibilityPublic)
	service := newServicesAssetsService(db, t.TempDir())

	_, err := service.ApplyRawAsset(gameID, "cover", []byte("not-an-image"), "text/plain", 0)
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("ApplyRawAsset error = %v, want domain.ErrValidation", err)
	}
}

func TestAssetsServiceApplyRawAssetWritesToStaging(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	assetsDir := filepath.Join(t.TempDir(), "assets")
	gameID := insertServicesTestGame(t, db, "staging-raw", "Staging Raw", domain.GameVisibilityPublic)
	service := newServicesAssetsService(db, assetsDir)

	path, err := service.ApplyRawAsset(gameID, "cover", []byte("fake-png-data"), "image/png", 0)
	if err != nil {
		t.Fatalf("ApplyRawAsset error = %v, want nil", err)
	}
	if !strings.HasPrefix(path, "/assets/staging-raw/") || !strings.HasSuffix(path, ".png") {
		t.Fatalf("path = %q, want staging-raw png path", path)
	}

	// File should be in staging, not permanent.
	stagingEntries, readErr := os.ReadDir(filepath.Join(assetsDir, "_staging"))
	if readErr != nil {
		t.Fatalf("ReadDir staging returned error: %v", readErr)
	}
	if len(stagingEntries) != 1 {
		t.Fatalf("expected 1 file in staging, got %d", len(stagingEntries))
	}

	// Permanent game directory should not exist.
	if _, err := os.Stat(filepath.Join(assetsDir, "staging-raw")); !os.IsNotExist(err) {
		t.Fatalf("expected no permanent directory, got err=%v", err)
	}
}
