package domain

const (
	StartScreenTileSizeSmall StartScreenTileSize = "small"
	StartScreenTileSizeWide  StartScreenTileSize = "wide"
	StartScreenTileSizeLarge StartScreenTileSize = "large"
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
	ID         int64   `db:"id"`
	GameID     int64   `db:"game_id"`
	PublicID   string  `db:"public_id"`
	Title      string  `db:"title"`
	CoverImage *string `db:"cover_image"`
	TileSize   string  `db:"tile_size"`
	SortOrder  int     `db:"sort_order"`
	CreatedAt  string  `db:"created_at"`
	UpdatedAt  string  `db:"updated_at"`
}

type StartScreenTileWrite struct {
	GameID   int64
	TileSize string
}
