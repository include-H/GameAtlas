import { describe, expect, it } from 'vitest'
import { packStartScreenTiles } from './start-screen-layout'
import type { StartScreenTile } from '@/services/types'

const makeTile = (gameId: number, tileSize: StartScreenTile['tile_size']): StartScreenTile => ({
  game_id: gameId,
  public_id: `game-${gameId}`,
  title: `Game ${gameId}`,
  cover_image: null,
  banner_image: null,
  tile_size: tileSize,
  image_small_path: null,
  image_wide_path: null,
  image_large_path: null,
  sort_order: 0,
})

describe('packStartScreenTiles', () => {
  it('fills a full column: one large, two wides, four smalls as 2x2', () => {
    const columns = packStartScreenTiles([
      makeTile(1, 'large'),
      makeTile(2, 'wide'),
      makeTile(3, 'wide'),
      makeTile(4, 'small'),
      makeTile(5, 'small'),
      makeTile(6, 'small'),
      makeTile(7, 'small'),
    ])

    expect(columns).toHaveLength(1)
    const slots = columns[0]?.slots ?? []
    expect(slots.find((s) => s.tile.game_id === 1)).toMatchObject({ row: 0, col: 0 })
    expect(slots.find((s) => s.tile.game_id === 2)).toMatchObject({ row: 2, col: 0 })
    expect(slots.find((s) => s.tile.game_id === 3)).toMatchObject({ row: 3, col: 0 })
    for (let index = 0; index < 4; index += 1) {
      expect(slots.find((s) => s.tile.game_id === 4 + index)).toMatchObject({
        row: 4 + Math.floor(index / 2),
        col: index % 2,
      })
    }
  })

  it('freely stacks three large tiles in one column', () => {
    const columns = packStartScreenTiles([
      makeTile(1, 'large'),
      makeTile(2, 'large'),
      makeTile(3, 'large'),
    ])

    expect(columns).toHaveLength(1)
    const slots = columns[0]?.slots ?? []
    expect(slots.map((s) => s.row)).toEqual([0, 2, 4])
  })

  it('opens a new column only when the current one is geometrically full', () => {
    const columns = packStartScreenTiles([
      makeTile(1, 'small'),
      makeTile(2, 'small'),
      makeTile(3, 'small'),
      makeTile(4, 'small'),
      makeTile(5, 'small'),
      makeTile(6, 'small'),
      makeTile(7, 'small'),
      makeTile(8, 'small'),
      makeTile(9, 'small'),
      makeTile(10, 'small'),
      makeTile(11, 'small'),
      makeTile(12, 'small'),
      makeTile(13, 'small'),
    ])

    expect(columns).toHaveLength(2)
    expect(columns[0]?.slots).toHaveLength(12)
    expect(columns[1]?.slots.map((s) => s.tile.game_id)).toEqual([13])
    expect(columns[1]?.slots[0]).toMatchObject({ row: 0, col: 0 })
  })

  it('lets a wide tile take the next row when a lone small blocks the first row', () => {
    const columns = packStartScreenTiles([
      makeTile(1, 'small'),
      makeTile(2, 'wide'),
    ])

    const slots = columns[0]?.slots ?? []
    expect(slots.find((s) => s.tile.game_id === 1)).toMatchObject({ row: 0, col: 0 })
    expect(slots.find((s) => s.tile.game_id === 2)).toMatchObject({ row: 1, col: 0 })
  })

  it('keeps global order indices for drag reordering', () => {
    const columns = packStartScreenTiles([
      makeTile(1, 'small'),
      makeTile(2, 'large'),
      makeTile(3, 'small'),
    ])

    expect(columns[0]?.slots.map((s) => s.globalIndex)).toEqual([0, 1, 2])
  })
})
