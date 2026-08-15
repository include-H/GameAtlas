import { ref } from 'vue'
import { getHttpErrorMessage, getHttpStatus } from '@/utils/http-error'
import { normalizeStartScreenTiles } from '@/utils/start-screen-layout'
import type {
  GameListItem,
  StartScreenColumn,
  StartScreenLayout,
  StartScreenLayoutInput,
  StartScreenTile,
  StartScreenTileSize,
} from '@/services/types'
import { createRequestGeneration } from '@/utils/request-generation'

interface UseStartScreenOptions {
  fetchTiles: () => Promise<StartScreenLayout>
  fetchFavorites: () => Promise<GameListItem[]>
  saveTiles: (input: StartScreenLayoutInput) => Promise<StartScreenLayout>
  addAlert: (message: string, type: 'success' | 'warning' | 'error') => void
}

const NEXT_TILE_SIZE: Record<StartScreenTileSize, StartScreenTileSize> = {
  small: 'wide',
  wide: 'large',
  large: 'small',
}

// 与后端 domain.StartScreenMaxFlipImages 保持一致：轮播总帧数 = 首帧 + 追加帧 ≤ 4。
export const START_SCREEN_MAX_FLIP_IMAGES = 3

const describeSaveError = (error: unknown): string => {
  const status = getHttpStatus(error)
  if (status === 401) return '保存失败：需要管理员登录'
  if (status === 404) return '保存失败：后端接口不存在，请确认后端已更新并重启'
  return `保存失败：${getHttpErrorMessage(error, '未知错误')}`
}

