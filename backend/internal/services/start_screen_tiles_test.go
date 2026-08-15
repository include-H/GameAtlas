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
	layout, err := service.List(true)
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
		[]domain.StartScreenColumnWrite{{Name: "第一列"}, {Name: "第二列"}},
		[]domain.StartScreenTileWrite{
			{GameID: secondID, TileSize: "wide", ColumnIndex: 1, GridRow: 3},
			{GameID: firstID, TileSize: "large", ColumnIndex: 0, GridRow: 2},
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
	if layout.Tiles[0].ColumnIndex != 1 || layout.Tiles[0].GridRow != 3 || layout.Tiles[0].GridCol != 0 {
		t.Fatalf("first tile position = %+v, want column 1 row 3 col 0", layout.Tiles[0])
	}
	if layout.Tiles[1].GameID != firstID || layout.Tiles[1].TileSize != "large" || layout.Tiles[1].SortOrder != 1 {
		t.Fatalf("second tile = %+v, want first game second with large size", layout.Tiles[1])
	}
	if layout.Tiles[1].ColumnIndex != 0 || layout.Tiles[1].GridRow != 2 || layout.Tiles[1].GridCol != 0 {
		t.Fatalf("second tile position = %+v, want column 0 row 2 col 0", layout.Tiles[1])
	}
	if layout.Tiles[0].PublicID != "tile-b" || layout.Tiles[0].Title != "Tile B" {
		t.Fatalf("first tile join = %+v, want tile-b metadata", layout.Tiles[0])
	}
	if len(layout.Columns) != 2 || layout.Columns[0].Name != "第一列" || layout.Columns[1].Name != "第二列" {
		t.Fatalf("columns = %+v, want two columns with names", layout.Columns)
	}
}

