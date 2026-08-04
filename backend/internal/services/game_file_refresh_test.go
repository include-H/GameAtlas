package services

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/files"
	"github.com/hao/game/internal/repositories"
)

func TestGameFileRefreshServiceRefreshesFilesWithExistingSize(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	romRoot := t.TempDir()
	target := filepath.Join(romRoot, "game.bin")
	if err := os.WriteFile(target, []byte("updated-content"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	gameID := insertServicesTestGame(t, db, "refresh-sizes", "Refresh Sizes", domain.GameVisibilityPublic)
	fileID := insertServicesGameFile(t, db, gameID, target, 0)
	// 预置一个已过期的大小，验证"刷新"会覆盖而非只补 NULL。
	if _, err := db.Exec(`UPDATE game_files SET size_bytes = 1 WHERE id = ?`, fileID); err != nil {
		t.Fatalf("preset stale size: %v", err)
	}

	service := NewGameFileRefreshService(
		repositories.NewGameFilesRepository(db),
		files.NewGuard(romRoot),
	)

	result, err := service.RefreshFileSizes()
	if err != nil {
		t.Fatalf("RefreshFileSizes returned error: %v", err)
	}
	if result.Updated != 1 || result.Errors != 0 {
		t.Fatalf("RefreshFileSizes result = %+v, want updated=1 errors=0", result)
	}

	var size int64
	if err := db.Get(&size, `SELECT size_bytes FROM game_files WHERE id = ?`, fileID); err != nil {
		t.Fatalf("read refreshed size: %v", err)
	}
	if size != int64(len("updated-content")) {
		t.Fatalf("size_bytes = %d, want %d", size, len("updated-content"))
	}

	var sourceCreatedAt string
	if err := db.Get(&sourceCreatedAt, `SELECT source_created_at FROM game_files WHERE id = ?`, fileID); err != nil {
		t.Fatalf("read source_created_at: %v", err)
	}
	if sourceCreatedAt == "" {
		t.Fatalf("source_created_at is empty after refresh")
	}
}

func TestGameFileRefreshServiceCountsMissingFilesAsErrors(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	romRoot := t.TempDir()
	gameID := insertServicesTestGame(t, db, "refresh-missing", "Refresh Missing", domain.GameVisibilityPublic)
	insertServicesGameFile(t, db, gameID, filepath.Join(romRoot, "missing.bin"), 0)

	service := NewGameFileRefreshService(
		repositories.NewGameFilesRepository(db),
		files.NewGuard(romRoot),
	)

	result, err := service.RefreshFileSizes()
	if err != nil {
		t.Fatalf("RefreshFileSizes returned error: %v", err)
	}
	if result.Updated != 0 || result.Errors != 1 {
		t.Fatalf("RefreshFileSizes result = %+v, want updated=0 errors=1", result)
	}
}
