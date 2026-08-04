package services

import (
	"errors"
	"testing"

	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/repositories"
)

func openStartScreenTilesService(t *testing.T) *StartScreenTilesService {
	t.Helper()
	db := openServicesTestDB(t)
	t.Cleanup(func() { _ = db.Close() })
	return NewStartScreenTilesService(
		repositories.NewStartScreenTilesRepository(db),
		repositories.NewGamesRepository(db),
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
