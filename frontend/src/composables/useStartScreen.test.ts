import { describe, expect, it, vi } from 'vitest'
import { useStartScreen } from './useStartScreen'
import type {
  StartScreenColumn,
  StartScreenLayout,
  StartScreenTile,
} from '@/services/types'

const makeTile = (
  gameId: number,
  publicId: string,
  tileSizeOrPosition: StartScreenTile['tile_size'] | Partial<Pick<StartScreenTile, 'column_index' | 'grid_row' | 'grid_col'>> = 'small',
): StartScreenTile => {
  const options = typeof tileSizeOrPosition === 'string' ? {} : tileSizeOrPosition
  const tileSize = typeof tileSizeOrPosition === 'string' ? tileSizeOrPosition : 'small'
  return {
    game_id: gameId,
    public_id: publicId,
    title: publicId,
    tile_size: tileSize,
    image_path: null,
    focus_x: 50,
    focus_y: 50,
    flip_images: null,
    sort_order: 0,
    column_index: 0,
    grid_row: 0,
    grid_col: 0,
    ...options,
  }
}

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
  saveTiles: () => Promise<StartScreenLayout>
  addAlert: () => void
}> = {}) => {
  const fetchTiles = overrides.fetchTiles ?? vi.fn().mockResolvedValue(makeLayout([]))
  const saveTiles = overrides.saveTiles ?? vi.fn().mockResolvedValue(makeLayout([]))
  const addAlert = overrides.addAlert ?? vi.fn()
  const screen = useStartScreen({
    fetchTiles,
    saveTiles,
    addAlert,
  })
  return { screen, fetchTiles, saveTiles, addAlert }
}

