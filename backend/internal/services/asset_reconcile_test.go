package services

import (
	"os"
	"path/filepath"
	"testing"

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
