import { describe, expect, it, vi } from 'vitest'
import { useStartScreen } from './useStartScreen'
import type { GameListItem, StartScreenTile } from '@/services/types'

const makeGame = (publicId: string, id = 1): GameListItem => ({
  id,
  public_id: publicId,
  title: publicId,
  title_alt: null,
  visibility: 'public',
  summary: null,
  release_date: null,
  cover_image: null,
  banner_image: null,
  wiki_content: null,
  downloads: 0,
  primary_screenshot: null,
  logo_visible: true,
  isFavorite: false,
  series: null,
  created_at: '',
  updated_at: '',
} as unknown as GameListItem)

const makeTile = (gameId: number, publicId: string, tileSize: StartScreenTile['tile_size'] = 'small'): StartScreenTile => ({
  game_id: gameId,
  public_id: publicId,
  title: publicId,
  cover_image: null,
  tile_size: tileSize,
  sort_order: 0,
})

const createScreen = (overrides: Partial<{
  fetchTiles: () => Promise<StartScreenTile[]>
  fetchFavorites: () => Promise<GameListItem[]>
  saveTiles: () => Promise<StartScreenTile[]>
  addAlert: () => void
}> = {}) => {
  const fetchTiles = overrides.fetchTiles ?? vi.fn().mockResolvedValue([])
  const fetchFavorites = overrides.fetchFavorites ?? vi.fn().mockResolvedValue([])
  const saveTiles = overrides.saveTiles ?? vi.fn().mockResolvedValue([])
  const addAlert = overrides.addAlert ?? vi.fn()
  const screen = useStartScreen({
    fetchTiles,
    fetchFavorites,
    saveTiles,
    addAlert,
  })
  return { screen, fetchTiles, fetchFavorites, saveTiles, addAlert }
}

describe('useStartScreen', () => {
  it('falls back to favorites as default tiles when no tiles are saved', async () => {
    const { screen, fetchTiles, fetchFavorites } = createScreen({
      fetchTiles: vi.fn().mockResolvedValue([]),
      fetchFavorites: vi.fn().mockResolvedValue([makeGame('a', 1), makeGame('b', 2)]),
    })

    screen.open()
    await vi.waitFor(() => expect(screen.isLoading.value).toBe(false))

    expect(fetchTiles).toHaveBeenCalledTimes(1)
    expect(fetchFavorites).toHaveBeenCalledTimes(1)
    expect(screen.tiles.value.map((tile) => tile.public_id)).toEqual(['a', 'b'])
    expect(screen.tiles.value.every((tile) => tile.tile_size === 'small')).toBe(true)
  })

  it('uses saved tiles and skips the favorites fallback', async () => {
    const { screen, fetchFavorites } = createScreen({
      fetchTiles: vi.fn().mockResolvedValue([makeTile(1, 'a', 'wide')]),
      fetchFavorites: vi.fn(),
    })

    screen.open()
    await vi.waitFor(() => expect(screen.isLoading.value).toBe(false))

    expect(fetchFavorites).not.toHaveBeenCalled()
    expect(screen.tiles.value).toEqual([makeTile(1, 'a', 'wide')])
  })

  it('alerts and keeps a retryable failure state when loading fails', async () => {
    const addAlert = vi.fn()
    const { screen } = createScreen({
      fetchTiles: vi.fn().mockRejectedValue(new Error('boom')),
      addAlert,
    })

    screen.open()
    await vi.waitFor(() => expect(screen.hasLoadFailure.value).toBe(true))

    expect(addAlert).toHaveBeenCalledWith('开始屏幕加载失败，请稍后重试', 'error')
  })

  it('refetches tiles every time the screen is opened', async () => {
    const fetchTiles = vi.fn().mockResolvedValue([makeTile(1, 'a')])
    const { screen } = createScreen({ fetchTiles })

    screen.open()
    await vi.waitFor(() => expect(screen.isLoading.value).toBe(false))
    screen.close()
    screen.open()
    await vi.waitFor(() => expect(screen.isLoading.value).toBe(false))

    expect(fetchTiles).toHaveBeenCalledTimes(2)
  })

  it('cancels editing and restores the original tiles', async () => {
    const { screen } = createScreen({
      fetchTiles: vi.fn().mockResolvedValue([makeTile(1, 'a'), makeTile(2, 'b')]),
    })

    screen.open()
    await vi.waitFor(() => expect(screen.isLoading.value).toBe(false))
    await screen.startEdit()
    screen.removeTile(1)
    expect(screen.tiles.value.map((tile) => tile.public_id)).toEqual(['b'])

    screen.cancelEdit()
    expect(screen.tiles.value.map((tile) => tile.public_id)).toEqual(['a', 'b'])
    expect(screen.isEditing.value).toBe(false)
  })

  it('saves the current tile arrangement', async () => {
    const saveTiles = vi.fn().mockResolvedValue([makeTile(1, 'a', 'wide')])
    const addAlert = vi.fn()
    const { screen } = createScreen({
      fetchTiles: vi.fn().mockResolvedValue([makeTile(1, 'a', 'small')]),
      saveTiles,
      addAlert,
    })

    screen.open()
    await vi.waitFor(() => expect(screen.isLoading.value).toBe(false))
    await screen.startEdit()
    screen.resizeTile(1)
    await screen.saveEdit()

    expect(saveTiles).toHaveBeenCalledWith([{ game_id: 1, tile_size: 'wide' }])
    expect(screen.tiles.value[0]?.tile_size).toBe('wide')
    expect(screen.isEditing.value).toBe(false)
    expect(addAlert).toHaveBeenCalledWith('开始屏幕已保存', 'success')
  })

  it('cycles tile sizes small -> wide -> large -> small', async () => {
    const { screen } = createScreen({
      fetchTiles: vi.fn().mockResolvedValue([makeTile(1, 'a', 'small')]),
    })
    screen.open()
    await vi.waitFor(() => expect(screen.isLoading.value).toBe(false))

    screen.resizeTile(1)
    expect(screen.tiles.value[0]?.tile_size).toBe('wide')
    screen.resizeTile(1)
    expect(screen.tiles.value[0]?.tile_size).toBe('large')
    screen.resizeTile(1)
    expect(screen.tiles.value[0]?.tile_size).toBe('small')
  })

  it('reorders tiles with moveTile', async () => {
    const { screen } = createScreen({
      fetchTiles: vi.fn().mockResolvedValue([makeTile(1, 'a'), makeTile(2, 'b'), makeTile(3, 'c')]),
    })
    screen.open()
    await vi.waitFor(() => expect(screen.isLoading.value).toBe(false))

    screen.moveTile(0, 2)
    expect(screen.tiles.value.map((tile) => tile.public_id)).toEqual(['b', 'c', 'a'])
  })

  it('adds a tile from the favorite pool and ignores duplicates', async () => {
    const { screen } = createScreen({
      fetchTiles: vi.fn().mockResolvedValue([makeTile(1, 'a')]),
    })
    screen.open()
    await vi.waitFor(() => expect(screen.isLoading.value).toBe(false))

    screen.addTile(makeGame('b', 2))
    screen.addTile(makeGame('b', 2))
    expect(screen.tiles.value.map((tile) => tile.public_id)).toEqual(['a', 'b'])
  })
})
