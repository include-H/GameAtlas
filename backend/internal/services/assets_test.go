package services

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/repositories"
)

func TestAssetsServiceDeletePrimaryVideoFallsBackToNextVideo(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	assetsDir := filepath.Join(t.TempDir(), "assets")
	gameID := insertServicesTestGame(t, db, "asset-game", "Asset Game", domain.GameVisibilityPublic)
	insertServicesGameAsset(t, db, gameID, "video-a", "video", "/assets/asset-game/video-a.mp4", 0)
	insertServicesGameAsset(t, db, gameID, "video-b", "video", "/assets/asset-game/video-b.mp4", 1)
	writeServicesAssetFile(t, assetsDir, "asset-game", "video-a.mp4", []byte("a"))
	writeServicesAssetFile(t, assetsDir, "asset-game", "video-b.mp4", []byte("b"))

	service := newServicesAssetsService(db, assetsDir)
	if err := service.Delete(domain.DeleteAssetInput{
		GameID:    gameID,
		AssetType: "video",
		AssetUID:  "video-a",
		Path:      "/assets/asset-game/video-a.mp4",
	}); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	if _, err := repositories.NewAssetsRepository(db).GetAssetByUID(gameID, "video-a", "video"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected deleted video asset row, got err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(assetsDir, "asset-game", "video-a.mp4")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected asset file to be deleted, got err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(assetsDir, "asset-game", "video-b.mp4")); err != nil {
		t.Fatalf("expected fallback asset file to remain, got err=%v", err)
	}
}

func TestAssetsServiceDeleteLastPrimaryVideoClearsPreviewUID(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	assetsDir := filepath.Join(t.TempDir(), "assets")
	gameID := insertServicesTestGame(t, db, "solo-video", "Solo Video", domain.GameVisibilityPublic)
	insertServicesGameAsset(t, db, gameID, "video-only", "video", "/assets/solo-video/video-only.mp4", 0)
	writeServicesAssetFile(t, assetsDir, "solo-video", "video-only.mp4", []byte("only"))

	service := newServicesAssetsService(db, assetsDir)
	if err := service.Delete(domain.DeleteAssetInput{
		GameID:    gameID,
		AssetType: "video",
		AssetUID:  "video-only",
		Path:      "/assets/solo-video/video-only.mp4",
	}); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

}

func TestAssetsServiceReorderScreenshotsRejectsEmptySelection(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertServicesTestGame(t, db, "reorder-game", "Reorder Game", domain.GameVisibilityPublic)
	service := newServicesAssetsService(db, t.TempDir())

	err := service.ReorderScreenshots(domain.ScreenshotOrderUpdateInput{
		GameID:    gameID,
		AssetUIDs: nil,
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("ReorderScreenshots error = %v, want ErrValidation", err)
	}
}

func TestAssetsServiceDeleteScreenshotReturnsNotFoundWhenAssetMissing(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	assetsDir := filepath.Join(t.TempDir(), "assets")
	gameID := insertServicesTestGame(t, db, "missing-shot", "Missing Shot", domain.GameVisibilityPublic)
	service := newServicesAssetsService(db, assetsDir)

	err := service.Delete(domain.DeleteAssetInput{
		GameID:    gameID,
		AssetType: "screenshot",
		Path:      "/assets/missing-shot/not-found.png",
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Delete error = %v, want ErrNotFound", err)
	}
}

func TestAssetsServiceDeleteCoverClearsImageAndRemovesFile(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	assetsDir := filepath.Join(t.TempDir(), "assets")
	gameID := insertServicesTestGame(t, db, "cover-game", "Cover Game", domain.GameVisibilityPublic)
	coverPath := "/assets/cover-game/cover.png"
	if _, err := db.Exec(`
		UPDATE games
		SET cover_image = ?
		WHERE id = ?
	`, coverPath, gameID); err != nil {
		t.Fatalf("set test cover image: %v", err)
	}
	writeServicesAssetFile(t, assetsDir, "cover-game", "cover.png", []byte("cover"))

	service := newServicesAssetsService(db, assetsDir)
	if err := service.Delete(domain.DeleteAssetInput{
		GameID:    gameID,
		AssetType: "cover",
		Path:      coverPath,
	}); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	game := mustLoadServicesGame(t, db, gameID)
	if game.CoverImage != nil {
		t.Fatalf("CoverImage = %v, want nil", game.CoverImage)
	}
	if _, err := os.Stat(filepath.Join(assetsDir, "cover-game", "cover.png")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected cover file to be deleted, got err=%v", err)
	}
}

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

func TestAssetsServiceReorderVideosReturnsNotFoundForMissingUID(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertServicesTestGame(t, db, "reorder-videos", "Reorder Videos", domain.GameVisibilityPublic)
	insertServicesGameAsset(t, db, gameID, "video-a", "video", "/assets/reorder-videos/video-a.mp4", 0)
	service := newServicesAssetsService(db, t.TempDir())

	err := service.ReorderVideos(domain.VideoOrderUpdateInput{
		GameID:    gameID,
		AssetUIDs: []string{"missing-video"},
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("ReorderVideos error = %v, want ErrNotFound", err)
	}
}
