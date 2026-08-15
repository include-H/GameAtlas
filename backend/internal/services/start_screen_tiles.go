package services

import (
	"strings"

	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/files"
	"github.com/hao/game/internal/repositories"
)

type StartScreenTilesService struct {
	tilesRepo   *repositories.StartScreenTilesRepository
	columnsRepo *repositories.StartScreenColumnsRepository
	gamesRepo   *repositories.GamesRepository
	store       *files.AssetStore
}

func NewStartScreenTilesService(
	tilesRepo *repositories.StartScreenTilesRepository,
	columnsRepo *repositories.StartScreenColumnsRepository,
	gamesRepo *repositories.GamesRepository,
	store *files.AssetStore,
) *StartScreenTilesService {
	return &StartScreenTilesService{
		tilesRepo:   tilesRepo,
		columnsRepo: columnsRepo,
		gamesRepo:   gamesRepo,
		store:       store,
	}
}

func (s *StartScreenTilesService) List(includeAll bool) (*domain.StartScreenLayout, error) {
	columns, err := s.columnsRepo.List()
	if err != nil {
		return nil, err
	}
	tiles, err := s.tilesRepo.List(includeAll)
	if err != nil {
		return nil, err
	}
	return &domain.StartScreenLayout{Columns: columns, Tiles: tiles}, nil
}

func (s *StartScreenTilesService) Update(
	columns []domain.StartScreenColumnWrite,
	tiles []domain.StartScreenTileWrite,
) (*domain.StartScreenLayout, error) {
	normalizedColumns, err := s.validateColumns(columns)
	if err != nil {
		return nil, err
	}
	normalized, err := s.validateTiles(tiles)
	if err != nil {
		return nil, err
	}
	normalizedColumns = ensureStartScreenColumns(normalizedColumns, normalized)
	if err := s.tilesRepo.ReplaceLayout(normalizedColumns, normalized); err != nil {
		return nil, err
	}
	return s.List(true)
}

// AddTile 从游戏库卡片入口追加一个磁贴到当前布局的第一个空位；已在开始屏幕时幂等返回当前布局。
func (s *StartScreenTilesService) AddTile(tile domain.StartScreenTileWrite) (*domain.StartScreenLayout, error) {
	normalized, err := s.validateTiles([]domain.StartScreenTileWrite{tile})
	if err != nil {
		return nil, err
	}
	if len(normalized) == 0 {
		return nil, domain.ErrValidation
	}
	layout, err := s.List(true)
	if err != nil {
		return nil, err
	}
	for _, existing := range layout.Tiles {
		if existing.GameID == normalized[0].GameID {
			return layout, nil
		}
	}
	position := findStartScreenFreePosition(layout.Tiles, len(layout.Columns), normalized[0].TileSize)
	if position.columnIndex >= 100 {
		return nil, domain.ErrValidation
	}
	normalized[0].ColumnIndex = position.columnIndex
	normalized[0].GridRow = position.gridRow
	normalized[0].GridCol = position.gridCol
	// 未指定图片时默认用游戏首张截图（高分辨率内容图），没有截图则用封面，
	// 两者都没有则保持空图（显示首字母色块）。
	if normalized[0].ImagePath == nil {
		game, err := s.gamesRepo.GetByID(normalized[0].GameID)
		if err != nil {
			return nil, normalizeRepoError(err)
		}
		screenshot, err := s.gamesRepo.FirstScreenshotPath(normalized[0].GameID)
		if err != nil {
			return nil, err
		}
		if trimmed := strings.TrimSpace(screenshot); trimmed != "" {
			normalized[0].ImagePath = &trimmed
		} else if game.CoverImage != nil {
			cover := strings.TrimSpace(*game.CoverImage)
			if cover != "" {
				normalized[0].ImagePath = &cover
			}
		}
	}
	if _, err := s.tilesRepo.Append(normalized[0]); err != nil {
		return nil, err
	}
	return s.List(true)
}

// RemoveTile 从开始屏幕移除一个游戏的磁贴；不存在时幂等返回当前布局。
func (s *StartScreenTilesService) RemoveTile(gameID int64) (*domain.StartScreenLayout, error) {
	if gameID <= 0 {
		return nil, domain.ErrValidation
	}
	if _, err := s.tilesRepo.RemoveByGameID(gameID); err != nil {
		return nil, err
	}
	return s.List(true)
}