func TestStartScreenTilesUpdateKeepsEmptyColumns(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertServicesTestGame(t, db, "tile-empty-column", "Tile Empty Column", domain.GameVisibilityPublic)
	service := NewStartScreenTilesService(
		repositories.NewStartScreenTilesRepository(db),
		repositories.NewStartScreenColumnsRepository(db),
		repositories.NewGamesRepository(db),
		files.NewAssetStore(t.TempDir()),
	)

	layout, err := service.Update(
		[]domain.StartScreenColumnWrite{{Name: "第一列"}, {Name: ""}, {Name: "第三列"}},
		[]domain.StartScreenTileWrite{{GameID: gameID, TileSize: "small", ColumnIndex: 2}},
	)
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if len(layout.Columns) != 3 || layout.Columns[1].Name != "" || layout.Columns[2].Name != "第三列" {
		t.Fatalf("columns = %+v, want empty middle column preserved", layout.Columns)
	}
	if len(layout.Tiles) != 1 || layout.Tiles[0].ColumnIndex != 2 {
		t.Fatalf("tiles = %+v, want tile in column 2", layout.Tiles)
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
	if _, err := service.Update(nil, []domain.StartScreenTileWrite{{GameID: gameID, TileSize: "small", GridRow: 199}}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("out of bounds row error = %v, want ErrValidation", err)
	}
	if _, err := service.Update(nil, []domain.StartScreenTileWrite{{GameID: gameID, TileSize: "wide", GridCol: 9}}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("wide tile out of bounds col error = %v, want ErrValidation", err)
	}
	if _, err := service.Update(nil, []domain.StartScreenTileWrite{{GameID: gameID, TileSize: "large", GridRow: 197}}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("large tile low row error = %v, want ErrValidation", err)
	}
}

func TestStartScreenTilesUpdateValidatesTileImages(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	assetsDir := t.TempDir()
	gameID := insertServicesTestGame(t, db, "tile-image", "Tile Image", domain.GameVisibilityPublic)

	imagePath := "/assets/tile-image/11111111-1111-4111-8111-111111111111.jpg"
	imageFile := filepath.Join(assetsDir, "tile-image", "11111111-1111-4111-8111-111111111111.jpg")
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

	invalidPath := "/assets/tile-image/missing.jpg"
	if _, err := service.Update(nil, []domain.StartScreenTileWrite{{
		GameID:    gameID,
		TileSize:  "small",
		ImagePath: &invalidPath,
	}}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("missing image error = %v, want ErrValidation", err)
	}

	badPrefix := "/etc/passwd"
	if _, err := service.Update(nil, []domain.StartScreenTileWrite{{
		GameID:    gameID,
		TileSize:  "small",
		ImagePath: &badPrefix,
	}}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("bad prefix error = %v, want ErrValidation", err)
	}

	// 引用其他游戏的素材：不属于本游戏，拒绝。
	otherImage := "/assets/tile-other/99999999-9999-4999-8999-999999999999.jpg"
	otherFile := filepath.Join(assetsDir, "tile-other", "99999999-9999-4999-8999-999999999999.jpg")
	if err := os.MkdirAll(filepath.Dir(otherFile), 0o755); err != nil {
		t.Fatalf("create other image dir: %v", err)
	}
	if err := os.WriteFile(otherFile, []byte("fake-image"), 0o644); err != nil {
		t.Fatalf("write other image: %v", err)
	}
	if _, err := service.Update(nil, []domain.StartScreenTileWrite{{
		GameID:    gameID,
		TileSize:  "small",
		ImagePath: &otherImage,
	}}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("foreign image error = %v, want ErrValidation", err)
	}

	// 旧版裁剪图路径（/assets/start-screen/…）不是本游戏素材，必须拒绝。
	legacyPath := "/assets/start-screen/55555555-5555-4555-8555-555555555555.jpg"
	if _, err := service.Update(nil, []domain.StartScreenTileWrite{{
		GameID:    gameID,
		TileSize:  "small",
		ImagePath: &legacyPath,
	}}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("legacy crop path error = %v, want ErrValidation", err)
	}

	layout, err := service.Update(nil, []domain.StartScreenTileWrite{{
		GameID:    gameID,
		TileSize:  "small",
		ImagePath: &imagePath,
		FocusX:    35,
		FocusY:    62,
	}})
	if err != nil {
		t.Fatalf("valid image update returned error: %v", err)
	}
	if len(layout.Tiles) != 1 || layout.Tiles[0].ImagePath == nil || *layout.Tiles[0].ImagePath != imagePath {
		t.Fatalf("tiles = %+v, want image path %q", layout.Tiles, imagePath)
	}
	if layout.Tiles[0].FocusX != 35 || layout.Tiles[0].FocusY != 62 {
		t.Fatalf("focus = %d/%d, want 35/62", layout.Tiles[0].FocusX, layout.Tiles[0].FocusY)
	}

	if _, err := service.Update(nil, []domain.StartScreenTileWrite{{
		GameID:    gameID,
		TileSize:  "small",
		ImagePath: &imagePath,
		FocusX:    101,
	}}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("focus out of range error = %v, want ErrValidation", err)
	}
	if _, err := service.Update(nil, []domain.StartScreenTileWrite{{
		GameID:    gameID,
		TileSize:  "small",
		ImagePath: &imagePath,
		FocusY:    -1,
	}}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("focus out of range error = %v, want ErrValidation", err)
	}
}

func TestStartScreenTilesUpdateRollsBackColumnsWhenTileWriteFails(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	firstID := insertServicesTestGame(t, db, "tile-atomic-a", "Tile Atomic A", domain.GameVisibilityPublic)
	secondID := insertServicesTestGame(t, db, "tile-atomic-b", "Tile Atomic B", domain.GameVisibilityPublic)
	service := NewStartScreenTilesService(
		repositories.NewStartScreenTilesRepository(db),
		repositories.NewStartScreenColumnsRepository(db),
		repositories.NewGamesRepository(db),
		files.NewAssetStore(t.TempDir()),
	)

	if _, err := service.Update(
		[]domain.StartScreenColumnWrite{{Name: "旧列"}},
		[]domain.StartScreenTileWrite{{GameID: firstID, TileSize: "small"}},
	); err != nil {
		t.Fatalf("initial Update returned error: %v", err)
	}
	if _, err := db.Exec(`
		CREATE TRIGGER fail_start_screen_tile_insert
		BEFORE INSERT ON start_screen_tiles
		BEGIN
			SELECT RAISE(ABORT, 'test tile insert failure');
		END
	`); err != nil {
		t.Fatalf("create failure trigger: %v", err)
	}
	defer func() { _, _ = db.Exec(`DROP TRIGGER fail_start_screen_tile_insert`) }()

	if _, err := service.Update(
		[]domain.StartScreenColumnWrite{{Name: "新列"}},
		[]domain.StartScreenTileWrite{{GameID: secondID, TileSize: "wide"}},
	); err == nil {
		t.Fatal("Update returned nil error with forced tile insert failure")
	}

	layout, err := service.List(true)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(layout.Columns) != 1 || layout.Columns[0].Name != "旧列" {
		t.Fatalf("columns after failed update = %+v, want old layout", layout.Columns)
	}
	if len(layout.Tiles) != 1 || layout.Tiles[0].GameID != firstID {
		t.Fatalf("tiles after failed update = %+v, want old layout", layout.Tiles)
	}
}

func TestStartScreenTilesUpdateValidatesFlipImages(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	assetsDir := t.TempDir()
	gameID := insertServicesTestGame(t, db, "tile-flip", "Tile Flip", domain.GameVisibilityPublic)
	writeServicesAssetFile(t, assetsDir, "tile-flip", "first.jpg", []byte("first"))
	writeServicesAssetFile(t, assetsDir, "tile-flip", "flip-a.jpg", []byte("a"))
	writeServicesAssetFile(t, assetsDir, "tile-flip", "flip-b.jpg", []byte("b"))

	service := NewStartScreenTilesService(
		repositories.NewStartScreenTilesRepository(db),
		repositories.NewStartScreenColumnsRepository(db),
		repositories.NewGamesRepository(db),
		files.NewAssetStore(assetsDir),
	)

	first := "/assets/tile-flip/first.jpg"
	flipA := "/assets/tile-flip/flip-a.jpg"
	flipB := "/assets/tile-flip/flip-b.jpg"

	// 轮播帧依赖首帧 image_path。
	if _, err := service.Update(nil, []domain.StartScreenTileWrite{{
		GameID:     gameID,
		TileSize:   "wide",
		FlipImages: []string{flipA},
	}}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("flip without image_path error = %v, want ErrValidation", err)
	}

	// 超过上限（3 张追加帧）。
	if _, err := service.Update(nil, []domain.StartScreenTileWrite{{
		GameID:      gameID,
		TileSize:    "wide",
		ImagePath:   &first,
		FlipImages:  []string{flipA, flipB, flipA, flipB},
	}}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("too many flip images error = %v, want ErrValidation", err)
	}

	// 与首帧重复。
	if _, err := service.Update(nil, []domain.StartScreenTileWrite{{
		GameID:      gameID,
		TileSize:    "wide",
		ImagePath:   &first,
		FlipImages:  []string{first},
	}}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("duplicate with image_path error = %v, want ErrValidation", err)
	}

	// 非本游戏素材。
	foreignPath := "/assets/tile-other/flip.jpg"
	writeServicesAssetFile(t, assetsDir, "tile-other", "flip.jpg", []byte("other"))
	if _, err := service.Update(nil, []domain.StartScreenTileWrite{{
		GameID:      gameID,
		TileSize:    "wide",
		ImagePath:   &first,
		FlipImages:  []string{foreignPath},
	}}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("foreign flip image error = %v, want ErrValidation", err)
	}

	// 合法轮播帧保存后按序返回。
	layout, err := service.Update(nil, []domain.StartScreenTileWrite{{
		GameID:      gameID,
		TileSize:    "wide",
		ImagePath:   &first,
		FlipImages:  []string{flipA, flipB},
	}})
	if err != nil {
		t.Fatalf("valid flip update returned error: %v", err)
	}
	if len(layout.Tiles) != 1 {
		t.Fatalf("tiles = %d, want 1", len(layout.Tiles))
	}
	got := layout.Tiles[0]
	if got.ImagePath == nil || *got.ImagePath != first {
		t.Fatalf("image_path = %v, want %q", got.ImagePath, first)
	}
	if len(got.FlipImages) != 2 || got.FlipImages[0] != flipA || got.FlipImages[1] != flipB {
		t.Fatalf("flip_images = %v, want [%q %q]", got.FlipImages, flipA, flipB)
	}
}

func TestStartScreenTilesAddTileDefaultsToFirstScreenshot(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	gameID := insertServicesTestGame(t, db, "tile-default-shot", "Tile Default Shot", domain.GameVisibilityPublic)
	insertServicesGameAsset(t, db, gameID, "shot-1", "screenshot", "/assets/tile-default-shot/first.jpg", 1)
	insertServicesGameAsset(t, db, gameID, "shot-2", "screenshot", "/assets/tile-default-shot/second.jpg", 2)
	if _, err := db.Exec(`UPDATE games SET cover_image = ? WHERE id = ?`, "/assets/tile-default-shot/cover.jpg", gameID); err != nil {
		t.Fatalf("set cover: %v", err)
	}

	service := NewStartScreenTilesService(
		repositories.NewStartScreenTilesRepository(db),
		repositories.NewStartScreenColumnsRepository(db),
		repositories.NewGamesRepository(db),
		files.NewAssetStore(t.TempDir()),
	)

	layout, err := service.AddTile(domain.StartScreenTileWrite{GameID: gameID, TileSize: "small"})
	if err != nil {
		t.Fatalf("AddTile returned error: %v", err)
	}
	if len(layout.Tiles) != 1 || layout.Tiles[0].ImagePath == nil {
		t.Fatalf("tiles = %+v, want one tile with default image", layout.Tiles)
	}
	if *layout.Tiles[0].ImagePath != "/assets/tile-default-shot/first.jpg" {
		t.Fatalf("default image = %q, want first screenshot", *layout.Tiles[0].ImagePath)
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
	// 12 列网格：small（2x2）占 col0-1，wide（2x4）就近放在同行 col2-5。
	if layout.Tiles[1].ColumnIndex != 0 || layout.Tiles[1].GridRow != 0 || layout.Tiles[1].GridCol != 2 {
		t.Fatalf("appended tile position = %+v, want next free wide cell in the group", layout.Tiles[1])
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

func TestStartScreenTilesAddTileFillsEmptyColumnFirst(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	firstID := insertServicesTestGame(t, db, "tile-empty-add-a", "Tile Empty Add A", domain.GameVisibilityPublic)
	secondID := insertServicesTestGame(t, db, "tile-empty-add-b", "Tile Empty Add B", domain.GameVisibilityPublic)
	thirdID := insertServicesTestGame(t, db, "tile-empty-add-c", "Tile Empty Add C", domain.GameVisibilityPublic)
	fourthID := insertServicesTestGame(t, db, "tile-empty-add-d", "Tile Empty Add D", domain.GameVisibilityPublic)
	fifthID := insertServicesTestGame(t, db, "tile-empty-add-e", "Tile Empty Add E", domain.GameVisibilityPublic)

	service := NewStartScreenTilesService(
		repositories.NewStartScreenTilesRepository(db),
		repositories.NewStartScreenColumnsRepository(db),
		repositories.NewGamesRepository(db),
		files.NewAssetStore(t.TempDir()),
	)

	if _, err := service.Update(
		[]domain.StartScreenColumnWrite{{Name: "第一列"}, {Name: ""}, {Name: "第三列"}},
		[]domain.StartScreenTileWrite{
			{GameID: thirdID, TileSize: "large", ColumnIndex: 0},
			{GameID: fourthID, TileSize: "large", ColumnIndex: 0, GridRow: 2},
			{GameID: fifthID, TileSize: "large", ColumnIndex: 0, GridRow: 4},
			{GameID: firstID, TileSize: "small", ColumnIndex: 2},
		},
	); err != nil {
		t.Fatalf("initial Update returned error: %v", err)
	}

	layout, err := service.AddTile(domain.StartScreenTileWrite{GameID: secondID, TileSize: "wide"})
	if err != nil {
		t.Fatalf("AddTile returned error: %v", err)
	}
	if len(layout.Tiles) != 5 {
		t.Fatalf("AddTile returned %d tiles, want 5", len(layout.Tiles))
	}
	var added *domain.StartScreenTile
	for index := range layout.Tiles {
		if layout.Tiles[index].GameID == secondID {
			added = &layout.Tiles[index]
			break
		}
	}
	// 12 列网格：组 0 前三行被三个 large（4x4）占满 col0-3，wide 就近落在同行 col4-7，
	// 不再需要等"整列放满"才开新列。
	if added == nil || added.ColumnIndex != 0 || added.GridRow != 0 || added.GridCol != 4 {
		t.Fatalf("appended tile = %+v, want nearest free cell in the group", added)
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

func TestStartScreenTilesRemoveTile(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	firstID := insertServicesTestGame(t, db, "tile-rm-a", "Tile Remove A", domain.GameVisibilityPublic)
	secondID := insertServicesTestGame(t, db, "tile-rm-b", "Tile Remove B", domain.GameVisibilityPublic)

	service := NewStartScreenTilesService(
		repositories.NewStartScreenTilesRepository(db),
		repositories.NewStartScreenColumnsRepository(db),
		repositories.NewGamesRepository(db),
		files.NewAssetStore(t.TempDir()),
	)

	if _, err := service.Update(
		[]domain.StartScreenColumnWrite{{Name: "第一列"}},
		[]domain.StartScreenTileWrite{
			{GameID: firstID, TileSize: "small"},
			{GameID: secondID, TileSize: "wide"},
		},
	); err != nil {
		t.Fatalf("initial Update returned error: %v", err)
	}

	layout, err := service.RemoveTile(firstID)
	if err != nil {
		t.Fatalf("RemoveTile returned error: %v", err)
	}
	if len(layout.Tiles) != 1 || layout.Tiles[0].GameID != secondID {
		t.Fatalf("after RemoveTile = %d tiles, want only second game", len(layout.Tiles))
	}

	// 移除不存在的磁贴：幂等，返回当前布局。
	layout, err = service.RemoveTile(firstID)
	if err != nil {
		t.Fatalf("idempotent RemoveTile returned error: %v", err)
	}
	if len(layout.Tiles) != 1 {
		t.Fatalf("idempotent RemoveTile returned %d tiles, want 1", len(layout.Tiles))
	}

	if _, err := service.RemoveTile(0); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("invalid RemoveTile error = %v, want ErrValidation", err)
	}
}

func TestStartScreenTilesListHidesPrivateGamesForPublicCallers(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	publicID := insertServicesTestGame(t, db, "tile-public", "Tile Public", domain.GameVisibilityPublic)
	privateID := insertServicesTestGame(t, db, "tile-private", "Tile Private", domain.GameVisibilityPrivate)

	service := NewStartScreenTilesService(
		repositories.NewStartScreenTilesRepository(db),
		repositories.NewStartScreenColumnsRepository(db),
		repositories.NewGamesRepository(db),
		files.NewAssetStore(t.TempDir()),
	)
	if _, err := service.Update(
		[]domain.StartScreenColumnWrite{{Name: "列"}},
		[]domain.StartScreenTileWrite{
			{GameID: publicID, TileSize: "small"},
			{GameID: privateID, TileSize: "small"},
		},
	); err != nil {
		t.Fatalf("Update returned error: %v", err)
	}

	publicLayout, err := service.List(false)
	if err != nil {
		t.Fatalf("public List returned error: %v", err)
	}
	if len(publicLayout.Tiles) != 1 || publicLayout.Tiles[0].GameID != publicID {
		t.Fatalf("public tiles = %+v, want only the public game", publicLayout.Tiles)
	}

	adminLayout, err := service.List(true)
	if err != nil {
		t.Fatalf("admin List returned error: %v", err)
	}
	if len(adminLayout.Tiles) != 2 {
		t.Fatalf("admin tiles = %d, want 2", len(adminLayout.Tiles))
	}
}
