package services

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hao/game/internal/config"
	"github.com/hao/game/internal/domain"
)

func TestAssetReconcileServiceRemovesMissingReferencesForSingleGame(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	assetsDir := filepath.Join(t.TempDir(), "assets")
	gameID := insertServicesTestGame(t, db, "reconcile-game", "Reconcile Game", domain.GameVisibilityPublic)
	if _, err := db.Exec(`
		UPDATE games
		SET cover_image = ?, banner_image = ?
		WHERE id = ?
	`, "/assets/reconcile-game/cover.png", "/assets/reconcile-game/banner.png", gameID); err != nil {
		t.Fatalf("set game image references: %v", err)
	}
	insertServicesGameAsset(t, db, gameID, "shot-a", "screenshot", "/assets/reconcile-game/shot-a.png", 0)
	insertServicesGameAsset(t, db, gameID, "video-a", "video", "/assets/reconcile-game/video-a.mp4", 1)

	writeServicesAssetFile(t, assetsDir, "reconcile-game", "banner.png", []byte("banner"))
	writeServicesAssetFile(t, assetsDir, "reconcile-game", "video-a.mp4", []byte("video"))

	service := NewAssetReconcileService(config.Config{AssetsDir: assetsDir}, db)
	changed, err := service.ReconcileGameMissingAssets(gameID)
	if err != nil {
		t.Fatalf("ReconcileGameMissingAssets returned error: %v", err)
	}
	if !changed {
		t.Fatal("ReconcileGameMissingAssets changed = false, want true")
	}

	game := mustLoadServicesGame(t, db, gameID)
	if game.CoverImage != nil {
		t.Fatalf("CoverImage = %#v, want nil", game.CoverImage)
	}
	if game.BannerImage == nil || *game.BannerImage != "/assets/reconcile-game/banner.png" {
		t.Fatalf("BannerImage = %#v, want surviving banner", game.BannerImage)
	}

	var assetCount int
	if err := db.Get(&assetCount, `SELECT COUNT(*) FROM game_assets WHERE game_id = ?`, gameID); err != nil {
		t.Fatalf("count game assets: %v", err)
	}
	if assetCount != 1 {
		t.Fatalf("assetCount = %d, want 1", assetCount)
	}
}

func TestAssetReconcileServiceFullSweepRemovesMissingReferencesAcrossGames(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	assetsDir := filepath.Join(t.TempDir(), "assets")
	firstGameID := insertServicesTestGame(t, db, "reconcile-a", "Reconcile A", domain.GameVisibilityPublic)
	secondGameID := insertServicesTestGame(t, db, "reconcile-b", "Reconcile B", domain.GameVisibilityPublic)

	if _, err := db.Exec(`UPDATE games SET cover_image = ? WHERE id = ?`, "/assets/reconcile-a/cover.png", firstGameID); err != nil {
		t.Fatalf("set first cover image: %v", err)
	}
	insertServicesGameAsset(t, db, firstGameID, "shot-a", "screenshot", "/assets/reconcile-a/shot-a.png", 0)
	insertServicesGameAsset(t, db, secondGameID, "video-b", "video", "/assets/reconcile-b/video-b.mp4", 0)

	writeServicesAssetFile(t, assetsDir, "reconcile-b", "video-b.mp4", []byte("video"))

	service := NewAssetReconcileService(config.Config{AssetsDir: assetsDir}, db)
	changed, err := service.ReconcileAllMissingAssets()
	if err != nil {
		t.Fatalf("ReconcileAllMissingAssets returned error: %v", err)
	}
	if changed != 1 {
		t.Fatalf("changed = %d, want 1 affected game", changed)
	}

	firstGame := mustLoadServicesGame(t, db, firstGameID)
	if firstGame.CoverImage != nil {
		t.Fatalf("firstGame.CoverImage = %#v, want nil", firstGame.CoverImage)
	}

	var firstAssetCount int
	if err := db.Get(&firstAssetCount, `SELECT COUNT(*) FROM game_assets WHERE game_id = ?`, firstGameID); err != nil {
		t.Fatalf("count first game assets: %v", err)
	}
	if firstAssetCount != 0 {
		t.Fatalf("firstAssetCount = %d, want 0", firstAssetCount)
	}

	if _, err := os.Stat(filepath.Join(assetsDir, "reconcile-b", "video-b.mp4")); err != nil {
		t.Fatalf("expected surviving asset file, got err=%v", err)
	}
}