describe('useStartScreen', () => {
  it('keeps an empty layout when nothing is saved', async () => {
    const { screen, fetchTiles } = createScreen({
      fetchTiles: vi.fn().mockResolvedValue(makeLayout([])),
    })

    screen.open()
    await vi.waitFor(() => expect(screen.isLoading.value).toBe(false))

    expect(fetchTiles).toHaveBeenCalledTimes(1)
    expect(screen.tiles.value).toEqual([])
    expect(screen.columns.value).toEqual([])
  })

  it('uses the saved layout with column names', async () => {
    const { screen } = createScreen({
      fetchTiles: vi.fn().mockResolvedValue(makeLayout(
        [makeTile(1, 'a', 'wide')],
        [makeColumn('我的收藏')],
      )),
    })

    screen.open()
    await vi.waitFor(() => expect(screen.isLoading.value).toBe(false))

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
        image_path: null,
        focus_x: 50,
        focus_y: 50,
        flip_images: null,
        column_index: 0,
        grid_row: 0,
        grid_col: 0,
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

  it('applies a selected original image with focus point', async () => {
    const addAlert = vi.fn()
    const { screen } = createScreen({
      fetchTiles: vi.fn().mockResolvedValue(makeLayout([makeTile(1, 'a')])),
      addAlert,
    })

    screen.open()
    await vi.waitFor(() => expect(screen.isLoading.value).toBe(false))
    screen.applyTileImage(1, '/assets/a/covers/cover.jpg', 42, 73)

    expect(screen.tiles.value[0]?.image_path).toBe('/assets/a/covers/cover.jpg')
    expect(screen.tiles.value[0]?.focus_x).toBe(42)
    expect(screen.tiles.value[0]?.focus_y).toBe(73)
    expect(addAlert).toHaveBeenCalledWith('磁贴图片已更新', 'success')
  })

  it('applies flip frames for live wide tiles, capped and deduped', async () => {
    const addAlert = vi.fn()
    const { screen } = createScreen({
      fetchTiles: vi.fn().mockResolvedValue(makeLayout([makeTile(1, 'a', 'wide')])),
      addAlert,
    })

    screen.open()
    await vi.waitFor(() => expect(screen.isLoading.value).toBe(false))
    screen.applyTileImage(
      1,
      '/assets/a/first.jpg',
      50,
      50,
      [
        '/assets/a/flip-1.jpg',
        '/assets/a/first.jpg',
        '/assets/a/flip-2.jpg',
        '/assets/a/flip-3.jpg',
        '/assets/a/flip-4.jpg',
      ],
    )

    // 首帧去重 + 上限 3 张追加帧。
    expect(screen.tiles.value[0]?.flip_images).toEqual([
      '/assets/a/flip-1.jpg',
      '/assets/a/flip-2.jpg',
      '/assets/a/flip-3.jpg',
    ])
  })

  it('moves a whole column and shifts tile column indexes', async () => {
    const { screen } = createScreen({
      fetchTiles: vi.fn().mockResolvedValue(makeLayout(
        [
          makeTile(1, 'a', { column_index: 0 }),
          makeTile(2, 'b', { column_index: 1 }),
          makeTile(3, 'c', { column_index: 2 }),
        ],
        [makeColumn('一'), makeColumn('二'), makeColumn('三')],
      )),
    })

    screen.open()
    await vi.waitFor(() => expect(screen.isLoading.value).toBe(false))
    screen.moveColumn(0, 2)

    expect(screen.columns.value.map((column) => column.name)).toEqual(['二', '三', '一'])
    const tileOf = (gameId: number) => screen.tiles.value.find((tile) => tile.game_id === gameId)
    expect(tileOf(1)?.column_index).toBe(2)
    expect(tileOf(2)?.column_index).toBe(0)
    expect(tileOf(3)?.column_index).toBe(1)
  })

  it('ignores no-op and out-of-range column moves', async () => {
    const { screen } = createScreen({
      fetchTiles: vi.fn().mockResolvedValue(makeLayout(
        [makeTile(1, 'a', { column_index: 0 }), makeTile(2, 'b', { column_index: 1 })],
        [makeColumn('一'), makeColumn('二')],
      )),
    })

    screen.open()
    await vi.waitFor(() => expect(screen.isLoading.value).toBe(false))
    screen.moveColumn(1, 1)
    screen.moveColumn(0, 5)

    expect(screen.columns.value.map((column) => column.name)).toEqual(['一', '二'])
    expect(screen.tiles.value.find((tile) => tile.game_id === 1)?.column_index).toBe(0)
  })

  it('adds and removes empty columns without moving occupied tiles', async () => {
    const { screen } = createScreen({
      fetchTiles: vi.fn().mockResolvedValue(makeLayout(
        [makeTile(1, 'a', { column_index: 1 })],
        [makeColumn('一'), makeColumn('二')],
      )),
    })

    screen.open()
    await vi.waitFor(() => expect(screen.isLoading.value).toBe(false))
    await screen.startEdit()
    screen.addColumn()
    expect(screen.columns.value).toHaveLength(3)

    screen.removeColumn(2)
    expect(screen.tiles.value[0]?.column_index).toBe(1)
    expect(screen.columns.value.map((column) => column.name)).toEqual(['一', '二'])
  })

  it('places a tile into an explicit empty column and saves it', async () => {
    const saveTiles = vi.fn().mockResolvedValue(makeLayout([], []))
    const { screen } = createScreen({
      fetchTiles: vi.fn().mockResolvedValue(makeLayout([makeTile(1, 'a')])),
      saveTiles,
    })

    screen.open()
    await vi.waitFor(() => expect(screen.isLoading.value).toBe(false))
    await screen.startEdit()
    screen.applyTilePlacement(1, 2, 3, 0)
    expect(screen.tiles.value[0]).toMatchObject({ column_index: 2, grid_row: 3, grid_col: 0 })
    await screen.saveEdit()

    expect(saveTiles).toHaveBeenCalledWith({
      columns: [{ name: '' }, { name: '' }, { name: '' }],
      tiles: [{
        game_id: 1,
        tile_size: 'small',
        image_path: null,
        focus_x: 50,
        focus_y: 50,
        flip_images: null,
        column_index: 2,
        grid_row: 3,
        grid_col: 0,
      }],
    })
  })
})
