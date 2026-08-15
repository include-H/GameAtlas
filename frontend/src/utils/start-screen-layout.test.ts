import { describe, expect, it } from 'vitest'
import {
  findStartScreenDropTarget,
  layoutStartScreenTiles,
  normalizeStartScreenTiles,
  packStartScreenTiles,
  START_SCREEN_FREE_COLS,
} from './start-screen-layout'
import type { StartScreenTile } from '@/services/types'

const makeTile = (
  gameId: number,
  tileSize: StartScreenTile['tile_size'],
  position: Partial<Pick<StartScreenTile, 'column_index' | 'grid_row' | 'grid_col'>> = {},
): StartScreenTile => ({
  game_id: gameId,
  public_id: `game-${gameId}`,
  title: `Game ${gameId}`,
  tile_size: tileSize,
  image_path: null,
  focus_x: 50,
  focus_y: 50,
  flip_images: null,
  sort_order: 0,
  column_index: 0,
  grid_row: 0,
  grid_col: 0,
  ...position,
})

describe('packStartScreenTiles', () => {
  it('auto-flows Win10-ratio tiles row-first into the 12-col free grid', () => {
    const groups = packStartScreenTiles([
      makeTile(1, 'large'),
      makeTile(2, 'wide'),
      makeTile(3, 'wide'),
      makeTile(4, 'small'),
      makeTile(5, 'small'),
      makeTile(6, 'small'),
      makeTile(7, 'small'),
    ])

    expect(groups).toHaveLength(1)
    const slots = groups[0]?.slots ?? []
    // large 4x4 占 col0-3（row0-3）；两个 wide 2x4 依次落在 col4、col8（row0-1）
    expect(slots.find((s) => s.tile.game_id === 1)).toMatchObject({ row: 0, col: 0 })
    expect(slots.find((s) => s.tile.game_id === 2)).toMatchObject({ row: 0, col: 4 })
    expect(slots.find((s) => s.tile.game_id === 3)).toMatchObject({ row: 0, col: 8 })
    // row2 起 large 只占 col0-3，四个 small 2x2 填进 col4-11
    for (let index = 0; index < 4; index += 1) {
      expect(slots.find((s) => s.tile.game_id === 4 + index)).toMatchObject({
        row: 2,
        col: 4 + index * 2,
      })
    }
  })

  it('places four large tiles side by side, then wraps to the next row', () => {
    const groups = packStartScreenTiles([
      makeTile(1, 'large'),
      makeTile(2, 'large'),
      makeTile(3, 'large'),
      makeTile(4, 'large'),
    ])

    expect(groups).toHaveLength(1)
    const slots = groups[0]?.slots ?? []
    expect(slots.map((s) => s.row)).toEqual([0, 0, 0, 4])
    expect(slots.map((s) => s.col)).toEqual([0, 4, 8, 0])
  })

  it('wraps to the next row when a full 12-col row is packed', () => {
    const groups = packStartScreenTiles(
      Array.from({ length: START_SCREEN_FREE_COLS + 1 }, (_, index) =>
        makeTile(index + 10, 'small'),
      ),
    )

    expect(groups).toHaveLength(1)
    const slots = groups[0]?.slots ?? []
    expect(slots).toHaveLength(START_SCREEN_FREE_COLS + 1)
    // 2x2 磁贴每行 6 个（占两行高），第 7 个换到第 3 行
    expect(slots[6]).toMatchObject({ row: 2, col: 0 })
    expect(slots[12]).toMatchObject({ row: 4, col: 0 })
  })

  it('fills the wide tile into the same row after a small tile', () => {
    const groups = packStartScreenTiles([
      makeTile(1, 'small'),
      makeTile(2, 'wide'),
    ])

    const slots = groups[0]?.slots ?? []
    expect(slots.find((s) => s.tile.game_id === 1)).toMatchObject({ row: 0, col: 0 })
    expect(slots.find((s) => s.tile.game_id === 2)).toMatchObject({ row: 0, col: 2 })
  })

  it('keeps global order indices for drag reordering', () => {
    const groups = packStartScreenTiles([
      makeTile(1, 'small'),
      makeTile(2, 'large'),
      makeTile(3, 'small'),
    ])

    expect(groups[0]?.slots.map((s) => s.globalIndex)).toEqual([0, 1, 2])
  })

  it('normalizes overlapping explicit placements within a group', () => {
    const normalized = normalizeStartScreenTiles([
      makeTile(1, 'large', { column_index: 0, grid_row: 0, grid_col: 0 }),
      makeTile(2, 'small', { column_index: 0, grid_row: 0, grid_col: 0 }),
    ])

    expect(normalized[0]).toMatchObject({ column_index: 0, grid_row: 0, grid_col: 0 })
    expect(normalized[1]).toMatchObject({ column_index: 0, grid_row: 0, grid_col: 4 })
  })

  it('keeps explicit empty groups in the layout', () => {
    const groups = layoutStartScreenTiles([
      makeTile(1, 'small', { column_index: 2, grid_row: 1, grid_col: 1 }),
    ], 4)

    expect(groups).toHaveLength(4)
    expect(groups[0]?.slots).toEqual([])
    expect(groups[1]?.slots).toEqual([])
    expect(groups[2]?.slots[0]).toMatchObject({
      row: 1,
      col: 1,
      tile: expect.objectContaining({ game_id: 1 }),
    })
  })

  it('finds the nearest free cell in the same group, row-boundary free', () => {
    const tiles = [
      makeTile(1, 'small', { column_index: 0, grid_row: 0, grid_col: 0 }),
    ]

    expect(findStartScreenDropTarget(tiles, 99, 0, 0, 0, 'small'))
      .toEqual({ columnIndex: 0, row: 0, col: 2 })

    // 6 个 2x2 占满前两行后，目标落到第 3 行，而不是另开一列
    const fullRow = Array.from({ length: 6 }, (_, index) =>
      makeTile(index + 10, 'small', { column_index: 0, grid_row: 0, grid_col: index * 2 }),
    )
    expect(findStartScreenDropTarget(fullRow, 99, 0, 0, 0, 'small'))
      .toEqual({ columnIndex: 0, row: 2, col: 0 })
  })
})
