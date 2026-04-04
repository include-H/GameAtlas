package services

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/repositories"
)

func TestAssetsServiceApplyRemoteAssetRejectsBlockedOrInvalidURLs(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertServicesTestGame(t, db, "remote-asset", "Remote Asset", domain.GameVisibilityPublic)
	service := newServicesAssetsService(db, t.TempDir())

	_, err := service.ApplyRemoteAsset(gameID, "cover", "http://localhost/image.png", 0)
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("ApplyRemoteAsset localhost error = %v, want ErrValidation", err)
	}

	_, err = service.ApplyRemoteAsset(gameID, "cover", "not-a-url", 0)
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("ApplyRemoteAsset invalid url error = %v, want ErrValidation", err)
	}
}

func TestAssetsServiceApplyRawAssetRejectsInvalidContentType(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertServicesTestGame(t, db, "raw-asset", "Raw Asset", domain.GameVisibilityPublic)
	service := newServicesAssetsService(db, t.TempDir())

	_, err := service.ApplyRawAsset(gameID, "cover", []byte("not-an-image"), "text/plain", 0)
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("ApplyRawAsset error = %v, want ErrValidation", err)
	}
}

func TestAssetsServiceApplyRawAssetCleansUpSavedFileWhenPersistFails(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	assetsDir := filepath.Join(t.TempDir(), "assets")
	gameID := insertServicesTestGame(t, db, "persist-fail", "Persist Fail", domain.GameVisibilityPublic)
	service := newServicesAssetsService(db, assetsDir)

	_, err := service.ApplyRawAsset(gameID, "unexpected", []byte("png"), "image/png", 0)
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("ApplyRawAsset error = %v, want ErrValidation", err)
	}

	entries, readErr := os.ReadDir(filepath.Join(assetsDir, "persist-fail"))
	if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
		t.Fatalf("ReadDir returned error: %v", readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no saved files after persist failure, got %d", len(entries))
	}

	tasksRepo := repositories.NewAssetCleanupTasksRepository(db)
	tasks, listErr := tasksRepo.ListPending(10)
	if listErr != nil {
		t.Fatalf("ListPending returned error: %v", listErr)
	}
	if len(tasks) != 0 {
		t.Fatalf("expected no cleanup task when rollback succeeds, got %d", len(tasks))
	}
}
