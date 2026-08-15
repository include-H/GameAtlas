import { describe, expect, it } from 'vitest'
import {
  findStartScreenDropTarget,
  layoutStartScreenTiles,
  normalizeStartScreenTiles,
  packStartScreenTiles,
  planStartScreenInsertion,
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

  it('preserves explicit positions for the drag preview', () => {
    const groups = layoutStartScreenTiles([
      makeTile(1, 'wide', { column_index: 0, grid_row: 4, grid_col: 6 }),
    ], 1, false)

    expect(groups[0]?.slots[0]).toMatchObject({ row: 4, col: 6 })
  })

  it('normalizes overlapping explicit placements within a group', () => {
    const normalized = normalizeStartScreenTiles([
      makeTile(1, 'large', { column_index: 0, grid_row: 0, grid_col: 0 }),
      makeTile(2, 'small', { column_index: 0, grid_row: 0, grid_col: 0 }),
    ])

    expect(normalized[0]).toMatchObject({ column_index: 0, grid_row: 0, grid_col: 0 })
    expect(normalized[1]).toMatchObject({ column_index: 0, grid_row: 0, grid_col: 4 })
  })

  it('compresses empty rows: 1 2 空 3 4 空空 5 重映射为连续行带', () => {
    // row0:1,2  row2:3,4  row5:5（row4 空带被压缩；2x2 磁贴行带间距保留）
    const normalized = normalizeStartScreenTiles([
      makeTile(1, 'small', { column_index: 0, grid_row: 0, grid_col: 0 }),
      makeTile(2, 'small', { column_index: 0, grid_row: 0, grid_col: 2 }),
      makeTile(3, 'small', { column_index: 0, grid_row: 2, grid_col: 0 }),
      makeTile(4, 'small', { column_index: 0, grid_row: 2, grid_col: 2 }),
      makeTile(5, 'small', { column_index: 0, grid_row: 5, grid_col: 0 }),
    ])

    expect(normalized.find((t) => t.game_id === 1)).toMatchObject({ grid_row: 0, grid_col: 0 })
    expect(normalized.find((t) => t.game_id === 2)).toMatchObject({ grid_row: 0, grid_col: 2 })
    expect(normalized.find((t) => t.game_id === 3)).toMatchObject({ grid_row: 2, grid_col: 0 })
    expect(normalized.find((t) => t.game_id === 4)).toMatchObject({ grid_row: 2, grid_col: 2 })
    expect(normalized.find((t) => t.game_id === 5)).toMatchObject({ grid_row: 4, grid_col: 0 })
  })

  it('keeps explicit empty groups in the layout', () => {
    const groups = layoutStartScreenTiles([
      makeTile(1, 'small', { column_index: 2, grid_row: 1, grid_col: 1 }),
    ], 4)

    expect(groups).toHaveLength(4)
    expect(groups[0]?.slots).toEqual([])
    expect(groups[1]?.slots).toEqual([])
    expect(groups[2]?.slots[0]).toMatchObject({
      row: 0, // 空行压缩：组内单磁贴 row1 → row0
      col: 1,
      tile: expect.objectContaining({ game_id: 1 }),
    })
  })

  it('keeps the requested target when the spot is occupied', () => {
    const tiles = [
      makeTile(1, 'small', { column_index: 0, grid_row: 0, grid_col: 0 }),
    ]

    // 请求坐标可放 → 直落
    expect(findStartScreenDropTarget(tiles, 99, 0, 0, 2, 'small'))
      .toEqual({ columnIndex: 0, row: 0, col: 2 })

    // 6 个 2x2 占满前两行：请求坐标冲突、附近没有精确空位 → 保留请求坐标
    const fullRow = Array.from({ length: 6 }, (_, index) =>
      makeTile(index + 10, 'small', { column_index: 0, grid_row: 0, grid_col: index * 2 }),
    )
    expect(findStartScreenDropTarget(fullRow, 99, 0, 0, 0, 'small'))
      .toEqual({ columnIndex: 0, row: 0, col: 0 })
  })

  it('does not auto-place a conflicted drop into a nearby gap', () => {
    // row0-1 占满 12 列 → row2-3 只留 col4-7（2x4 精确空位）
    const tiles = [
      makeTile(1, 'small', { column_index: 0, grid_row: 0, grid_col: 0 }),
      makeTile(2, 'small', { column_index: 0, grid_row: 0, grid_col: 2 }),
      makeTile(3, 'small', { column_index: 0, grid_row: 0, grid_col: 4 }),
      makeTile(4, 'small', { column_index: 0, grid_row: 0, grid_col: 6 }),
      makeTile(5, 'small', { column_index: 0, grid_row: 0, grid_col: 8 }),
      makeTile(6, 'small', { column_index: 0, grid_row: 0, grid_col: 10 }),
      makeTile(7, 'small', { column_index: 0, grid_row: 2, grid_col: 0 }),
      makeTile(8, 'small', { column_index: 0, grid_row: 2, grid_col: 2 }),
      makeTile(9, 'small', { column_index: 0, grid_row: 2, grid_col: 8 }),
      makeTile(10, 'small', { column_index: 0, grid_row: 2, grid_col: 10 }),
    ]

    // 请求坐标冲突，但附近存在精确 2x4 空位 → 仍保留请求坐标
    expect(findStartScreenDropTarget(tiles, 99, 0, 3, 7, 'wide'))
      .toEqual({ columnIndex: 0, row: 3, col: 7 })
    expect(findStartScreenDropTarget(tiles, 99, 0, 2, 6, 'wide'))
      .toEqual({ columnIndex: 0, row: 2, col: 6 })
    expect(findStartScreenDropTarget(tiles, 99, 0, 1, 5, 'wide'))
      .toEqual({ columnIndex: 0, row: 1, col: 5 })
  })

  it('keeps an occupied request for insertion planning', () => {
    // row0-1 满，row2-3 只留 col4-11（2x8 空位，大于 2x4）→ 不吸附
    const tiles = [
      makeTile(1, 'small', { column_index: 0, grid_row: 0, grid_col: 0 }),
      makeTile(2, 'small', { column_index: 0, grid_row: 0, grid_col: 2 }),
      makeTile(3, 'small', { column_index: 0, grid_row: 0, grid_col: 4 }),
      makeTile(4, 'small', { column_index: 0, grid_row: 0, grid_col: 6 }),
      makeTile(5, 'small', { column_index: 0, grid_row: 0, grid_col: 8 }),
      makeTile(6, 'small', { column_index: 0, grid_row: 0, grid_col: 10 }),
      makeTile(7, 'small', { column_index: 0, grid_row: 2, grid_col: 0 }),
      makeTile(8, 'small', { column_index: 0, grid_row: 2, grid_col: 2 }),
    ]

    // 指针在空位内（row2 col10，clamp 到 col8），2x4 放得下 → 直接落点
    expect(findStartScreenDropTarget(tiles, 99, 0, 2, 10, 'wide'))
      .toEqual({ columnIndex: 0, row: 2, col: 8 })
    // 构造：row0-1 全满、row2-3 只留 col4-11，指针在 row0 col2（被占）
    const fullTop = [
      ...tiles,
      makeTile(9, 'small', { column_index: 0, grid_row: 2, grid_col: 4 }),
      makeTile(10, 'small', { column_index: 0, grid_row: 2, grid_col: 6 }),
      makeTile(11, 'small', { column_index: 0, grid_row: 2, grid_col: 8 }),
      makeTile(12, 'small', { column_index: 0, grid_row: 2, grid_col: 10 }),
    ]
    const target = findStartScreenDropTarget(fullTop, 99, 0, 0, 2, 'wide')
    // 请求坐标冲突 → 仍保留请求坐标，由插入规划处理
    expect(target).toEqual({ columnIndex: 0, row: 0, col: 2 })
  })

  it('plans insertion between two tiles by pushing both down', () => {
    const tiles = [
      makeTile(1, 'small', { column_index: 0, grid_row: 0, grid_col: 0 }),
      makeTile(2, 'small', { column_index: 0, grid_row: 0, grid_col: 2 }),
    ]

    const plan = planStartScreenInsertion(tiles, 99, 0, 0, 1, 'small')
    expect(plan.target).toEqual({ columnIndex: 0, row: 0, col: 1 })
    expect(plan.moves).toEqual([
      { gameId: 1, columnIndex: 0, row: 2, col: 0 },
      { gameId: 2, columnIndex: 0, row: 2, col: 2 },
    ])
  })

  it('plans chained displacement when a pushed tile meets another tile', () => {
    const tiles = [
      makeTile(1, 'small', { column_index: 0, grid_row: 0, grid_col: 0 }),
      makeTile(2, 'small', { column_index: 0, grid_row: 0, grid_col: 2 }),
      makeTile(3, 'small', { column_index: 0, grid_row: 0, grid_col: 4 }),
      makeTile(4, 'small', { column_index: 0, grid_row: 2, grid_col: 0 }),
      makeTile(5, 'small', { column_index: 0, grid_row: 2, grid_col: 2 }),
    ]

    const plan = planStartScreenInsertion(tiles, 3, 0, 0, 0, 'small')
    expect(plan.target).toEqual({ columnIndex: 0, row: 0, col: 0 })
    expect(plan.moves.map((move) => move.gameId)).toEqual([4, 1])
    expect(plan.moves.find((move) => move.gameId === 4)).toMatchObject({ row: 4, col: 0 })
    expect(plan.moves.find((move) => move.gameId === 1)).toMatchObject({ row: 2, col: 0 })
  })

  it('spills a pushed tile into the next group after the 12-row limit', () => {
    const tiles = Array.from({ length: 36 }, (_, index) =>
      makeTile(index + 1, 'small', {
        column_index: 0,
        grid_row: Math.floor(index / 6) * 2,
        grid_col: (index % 6) * 2,
      }),
    )
    tiles.push(makeTile(100, 'large', { column_index: 1, grid_row: 0, grid_col: 0 }))

    const plan = planStartScreenInsertion(tiles, 99, 0, 10, 0, 'small')
    expect(plan.target).toEqual({ columnIndex: 0, row: 10, col: 0 })
    expect(plan.moves.find((move) => move.gameId === 31)).toMatchObject({
      columnIndex: 1,
      row: 0,
      col: 0,
    })
    expect(plan.moves.find((move) => move.gameId === 100)).toMatchObject({ row: 4, col: 0 })
  })
})
