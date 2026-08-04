import { ref } from 'vue'
import type { GameListItem } from '@/services/types'

interface UseStartScreenOptions {
  fetchFavorites: () => Promise<GameListItem[]>
  removeFavorite: (publicId: string) => Promise<void>
  addAlert: (message: string, type: 'success' | 'warning' | 'error') => void
}

export const useStartScreen = (options: UseStartScreenOptions) => {
  const visible = ref(false)
  const games = ref<GameListItem[]>([])
  const isLoading = ref(false)
  const hasLoadFailure = ref(false)
  const loadedOnce = ref(false)

  const refresh = async () => {
    isLoading.value = true
    hasLoadFailure.value = false
    try {
      games.value = await options.fetchFavorites()
      loadedOnce.value = true
    } catch {
      hasLoadFailure.value = true
      options.addAlert('开始屏幕加载失败，请稍后重试', 'error')
    } finally {
      isLoading.value = false
    }
  }

  const open = () => {
    visible.value = true
    if (!loadedOnce.value && !isLoading.value && !hasLoadFailure.value) {
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

  const unpin = async (publicId: string) => {
    const previous = games.value
    games.value = games.value.filter((game) => game.public_id !== publicId)
    try {
      await options.removeFavorite(publicId)
    } catch {
      games.value = previous
      options.addAlert('取消收藏失败，请稍后重试', 'error')
    }
  }

  return {
    visible,
    games,
    isLoading,
    hasLoadFailure,
    open,
    close,
    toggle,
    retry,
    unpin,
  }
}
