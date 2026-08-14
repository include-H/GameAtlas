import type { StartScreenTile, StartScreenTileSize } from '@/services/types'

export const START_SCREEN_FREE_COLS = 12

// Win10 磁贴比例体系：没有 1x1 那么小的资源，最小单元从 2x2 起，
// 2x2（小）→ 2x4（宽）→ 4x4（大）。
const START_SCREEN_TILE_SPANS: Record<StartScreenTileSize, { rows: number; cols: number }> = {
  small: { rows: 2, cols: 2 },
  wide: { rows: 2, cols: 4 },
  large: { rows: 4, cols: 4 },
}

export interface PackedStartScreenSlot {
  tile: StartScreenTile
  globalIndex: number
  row: number
  col: number
}

export interface PackedStartScreenGroup {
  columnIndex: number
  slots: PackedStartScreenSlot[]
}

export interface StartScreenDropTarget {
  columnIndex: number
  row: number
  col: number
}

// 组内 12 列 × 行数不限的稀疏占用表：row -> 已占用的列位
type Occupancy = Map<number, boolean[]>

const createOccupancy = (): Occupancy => new Map()

const fits = (occupied: Occupancy, row: number, col: number, rows: number, cols: number): boolean => {
  for (let r = row; r < row + rows; r += 1) {
    const line = occupied.get(r)
    if (!line) continue
    for (let c = col; c < col + cols; c += 1) {
      if (line[c]) return false
    }
  }
  return true
}

const findFirstFit = (
  occupied: Occupancy,
  rows: number,
  cols: number,
  fromRow = 0,
): { row: number; col: number } => {
  let row = fromRow
  for (;;) {
    for (let col = 0; col <= START_SCREEN_FREE_COLS - cols; col += 1) {
      if (fits(occupied, row, col, rows, cols)) {
        return { row, col }
      }
    }
    row += 1
  }
}

const occupy = (occupied: Occupancy, row: number, col: number, rows: number, cols: number) => {
  for (let r = row; r < row + rows; r += 1) {
    const line = occupied.get(r) ?? []
    occupied.set(r, line)
    for (let c = col; c < col + cols; c += 1) {
      line[c] = true
    }
  }
}

const normalizedPosition = (value: number, fallback: number): number => {
  if (!Number.isFinite(value)) return fallback
  return Math.max(0, Math.trunc(value))
}

const groupIndexesOf = (tiles: StartScreenTile[]): number[] => {
  const set = new Set(tiles.map((tile) => normalizedPosition(tile.column_index, 0)))
  return [...set].sort((a, b) => a - b)
}

/**
 * 全屏自定义网格：组（列）只是顶部标签，磁贴全部摆进 12 列无限行的自由网格，
 * 行优先 first-fit 自动摆放，任意尺寸组合都能自然铺开。
 */
export function packStartScreenTiles(tiles: StartScreenTile[]): PackedStartScreenGroup[] {
  if (tiles.length === 0) return []
  const occupied = createOccupancy()
  const slots: PackedStartScreenSlot[] = tiles.map((tile, globalIndex) => {
    const span = START_SCREEN_TILE_SPANS[tile.tile_size]
    const position = findFirstFit(occupied, span.rows, span.cols)
    occupy(occupied, position.row, position.col, span.rows, span.cols)
    return {
      tile: { ...tile, grid_row: position.row, grid_col: position.col },
      globalIndex,
      row: position.row,
      col: position.col,
    }
  })
  return [{ columnIndex: 0, slots }]
}

/**
 * 显式坐标纠正为不重叠、不越界的布局：组内就近找空位，行数不限总放得下，
 * 不再有"列满开新列"的硬边界。
 */
