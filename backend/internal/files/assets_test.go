package files

import (
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeTestImage(t *testing.T, path string, width, height int, encode func(io.Writer, image.Image) error) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 255), G: uint8(y % 255), B: 128, A: 255})
		}
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create test image: %v", err)
	}
	defer file.Close()
	if err := encode(file, img); err != nil {
		t.Fatalf("encode test image: %v", err)
	}
}

func writeTestAsset(t *testing.T, baseDir string, gameID string, filename string, width, height int, encode func(io.Writer, image.Image) error) string {
	t.Helper()
	gameDir := filepath.Join(baseDir, gameID)
	if err := os.MkdirAll(gameDir, 0o755); err != nil {
		t.Fatalf("mkdir game dir: %v", err)
	}
	src := filepath.Join(gameDir, filename)
	writeTestImage(t, src, width, height, encode)
	return src
}

func TestAssetStoreEnsureVariantGeneratesWebP(t *testing.T) {
	baseDir := t.TempDir()
	store := NewAssetStore(baseDir)
	writeTestAsset(t, baseDir, "game-a", "shot.jpg", 1600, 900, func(w io.Writer, m image.Image) error {
		return jpeg.Encode(w, m, nil)
	})

	variant, err := store.EnsureVariant("/assets/game-a/shot.jpg", 480)
	if err != nil {
		t.Fatalf("EnsureVariant returned error: %v", err)
	}

	want := filepath.Join(baseDir, "game-a", "shot.w480.webp")
	if variant != want {
		t.Fatalf("variant path = %q, want %q", variant, want)
	}
	variantInfo, err := os.Stat(want)
	if err != nil {
		t.Fatalf("variant file missing: %v", err)
	}
	srcInfo, err := os.Stat(filepath.Join(baseDir, "game-a", "shot.jpg"))
	if err != nil {
		t.Fatalf("original missing: %v", err)
	}
	if variantInfo.Size() >= srcInfo.Size() {
		t.Fatalf("variant size %d not smaller than original %d", variantInfo.Size(), srcInfo.Size())
	}
}

func TestAssetStoreEnsureVariantReusesExistingFile(t *testing.T) {
	baseDir := t.TempDir()
	store := NewAssetStore(baseDir)
	writeTestAsset(t, baseDir, "game-a", "shot.png", 1600, 900, png.Encode)

	first, err := store.EnsureVariant("/assets/game-a/shot.png", 480)
	if err != nil {
		t.Fatalf("first EnsureVariant returned error: %v", err)
	}
	firstInfo, err := os.Stat(first)
	if err != nil {
		t.Fatalf("first variant missing: %v", err)
	}
	time.Sleep(20 * time.Millisecond)

	second, err := store.EnsureVariant("/assets/game-a/shot.png", 480)
	if err != nil {
		t.Fatalf("second EnsureVariant returned error: %v", err)
	}
	secondInfo, err := os.Stat(second)
	if err != nil {
		t.Fatalf("second variant missing: %v", err)
	}
	if first != second {
		t.Fatalf("variant path changed: %q vs %q", first, second)
	}
	if !firstInfo.ModTime().Equal(secondInfo.ModTime()) {
		t.Fatalf("variant was regenerated on reuse")
	}
}

func TestAssetStoreEnsureVariantSkipsNarrowerOriginal(t *testing.T) {
	baseDir := t.TempDir()
	store := NewAssetStore(baseDir)
	writeTestAsset(t, baseDir, "game-a", "small.jpg", 400, 300, func(w io.Writer, m image.Image) error {
		return jpeg.Encode(w, m, nil)
	})

	variant, err := store.EnsureVariant("/assets/game-a/small.jpg", 480)
	if err != nil {
		t.Fatalf("EnsureVariant returned error: %v", err)
	}
	if variant != filepath.Join(baseDir, "game-a", "small.jpg") {
		t.Fatalf("narrower original should be served as-is, got %q", variant)
	}
	if _, err := os.Stat(filepath.Join(baseDir, "game-a", "small.w480.webp")); err == nil {
		t.Fatalf("variant should not be generated for narrower original")
	}
}

func TestAssetStoreEnsureVariantSkipsAnimatedGIF(t *testing.T) {
	baseDir := t.TempDir()
	store := NewAssetStore(baseDir)
	writeTestAsset(t, baseDir, "game-a", "anim.gif", 800, 600, func(w io.Writer, m image.Image) error {
		return gif.Encode(w, m, nil)
	})

	variant, err := store.EnsureVariant("/assets/game-a/anim.gif", 480)
	if err != nil {
		t.Fatalf("EnsureVariant returned error: %v", err)
	}
	if variant != filepath.Join(baseDir, "game-a", "anim.gif") {
		t.Fatalf("gif should be served as-is, got %q", variant)
	}
	if _, err := os.Stat(filepath.Join(baseDir, "game-a", "anim.w480.webp")); err == nil {
		t.Fatalf("variant should not be generated for gif")
	}
}

func TestAssetStoreEnsureVariantRejectsInvalidWidth(t *testing.T) {
	baseDir := t.TempDir()
	store := NewAssetStore(baseDir)
	writeTestAsset(t, baseDir, "game-a", "shot.jpg", 1600, 900, func(w io.Writer, m image.Image) error {
		return jpeg.Encode(w, m, nil)
	})

	variant, err := store.EnsureVariant("/assets/game-a/shot.jpg", 1111)
	if err != nil {
		t.Fatalf("EnsureVariant returned error: %v", err)
	}
	if variant != filepath.Join(baseDir, "game-a", "shot.jpg") {
		t.Fatalf("invalid width should be served as-is, got %q", variant)
	}
	if _, err := os.Stat(filepath.Join(baseDir, "game-a", "shot.w1111.webp")); err == nil {
		t.Fatalf("variant should not be generated for invalid width")
	}
}