func (s *StartScreenTilesService) validateColumns(columns []domain.StartScreenColumnWrite) ([]domain.StartScreenColumnWrite, error) {
	if len(columns) > 100 {
		return nil, domain.ErrValidation
	}
	normalized := make([]domain.StartScreenColumnWrite, 0, len(columns))
	for _, column := range columns {
		name := strings.TrimSpace(column.Name)
		if len([]rune(name)) > 30 {
			return nil, domain.ErrValidation
		}
		normalized = append(normalized, domain.StartScreenColumnWrite{Name: name})
	}
	return normalized, nil
}

func (s *StartScreenTilesService) validateTiles(tiles []domain.StartScreenTileWrite) ([]domain.StartScreenTileWrite, error) {
	normalized := make([]domain.StartScreenTileWrite, 0, len(tiles))
	seen := make(map[int64]struct{}, len(tiles))
	for _, tile := range tiles {
		if tile.GameID <= 0 {
			return nil, domain.ErrValidation
		}
		tileSize := strings.TrimSpace(tile.TileSize)
		if !domain.IsAllowedStartScreenTileSize(tileSize) {
			return nil, domain.ErrValidation
		}
		if tile.ColumnIndex < 0 || tile.ColumnIndex >= 100 || tile.GridRow < 0 || tile.GridCol < 0 {
			return nil, domain.ErrValidation
		}
		rows, cols := startScreenTileSpan(tileSize)
		if tile.GridRow+rows > domain.StartScreenMaxRows || tile.GridCol+cols > domain.StartScreenFreeCols {
			return nil, domain.ErrValidation
		}
		if _, exists := seen[tile.GameID]; exists {
			return nil, domain.ErrValidation
		}
		seen[tile.GameID] = struct{}{}

		game, err := s.gamesRepo.GetByID(tile.GameID)
		if err != nil {
			return nil, normalizeRepoError(err)
		}

		imagePath, err := s.validateTileImagePath(game, tile.ImagePath)
		if err != nil {
			return nil, err
		}
		if tile.FocusX < 0 || tile.FocusX > 100 || tile.FocusY < 0 || tile.FocusY > 100 {
			return nil, domain.ErrValidation
		}

		flipImages, err := s.validateFlipImages(game, imagePath, tile.FlipImages)
		if err != nil {
			return nil, err
		}

		normalized = append(normalized, domain.StartScreenTileWrite{
			GameID:      tile.GameID,
			TileSize:    tileSize,
			ImagePath:   imagePath,
			FocusX:      tile.FocusX,
			FocusY:      tile.FocusY,
			FlipImages:  flipImages,
			ColumnIndex: tile.ColumnIndex,
			GridRow:     tile.GridRow,
			GridCol:     tile.GridCol,
		})
	}
	return normalized, nil
}

// Win10 磁贴比例体系：没有 1x1 那么小的资源，最小单元从 2x2 起，
// 2x2（小）→ 2x4（宽）→ 4x4（大）。
func startScreenTileSpan(tileSize string) (int, int) {
	switch domain.StartScreenTileSize(tileSize) {
	case domain.StartScreenTileSizeLarge:
		return 4, 4
	case domain.StartScreenTileSizeWide:
		return 2, 4
	default:
		return 2, 2
	}
}

func ensureStartScreenColumns(
	columns []domain.StartScreenColumnWrite,
	tiles []domain.StartScreenTileWrite,
) []domain.StartScreenColumnWrite {
	maxColumnIndex := -1
	for _, tile := range tiles {
		if tile.ColumnIndex > maxColumnIndex {
			maxColumnIndex = tile.ColumnIndex
		}
	}
	for len(columns) <= maxColumnIndex {
		columns = append(columns, domain.StartScreenColumnWrite{Name: ""})
	}
	return columns
}

type startScreenTilePosition struct {
	columnIndex int
	gridRow     int
	gridCol     int
}

