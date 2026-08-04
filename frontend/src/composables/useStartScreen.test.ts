import { describe, expect, it, vi } from 'vitest'
import { useStartScreen } from './useStartScreen'
import type {
  GameListItem,
  StartScreenColumn,
  StartScreenLayout,
  StartScreenTile,
} from '@/services/types'

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
  banner_image: null,
  tile_size: tileSize,
  image_small_path: null,
  image_wide_path: null,
  image_large_path: null,
  sort_order: 0,
})

const makeColumn = (name: string, id = 1): StartScreenColumn => ({
  id,
  name,
  sort_order: 0,
})

const makeLayout = (tiles: StartScreenTile[], columns: StartScreenColumn[] = []): StartScreenLayout => ({
  tiles,
  columns,
})

const createScreen = (overrides: Partial<{
  fetchTiles: () => Promise<StartScreenLayout>
  fetchFavorites: () => Promise<GameListItem[]>
  saveTiles: () => Promise<StartScreenLayout>
  uploadTileImage: () => Promise<string>
  addAlert: () => void
}> = {}) => {
  const fetchTiles = overrides.fetchTiles ?? vi.fn().mockResolvedValue(makeLayout([]))
  const fetchFavorites = overrides.fetchFavorites ?? vi.fn().mockResolvedValue([])
  const saveTiles = overrides.saveTiles ?? vi.fn().mockResolvedValue(makeLayout([]))
  const uploadTileImage = overrides.uploadTileImage ?? vi.fn().mockResolvedValue('/assets/start-screen/tile.png')
  const addAlert = overrides.addAlert ?? vi.fn()
  const screen = useStartScreen({
    fetchTiles,
    fetchFavorites,
    saveTiles,
    uploadTileImage,
    addAlert,
  })
  return { screen, fetchTiles, fetchFavorites, saveTiles, uploadTileImage, addAlert }
}

