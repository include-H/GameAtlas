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
  it('builds the canonical column: one large, two wide, four small', () => {
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
      expect(slots.find((s) => s.tile.game_id === 4 + index)).toMatchObject({ row: 4, col: index })
    }
  })

  it('opens a new column when a size quota is exceeded', () => {
    const columns = packStartScreenTiles([
      makeTile(1, 'large'),
      makeTile(2, 'large'),
      makeTile(3, 'wide'),
      makeTile(4, 'wide'),
      makeTile(5, 'wide'),
    ])

    expect(columns).toHaveLength(2)
    expect(columns[0]?.slots.map((s) => s.tile.game_id)).toEqual([1, 3, 4])
    expect(columns[1]?.slots.map((s) => s.tile.game_id)).toEqual([2, 5])
    expect(columns[1]?.slots.find((s) => s.tile.game_id === 5)).toMatchObject({ row: 2, col: 0 })
  })

  it('compacts columns without a large tile so wides and smalls stay at the top', () => {
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
    expect(slots.find((s) => s.tile.game_id === 6)).toMatchObject({ row: 2, col: 3 })
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