func findStartScreenFreePosition(
	tiles []domain.StartScreenTile,
	columnCount int,
	tileSize string,
) startScreenTilePosition {
	rows, cols := startScreenTileSpan(tileSize)
	occupied := map[int][][]bool{}

	ensureColumn := func(columnIndex int) [][]bool {
		column, exists := occupied[columnIndex]
		if exists {
			return column
		}
		column = make([][]bool, 0, 8)
		occupied[columnIndex] = column
		return column
	}

	ensureRow := func(column [][]bool, row int) [][]bool {
		for len(column) <= row {
			column = append(column, make([]bool, domain.StartScreenFreeCols))
		}
		return column
	}

	occupy := func(columnIndex, row, col, spanRows, spanCols int) {
		column := ensureColumn(columnIndex)
		for r := row; r < row+spanRows; r += 1 {
			column = ensureRow(column, r)
			occupied[columnIndex] = column
			for c := col; c < col+spanCols; c += 1 {
				column[r][c] = true
			}
		}
	}

	fits := func(columnIndex, row, col, spanRows, spanCols int) bool {
		column := ensureColumn(columnIndex)
		for r := row; r < row+spanRows; r += 1 {
			if r >= len(column) {
				continue
			}
			line := column[r]
			for c := col; c < col+spanCols; c += 1 {
				if line[c] {
					return false
				}
			}
		}
		return true
	}

	for _, tile := range tiles {
		tileRows, tileCols := startScreenTileSpan(tile.TileSize)
		row := tile.GridRow
		if row < 0 {
			row = 0
		}
		col := tile.GridCol
		if col < 0 {
			col = 0
		}
		if col > domain.StartScreenFreeCols-tileCols {
			col = domain.StartScreenFreeCols - tileCols
		}
		occupy(tile.ColumnIndex, row, col, tileRows, tileCols)
	}

	maxColumnIndex := columnCount - 1
	for _, tile := range tiles {
		if tile.ColumnIndex > maxColumnIndex {
			maxColumnIndex = tile.ColumnIndex
		}
	}

	// 组内 12 列行优先找空位，行数不限（空行必然可放，不会死循环）。
	for columnIndex := 0; columnIndex <= maxColumnIndex; columnIndex += 1 {
		for row := 0; ; row += 1 {
			for col := 0; col <= domain.StartScreenFreeCols-cols; col += 1 {
				if fits(columnIndex, row, col, rows, cols) {
					return startScreenTilePosition{
						columnIndex: columnIndex,
						gridRow:     row,
						gridCol:     col,
					}
				}
			}
		}
	}

	return startScreenTilePosition{
		columnIndex: maxColumnIndex + 1,
		gridRow:     0,
		gridCol:     0,
	}
}

// validateTileImagePath 校验磁贴图片必须是该游戏自己的本地素材（/assets/{publicId}/…）且文件存在。
// 只允许原图直接引用，不再上传派生裁剪图。
func (s *StartScreenTilesService) validateTileImagePath(game *domain.Game, path *string) (*string, error) {
	if path == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*path)
	if trimmed == "" {
		return nil, nil
	}
	if err := s.validateGameAssetPath(game, trimmed); err != nil {
		return nil, err
	}
	return &trimmed, nil
}

// validateFlipImages 校验宽磁贴轮播帧：必须是本游戏素材、去重、不与首帧重复、
// 数量不超过 StartScreenMaxFlipImages，且依赖首帧 image_path（首帧即轮播第 0 帧）。
func (s *StartScreenTilesService) validateFlipImages(
	game *domain.Game,
	imagePath *string,
	flipImages []string,
) ([]string, error) {
	if len(flipImages) == 0 {
		return nil, nil
	}
	if imagePath == nil {
		return nil, domain.ErrValidation
	}
	if len(flipImages) > domain.StartScreenMaxFlipImages {
		return nil, domain.ErrValidation
	}
	seen := make(map[string]struct{}, len(flipImages)+1)
	seen[*imagePath] = struct{}{}
	normalized := make([]string, 0, len(flipImages))
	for _, path := range flipImages {
		trimmed := strings.TrimSpace(path)
		if trimmed == "" {
			return nil, domain.ErrValidation
		}
		if _, dup := seen[trimmed]; dup {
			return nil, domain.ErrValidation
		}
		if err := s.validateGameAssetPath(game, trimmed); err != nil {
			return nil, err
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	return normalized, nil
}

func (s *StartScreenTilesService) validateGameAssetPath(game *domain.Game, path string) error {
	expectedPrefix := "/assets/" + game.PublicID + "/"
	if !strings.HasPrefix(path, expectedPrefix) {
		return domain.ErrValidation
	}
	if s.store == nil || !s.store.AssetExists(path) {
		return domain.ErrValidation
	}
	return nil
}
