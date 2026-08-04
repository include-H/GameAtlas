package services

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
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
		repositories.NewStartScreenColumnsRepository(db),
		repositories.NewGamesRepository(db),
		files.NewAssetStore(t.TempDir()),
	)
}

func TestStartScreenTilesListEmpty(t *testing.T) {
	service := openStartScreenTilesService(t)
	layout, err := service.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(layout.Tiles) != 0 || len(layout.Columns) != 0 {
		t.Fatalf("List = %d tiles / %d columns, want empty", len(layout.Tiles), len(layout.Columns))
	}
}

func TestStartScreenTilesUpdatePersistsOrderAndSizes(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	firstID := insertServicesTestGame(t, db, "tile-a", "Tile A", domain.GameVisibilityPublic)
	secondID := insertServicesTestGame(t, db, "tile-b", "Tile B", domain.GameVisibilityPublic)

	service := NewStartScreenTilesService(
		repositories.NewStartScreenTilesRepository(db),
		repositories.NewStartScreenColumnsRepository(db),
		repositories.NewGamesRepository(db),
		files.NewAssetStore(t.TempDir()),
	)

	layout, err := service.Update(
		[]domain.StartScreenColumnWrite{{Name: "第一列"}},
		[]domain.StartScreenTileWrite{
		{GameID: secondID, TileSize: "wide"},
		{GameID: firstID, TileSize: "large"},
		},
	)
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if len(layout.Tiles) != 2 {
		t.Fatalf("Update returned %d tiles, want 2", len(layout.Tiles))
	}
	if layout.Tiles[0].GameID != secondID || layout.Tiles[0].TileSize != "wide" || layout.Tiles[0].SortOrder != 0 {
		t.Fatalf("first tile = %+v, want second game first with wide size", layout.Tiles[0])
	}
	if layout.Tiles[1].GameID != firstID || layout.Tiles[1].TileSize != "large" || layout.Tiles[1].SortOrder != 1 {
		t.Fatalf("second tile = %+v, want first game second with large size", layout.Tiles[1])
	}
	if layout.Tiles[0].PublicID != "tile-b" || layout.Tiles[0].Title != "Tile B" {
		t.Fatalf("first tile join = %+v, want tile-b metadata", layout.Tiles[0])
	}
	if len(layout.Columns) != 1 || layout.Columns[0].Name != "第一列" {
		t.Fatalf("columns = %+v, want one column named 第一列", layout.Columns)
	}
}

func TestStartScreenTilesUpdateRejectsInvalidInput(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertServicesTestGame(t, db, "tile-invalid", "Tile Invalid", domain.GameVisibilityPublic)
	service := NewStartScreenTilesService(
		repositories.NewStartScreenTilesRepository(db),
		repositories.NewStartScreenColumnsRepository(db),
		repositories.NewGamesRepository(db),
		files.NewAssetStore(t.TempDir()),
	)

	if _, err := service.Update(nil, []domain.StartScreenTileWrite{{GameID: gameID, TileSize: "huge"}}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("invalid size error = %v, want ErrValidation", err)
	}
	if _, err := service.Update(nil, []domain.StartScreenTileWrite{
		{GameID: gameID, TileSize: "small"},
		{GameID: gameID, TileSize: "wide"},
	}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("duplicate game error = %v, want ErrValidation", err)
	}
	if _, err := service.Update(nil, []domain.StartScreenTileWrite{{GameID: 999999, TileSize: "small"}}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing game error = %v, want ErrNotFound", err)
	}
	if _, err := service.Update([]domain.StartScreenColumnWrite{{Name: strings.Repeat("名", 31)}}, nil); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("long column name error = %v, want ErrValidation", err)
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
		repositories.NewStartScreenColumnsRepository(db),
		repositories.NewGamesRepository(db),
		files.NewAssetStore(assetsDir),
	)

	invalidPath := "/assets/start-screen/missing.jpg"
	if _, err := service.Update(nil, []domain.StartScreenTileWrite{{
		GameID:         gameID,
		TileSize:       "small",
		ImageSmallPath: &invalidPath,
	}}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("missing image error = %v, want ErrValidation", err)
	}

	badPrefix := "/etc/passwd"
	if _, err := service.Update(nil, []domain.StartScreenTileWrite{{
		GameID:         gameID,
		TileSize:       "small",
		ImageSmallPath: &badPrefix,
	}}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("bad prefix error = %v, want ErrValidation", err)
	}

	layout, err := service.Update(nil, []domain.StartScreenTileWrite{{
		GameID:         gameID,
		TileSize:       "small",
		ImageSmallPath: &imagePath,
	}})
	if err != nil {
		t.Fatalf("valid image update returned error: %v", err)
	}
	if len(layout.Tiles) != 1 || layout.Tiles[0].ImageSmallPath == nil || *layout.Tiles[0].ImageSmallPath != imagePath {
		t.Fatalf("tiles = %+v, want image path %q", layout.Tiles, imagePath)
	}
}

func TestStartScreenTilesAddTileAppendsAtEnd(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	firstID := insertServicesTestGame(t, db, "tile-add-a", "Tile Add A", domain.GameVisibilityPublic)
	secondID := insertServicesTestGame(t, db, "tile-add-b", "Tile Add B", domain.GameVisibilityPublic)

	service := NewStartScreenTilesService(
		repositories.NewStartScreenTilesRepository(db),
		repositories.NewStartScreenColumnsRepository(db),
		repositories.NewGamesRepository(db),
		files.NewAssetStore(t.TempDir()),
	)

	if _, err := service.Update(
		[]domain.StartScreenColumnWrite{{Name: "第一列"}},
		[]domain.StartScreenTileWrite{{GameID: firstID, TileSize: "small"}},
	); err != nil {
		t.Fatalf("initial Update returned error: %v", err)
	}

	layout, err := service.AddTile(domain.StartScreenTileWrite{GameID: secondID, TileSize: "wide"})
	if err != nil {
		t.Fatalf("AddTile returned error: %v", err)
	}
	if len(layout.Tiles) != 2 {
		t.Fatalf("AddTile returned %d tiles, want 2", len(layout.Tiles))
	}
	if layout.Tiles[1].GameID != secondID || layout.Tiles[1].TileSize != "wide" || layout.Tiles[1].SortOrder != 1 {
		t.Fatalf("appended tile = %+v, want second game at the end", layout.Tiles[1])
	}

	// 重复添加同一游戏：幂等，不产生重复磁贴。
	layout, err = service.AddTile(domain.StartScreenTileWrite{GameID: secondID, TileSize: "large"})
	if err != nil {
		t.Fatalf("duplicate AddTile returned error: %v", err)
	}
	if len(layout.Tiles) != 2 {
		t.Fatalf("duplicate AddTile returned %d tiles, want 2", len(layout.Tiles))
	}
}

func TestStartScreenTilesAddTileRejectsInvalidInput(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	service := NewStartScreenTilesService(
		repositories.NewStartScreenTilesRepository(db),
		repositories.NewStartScreenColumnsRepository(db),
		repositories.NewGamesRepository(db),
		files.NewAssetStore(t.TempDir()),
	)

	if _, err := service.AddTile(domain.StartScreenTileWrite{GameID: 1, TileSize: "huge"}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("invalid size error = %v, want ErrValidation", err)
	}
	if _, err := service.AddTile(domain.StartScreenTileWrite{GameID: 999999, TileSize: "small"}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing game error = %v, want ErrNotFound", err)
	}
}