describe('useStartScreen', () => {
  it('falls back to favorites as default tiles when nothing is saved', async () => {
    const { screen, fetchTiles, fetchFavorites } = createScreen({
      fetchTiles: vi.fn().mockResolvedValue(makeLayout([])),
      fetchFavorites: vi.fn().mockResolvedValue([makeGame('a', 1), makeGame('b', 2)]),
    })

    screen.open()
    await vi.waitFor(() => expect(screen.isLoading.value).toBe(false))

    expect(fetchTiles).toHaveBeenCalledTimes(1)
    expect(fetchFavorites).toHaveBeenCalledTimes(1)
    expect(screen.tiles.value.map((tile) => tile.public_id)).toEqual(['a', 'b'])
    expect(screen.columns.value).toEqual([])
  })

  it('uses the saved layout with column names', async () => {
    const { screen, fetchFavorites } = createScreen({
      fetchTiles: vi.fn().mockResolvedValue(makeLayout(
        [makeTile(1, 'a', 'wide')],
        [makeColumn('我的收藏')],
      )),
      fetchFavorites: vi.fn(),
    })

    screen.open()
    await vi.waitFor(() => expect(screen.isLoading.value).toBe(false))

    expect(fetchFavorites).not.toHaveBeenCalled()
    expect(screen.tiles.value).toEqual([makeTile(1, 'a', 'wide')])
    expect(screen.columns.value[0]?.name).toBe('我的收藏')
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

  it('refetches the layout every time the screen is opened', async () => {
    const fetchTiles = vi.fn().mockResolvedValue(makeLayout([makeTile(1, 'a')]))
    const { screen } = createScreen({ fetchTiles })

    screen.open()
    await vi.waitFor(() => expect(screen.isLoading.value).toBe(false))
    screen.close()
    screen.open()
    await vi.waitFor(() => expect(screen.isLoading.value).toBe(false))

    expect(fetchTiles).toHaveBeenCalledTimes(2)
  })

  it('cancels editing and restores both tiles and column names', async () => {
    const { screen } = createScreen({
      fetchTiles: vi.fn().mockResolvedValue(makeLayout(
        [makeTile(1, 'a'), makeTile(2, 'b')],
        [makeColumn('第一列')],
      )),
    })

    screen.open()
    await vi.waitFor(() => expect(screen.isLoading.value).toBe(false))
    await screen.startEdit()
    screen.removeTile(1)
    screen.renameColumn(0, '改过的名字')
    screen.cancelEdit()

    expect(screen.tiles.value.map((tile) => tile.public_id)).toEqual(['a', 'b'])
    expect(screen.columns.value[0]?.name).toBe('第一列')
    expect(screen.isEditing.value).toBe(false)
  })

  it('saves tiles together with column names', async () => {
    const saveTiles = vi.fn().mockResolvedValue(makeLayout(
      [makeTile(1, 'a', 'wide')],
      [makeColumn('改名')],
    ))
    const addAlert = vi.fn()
    const { screen } = createScreen({
      fetchTiles: vi.fn().mockResolvedValue(makeLayout(
        [makeTile(1, 'a', 'small')],
        [makeColumn('第一列')],
      )),
      saveTiles,
      addAlert,
    })

    screen.open()
    await vi.waitFor(() => expect(screen.isLoading.value).toBe(false))
    await screen.startEdit()
    screen.resizeTile(1)
    screen.renameColumn(0, '改名')
    await screen.saveEdit()

    expect(saveTiles).toHaveBeenCalledWith({
      columns: [{ name: '改名' }],
      tiles: [{
        game_id: 1,
        tile_size: 'wide',
        image_small_path: null,
        image_wide_path: null,
        image_large_path: null,
      }],
    })
    expect(screen.columns.value[0]?.name).toBe('改名')
    expect(screen.isEditing.value).toBe(false)
    expect(addAlert).toHaveBeenCalledWith('开始屏幕已保存', 'success')
  })

  it('pads column names to match packed columns when saving', async () => {
    const saveTiles = vi.fn().mockResolvedValue(makeLayout([], [makeColumn('一'), makeColumn('二')]))
    const { screen } = createScreen({
      fetchTiles: vi.fn().mockResolvedValue(makeLayout([
        makeTile(1, 'a'),
        makeTile(2, 'b'),
        makeTile(3, 'c'),
        makeTile(4, 'd'),
        makeTile(5, 'e'),
        makeTile(6, 'f'),
        makeTile(7, 'g'),
        makeTile(8, 'h'),
        makeTile(9, 'i'),
        makeTile(10, 'j'),
        makeTile(11, 'k'),
        makeTile(12, 'l'),
        makeTile(13, 'm'),
      ])),
      saveTiles,
    })

    screen.open()
    await vi.waitFor(() => expect(screen.isLoading.value).toBe(false))
    await screen.startEdit()
    screen.renameColumn(1, '二')
    await screen.saveEdit()

    const input = saveTiles.mock.calls[0]?.[0] as { columns: Array<{ name: string }> }
    expect(input.columns).toEqual([{ name: '' }, { name: '二' }])
  })

  it('keeps edit mode and reports the real reason when saving fails with 401', async () => {
    const axiosError = Object.assign(new Error('Request failed with status code 401'), {
      isAxiosError: true,
      response: { status: 401, data: { error: '需要管理员登录' } },
    })
    const saveTiles = vi.fn().mockRejectedValue(axiosError)
    const addAlert = vi.fn()
    const { screen } = createScreen({
      fetchTiles: vi.fn().mockResolvedValue(makeLayout([makeTile(1, 'a')])),
      saveTiles,
      addAlert,
    })

    screen.open()
    await vi.waitFor(() => expect(screen.isLoading.value).toBe(false))
    await screen.startEdit()
    await screen.saveEdit()

    expect(screen.isEditing.value).toBe(true)
    expect(screen.saveError.value).toBe('保存失败：需要管理员登录')
    expect(addAlert).toHaveBeenCalledWith('保存失败：需要管理员登录', 'error')
  })

  it('applies cropped images for all three tile sizes', async () => {
    const uploadTileImage = vi.fn().mockResolvedValueOnce('/assets/start-screen/small.png')
      .mockResolvedValueOnce('/assets/start-screen/wide.png')
      .mockResolvedValueOnce('/assets/start-screen/large.png')
    const addAlert = vi.fn()
    const { screen } = createScreen({
      fetchTiles: vi.fn().mockResolvedValue(makeLayout([makeTile(1, 'a')])),
      uploadTileImage,
      addAlert,
    })

    screen.open()
    await vi.waitFor(() => expect(screen.isLoading.value).toBe(false))
    await screen.applyTileCrop(1, {
      small: new Blob(['s'], { type: 'image/png' }),
      wide: new Blob(['w'], { type: 'image/png' }),
      large: new Blob(['l'], { type: 'image/png' }),
    })

    expect(uploadTileImage).toHaveBeenCalledTimes(3)
    expect(screen.tiles.value[0]?.image_small_path).toBe('/assets/start-screen/small.png')
    expect(screen.tiles.value[0]?.image_wide_path).toBe('/assets/start-screen/wide.png')
    expect(screen.tiles.value[0]?.image_large_path).toBe('/assets/start-screen/large.png')
    expect(addAlert).toHaveBeenCalledWith('磁贴图片已更新', 'success')
  })
})
