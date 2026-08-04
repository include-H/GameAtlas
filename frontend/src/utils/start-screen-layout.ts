import type { StartScreenTile, StartScreenTileSize } from '@/services/types'

interface PackedStartScreenSlot {
  tile: StartScreenTile
  globalIndex: number
  row: number
  col: number
}

interface PackedStartScreenColumn {
  slots: PackedStartScreenSlot[]
}

// 列 = 高度容器：2 格宽 × 6 行高（约三个 2x2 大正方形占地），横向无限延伸。
const START_SCREEN_COLUMN_ROWS = 6
const START_SCREEN_COLUMN_COLS = 2

const SPANS: Record<StartScreenTileSize, { rows: number; cols: number }> = {
  small: { rows: 1, cols: 1 },
  wide: { rows: 1, cols: 2 },
  large: { rows: 2, cols: 2 },
}

/**
 * 自由排列：没有按尺寸的配额。磁贴按全局顺序放入当前列，行优先 first-fit；
 * 当前列放不下就另开一列。满配时自然呈现"1 大 + 2 长 + 4 小（2x2）"的整列，
 * 其他组合（如三个大磁铁叠满、12 个 1x1）同样自由成立。
 */
export function packStartScreenTiles(tiles: StartScreenTile[]): PackedStartScreenColumn[] {
  const columns: PackedStartScreenColumn[] = []
  const occupied: boolean[][][] = []

  if (tiles.length === 0) return columns

  const ensureColumn = () => {
    columns.push({ slots: [] })
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

  const findFirstFit = (
    columnIndex: number,
    rows: number,
    cols: number,
  ): { row: number; col: number } | null => {
    for (let row = 0; row <= START_SCREEN_COLUMN_ROWS - rows; row += 1) {
      for (let col = 0; col <= START_SCREEN_COLUMN_COLS - cols; col += 1) {
        if (fits(columnIndex, row, col, rows, cols)) {
          return { row, col }
        }
      }
    }
    return null
  }

  const place = (columnIndex: number, tile: StartScreenTile, globalIndex: number, row: number, col: number) => {
    const span = SPANS[tile.tile_size]
    for (let r = row; r < row + span.rows; r += 1) {
      for (let c = col; c < col + span.cols; c += 1) {
        occupied[columnIndex][r][c] = true
      }
    }
    columns[columnIndex].slots.push({ tile, globalIndex, row, col })
  }

  ensureColumn()
  tiles.forEach((tile, globalIndex) => {
    const span = SPANS[tile.tile_size]
    const current = columns.length - 1
    const position = findFirstFit(current, span.rows, span.cols)
    if (position) {
      place(current, tile, globalIndex, position.row, position.col)
      return
    }
    ensureColumn()
    place(columns.length - 1, tile, globalIndex, 0, 0)
  })

  return columns
}
