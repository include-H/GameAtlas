package services

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/files"
	"github.com/hao/game/internal/repositories"
)

func openStartScreenTilesService(t *testing.T) *StartScreenTilesService {
	t.Helper()
	db := openServicesTestDB(t)
	t.Cleanup(func() { _ = db.Close() })
	return NewStartScreenTilesService(
		repositories.NewStartScreenTilesRepository(db),
		repositories.NewGamesRepository(db),
		files.NewAssetStore(t.TempDir()),
	)
}

func TestStartScreenTilesListEmpty(t *testing.T) {
	service := openStartScreenTilesService(t)
	tiles, err := service.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(tiles) != 0 {
		t.Fatalf("List = %d tiles, want 0", len(tiles))
	}
}

func TestStartScreenTilesUpdatePersistsOrderAndSizes(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	firstID := insertServicesTestGame(t, db, "tile-a", "Tile A", domain.GameVisibilityPublic)
	secondID := insertServicesTestGame(t, db, "tile-b", "Tile B", domain.GameVisibilityPublic)

	service := NewStartScreenTilesService(
		repositories.NewStartScreenTilesRepository(db),
		repositories.NewGamesRepository(db),
		files.NewAssetStore(t.TempDir()),
	)

	tiles, err := service.Update([]domain.StartScreenTileWrite{
		{GameID: secondID, TileSize: "wide"},
		{GameID: firstID, TileSize: "large"},
	})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if len(tiles) != 2 {
		t.Fatalf("Update returned %d tiles, want 2", len(tiles))
	}
	if tiles[0].GameID != secondID || tiles[0].TileSize != "wide" || tiles[0].SortOrder != 0 {
		t.Fatalf("first tile = %+v, want second game first with wide size", tiles[0])
	}
	if tiles[1].GameID != firstID || tiles[1].TileSize != "large" || tiles[1].SortOrder != 1 {
		t.Fatalf("second tile = %+v, want first game second with large size", tiles[1])
	}
	if tiles[0].PublicID != "tile-b" || tiles[0].Title != "Tile B" {
		t.Fatalf("first tile join = %+v, want tile-b metadata", tiles[0])
	}
}

func TestStartScreenTilesUpdateRejectsInvalidInput(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertServicesTestGame(t, db, "tile-invalid", "Tile Invalid", domain.GameVisibilityPublic)
	service := NewStartScreenTilesService(
		repositories.NewStartScreenTilesRepository(db),
		repositories.NewGamesRepository(db),
		files.NewAssetStore(t.TempDir()),
	)

	if _, err := service.Update([]domain.StartScreenTileWrite{{GameID: gameID, TileSize: "huge"}}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("invalid size error = %v, want ErrValidation", err)
	}
	if _, err := service.Update([]domain.StartScreenTileWrite{
		{GameID: gameID, TileSize: "small"},
		{GameID: gameID, TileSize: "wide"},
	}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("duplicate game error = %v, want ErrValidation", err)
	}
	if _, err := service.Update([]domain.StartScreenTileWrite{{GameID: 999999, TileSize: "small"}}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing game error = %v, want ErrNotFound", err)
	}
}

func TestStartScreenTilesUpdateValidatesTileImages(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	assetsDir := t.TempDir()
	gameID := insertServicesTestGame(t, db, "tile-image", "Tile Image", domain.GameVisibilityPublic)

	imagePath := "/assets/start-screen/11111111-1111-4111-8111-111111111111.jpg"
	imageFile := filepath.Join(assetsDir, "start-screen", "11111111-1111-4111-8111-111111111111.jpg")
	if err := os.MkdirAll(filepath.Dir(imageFile), 0o755); err != nil {
		t.Fatalf("create tile image dir: %v", err)
	}
	if err := os.WriteFile(imageFile, []byte("fake-image"), 0o644); err != nil {
		t.Fatalf("write tile image: %v", err)
	}

	service := NewStartScreenTilesService(
		repositories.NewStartScreenTilesRepository(db),
		repositories.NewGamesRepository(db),
		files.NewAssetStore(assetsDir),
	)

	invalidPath := "/assets/start-screen/missing.jpg"
	if _, err := service.Update([]domain.StartScreenTileWrite{{
		GameID:         gameID,
		TileSize:       "small",
		ImageSmallPath: &invalidPath,
	}}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("missing image error = %v, want ErrValidation", err)
	}

	badPrefix := "/etc/passwd"
	if _, err := service.Update([]domain.StartScreenTileWrite{{
		GameID:         gameID,
		TileSize:       "small",
		ImageSmallPath: &badPrefix,
	}}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("bad prefix error = %v, want ErrValidation", err)
	}

	tiles, err := service.Update([]domain.StartScreenTileWrite{{
		GameID:         gameID,
		TileSize:       "small",
		ImageSmallPath: &imagePath,
	}})
	if err != nil {
		t.Fatalf("valid image update returned error: %v", err)
	}
	if len(tiles) != 1 || tiles[0].ImageSmallPath == nil || *tiles[0].ImageSmallPath != imagePath {
		t.Fatalf("tiles = %+v, want image path %q", tiles, imagePath)
	}
}
