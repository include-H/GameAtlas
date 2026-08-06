package services

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/textproto"
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
	if _, err := service.Update(nil, []domain.StartScreenTileWrite{{GameID: gameID, TileSize: "small", GridRow: 6}}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("out of bounds row error = %v, want ErrValidation", err)
	}
	if _, err := service.Update(nil, []domain.StartScreenTileWrite{{GameID: gameID, TileSize: "wide", GridCol: 1}}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("wide tile second column error = %v, want ErrValidation", err)
	}
	if _, err := service.Update(nil, []domain.StartScreenTileWrite{{GameID: gameID, TileSize: "large", GridRow: 5}}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("large tile low row error = %v, want ErrValidation", err)
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

func TestStartScreenTilesUpdateMovesStagedImagesToPermanent(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	assetsDir := t.TempDir()
	gameID := insertServicesTestGame(t, db, "tile-stage-move", "Tile Stage Move", domain.GameVisibilityPublic)

	filename := "22222222-2222-4222-8222-222222222222.jpg"
	stagingFile := filepath.Join(assetsDir, "_staging", filename)
	if err := os.MkdirAll(filepath.Dir(stagingFile), 0o755); err != nil {
		t.Fatalf("create staging dir: %v", err)
	}
	if err := os.WriteFile(stagingFile, []byte("fake-image"), 0o644); err != nil {
		t.Fatalf("write staged image: %v", err)
	}

	service := NewStartScreenTilesService(
		repositories.NewStartScreenTilesRepository(db),
		repositories.NewStartScreenColumnsRepository(db),
		repositories.NewGamesRepository(db),
		files.NewAssetStore(assetsDir),
	)

	imagePath := "/assets/start-screen/" + filename
	layout, err := service.Update(nil, []domain.StartScreenTileWrite{{
		GameID:         gameID,
		TileSize:       "small",
		ImageSmallPath: &imagePath,
	}})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if len(layout.Tiles) != 1 || layout.Tiles[0].ImageSmallPath == nil || *layout.Tiles[0].ImageSmallPath != imagePath {
		t.Fatalf("tiles = %+v, want image path %q", layout.Tiles, imagePath)
	}
	if _, err := os.Stat(filepath.Join(assetsDir, "start-screen", filename)); err != nil {
		t.Fatalf("permanent image missing after save: %v", err)
	}
	if _, err := os.Stat(stagingFile); !os.IsNotExist(err) {
		t.Fatalf("staging image still present after save, want moved to permanent")
	}
}

func TestStartScreenTilesUploadTileImageStagesOnly(t *testing.T) {
	assetsDir := t.TempDir()
	service := NewStartScreenTilesService(
		nil,
		nil,
		nil,
		files.NewAssetStore(assetsDir),
	)

	path, err := service.UploadTileImage(buildTileImageUploadHeader(t, "tile.png"))
	if err != nil {
		t.Fatalf("UploadTileImage returned error: %v", err)
	}
	if !strings.HasPrefix(path, "/assets/start-screen/") {
		t.Fatalf("path = %q, want /assets/start-screen/ prefix", path)
	}

	filename := filepath.Base(path)
	if _, err := os.Stat(filepath.Join(assetsDir, "_staging", filename)); err != nil {
		t.Fatalf("staged image missing after upload: %v", err)
	}
	if _, err := os.Stat(filepath.Join(assetsDir, "start-screen", filename)); !os.IsNotExist(err) {
		t.Fatalf("permanent image exists before layout save, want staged only")
	}
}

func buildTileImageUploadHeader(t *testing.T, filename string) *multipart.FileHeader {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	partHeader := textproto.MIMEHeader{}
	partHeader.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	partHeader.Set("Content-Type", "image/png")
	part, err := writer.CreatePart(partHeader)
	if err != nil {
		t.Fatalf("CreatePart returned error: %v", err)
	}
	if _, err := part.Write([]byte("fake-image")); err != nil {
		t.Fatalf("Write file part returned error: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close returned error: %v", err)
	}

	form, err := multipart.NewReader(body, writer.Boundary()).ReadForm(1 << 20)
	if err != nil {
		t.Fatalf("ReadForm returned error: %v", err)
	}
	t.Cleanup(func() { _ = form.RemoveAll() })
	files := form.File["file"]
	if len(files) != 1 {
		t.Fatalf("form file count = %d, want 1", len(files))
	}
	return files[0]
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
	if layout.Tiles[1].ColumnIndex != 0 || layout.Tiles[1].GridRow != 1 || layout.Tiles[1].GridCol != 0 {
		t.Fatalf("appended tile position = %+v, want next free wide cell in first column", layout.Tiles[1])
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
	if added == nil || added.ColumnIndex != 1 || added.GridRow != 0 || added.GridCol != 0 {
		t.Fatalf("appended tile = %+v, want empty middle column first cell", added)
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
