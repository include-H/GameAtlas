package repositories

import (
	"database/sql"

	"github.com/hao/game/internal/domain"
)

type startScreenLayoutExecutor interface {
	Exec(query string, args ...any) (sql.Result, error)
}

func replaceStartScreenColumns(exec startScreenLayoutExecutor, columns []domain.StartScreenColumnWrite) error {
	if _, err := exec.Exec(`DELETE FROM start_screen_columns`); err != nil {
		return err
	}

	for index, column := range columns {
		if _, err := exec.Exec(`
			INSERT INTO start_screen_columns (name, sort_order)
			VALUES (?, ?)
		`, column.Name, index); err != nil {
			return err
		}
	}
	return nil
}

func replaceStartScreenTiles(exec startScreenLayoutExecutor, tiles []domain.StartScreenTileWrite) error {
	if _, err := exec.Exec(`DELETE FROM start_screen_tiles`); err != nil {
		return err
	}

	for index, tile := range tiles {
		if _, err := exec.Exec(`
			INSERT INTO start_screen_tiles (
				game_id, tile_size, image_small_path, image_wide_path, image_large_path,
				sort_order, column_index, grid_row, grid_col
			)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, tile.GameID, tile.TileSize, tile.ImageSmallPath, tile.ImageWidePath, tile.ImageLargePath,
			index, tile.ColumnIndex, tile.GridRow, tile.GridCol); err != nil {
			return err
		}
	}
	return nil
}