func TestCleanOrphanedAssetFilesDeletesUnreferencedFiles(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	assetsDir := filepath.Join(t.TempDir(), "assets")
	gameA := insertServicesTestGame(t, db, "orphan-a", "Orphan A", domain.GameVisibilityPublic)
	gameB := insertServicesTestGame(t, db, "orphan-b", "Orphan B", domain.GameVisibilityPublic)

	// DB references.
	if _, err := db.Exec(`UPDATE games SET cover_image = ? WHERE id = ?`, "/assets/orphan-a/cover.jpg", gameA); err != nil {
		t.Fatalf("set cover: %v", err)
	}
	insertServicesGameAsset(t, db, gameA, "shot-a", "screenshot", "/assets/orphan-a/shot.png", 0)
	insertServicesGameAsset(t, db, gameB, "vid-b", "video", "/assets/orphan-b/video.mp4", 0)

	// Files on disk: 2 referenced + 2 orphaned.
	writeServicesAssetFile(t, assetsDir, "orphan-a", "cover.jpg", []byte("cover"))
	writeServicesAssetFile(t, assetsDir, "orphan-a", "shot.png", []byte("shot"))
	writeServicesAssetFile(t, assetsDir, "orphan-a", "stale-old.jpg", []byte("orphan"))
	writeServicesAssetFile(t, assetsDir, "orphan-b", "video.mp4", []byte("video"))
	writeServicesAssetFile(t, assetsDir, "orphan-b", "stale-banner.png", []byte("orphan"))

	service := NewAssetReconcileService(config.Config{AssetsDir: assetsDir}, db)
	deleted, err := service.CleanOrphanedAssetFiles()
	if err != nil {
		t.Fatalf("CleanOrphanedAssetFiles returned error: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("deleted = %d, want 2", deleted)
	}

	// Referenced files survive.
	assertFileExists(t, filepath.Join(assetsDir, "orphan-a", "cover.jpg"))
	assertFileExists(t, filepath.Join(assetsDir, "orphan-a", "shot.png"))
	assertFileExists(t, filepath.Join(assetsDir, "orphan-b", "video.mp4"))

	// Orphaned files deleted.
	assertFileMissing(t, filepath.Join(assetsDir, "orphan-a", "stale-old.jpg"))
	assertFileMissing(t, filepath.Join(assetsDir, "orphan-b", "stale-banner.png"))
}

func TestCleanOrphanedAssetFilesQuarantinesUnreferencedFilesByDefault(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	tempDir := t.TempDir()
	assetsDir := filepath.Join(tempDir, "assets")
	gameID := insertServicesTestGame(t, db, "quarantine", "Quarantine", domain.GameVisibilityPublic)
	insertServicesGameAsset(t, db, gameID, "vid-1", "video", "/assets/quarantine/video.mp4", 0)

	orphanPath := writeServicesAssetFile(t, assetsDir, "quarantine", "old-cover.jpg", []byte("orphan"))
	quarantineDir := filepath.Join(tempDir, "orphaned-assets")

	service := NewAssetReconcileService(config.Config{
		AssetsDir: assetsDir,
	}, db)

	processed, err := service.CleanOrphanedAssetFiles()
	if err != nil {
		t.Fatalf("CleanOrphanedAssetFiles returned error: %v", err)
	}
	if processed != 1 {
		t.Fatalf("processed = %d, want 1", processed)
	}

	assertFileMissing(t, orphanPath)
	matches, err := filepath.Glob(filepath.Join(quarantineDir, "*", "quarantine", "old-cover.jpg"))
	if err != nil {
		t.Fatalf("Glob returned error: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("quarantined matches = %#v, want one quarantined file", matches)
	}
	assertFileExists(t, matches[0])
}

func TestCleanOldQuarantinedAssetsRemovesFilesAfterSevenDays(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	tempDir := t.TempDir()
	assetsDir := filepath.Join(tempDir, "assets")
	quarantinedPath := filepath.Join(tempDir, "orphaned-assets", "20260801-000000", "game-a", "cover.jpg")
	if err := os.MkdirAll(filepath.Dir(quarantinedPath), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(quarantinedPath, []byte("orphan"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	oldTime := time.Now().Add(-8 * 24 * time.Hour)
	if err := os.Chtimes(quarantinedPath, oldTime, oldTime); err != nil {
		t.Fatalf("Chtimes returned error: %v", err)
	}

	service := NewAssetReconcileService(config.Config{AssetsDir: assetsDir}, db)
	removed, err := service.cleanOldQuarantinedAssets()
	if err != nil {
		t.Fatalf("cleanOldQuarantinedAssets returned error: %v", err)
	}
	if removed != 1 {
		t.Fatalf("removed = %d, want 1", removed)
	}
	assertFileMissing(t, quarantinedPath)
}

func TestCleanOrphanedAssetFilesPreservesAllReferencedFiles(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	assetsDir := filepath.Join(t.TempDir(), "assets")
	gameID := insertServicesTestGame(t, db, "no-orphan", "No Orphan", domain.GameVisibilityPublic)

	if _, err := db.Exec(`
		UPDATE games SET cover_image = ?, banner_image = ? WHERE id = ?
	`, "/assets/no-orphan/cover.jpg", "/assets/no-orphan/banner.jpg", gameID); err != nil {
		t.Fatalf("set images: %v", err)
	}
	insertServicesGameAsset(t, db, gameID, "shot-1", "screenshot", "/assets/no-orphan/shot1.png", 0)
	insertServicesGameAsset(t, db, gameID, "shot-2", "screenshot", "/assets/no-orphan/shot2.png", 1)

	writeServicesAssetFile(t, assetsDir, "no-orphan", "cover.jpg", []byte("cover"))
	writeServicesAssetFile(t, assetsDir, "no-orphan", "banner.jpg", []byte("banner"))
	writeServicesAssetFile(t, assetsDir, "no-orphan", "shot1.png", []byte("s1"))
	writeServicesAssetFile(t, assetsDir, "no-orphan", "shot2.png", []byte("s2"))

	service := NewAssetReconcileService(config.Config{AssetsDir: assetsDir}, db)
	deleted, err := service.CleanOrphanedAssetFiles()
	if err != nil {
		t.Fatalf("CleanOrphanedAssetFiles returned error: %v", err)
	}
	if deleted != 0 {
		t.Fatalf("deleted = %d, want 0", deleted)
	}
}

func TestCleanOrphanedAssetFilesHandlesEmptyAssetsDir(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	assetsDir := filepath.Join(t.TempDir(), "nonexistent")
	service := NewAssetReconcileService(config.Config{AssetsDir: assetsDir}, db)
	deleted, err := service.CleanOrphanedAssetFiles()
	if err != nil {
		t.Fatalf("CleanOrphanedAssetFiles returned error: %v", err)
	}
	if deleted != 0 {
		t.Fatalf("deleted = %d, want 0", deleted)
	}
}

func TestCleanOrphanedAssetFilesRemovesEmptyGameDirectories(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	assetsDir := filepath.Join(t.TempDir(), "assets")
	gameID := insertServicesTestGame(t, db, "empty-dir", "Empty Dir", domain.GameVisibilityPublic)
	insertServicesGameAsset(t, db, gameID, "vid-1", "video", "/assets/empty-dir/video.mp4", 0)

	// File on disk is orphaned (different path than DB reference).
	writeServicesAssetFile(t, assetsDir, "empty-dir", "old-cover.jpg", []byte("orphan"))
	// The referenced file does NOT exist on disk — that's fine, orphan cleanup only cares about unreferenced files.

	service := NewAssetReconcileService(config.Config{AssetsDir: assetsDir}, db)
	deleted, err := service.CleanOrphanedAssetFiles()
	if err != nil {
		t.Fatalf("CleanOrphanedAssetFiles returned error: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("deleted = %d, want 1", deleted)
	}

	// Directory should be removed since it's now empty.
	if _, err := os.Stat(filepath.Join(assetsDir, "empty-dir")); !os.IsNotExist(err) {
		t.Fatalf("expected empty-dir to be removed, got err=%v", err)
	}
}

func TestCleanOrphanedAssetFilesDoesNotEmptyNonEmptyDirectories(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	assetsDir := filepath.Join(t.TempDir(), "assets")
	gameID := insertServicesTestGame(t, db, "keep-dir", "Keep Dir", domain.GameVisibilityPublic)
	insertServicesGameAsset(t, db, gameID, "vid-1", "video", "/assets/keep-dir/video.mp4", 0)

	// One orphaned asset file + one non-asset file.
	writeServicesAssetFile(t, assetsDir, "keep-dir", "stale.jpg", []byte("orphan"))
	nonAssetPath := filepath.Join(assetsDir, "keep-dir", "notes.txt")
	if err := os.WriteFile(nonAssetPath, []byte("notes"), 0o644); err != nil {
		t.Fatalf("write notes.txt: %v", err)
	}

	service := NewAssetReconcileService(config.Config{AssetsDir: assetsDir}, db)
	deleted, err := service.CleanOrphanedAssetFiles()
	if err != nil {
		t.Fatalf("CleanOrphanedAssetFiles returned error: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("deleted = %d, want 1", deleted)
	}

	// Directory survives because notes.txt remains.
	assertFileExists(t, nonAssetPath)
	assertFileMissing(t, filepath.Join(assetsDir, "keep-dir", "stale.jpg"))
}

func TestCleanOrphanedAssetFilesCooldown(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	assetsDir := filepath.Join(t.TempDir(), "assets")
	gameID := insertServicesTestGame(t, db, "cooldown", "Cooldown", domain.GameVisibilityPublic)
	insertServicesGameAsset(t, db, gameID, "vid-1", "video", "/assets/cooldown/video.mp4", 0)
	writeServicesAssetFile(t, assetsDir, "cooldown", "orphan.jpg", []byte("orphan"))

	service := NewAssetReconcileService(config.Config{AssetsDir: assetsDir}, db)

	first, err := service.CleanOrphanedAssetFiles()
	if err != nil {
		t.Fatalf("first call returned error: %v", err)
	}
	if first != 1 {
		t.Fatalf("first deleted = %d, want 1", first)
	}

	// Second call within cooldown should be a no-op.
	second, err := service.CleanOrphanedAssetFiles()
	if err != nil {
		t.Fatalf("second call returned error: %v", err)
	}
	if second != 0 {
		t.Fatalf("second deleted = %d, want 0 (cooldown)", second)
	}

	// After cooldown expires, it should work again.
	service.lastOrphanSweepAt = time.Now().Add(-3 * time.Second)
	writeServicesAssetFile(t, assetsDir, "cooldown", "orphan2.jpg", []byte("orphan2"))
	third, err := service.CleanOrphanedAssetFiles()
	if err != nil {
		t.Fatalf("third call returned error: %v", err)
	}
	if third != 1 {
		t.Fatalf("third deleted = %d, want 1", third)
	}
}

func TestFsPathToAssetPath(t *testing.T) {
	tests := []struct {
		name    string
		baseDir string
		fsPath  string
		want    string
		wantErr bool
	}{
		{
			name:    "normal file",
			baseDir: "/data/assets",
			fsPath:  "/data/assets/game-a/cover.jpg",
			want:    "/assets/game-a/cover.jpg",
		},
		{
			name:    "nested directory",
			baseDir: "/data/assets",
			fsPath:  "/data/assets/game-b/sub/deep.png",
			want:    "/assets/game-b/sub/deep.png",
		},
		{
			name:    "path outside base dir",
			baseDir: "/data/assets",
			fsPath:  "/data/other/file.jpg",
			wantErr: true,
		},
		{
			name:    "traversal attempt",
			baseDir: "/data/assets",
			fsPath:  "/data/assets/../../../etc/passwd",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fsPathToAssetPath(tt.baseDir, tt.fsPath)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("fsPathToAssetPath(%q, %q) = %q, want error", tt.baseDir, tt.fsPath, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("fsPathToAssetPath(%q, %q) returned error: %v", tt.baseDir, tt.fsPath, err)
			}
			if got != tt.want {
				t.Fatalf("fsPathToAssetPath(%q, %q) = %q, want %q", tt.baseDir, tt.fsPath, got, tt.want)
			}
		})
	}
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file %q to exist: %v", path, err)
	}
}

func assertFileMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected file %q to be missing: %v", path, err)
	}
}
