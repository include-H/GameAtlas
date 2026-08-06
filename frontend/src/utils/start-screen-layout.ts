import type { StartScreenTile, StartScreenTileSize } from '@/services/types'

export const START_SCREEN_COLUMN_ROWS = 6
export const START_SCREEN_COLUMN_COLS = 2

const START_SCREEN_TILE_SPANS: Record<StartScreenTileSize, { rows: number; cols: number }> = {
  small: { rows: 1, cols: 1 },
  wide: { rows: 1, cols: 2 },
  large: { rows: 2, cols: 2 },
}

interface PackedStartScreenSlot {
  tile: StartScreenTile
  globalIndex: number
  row: number
  col: number
}

interface PackedStartScreenColumn {
  slots: PackedStartScreenSlot[]
}

interface StartScreenDropTarget {
  columnIndex: number
  row: number
  col: number
}

type Occupancy = boolean[][]
type OccupancyMap = Map<number, Occupancy>

const createOccupancy = (): Occupancy =>
  Array.from({ length: START_SCREEN_COLUMN_ROWS }, () =>
    Array<boolean>(START_SCREEN_COLUMN_COLS).fill(false),
  )

const ensureOccupancyColumn = (occupied: OccupancyMap, columnIndex: number): Occupancy => {
  const column = occupied.get(columnIndex)
  if (column) return column
  const next = createOccupancy()
  occupied.set(columnIndex, next)
  return next
}

const fits = (
  column: Occupancy,
  row: number,
  col: number,
  rows: number,
  cols: number,
) => {
  for (let r = row; r < row + rows; r += 1) {
    for (let c = col; c < col + cols; c += 1) {
      if (column[r][c]) return false
    }
  }
  return true
}

const findFirstFit = (
  column: Occupancy,
  rows: number,
  cols: number,
  fromRow = 0,
): { row: number; col: number } | null => {
  for (let row = fromRow; row <= START_SCREEN_COLUMN_ROWS - rows; row += 1) {
    for (let col = 0; col <= START_SCREEN_COLUMN_COLS - cols; col += 1) {
      if (fits(column, row, col, rows, cols)) {
        return { row, col }
      }
    }
  }
  return null
}

const occupy = (
  column: Occupancy,
  row: number,
  col: number,
  rows: number,
  cols: number,
) => {
  for (let r = row; r < row + rows; r += 1) {
    for (let c = col; c < col + cols; c += 1) {
      column[r][c] = true
    }
  }
}

const normalizedPosition = (value: number, fallback: number): number => {
  if (!Number.isFinite(value)) return fallback
  return Math.max(0, Math.trunc(value))
}

/**
 * 自由排列：没有按尺寸的配额。磁贴按全局顺序放入当前列，行优先 first-fit；
 * 当前列放不下就另开一列。满配时自然呈现"1 大 + 2 长 + 4 小（2x2）"的整列，
 * 其他组合（如三个大磁铁叠满、12 个 1x1）同样自由成立。
 */
export function packStartScreenTiles(tiles: StartScreenTile[]): PackedStartScreenColumn[] {
  const columns: PackedStartScreenColumn[] = []
  const occupied: Occupancy[] = []

  if (tiles.length === 0) return columns

  const ensureColumn = () => {
    columns.push({ slots: [] })
    occupied.push(createOccupancy())
  }

  const place = (columnIndex: number, tile: StartScreenTile, globalIndex: number, row: number, col: number) => {
    const span = START_SCREEN_TILE_SPANS[tile.tile_size]
    occupy(occupied[columnIndex], row, col, span.rows, span.cols)
    columns[columnIndex].slots.push({ tile, globalIndex, row, col })
  }

  ensureColumn()
  tiles.forEach((tile, globalIndex) => {
    const span = START_SCREEN_TILE_SPANS[tile.tile_size]
    const current = columns.length - 1
    const position = findFirstFit(occupied[current], span.rows, span.cols)
    if (position) {
      place(current, tile, globalIndex, position.row, position.col)
      return
    }
    ensureColumn()
    place(columns.length - 1, tile, globalIndex, 0, 0)
  })

  return columns
}

/**
 * 把显式坐标纠正为不重叠、不越界的布局。旧数据或放大尺寸后出现的碰撞，
 * 会在同一列就近找空位，列满后自动向右开新列。
 */
