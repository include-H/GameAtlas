package domain

const (
	StartScreenTileSizeSmall StartScreenTileSize = "small"
	StartScreenTileSizeWide  StartScreenTileSize = "wide"
	StartScreenTileSizeLarge StartScreenTileSize = "large"

	// 全屏自定义网格：组（列）只是顶部标签，磁贴在 12 列无限行的自由网格内摆放。
	StartScreenFreeCols = 12
	StartScreenMaxRows  = 200
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

type StartScreenTile struct {
	ID             int64   `db:"id"`
	GameID         int64   `db:"game_id"`
	PublicID       string  `db:"public_id"`
	Title          string  `db:"title"`
	CoverImage     *string `db:"cover_image"`
	BannerImage    *string `db:"banner_image"`
	TileSize       string  `db:"tile_size"`
	ImageSmallPath *string `db:"image_small_path"`
	ImageWidePath  *string `db:"image_wide_path"`
	ImageLargePath *string `db:"image_large_path"`
	SortOrder      int     `db:"sort_order"`
	ColumnIndex    int     `db:"column_index"`
	GridRow        int     `db:"grid_row"`
	GridCol        int     `db:"grid_col"`
	CreatedAt      string  `db:"created_at"`
	UpdatedAt      string  `db:"updated_at"`
}

type StartScreenTileWrite struct {
	GameID         int64
	TileSize       string
	ImageSmallPath *string
	ImageWidePath  *string
	ImageLargePath *string
	ColumnIndex    int
	GridRow        int
	GridCol        int
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
