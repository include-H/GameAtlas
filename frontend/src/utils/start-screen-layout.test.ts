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

  it('keeps the requested target when the spot is occupied (displacement applied on drop)', () => {
    const tiles = [
      makeTile(1, 'small', { column_index: 0, grid_row: 0, grid_col: 0 }),
    ]

    expect(findStartScreenDropTarget(tiles, 99, 0, 0, 0, 'small'))
      .toEqual({ columnIndex: 0, row: 0, col: 0 })

    // 6 个 2x2 占满前两行：落点不再 first-fit 弹走，冲突保留请求坐标，
    // 由 planStartScreenInsertion 产出避让链（推土机：被覆盖者下移一格）
    const fullRow = Array.from({ length: 6 }, (_, index) =>
      makeTile(index + 10, 'small', { column_index: 0, grid_row: 0, grid_col: index * 2 }),
    )
    expect(findStartScreenDropTarget(fullRow, 99, 0, 0, 0, 'small'))
      .toEqual({ columnIndex: 0, row: 0, col: 0 })
    const plan = planStartScreenInsertion(fullRow, 99, 0, 0, 0, 'small')
    expect(plan?.target).toEqual({ columnIndex: 0, row: 0, col: 0 })
    // 被覆盖的磁贴（落点 row0 col0-1 相交者）顺延一格
    expect(plan?.moves.map((m) => m.gameId)).toEqual([10])
    expect(plan?.moves.find((m) => m.gameId === 10)).toMatchObject({ columnIndex: 0, row: 2, col: 0 })
  })

  // 吸附：指针附近"空闲矩形恰好等于磁贴尺寸"的精确空位，拖到区域内任意处即吸附
  it('snaps a wide tile into an exact 2x4 gap when the pointer is inside it', () => {
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

    // 指针在空位右下角格子（row3 col7），请求坐标放不下 2x4 → 吸附到左上角
    expect(findStartScreenDropTarget(tiles, 99, 0, 3, 7, 'wide'))
      .toEqual({ columnIndex: 0, row: 2, col: 4 })
    // 指针在空位内任意格子（row2 col6）同样吸附
    expect(findStartScreenDropTarget(tiles, 99, 0, 2, 6, 'wide'))
      .toEqual({ columnIndex: 0, row: 2, col: 4 })
    // 指针在空位上方 1 格也吸附
    expect(findStartScreenDropTarget(tiles, 99, 0, 1, 5, 'wide'))
      .toEqual({ columnIndex: 0, row: 2, col: 4 })
  })

  it('does not snap into a gap larger than the dragged tile', () => {
    // row0-1 满，row2-3 只留 col4-11（2x8 空位，大于 2x4）→ 不吸附，走 first-fit
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
    // 指针在空位边缘外 2 格（row2 col0 上方区域被占时）不吸附：走 first-fit
    // 构造：row0-1 全满、row2-3 只留 col4-11，指针在 row0 col2（被占）→ first-fit
    const fullTop = [
      ...tiles,
      makeTile(9, 'small', { column_index: 0, grid_row: 2, grid_col: 4 }),
      makeTile(10, 'small', { column_index: 0, grid_row: 2, grid_col: 6 }),
      makeTile(11, 'small', { column_index: 0, grid_row: 2, grid_col: 8 }),
      makeTile(12, 'small', { column_index: 0, grid_row: 2, grid_col: 10 }),
    ]
    const target = findStartScreenDropTarget(fullTop, 99, 0, 0, 2, 'wide')
    // 冲突不再 first-fit 弹走：保留请求坐标（clamp 后 col2），避让由落位层处理
    expect(target).toEqual({ columnIndex: 0, row: 0, col: 2 })
  })

  it('plans insertion between two tiles: the dragged tile lands, others dodge', () => {
    // 狙击手 (0,0)、最终幻想 (0,2) 相邻，黑手党 2x2 拖到"中间"（col1 row0）
    const tiles = [
      makeTile(1, 'small', { column_index: 0, grid_row: 0, grid_col: 0 }),
      makeTile(2, 'small', { column_index: 0, grid_row: 0, grid_col: 2 }),
    ]

    const plan = planStartScreenInsertion(tiles, 99, 0, 0, 1, 'small')
    // 落点 = 用户拖的位置（不被弹走）
    expect(plan?.target).toEqual({ columnIndex: 0, row: 0, col: 1 })
    // 推土机：两个被覆盖磁贴各下移一格（保持列对齐），不再向右借位
    expect(plan?.moves).toEqual([
      { gameId: 1, columnIndex: 0, row: 2, col: 0 },
      { gameId: 2, columnIndex: 0, row: 2, col: 2 },
    ])
  })

  it('plans chained displacement when a moved tile covers another', () => {
    // row0: 3 个 2x2 占 col0/2/4；row2: 2 个 2x2 占 col0/2
    // 把 col4 磁贴拖到 col0（覆盖 col0 磁贴），col0 磁贴被推后覆盖 row2 col0
    const tiles = [
      makeTile(1, 'small', { column_index: 0, grid_row: 0, grid_col: 0 }),
      makeTile(2, 'small', { column_index: 0, grid_row: 0, grid_col: 2 }),
      makeTile(3, 'small', { column_index: 0, grid_row: 0, grid_col: 4 }),
      makeTile(4, 'small', { column_index: 0, grid_row: 2, grid_col: 0 }),
      makeTile(5, 'small', { column_index: 0, grid_row: 2, grid_col: 2 }),
    ]

    // 磁贴3（col4）拖到 row0 col0
    const plan = planStartScreenInsertion(tiles, 3, 0, 0, 0, 'small')
    expect(plan?.target).toEqual({ columnIndex: 0, row: 0, col: 0 })
    // 磁贴1 被覆盖 → 下移一格 (2,0)；磁贴4（row2 col0）被磁贴1 新位置顶住 → 链式下移 (4,0)
    const moved = plan?.moves ?? []
    expect(moved.map((m) => m.gameId)).toEqual([4, 1])
    expect(moved.find((m) => m.gameId === 4)).toMatchObject({ columnIndex: 0, row: 4, col: 0 })
    expect(moved.find((m) => m.gameId === 1)).toMatchObject({ columnIndex: 0, row: 2, col: 0 })
    // 所有磁贴最终互不重叠
    const final = [
      { id: 3, row: 0, col: 0 },
      ...moved,
    ]
    for (let i = 0; i < final.length; i += 1) {
      for (let j = i + 1; j < final.length; j += 1) {
        const a = final[i]
        const b = final[j]
        expect(a.row + 2 <= b.row || b.row + 2 <= a.row || a.col + 2 <= b.col || b.col + 2 <= a.col)
          .toBe(true)
      }
    }
  })

  it('does not open a new column while the group has free horizontal space', () => {
    // 组0 只有 row10 的两个 2x2（横向 12 列远未占满）
    // 落点 row10 col0 覆盖 row10 col0 的磁贴 → 下移一格越界 → 组内还有空位 → 横向回填，不开新列
    const tiles = [
      makeTile(1, 'small', { column_index: 0, grid_row: 10, grid_col: 0 }),
      makeTile(2, 'small', { column_index: 0, grid_row: 10, grid_col: 2 }),
    ]

    const plan = planStartScreenInsertion(tiles, 99, 0, 10, 0, 'small')
    expect(plan?.target).toEqual({ columnIndex: 0, row: 10, col: 0 })
    const moved = plan?.moves ?? []
    // 磁贴1 被顶：组底越界 → 组内 12 行内找空位（row0 起横向 first-fit）→ 回填组内，不开新列
    expect(moved.find((m) => m.gameId === 1)).toMatchObject({ columnIndex: 0, row: 0, col: 0 })
    expect(moved.find((m) => m.gameId === 2)).toBeUndefined()
  })

  it('spills to the right column only when the group is fully packed (chain)', () => {
    // 组0：9 个 4x4 填满 12 行 × 12 列（3 并排 × 3 层）
    // 组1：1 个 4x4 在顶部
    const tiles = []
    for (let layer = 0; layer < 3; layer += 1) {
      for (let col = 0; col <= 8; col += 4) {
        tiles.push(
          makeTile(10 + layer * 3 + col / 4, 'large', {
            column_index: 0,
            grid_row: layer * 4,
            grid_col: col,
          }),
        )
      }
    }
    tiles.push(makeTile(99, 'large', { column_index: 1, grid_row: 0, grid_col: 0 }))

    // 落点 row10 col0 覆盖组0 底部 large（row8-11 col0-3）→ 下移越界 → 组内已满 → 溢右
    const plan = planStartScreenInsertion(tiles, 100, 0, 10, 0, 'large')
    expect(plan?.target).toEqual({ columnIndex: 0, row: 8, col: 0 })
    const moved = plan?.moves ?? []
    // 组0 底部的 large（gameId 16，row8 col0）被顶：组0 无空位 → 溢到组1 顶部，
    // 顶到组1 的 large(99) → 99 继续下移 (4,0)
    expect(moved.find((m) => m.gameId === 99)).toMatchObject({ columnIndex: 1, row: 4, col: 0 })
    expect(moved.find((m) => m.gameId === 16)).toMatchObject({ columnIndex: 1, row: 0, col: 0 })
    // 落点磁贴 row10 clamp 到 row8
    expect(plan?.target.row).toBe(8)
  })
})
