package repositories

import (
	"database/sql"
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

func (r *StartScreenTilesRepository) List(includeAll bool) ([]domain.StartScreenTile, error) {
	where := ""
	if !includeAll {
		where = "WHERE g.visibility = 'public'"
	}
	var tiles []domain.StartScreenTile
	err := r.db.Select(&tiles, `
		SELECT
			t.id, t.game_id, g.public_id, g.title, g.cover_image, g.banner_image,
			t.tile_size, t.image_small_path, t.image_wide_path, t.image_large_path,
			t.sort_order, t.column_index, t.grid_row, t.grid_col,
			t.created_at, t.updated_at
		FROM start_screen_tiles t
		INNER JOIN games g ON g.id = t.game_id
		`+where+`
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
				game_id, tile_size, image_small_path, image_wide_path, image_large_path,
				sort_order, column_index, grid_row, grid_col
			)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, tile.GameID, tile.TileSize, tile.ImageSmallPath, tile.ImageWidePath, tile.ImageLargePath,
			index, tile.ColumnIndex, tile.GridRow, tile.GridCol); err != nil {
			return fmt.Errorf("insert start screen tile: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit start screen tiles replace: %w", err)
	}
	return nil
}

// RemoveByGameID 删除指定游戏的磁贴；不存在时返回 false。
func (r *StartScreenTilesRepository) RemoveByGameID(gameID int64) (bool, error) {
	result, err := r.db.Exec(`DELETE FROM start_screen_tiles WHERE game_id = ?`, gameID)
	if err != nil {
		return false, fmt.Errorf("remove start screen tile: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read removed start screen tile rows: %w", err)
	}
	return rows > 0, nil
}

// Append 以新的 sort_order 追加一个磁贴，位置由服务层计算后写入；game_id 已存在时不做任何修改并返回 false。
func (r *StartScreenTilesRepository) Append(tile domain.StartScreenTileWrite) (bool, error) {
	var maxOrder sql.NullInt64
	if err := r.db.Get(&maxOrder, `SELECT MAX(sort_order) FROM start_screen_tiles`); err != nil {
		return false, fmt.Errorf("read max start screen tile order: %w", err)
	}
	nextOrder := 0
	if maxOrder.Valid {
		nextOrder = int(maxOrder.Int64) + 1
	}

	result, err := r.db.Exec(`
		INSERT OR IGNORE INTO start_screen_tiles (
			game_id, tile_size, image_small_path, image_wide_path, image_large_path,
			sort_order, column_index, grid_row, grid_col
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, tile.GameID, tile.TileSize, tile.ImageSmallPath, tile.ImageWidePath, tile.ImageLargePath,
		nextOrder, tile.ColumnIndex, tile.GridRow, tile.GridCol)
	if err != nil {
		return false, fmt.Errorf("append start screen tile: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read appended start screen tile rows: %w", err)
	}
	return rows > 0, nil
}