export function normalizeStartScreenTiles(tiles: StartScreenTile[]): StartScreenTile[] {
  const normalized = tiles.map((tile) => ({
    ...tile,
    column_index: normalizedPosition(tile.column_index, 0),
    grid_row: normalizedPosition(tile.grid_row, 0),
    grid_col: normalizedPosition(tile.grid_col, 0),
  }))
  const sorted = [...normalized].sort((a, b) =>
    a.column_index - b.column_index ||
    a.grid_row - b.grid_row ||
    a.grid_col - b.grid_col ||
    a.sort_order - b.sort_order ||
    a.game_id - b.game_id,
  )
  const occupied: OccupancyMap = new Map()

  return sorted.map((tile) => {
    const span = START_SCREEN_TILE_SPANS[tile.tile_size]
    const columnIndex = tile.column_index
    const row = Math.min(START_SCREEN_COLUMN_ROWS - span.rows, tile.grid_row)
    const col = Math.min(START_SCREEN_COLUMN_COLS - span.cols, tile.grid_col)
    const targetColumn = ensureOccupancyColumn(occupied, columnIndex)

    if (fits(targetColumn, row, col, span.rows, span.cols)) {
      occupy(targetColumn, row, col, span.rows, span.cols)
      return { ...tile, column_index: columnIndex, grid_row: row, grid_col: col }
    }

    const sameColumn = findFirstFit(targetColumn, span.rows, span.cols)
    if (sameColumn) {
      occupy(targetColumn, sameColumn.row, sameColumn.col, span.rows, span.cols)
      return {
        ...tile,
        column_index: columnIndex,
        grid_row: sameColumn.row,
        grid_col: sameColumn.col,
      }
    }

    let nextColumnIndex = columnIndex + 1
    while (true) {
      const nextColumn = ensureOccupancyColumn(occupied, nextColumnIndex)
      const position = findFirstFit(nextColumn, span.rows, span.cols)
      if (position) {
        occupy(nextColumn, position.row, position.col, span.rows, span.cols)
        return {
          ...tile,
          column_index: nextColumnIndex,
          grid_row: position.row,
          grid_col: position.col,
        }
      }
      nextColumnIndex += 1
    }
  })
}

export function layoutStartScreenTiles(
  tiles: StartScreenTile[],
  columnCount = 0,
): PackedStartScreenColumn[] {
  const normalized = normalizeStartScreenTiles(tiles)
  const originalIndices = new Map<number, number>()
  tiles.forEach((tile, index) => originalIndices.set(tile.game_id, index))

  const maxTileColumn = normalized.reduce((max, tile) => Math.max(max, tile.column_index), -1)
  const count = Math.max(1, columnCount, maxTileColumn + 1)
  const columns: PackedStartScreenColumn[] = Array.from({ length: count }, () => ({ slots: [] }))

  normalized.forEach((tile) => {
    const column = columns[tile.column_index]
    if (!column) return
    column.slots.push({
      tile,
      globalIndex: originalIndices.get(tile.game_id) ?? tile.sort_order,
      row: tile.grid_row,
      col: tile.grid_col,
    })
  })

  columns.forEach((column) => {
    column.slots.sort((a, b) => a.row - b.row || a.col - b.col || a.tile.sort_order - b.tile.sort_order)
  })

  return columns
}

export function findStartScreenDropTarget(
  tiles: StartScreenTile[],
  excludedGameId: number,
  columnIndex: number,
  row: number,
  col: number,
  tileSize: StartScreenTileSize,
): StartScreenDropTarget {
  const span = START_SCREEN_TILE_SPANS[tileSize]
  const occupied: OccupancyMap = new Map()

  for (const tile of tiles) {
    if (tile.game_id === excludedGameId) continue
    const tileSpan = START_SCREEN_TILE_SPANS[tile.tile_size]
    const tileColumn = ensureOccupancyColumn(occupied, normalizedPosition(tile.column_index, 0))
    const tileRow = Math.min(START_SCREEN_COLUMN_ROWS - tileSpan.rows, normalizedPosition(tile.grid_row, 0))
    const tileCol = Math.min(START_SCREEN_COLUMN_COLS - tileSpan.cols, normalizedPosition(tile.grid_col, 0))
    if (fits(tileColumn, tileRow, tileCol, tileSpan.rows, tileSpan.cols)) {
      occupy(tileColumn, tileRow, tileCol, tileSpan.rows, tileSpan.cols)
    }
  }

  const requestedColumn = Math.max(0, Math.trunc(columnIndex || 0))
  const requestedRow = Math.min(START_SCREEN_COLUMN_ROWS - span.rows, Math.max(0, Math.trunc(row || 0)))
  const requestedCol = Math.min(START_SCREEN_COLUMN_COLS - span.cols, Math.max(0, Math.trunc(col || 0)))
  const targetColumn = ensureOccupancyColumn(occupied, requestedColumn)

  if (fits(targetColumn, requestedRow, requestedCol, span.rows, span.cols)) {
    return { columnIndex: requestedColumn, row: requestedRow, col: requestedCol }
  }

  for (let r = requestedRow; r <= START_SCREEN_COLUMN_ROWS - span.rows; r += 1) {
    const position = findFirstFit(targetColumn, span.rows, span.cols, r)
    if (position) {
      return { columnIndex: requestedColumn, row: position.row, col: position.col }
    }
  }
  for (let r = 0; r < requestedRow; r += 1) {
    const position = findFirstFit(targetColumn, span.rows, span.cols, r)
    if (position) {
      return { columnIndex: requestedColumn, row: position.row, col: position.col }
    }
  }

  const maxExistingColumn = tiles.reduce(
    (max, tile) => Math.max(max, normalizedPosition(tile.column_index, 0)),
    requestedColumn,
  )
  for (let column = requestedColumn + 1; column <= maxExistingColumn; column += 1) {
    const position = findFirstFit(ensureOccupancyColumn(occupied, column), span.rows, span.cols)
    if (position) {
      return { columnIndex: column, row: position.row, col: position.col }
    }
  }

  let nextColumn = maxExistingColumn + 1
  while (true) {
    const position = findFirstFit(ensureOccupancyColumn(occupied, nextColumn), span.rows, span.cols)
    if (position) {
      return { columnIndex: nextColumn, row: position.row, col: position.col }
    }
    nextColumn += 1
  }
}
