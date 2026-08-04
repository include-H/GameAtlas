package repositories

import (
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/hao/game/internal/domain"
)

type StartScreenTilesRepository struct {
	db *sqlx.DB
}

func NewStartScreenTilesRepository(db *sqlx.DB) *StartScreenTilesRepository {
	return &StartScreenTilesRepository{db: db}
}

func (r *StartScreenTilesRepository) List() ([]domain.StartScreenTile, error) {
	var tiles []domain.StartScreenTile
	err := r.db.Select(&tiles, `
		SELECT
			t.id, t.game_id, g.public_id, g.title, g.cover_image, g.banner_image,
			t.tile_size, t.image_small_path, t.image_wide_path, t.image_large_path,
			t.sort_order, t.created_at, t.updated_at
		FROM start_screen_tiles t
		INNER JOIN games g ON g.id = t.game_id
		ORDER BY t.sort_order ASC, t.id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list start screen tiles: %w", err)
	}
	if tiles == nil {
		return []domain.StartScreenTile{}, nil
	}
	return tiles, nil
}

// Replace 是全量替换：sort_order 按传入顺序写入。单用户场景下无并发编辑，直接清空重建。
func (r *StartScreenTilesRepository) Replace(tiles []domain.StartScreenTileWrite) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return fmt.Errorf("begin start screen tiles replace: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM start_screen_tiles`); err != nil {
		return fmt.Errorf("clear start screen tiles: %w", err)
	}

	for index, tile := range tiles {
		if _, err := tx.Exec(`
			INSERT INTO start_screen_tiles (
				game_id, tile_size, image_small_path, image_wide_path, image_large_path, sort_order
			)
			VALUES (?, ?, ?, ?, ?, ?)
		`, tile.GameID, tile.TileSize, tile.ImageSmallPath, tile.ImageWidePath, tile.ImageLargePath, index); err != nil {
			return fmt.Errorf("insert start screen tile: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit start screen tiles replace: %w", err)
	}
	return nil
}
