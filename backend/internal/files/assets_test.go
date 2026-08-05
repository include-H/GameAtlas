package files

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func writeTestPNG(t *testing.T, path string, width int, height int) {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 120, A: 255})
		}
	}

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create test image: %v", err)
	}
	defer file.Close()
	if err := png.Encode(file, img); err != nil {
		t.Fatalf("encode test image: %v", err)
	}
}

func TestThumbURLFor(t *testing.T) {
	if got := ThumbURLFor("/assets/game-1/uid.png"); got != "/assets/game-1/uid.thumb.jpg" {
		t.Fatalf("ThumbURLFor = %q, want /assets/game-1/uid.thumb.jpg", got)
	}
}

func TestWriteThumbnailMovesWithOriginalAndDeletesWithIt(t *testing.T) {
	baseDir := t.TempDir()
	store := NewAssetStore(baseDir)

	stagingDir := filepath.Join(baseDir, "_staging")
	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		t.Fatalf("MkdirAll staging: %v", err)
	}

	uuid := "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	originalName := uuid + ".png"
	writeTestPNG(t, filepath.Join(stagingDir, originalName), 800, 400)

	assetPath := "/assets/game-1/" + originalName
	if err := store.WriteThumbnail(assetPath); err != nil {
		t.Fatalf("WriteThumbnail returned error: %v", err)
	}

	thumbStaging := filepath.Join(stagingDir, uuid+".thumb.jpg")
	if _, err := os.Stat(thumbStaging); err != nil {
		t.Fatalf("expected staging thumbnail, got err=%v", err)
	}

	permPath, err := store.MoveToPermanent(assetPath, "game-1")
	if err != nil {
		t.Fatalf("MoveToPermanent returned error: %v", err)
	}
	if permPath != assetPath {
		t.Fatalf("MoveToPermanent = %q, want %q", permPath, assetPath)
	}

	thumbPerm := filepath.Join(baseDir, "game-1", uuid+".thumb.jpg")
	if _, err := os.Stat(thumbPerm); err != nil {
		t.Fatalf("expected permanent thumbnail, got err=%v", err)
	}
	if _, err := os.Stat(thumbStaging); !os.IsNotExist(err) {
		t.Fatalf("staging thumbnail should be gone, got err=%v", err)
	}

	if err := store.DeleteAsset(assetPath); err != nil {
		t.Fatalf("DeleteAsset returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(baseDir, "game-1", originalName)); !os.IsNotExist(err) {
		t.Fatalf("original should be deleted, got err=%v", err)
	}
	if _, err := os.Stat(thumbPerm); !os.IsNotExist(err) {
		t.Fatalf("thumbnail should be deleted with original, got err=%v", err)
	}
}

func TestWriteThumbnailCreatesThumbnailForSmallImages(t *testing.T) {
	baseDir := t.TempDir()
	store := NewAssetStore(baseDir)

	stagingDir := filepath.Join(baseDir, "_staging")
	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		t.Fatalf("MkdirAll staging: %v", err)
	}

	uuid := "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
	originalName := uuid + ".png"
	writeTestPNG(t, filepath.Join(stagingDir, originalName), 200, 100)

	if err := store.WriteThumbnail("/assets/game-1/" + originalName); err != nil {
		t.Fatalf("WriteThumbnail returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(stagingDir, uuid+".thumb.jpg")); err != nil {
		t.Fatalf("small image should still get a thumbnail, got err=%v", err)
	}
}