export function normalizeStartScreenTiles(tiles: StartScreenTile[]): StartScreenTile[] {
  const normalized = tiles.map((tile) => ({
    ...tile,
    column_index: normalizedPosition(tile.column_index, 0),
    grid_row: normalizedPosition(tile.grid_row, 0),
    grid_col: normalizedPosition(tile.grid_col, 0),
  }))

  const result: StartScreenTile[] = []
  for (const columnIndex of groupIndexesOf(normalized)) {
    const groupTiles = normalized
      .filter((tile) => tile.column_index === columnIndex)
      .sort(
        (a, b) =>
          a.grid_row - b.grid_row ||
          a.grid_col - b.grid_col ||
          a.sort_order - b.sort_order ||
          a.game_id - b.game_id,
      )
    const occupied = createOccupancy()
    for (const tile of groupTiles) {
      const span = START_SCREEN_TILE_SPANS[tile.tile_size]
      const row = tile.grid_row
      const col = Math.min(START_SCREEN_FREE_COLS - span.cols, tile.grid_col)
      if (fits(occupied, row, col, span.rows, span.cols)) {
        occupy(occupied, row, col, span.rows, span.cols)
        result.push({ ...tile, grid_row: row, grid_col: col })
        continue
      }
      const position = findFirstFit(occupied, span.rows, span.cols, row)
      occupy(occupied, position.row, position.col, span.rows, span.cols)
      result.push({ ...tile, grid_row: position.row, grid_col: position.col })
    }
  }
  return result
}

export function layoutStartScreenTiles(
  tiles: StartScreenTile[],
  columnCount = 0,
): PackedStartScreenGroup[] {
  const normalized = normalizeStartScreenTiles(tiles)
  const originalIndices = new Map<number, number>()
  tiles.forEach((tile, index) => originalIndices.set(tile.game_id, index))

  const maxTileColumn = normalized.reduce((max, tile) => Math.max(max, tile.column_index), -1)
  const count = Math.max(1, columnCount, maxTileColumn + 1)

  const groups: PackedStartScreenGroup[] = []
  for (let columnIndex = 0; columnIndex < count; columnIndex += 1) {
    const groupTiles = normalized.filter((tile) => tile.column_index === columnIndex)
    const slots: PackedStartScreenSlot[] = groupTiles.map((tile) => ({
      tile,
      globalIndex: originalIndices.get(tile.game_id) ?? tile.sort_order,
      row: tile.grid_row,
      col: tile.grid_col,
    }))
    slots.sort((a, b) => a.row - b.row || a.col - b.col || a.tile.sort_order - b.tile.sort_order)
    groups.push({ columnIndex, slots })
  }
  return groups
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
  const occupiedByColumn = new Map<number, Occupancy>()
  const columnOf = (index: number): Occupancy => {
    const existing = occupiedByColumn.get(index)
    if (existing) return existing
    const next = createOccupancy()
    occupiedByColumn.set(index, next)
    return next
  }

  for (const tile of tiles) {
    if (tile.game_id === excludedGameId) continue
    const tileSpan = START_SCREEN_TILE_SPANS[tile.tile_size]
    const tileColumn = columnOf(normalizedPosition(tile.column_index, 0))
    const tileRow = normalizedPosition(tile.grid_row, 0)
    const tileCol = Math.min(
      START_SCREEN_FREE_COLS - tileSpan.cols,
      normalizedPosition(tile.grid_col, 0),
    )
    if (fits(tileColumn, tileRow, tileCol, tileSpan.rows, tileSpan.cols)) {
      occupy(tileColumn, tileRow, tileCol, tileSpan.rows, tileSpan.cols)
    }
  }

  const requestedColumnIndex = Math.max(0, Math.trunc(columnIndex || 0))
  const targetColumn = columnOf(requestedColumnIndex)
  const requestedRow = Math.max(0, Math.trunc(row || 0))
  const requestedCol = Math.min(
    START_SCREEN_FREE_COLS - span.cols,
    Math.max(0, Math.trunc(col || 0)),
  )

  if (fits(targetColumn, requestedRow, requestedCol, span.rows, span.cols)) {
    return { columnIndex: requestedColumnIndex, row: requestedRow, col: requestedCol }
  }

  const position = findFirstFit(targetColumn, span.rows, span.cols, requestedRow)
  return { columnIndex: requestedColumnIndex, row: position.row, col: position.col }
}
