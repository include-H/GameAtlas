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

export const START_SCREEN_COLUMN_ROWS = 5
export const START_SCREEN_COLUMN_COLS = 4

const COLUMN_QUOTA: Record<StartScreenTileSize, number> = {
  large: 1,
  wide: 2,
  small: 4,
}

/**
 * Win8 风格列模板：一列 = 1 个大正方形（2x2）+ 2 个宽长方形（2x1）+ 4 个小正方形（1x1）。
 * 某种尺寸超出配额就往右另开一列。列内布局：
 * - 有大磁贴：大占 0-1 行，宽占 2/3 行，小占第 4 行；
 * - 没有大磁贴：宽从第 0 行开始，小磁贴紧跟在宽下方，避免顶部留空。
 */
export function packStartScreenTiles(tiles: StartScreenTile[]): PackedStartScreenColumn[] {
  const columns: PackedStartScreenColumn[] = []
  const used: Record<StartScreenTileSize, number>[] = []

  if (tiles.length === 0) return columns

  const ensureColumn = () => {
    columns.push({ slots: [] })
    used.push({ large: 0, wide: 0, small: 0 })
  }

  // 第一遍：按尺寸配额把磁贴分配到列（保持各尺寸内部的原顺序）。
  tiles.forEach((tile, globalIndex) => {
    const size = tile.tile_size
    let columnIndex = columns.findIndex((_, index) => used[index][size] < COLUMN_QUOTA[size])
    if (columnIndex === -1) {
      ensureColumn()
      columnIndex = columns.length - 1
    }
    columns[columnIndex].slots.push({ tile, globalIndex, row: 0, col: 0 })
    used[columnIndex][size] += 1
  })

  // 第二遍：按列模板计算每个磁贴的网格位置。
  for (const column of columns) {
    const largeCount = column.slots.filter((slot) => slot.tile.tile_size === 'large').length
    const wideSlots = column.slots.filter((slot) => slot.tile.tile_size === 'wide')
    const smallSlots = column.slots.filter((slot) => slot.tile.tile_size === 'small')
    const largeSlot = column.slots.find((slot) => slot.tile.tile_size === 'large')

    if (largeSlot) {
      largeSlot.row = 0
      largeSlot.col = 0
    }
    wideSlots.forEach((slot, index) => {
      slot.row = largeCount > 0 ? 2 + index : index
      slot.col = 0
    })
    const smallStartRow = largeCount > 0 ? 4 : wideSlots.length
    smallSlots.forEach((slot, index) => {
      slot.row = smallStartRow
      slot.col = index
    })
  }

  return columns
}
