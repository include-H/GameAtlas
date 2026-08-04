import type { StartScreenTile, StartScreenTileSize } from '@/services/types'

export interface PackedStartScreenSlot {
  tile: StartScreenTile
  globalIndex: number
  row: number
  col: number
}

export interface PackedStartScreenColumn {
  slots: PackedStartScreenSlot[]
}

// 一列 = 三个"大正方形"占地纵向堆叠：大磁铁（2x2）+ 两个长磁铁（2x1 x2）+ 四个 1x1（2x2）。
// 因此列宽 2 格、列高 6 行。
export const START_SCREEN_COLUMN_ROWS = 6
export const START_SCREEN_COLUMN_COLS = 2

// 每列容量：1 个大（2x2）+ 2 个宽（2x1）+ 4 个小（1x1），超出即另开一列。
const COLUMN_QUOTA: Record<StartScreenTileSize, number> = {
  large: 1,
  wide: 2,
  small: 4,
}

const SPANS: Record<StartScreenTileSize, { rows: number; cols: number }> = {
  small: { rows: 1, cols: 1 },
  wide: { rows: 1, cols: 2 },
  large: { rows: 2, cols: 2 },
}

/**
 * 先把磁贴按容量分到各列（保持全局顺序），再在列内做行优先 first-fit 排布。
 * 满配时自然呈现：大磁铁在上，两个长磁铁在中，四个 1x1 以 2x2 在底。
 */
export function packStartScreenTiles(tiles: StartScreenTile[]): PackedStartScreenColumn[] {
  const columns: PackedStartScreenColumn[] = []
  const used: Record<StartScreenTileSize, number>[] = []
  const occupied: boolean[][][] = []

  if (tiles.length === 0) return columns

  const ensureColumn = () => {
    columns.push({ slots: [] })
    used.push({ large: 0, wide: 0, small: 0 })
    occupied.push(
      Array.from({ length: START_SCREEN_COLUMN_ROWS }, () =>
        Array<boolean>(START_SCREEN_COLUMN_COLS).fill(false),
      ),
    )
  }

  const fits = (columnIndex: number, row: number, col: number, rows: number, cols: number) => {
    for (let r = row; r < row + rows; r += 1) {
      for (let c = col; c < col + cols; c += 1) {
        if (occupied[columnIndex][r][c]) return false
      }
    }
    return true
  }

  const place = (columnIndex: number, tile: StartScreenTile, globalIndex: number) => {
    const span = SPANS[tile.tile_size]
    for (let row = 0; row <= START_SCREEN_COLUMN_ROWS - span.rows; row += 1) {
      for (let col = 0; col <= START_SCREEN_COLUMN_COLS - span.cols; col += 1) {
        if (!fits(columnIndex, row, col, span.rows, span.cols)) continue
        for (let r = row; r < row + span.rows; r += 1) {
          for (let c = col; c < col + span.cols; c += 1) {
            occupied[columnIndex][r][c] = true
          }
        }
        columns[columnIndex].slots.push({ tile, globalIndex, row, col })
        return
      }
    }
    // 容量配额保证列内放得下，这里只是防御性兜底。
    columns[columnIndex].slots.push({ tile, globalIndex, row: 0, col: 0 })
  }

  ensureColumn()
  tiles.forEach((tile, globalIndex) => {
    const size = tile.tile_size
    let columnIndex = columns.findIndex((_, index) => used[index][size] < COLUMN_QUOTA[size])
    if (columnIndex === -1) {
      ensureColumn()
      columnIndex = columns.length - 1
    }
    place(columnIndex, tile, globalIndex)
    used[columnIndex][size] += 1
  })

  return columns
}