export const useStartScreen = (options: UseStartScreenOptions) => {
  const visible = ref(false)
  const tiles = ref<StartScreenTile[]>([])
  const columns = ref<StartScreenColumn[]>([])
  const isLoading = ref(false)
  const hasLoadFailure = ref(false)
  const isEditing = ref(false)
  const isSaving = ref(false)
  const saveError = ref<string | null>(null)
  const originalTiles = ref<StartScreenTile[]>([])
  const originalColumns = ref<StartScreenColumn[]>([])
  const refreshRequests = createRequestGeneration()

  const applyLayout = (layout: StartScreenLayout) => {
    tiles.value = normalizeStartScreenTiles(layout.tiles)
    columns.value = layout.columns
  }

  const refresh = async () => {
    const request = refreshRequests.begin()
    isLoading.value = true
    hasLoadFailure.value = false
    try {
      const saved = await options.fetchTiles()
      if (!request.isCurrent()) return
      if (saved.tiles.length > 0) {
        applyLayout(saved)
        return
      }
      // 没有保存过自定义磁贴时，先用收藏游戏作为默认磁贴；列名留空，展示时按"列 N"兜底。
      const favorites = await options.fetchFavorites()
      if (!request.isCurrent()) return
      tiles.value = normalizeStartScreenTiles(
        favorites.map((game, index) => ({
          game_id: game.id,
          public_id: game.public_id,
          title: game.title,
          tile_size: 'small' as const,
          image_path: game.primary_screenshot || game.cover_image,
          focus_x: 50,
          focus_y: 50,
          flip_images: null,
          sort_order: index,
          column_index: 0,
          grid_row: 0,
          grid_col: 0,
        })),
      )
      columns.value = []
    } catch {
      if (!request.isCurrent()) return
      hasLoadFailure.value = true
      options.addAlert('开始屏幕加载失败，请稍后重试', 'error')
    } finally {
      if (request.isCurrent()) {
        isLoading.value = false
      }
    }
  }

  const open = () => {
    visible.value = true
    if (!isLoading.value) {
      void refresh()
    }
  }

  const close = () => {
    visible.value = false
  }

  const toggle = () => {
    if (visible.value) {
      close()
    } else {
      open()
    }
  }

  const retry = () => {
    if (isLoading.value) return
    void refresh()
  }

  const startEdit = async () => {
    if (isEditing.value) return
    saveError.value = null
    originalTiles.value = tiles.value.map((tile) => ({ ...tile }))
    originalColumns.value = columns.value.map((column) => ({ ...column }))
    isEditing.value = true
  }

  const cancelEdit = () => {
    tiles.value = originalTiles.value.map((tile) => ({ ...tile }))
    columns.value = originalColumns.value.map((column) => ({ ...column }))
    saveError.value = null
    isEditing.value = false
  }

  const renameColumn = (index: number, name: string) => {
    while (columns.value.length <= index) {
      columns.value.push({ id: 0, name: '', sort_order: columns.value.length })
    }
    columns.value[index].name = name.trim()
  }

  const addColumn = () => {
    columns.value.push({ id: 0, name: '', sort_order: columns.value.length })
  }

  const removeColumn = (index: number) => {
    if (index < 0 || index >= columns.value.length) return
    if (tiles.value.some((tile) => tile.column_index === index)) return
    columns.value.splice(index, 1)
    tiles.value = tiles.value.map((tile) =>
      tile.column_index > index ? { ...tile, column_index: tile.column_index - 1 } : tile,
    )
  }

  const applyTilePlacement = (gameId: number, columnIndex: number, row: number, col: number) => {
    const tile = tiles.value.find((item) => item.game_id === gameId)
    if (!tile) return
    while (columns.value.length <= columnIndex) {
      columns.value.push({ id: 0, name: '', sort_order: columns.value.length })
    }
    tile.column_index = columnIndex
    tile.grid_row = row
    tile.grid_col = col
    tiles.value = normalizeStartScreenTiles(tiles.value)
  }

  const saveEdit = async () => {
    if (isSaving.value) return
    isSaving.value = true
    saveError.value = null
    try {
      const columnCount = Math.max(
        1,
        columns.value.length,
        ...tiles.value.map((tile) => tile.column_index + 1),
      )
      const columnNames = Array.from({ length: columnCount }, (_, index) => columns.value[index]?.name ?? '')
      const saved = await options.saveTiles({
        columns: columnNames.map((name) => ({ name })),
        tiles: tiles.value.map((tile) => ({
          game_id: tile.game_id,
          tile_size: tile.tile_size,
          image_path: tile.image_path,
          focus_x: tile.focus_x,
          focus_y: tile.focus_y,
          flip_images: tile.flip_images,
          column_index: tile.column_index,
          grid_row: tile.grid_row,
          grid_col: tile.grid_col,
        })),
      })
      applyLayout(saved)
      isEditing.value = false
      options.addAlert('开始屏幕已保存', 'success')
    } catch (error) {
      const message = describeSaveError(error)
      saveError.value = message
      options.addAlert(message, 'error')
    } finally {
      isSaving.value = false
    }
  }

  const resizeTile = (gameId: number) => {
    const tile = tiles.value.find((item) => item.game_id === gameId)
    if (!tile) return
    tile.tile_size = NEXT_TILE_SIZE[tile.tile_size]
    tiles.value = normalizeStartScreenTiles(tiles.value)
  }

  const removeTile = (gameId: number) => {
    tiles.value = tiles.value.filter((tile) => tile.game_id !== gameId)
  }

  // 选择器确认：主图 + 焦点 + 宽磁贴轮播追加帧（首帧即主图，追加帧 ≤ 3）。
  const applyTileImage = (
    gameId: number,
    imagePath: string,
    focusX: number,
    focusY: number,
    flipImages: string[] = [],
  ) => {
    const tile = tiles.value.find((item) => item.game_id === gameId)
    if (!tile) return
    tile.image_path = imagePath
    tile.focus_x = Math.max(0, Math.min(100, Math.round(focusX)))
    tile.focus_y = Math.max(0, Math.min(100, Math.round(focusY)))
    tile.flip_images = Array.from(
      new Set(flipImages.filter((path) => path !== imagePath)),
    ).slice(0, START_SCREEN_MAX_FLIP_IMAGES)
    options.addAlert('磁贴图片已更新', 'success')
  }

  return {
    visible,
    tiles,
    columns,
    isLoading,
    hasLoadFailure,
    isEditing,
    isSaving,
    saveError,
    open,
    close,
    toggle,
    retry,
    startEdit,
    cancelEdit,
    saveEdit,
    renameColumn,
    addColumn,
    removeColumn,
    applyTilePlacement,
    resizeTile,
    removeTile,
    applyTileImage,
  }
}
