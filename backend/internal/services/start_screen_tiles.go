package services

import (
	"errors"
	"mime/multipart"
	"os"
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
	movedPaths := make([]string, 0)
	for i := range normalized {
		moved, err := s.moveTileImagesToPermanent(&normalized[i])
		movedPaths = append(movedPaths, moved...)
		if err != nil {
			return nil, errors.Join(err, s.cleanupMovedTileImages(movedPaths))
		}
	}
	if err := s.tilesRepo.ReplaceLayout(normalizedColumns, normalized); err != nil {
		return nil, errors.Join(err, s.cleanupMovedTileImages(movedPaths))
	}
	return s.List(true)
}

// moveTileImagesToPermanent 与游戏素材编辑一致：上传只进 staging，保存布局时才把
// 本次引用的裁剪图转正到 assets/start-screen/，未保存的裁剪图留在 staging 由启动清理兜底。
func (s *StartScreenTilesService) moveTileImagesToPermanent(tile *domain.StartScreenTileWrite) ([]string, error) {
	movedPaths := make([]string, 0, 3)
	for _, path := range []*string{tile.ImageSmallPath, tile.ImageWidePath, tile.ImageLargePath} {
		if path == nil || strings.TrimSpace(*path) == "" {
			continue
		}
		permanentPath, moved, err := s.store.MoveToPermanentWithStatus(*path, "start-screen")
		if err != nil {
			return movedPaths, err
		}
		*path = permanentPath
		if moved {
			movedPaths = append(movedPaths, permanentPath)
		}
	}
	return movedPaths, nil
}

func (s *StartScreenTilesService) cleanupMovedTileImages(paths []string) error {
	var cleanupErr error
	for _, path := range paths {
		if err := s.store.DeleteAsset(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			cleanupErr = errors.Join(cleanupErr, err)
		}
	}
	return cleanupErr
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
		if tile.GridRow+rows > domain.StartScreenColumnRows || tile.GridCol+cols > domain.StartScreenColumnCols {
			return nil, domain.ErrValidation
		}
		if _, exists := seen[tile.GameID]; exists {
			return nil, domain.ErrValidation
		}
		seen[tile.GameID] = struct{}{}

		if _, err := s.gamesRepo.GetByID(tile.GameID); err != nil {
			return nil, normalizeRepoError(err)
		}

		imageSmallPath, err := s.validateTileImagePath(tile.ImageSmallPath)
		if err != nil {
			return nil, err
		}
		imageWidePath, err := s.validateTileImagePath(tile.ImageWidePath)
		if err != nil {
			return nil, err
		}
		imageLargePath, err := s.validateTileImagePath(tile.ImageLargePath)
		if err != nil {
			return nil, err
		}

		normalized = append(normalized, domain.StartScreenTileWrite{
			GameID:         tile.GameID,
			TileSize:       tileSize,
			ImageSmallPath: imageSmallPath,
			ImageWidePath:  imageWidePath,
			ImageLargePath: imageLargePath,
			ColumnIndex:    tile.ColumnIndex,
			GridRow:        tile.GridRow,
			GridCol:        tile.GridCol,
		})
	}
	return normalized, nil
}

func startScreenTileSpan(tileSize string) (int, int) {
	switch domain.StartScreenTileSize(tileSize) {
	case domain.StartScreenTileSizeLarge:
		return 2, 2
	case domain.StartScreenTileSizeWide:
		return 1, 2
	default:
		return 1, 1
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
		column = make([][]bool, domain.StartScreenColumnRows)
		for row := range column {
			column[row] = make([]bool, domain.StartScreenColumnCols)
		}
		occupied[columnIndex] = column
		return column
	}

	occupy := func(columnIndex, row, col, spanRows, spanCols int) {
		column := ensureColumn(columnIndex)
		for r := row; r < row+spanRows; r += 1 {
			for c := col; c < col+spanCols; c += 1 {
				column[r][c] = true
			}
		}
	}

	fits := func(columnIndex, row, col, spanRows, spanCols int) bool {
		column := ensureColumn(columnIndex)
		for r := row; r < row+spanRows; r += 1 {
			for c := col; c < col+spanCols; c += 1 {
				if column[r][c] {
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
		if row > domain.StartScreenColumnRows-tileRows {
			row = domain.StartScreenColumnRows - tileRows
		}
		col := tile.GridCol
		if col < 0 {
			col = 0
		}
		if col > domain.StartScreenColumnCols-tileCols {
			col = domain.StartScreenColumnCols - tileCols
		}
		occupy(tile.ColumnIndex, row, col, tileRows, tileCols)
	}

	maxColumnIndex := columnCount - 1
	for _, tile := range tiles {
		if tile.ColumnIndex > maxColumnIndex {
			maxColumnIndex = tile.ColumnIndex
		}
	}

	for columnIndex := 0; columnIndex <= maxColumnIndex; columnIndex += 1 {
		for row := 0; row <= domain.StartScreenColumnRows-rows; row += 1 {
			for col := 0; col <= domain.StartScreenColumnCols-cols; col += 1 {
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

func (s *StartScreenTilesService) validateTileImagePath(path *string) (*string, error) {
	if path == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*path)
	if trimmed == "" {
		return nil, nil
	}
	if !strings.HasPrefix(trimmed, "/assets/") {
		return nil, domain.ErrValidation
	}
	if s.store == nil || !s.store.AssetExists(trimmed) {
		return nil, domain.ErrValidation
	}
	return &trimmed, nil
}

// UploadTileImage 保存磁贴裁剪图到 staging，返回 /assets/start-screen/{uid}.{ext} 路径；
// 文件在布局保存（Update）时才转正到 assets/start-screen/。
func (s *StartScreenTilesService) UploadTileImage(header *multipart.FileHeader) (string, error) {
	if s.store == nil {
		return "", domain.ErrMissingConfig
	}
	src, err := header.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	contentType := header.Header.Get("Content-Type")
	assetUID := newAssetUID()
	stagingPath, err := s.store.SaveToStaging("start-screen", "cover", assetUID, src, contentType)
	if err != nil {
		return "", normalizeAssetError(err)
	}
	return stagingPath, nil
}
