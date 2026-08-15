package domain

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

const (
	StartScreenTileSizeSmall StartScreenTileSize = "small"
	StartScreenTileSizeWide  StartScreenTileSize = "wide"
	StartScreenTileSizeLarge StartScreenTileSize = "large"

	// 全屏自定义网格：组（列）只是顶部标签，磁贴在 12 列无限行的自由网格内摆放。
	StartScreenFreeCols = 12
	StartScreenMaxRows  = 200

	// 宽磁贴活磁贴：轮播帧 = image_path 首帧 + flip_images 追加帧，共不超过 4 帧。
	StartScreenMaxFlipImages = 3
)

type StartScreenTileSize string

func IsAllowedStartScreenTileSize(value string) bool {
	switch StartScreenTileSize(value) {
	case StartScreenTileSizeSmall, StartScreenTileSizeWide, StartScreenTileSizeLarge:
		return true
	default:
		return false
	}
}

// AssetPathList 以 JSON 数组形式存储在 TEXT 列（如 start_screen_tiles.flip_images），
// 实现 sql.Scanner / driver.Valuer 供 sqlx 直接读写。
type AssetPathList []string

func (l *AssetPathList) Scan(value any) error {
	if value == nil {
		*l = nil
		return nil
	}
	var raw []byte
	switch v := value.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return fmt.Errorf("scan asset path list from unsupported type %T", value)
	}
	if len(raw) == 0 {
		*l = nil
		return nil
	}
	if err := json.Unmarshal(raw, l); err != nil {
		return fmt.Errorf("unmarshal asset path list: %w", err)
	}
	return nil
}

func (l AssetPathList) Value() (driver.Value, error) {
	if len(l) == 0 {
		return nil, nil
	}
	data, err := json.Marshal(l)
	if err != nil {
		return nil, err
	}
	return string(data), nil
}

type StartScreenTile struct {
	ID          int64         `db:"id"`
	GameID      int64         `db:"game_id"`
	PublicID    string        `db:"public_id"`
	Title       string        `db:"title"`
	TileSize    string        `db:"tile_size"`
	ImagePath   *string       `db:"image_path"`
	FocusX      int           `db:"focus_x"`
	FocusY      int           `db:"focus_y"`
	FlipImages  AssetPathList `db:"flip_images"`
	SortOrder   int           `db:"sort_order"`
	ColumnIndex int           `db:"column_index"`
	GridRow     int           `db:"grid_row"`
	GridCol     int           `db:"grid_col"`
	CreatedAt   string        `db:"created_at"`
	UpdatedAt   string        `db:"updated_at"`
}

type StartScreenTileWrite struct {
	GameID      int64
	TileSize    string
	ImagePath   *string
	FocusX      int
	FocusY      int
	FlipImages  []string
	ColumnIndex int
	GridRow     int
	GridCol     int
}

type StartScreenColumn struct {
	ID        int64  `db:"id"`
	Name      string `db:"name"`
	SortOrder int    `db:"sort_order"`
	CreatedAt string `db:"created_at"`
	UpdatedAt string `db:"updated_at"`
}

type StartScreenColumnWrite struct {
	Name string
}

type StartScreenLayout struct {
	Columns []StartScreenColumn
	Tiles   []StartScreenTile
}
