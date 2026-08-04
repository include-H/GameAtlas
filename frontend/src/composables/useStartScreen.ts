import { ref } from 'vue'
import { getHttpErrorMessage, getHttpStatus } from '@/utils/http-error'
import type { GameListItem, StartScreenTile, StartScreenTileSize, StartScreenTileWrite } from '@/services/types'

interface UseStartScreenOptions {
  fetchTiles: () => Promise<StartScreenTile[]>
  fetchFavorites: () => Promise<GameListItem[]>
  saveTiles: (tiles: StartScreenTileWrite[]) => Promise<StartScreenTile[]>
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
  const favoritePool = ref<GameListItem[]>([])
  const isLoading = ref(false)
  const hasLoadFailure = ref(false)
  const isEditing = ref(false)
  const isSaving = ref(false)
  const saveError = ref<string | null>(null)
  const originalTiles = ref<StartScreenTile[]>([])

  const refresh = async () => {
    isLoading.value = true
    hasLoadFailure.value = false
    try {
      const saved = await options.fetchTiles()
      if (saved.length > 0) {
        tiles.value = saved
        return
      }
      // 没有保存过自定义磁贴时，先用收藏游戏作为默认磁贴。
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
    favoritePool.value = []
    saveError.value = null
    isEditing.value = false
  }

  const saveEdit = async () => {
    if (isSaving.value) return
    isSaving.value = true
    saveError.value = null
    try {
      const saved = await options.saveTiles(
        tiles.value.map((tile) => ({
          game_id: tile.game_id,
          tile_size: tile.tile_size,
          image_small_path: tile.image_small_path,
          image_wide_path: tile.image_wide_path,
          image_large_path: tile.image_large_path,
        })),
      )
      tiles.value = saved
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
    resizeTile,
    removeTile,
    moveTile,
    addTile,
    applyTileCrop,
  }
}
