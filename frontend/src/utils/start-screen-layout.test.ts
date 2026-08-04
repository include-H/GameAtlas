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
  it('packs one large, two wides and four smalls into one column', () => {
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

  it('opens a new column when the small quota is exceeded', () => {
    const columns = packStartScreenTiles([
      makeTile(1, 'small'),
      makeTile(2, 'small'),
      makeTile(3, 'small'),
      makeTile(4, 'small'),
      makeTile(5, 'small'),
    ])

    expect(columns).toHaveLength(2)
    expect(columns[0]?.slots.map((s) => s.tile.game_id)).toEqual([1, 2, 3, 4])
    expect(columns[1]?.slots.map((s) => s.tile.game_id)).toEqual([5])
    expect(columns[1]?.slots[0]).toMatchObject({ row: 0, col: 0 })
  })

  it('compacts a column without a large tile', () => {
    const columns = packStartScreenTiles([
      makeTile(1, 'wide'),
      makeTile(2, 'wide'),
      makeTile(3, 'small'),
      makeTile(4, 'small'),
      makeTile(5, 'small'),
      makeTile(6, 'small'),
    ])

    expect(columns).toHaveLength(1)
    const slots = columns[0]?.slots ?? []
    expect(slots.find((s) => s.tile.game_id === 1)).toMatchObject({ row: 0, col: 0 })
    expect(slots.find((s) => s.tile.game_id === 2)).toMatchObject({ row: 1, col: 0 })
    expect(slots.find((s) => s.tile.game_id === 3)).toMatchObject({ row: 2, col: 0 })
    expect(slots.find((s) => s.tile.game_id === 4)).toMatchObject({ row: 2, col: 1 })
    expect(slots.find((s) => s.tile.game_id === 5)).toMatchObject({ row: 3, col: 0 })
    expect(slots.find((s) => s.tile.game_id === 6)).toMatchObject({ row: 3, col: 1 })
  })

  it('keeps global order indices for drag reordering', () => {
    const columns = packStartScreenTiles([
      makeTile(1, 'small'),
      makeTile(2, 'small'),
      makeTile(3, 'large'),
    ])

    expect(columns[0]?.slots.map((s) => s.globalIndex)).toEqual([0, 1, 2])
  })
})
