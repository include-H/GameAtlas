package repositories

import (
	"testing"

	"github.com/hao/game/internal/domain"
)

func TestStartScreenTilesRepositoryGetGameVisibilityByImagePath(t *testing.T) {
	db := openRepositoryTestDB(t)
	defer func() { _ = db.Close() }()

	publicID := insertRepositoryGame(t, db, "tile-visibility-public", "Tile Visibility Public", domain.GameVisibilityPublic)
	privateID := insertRepositoryGame(t, db, "tile-visibility-private", "Tile Visibility Private", domain.GameVisibilityPrivate)

	if _, err := db.Exec(`
		INSERT INTO start_screen_tiles (
			game_id, tile_size, image_small_path, image_wide_path, image_large_path,
			sort_order, column_index, grid_row, grid_col
		)
		VALUES
			(?, 'small', '/assets/start-screen/public-small.png', '/assets/start-screen/public-wide.png', '/assets/start-screen/public-large.png', 0, 0, 0, 0),
			(?, 'small', '/assets/start-screen/private-small.png', NULL, NULL, 1, 0, 0, 0)
	`, publicID, privateID); err != nil {
		t.Fatalf("insert start screen tiles: %v", err)
	}

	repo := NewStartScreenTilesRepository(db)
	for _, tc := range []struct {
		path string
		want string
	}{
		{path: "/assets/start-screen/public-small.png", want: domain.GameVisibilityPublic},
		{path: "/assets/start-screen/public-wide.png", want: domain.GameVisibilityPublic},
		{path: "/assets/start-screen/public-large.png", want: domain.GameVisibilityPublic},
		{path: "/assets/start-screen/private-small.png", want: domain.GameVisibilityPrivate},
	} {
		got, err := repo.GetGameVisibilityByImagePath(tc.path)
		if err != nil {
			t.Fatalf("GetGameVisibilityByImagePath(%q) returned error: %v", tc.path, err)
		}
		if got != tc.want {
			t.Fatalf("GetGameVisibilityByImagePath(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}

	if _, err := repo.GetGameVisibilityByImagePath("/assets/start-screen/missing.png"); err == nil {
		t.Fatal("GetGameVisibilityByImagePath returned nil error for missing image path")
	}
}
