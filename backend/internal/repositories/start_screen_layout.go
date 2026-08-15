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
				game_id, tile_size, image_path, focus_x, focus_y, flip_images,
				sort_order, column_index, grid_row, grid_col
			)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, tile.GameID, tile.TileSize, tile.ImagePath, tile.FocusX, tile.FocusY,
			domain.AssetPathList(tile.FlipImages),
			index, tile.ColumnIndex, tile.GridRow, tile.GridCol); err != nil {
			return err
		}
	}
	return nil
}
