import { ref } from 'vue'
import { getHttpErrorMessage, getHttpStatus } from '@/utils/http-error'
import { packStartScreenTiles } from '@/utils/start-screen-layout'
import type {
  GameListItem,
  StartScreenColumn,
  StartScreenLayout,
  StartScreenLayoutInput,
  StartScreenTile,
  StartScreenTileSize,
} from '@/services/types'

interface UseStartScreenOptions {
  fetchTiles: () => Promise<StartScreenLayout>
  fetchFavorites: () => Promise<GameListItem[]>
  saveTiles: (input: StartScreenLayoutInput) => Promise<StartScreenLayout>
  uploadTileImage: (file: File, size: StartScreenTileSize) => Promise<string>
  addAlert: (message: string, type: 'success' | 'warning' | 'error') => void
}

const NEXT_TILE_SIZE: Record<StartScreenTileSize, StartScreenTileSize> = {
  small: 'wide',
  wide: 'large',
  large: 'small',
}

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
  const favoritePool = ref<GameListItem[]>([])
  const isLoading = ref(false)
  const hasLoadFailure = ref(false)
  const isEditing = ref(false)
  const isSaving = ref(false)
  const saveError = ref<string | null>(null)
  const originalTiles = ref<StartScreenTile[]>([])
  const originalColumns = ref<StartScreenColumn[]>([])

  const applyLayout = (layout: StartScreenLayout) => {
    tiles.value = layout.tiles
    columns.value = layout.columns
  }

  const refresh = async () => {
    isLoading.value = true
    hasLoadFailure.value = false
    try {
      const saved = await options.fetchTiles()
      if (saved.tiles.length > 0) {
        applyLayout(saved)
        return
      }
      // 没有保存过自定义磁贴时，先用收藏游戏作为默认磁贴；列名留空，展示时按"列 N"兜底。
      const favorites = await options.fetchFavorites()
      tiles.value = favorites.map((game, index) => ({
        game_id: game.id,
        public_id: game.public_id,
        title: game.title,
        cover_image: game.cover_image,
        banner_image: game.banner_image,
        tile_size: 'small',
        image_small_path: null,
        image_wide_path: null,
        image_large_path: null,
        sort_order: index,
      }))
      columns.value = []
    } catch {
      hasLoadFailure.value = true
      options.addAlert('开始屏幕加载失败，请稍后重试', 'error')
    } finally {
      isLoading.value = false
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
    try {
      favoritePool.value = await options.fetchFavorites()
    } catch {
      favoritePool.value = []
      options.addAlert('收藏列表加载失败，暂时无法添加磁贴', 'warning')
    }
  }

  const cancelEdit = () => {
    tiles.value = originalTiles.value.map((tile) => ({ ...tile }))
    columns.value = originalColumns.value.map((column) => ({ ...column }))
    favoritePool.value = []
    saveError.value = null
    isEditing.value = false
  }

  const renameColumn = (index: number, name: string) => {
    while (columns.value.length <= index) {
      columns.value.push({ id: 0, name: '', sort_order: columns.value.length })
    }
    columns.value[index].name = name.trim()
  }

  const saveEdit = async () => {
    if (isSaving.value) return
    isSaving.value = true
    saveError.value = null
    try {
      const packed = packStartScreenTiles(tiles.value)
      const columnNames = packed.map((_, index) => columns.value[index]?.name ?? '')
      const saved = await options.saveTiles({
        columns: columnNames.map((name) => ({ name })),
        tiles: tiles.value.map((tile) => ({
          game_id: tile.game_id,
          tile_size: tile.tile_size,
          image_small_path: tile.image_small_path,
          image_wide_path: tile.image_wide_path,
          image_large_path: tile.image_large_path,
        })),
      })
      applyLayout(saved)
      favoritePool.value = []
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
  }

  const removeTile = (gameId: number) => {
    tiles.value = tiles.value.filter((tile) => tile.game_id !== gameId)
  }

  const moveTile = (fromIndex: number, toIndex: number) => {
    if (fromIndex === toIndex) return
    const next = [...tiles.value]
    const [moved] = next.splice(fromIndex, 1)
    if (!moved) return
    next.splice(toIndex, 0, moved)
    tiles.value = next
  }

  const applyTileOrder = (ordered: StartScreenTile[]) => {
    tiles.value = ordered.map((tile) => ({ ...tile }))
  }

  const addTile = (game: GameListItem) => {
    if (tiles.value.some((tile) => tile.game_id === game.id)) return
    tiles.value.push({
      game_id: game.id,
      public_id: game.public_id,
      title: game.title,
      cover_image: game.cover_image,
      banner_image: game.banner_image,
      tile_size: 'small',
      image_small_path: null,
      image_wide_path: null,
      image_large_path: null,
      sort_order: tiles.value.length,
    })
  }

  const applyTileCrop = async (gameId: number, blobs: Record<StartScreenTileSize, Blob>) => {
    const tile = tiles.value.find((item) => item.game_id === gameId)
    if (!tile) return
    try {
      const [smallPath, widePath, largePath] = await Promise.all([
        options.uploadTileImage(new File([blobs.small], 'tile-small.png', { type: 'image/png' }), 'small'),
        options.uploadTileImage(new File([blobs.wide], 'tile-wide.png', { type: 'image/png' }), 'wide'),
        options.uploadTileImage(new File([blobs.large], 'tile-large.png', { type: 'image/png' }), 'large'),
      ])
      tile.image_small_path = smallPath
      tile.image_wide_path = widePath
      tile.image_large_path = largePath
      options.addAlert('磁贴图片已更新', 'success')
    } catch (error) {
      options.addAlert(`磁贴图片更新失败：${getHttpErrorMessage(error, '未知错误')}`, 'error')
    }
  }

  return {
    visible,
    tiles,
    columns,
    favoritePool,
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
    resizeTile,
    removeTile,
    moveTile,
    applyTileOrder,
    addTile,
    applyTileCrop,
  }
}
