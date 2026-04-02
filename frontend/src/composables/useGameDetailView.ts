import { computed, nextTick, onMounted, onUnmounted, ref, watch, type Ref } from 'vue'
import type { RouteLocationNormalizedLoaded, Router } from 'vue-router'
import wikiService, { type WikiDocumentResponse } from '@/services/wiki.service'
import downloadService from '@/services/download.service'
import type { GameVersion } from '@/services/types'
import { getHttpStatus } from '@/utils/http-error'
import { formatDisplayDate } from '@/utils/date'
import { useNamedRouteGuard, watchRouteParamWhenActive } from '@/composables/useNamedRouteGuard'
import { createDetailRouteQuery, resolveReturnRoute } from '@/utils/navigation'
import { useGamesStore } from '@/stores/games'
import { useUiStore } from '@/stores/ui'

interface UseGameDetailViewOptions {
  route: RouteLocationNormalizedLoaded
  router: Router
  gamesStore: ReturnType<typeof useGamesStore>
  uiStore: ReturnType<typeof useUiStore>
  isAdmin: Ref<boolean>
}

export const formatGameDetailDate = (dateStr: string) => {
  return formatDisplayDate(dateStr)
}

export const formatGameDetailSize = (bytes: number) => {
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let size = bytes
  let unitIndex = 0

  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024
    unitIndex++
  }

  return `${size.toFixed(1)} ${units[unitIndex]}`
}

export const shouldSpanGameMetadataRow = (items?: { length: number } | null) => {
  return (items?.length || 0) > 2
}

export const useGameDetailView = ({
  route,
  router,
  gamesStore,
  uiStore,
  isAdmin,
}: UseGameDetailViewOptions) => {
  const { runWhenActive } = useNamedRouteGuard(route, 'game-detail')

  const game = computed(() => gamesStore.currentGame)
  const versions = computed(() => gamesStore.currentVersions)
  const wiki = ref<WikiDocumentResponse | null>(null)
  const showEditModal = ref(false)
  const topSectionRef = ref<HTMLElement | null>(null)
  const topSectionHeight = ref<number | undefined>(undefined)
  const isDesktopTopLayout = ref(false)
  let topSectionObserver: ResizeObserver | null = null

  const developerNames = computed(() => (game.value?.developers || []).map((item) => item.name).join(' / '))
  const publisherNames = computed(() => (game.value?.publishers || []).map((item) => item.name).join(' / '))
  const hasWikiContent = computed(() => Boolean(wiki.value?.content?.trim()))
  const canEdit = computed(() => isAdmin.value)

  const handleDownloadVersion = async (version: GameVersion) => {
    if (!game.value?.public_id) return

    try {
      await downloadService.startDownload(game.value.public_id, version.id)
      uiStore.addAlert(`已开始下载 ${version.version}`, 'success')
    } catch {
      uiStore.addAlert('下载启动失败', 'error')
    }
  }

  const handleDownloadLaunchScript = (version: GameVersion) => {
    if (!game.value?.public_id) return

    try {
      downloadService.downloadLaunchScript(game.value.public_id, version.id)
      uiStore.addAlert(`已为 ${version.version} 生成启动脚本`, 'success')
    } catch {
      uiStore.addAlert('开始游玩失败', 'error')
    }
  }

  const handleEditSuccess = async () => {
    if (game.value?.public_id) {
      await gamesStore.fetchGame(game.value.public_id)
    }
  }

  const handleGoBack = () => {
    router.push(resolveReturnRoute(route, { name: 'games' }))
  }

  const openWikiEditor = () => {
    if (!game.value?.public_id) return
    router.push({
      name: 'wiki-edit',
      params: { publicId: game.value.public_id },
      query: createDetailRouteQuery(route),
    })
  }

  const handleToggleFavorite = async () => {
    if (!game.value?.public_id) return
    try {
      await gamesStore.toggleFavorite(game.value.public_id)
      uiStore.addAlert('收藏已更新', 'success')
    } catch {
      uiStore.addAlert('更新收藏失败', 'error')
    }
  }

  const carouselHeight = computed(() => {
    if (!isDesktopTopLayout.value) return undefined
    if (!topSectionHeight.value) return undefined
    return Math.max(Math.round(topSectionHeight.value), 420)
  })

  const disconnectTopSectionObserver = () => {
    if (topSectionObserver) {
      topSectionObserver.disconnect()
      topSectionObserver = null
    }
  }

  const loadGameDetail = async (gameId: string) => {
    await runWhenActive(async () => {
      try {
        await gamesStore.fetchGame(gameId)

        wiki.value = null
        try {
          wiki.value = await wikiService.getWikiPage(gameId)
        } catch {
          // Wiki doesn't exist.
        }
      } catch (error) {
        const status = getHttpStatus(error)
        if (status === 404) {
          router.replace({ name: 'not-found' })
          return
        }
        uiStore.addAlert('加载游戏详情失败', 'error')
      }
    })
  }

  watchRouteParamWhenActive(
    route,
    'game-detail',
    'publicId',
    async (gameId) => {
      showEditModal.value = false
      await loadGameDetail(gameId)
    },
  )

  const syncTopSectionHeight = () => {
    const element = topSectionRef.value
    if (!element) {
      topSectionHeight.value = undefined
      return
    }

    if (typeof window !== 'undefined') {
      isDesktopTopLayout.value = window.innerWidth > 992
    }
    if (!isDesktopTopLayout.value) {
      topSectionHeight.value = undefined
      return
    }

    const nextHeight = Math.round(element.getBoundingClientRect().height)
    topSectionHeight.value = nextHeight > 0 ? nextHeight : undefined
  }

  const setupTopSectionObserver = async () => {
    await nextTick()
    disconnectTopSectionObserver()
    syncTopSectionHeight()

    if (!topSectionRef.value || typeof ResizeObserver === 'undefined') return

    topSectionObserver = new ResizeObserver(() => {
      syncTopSectionHeight()
    })
    topSectionObserver.observe(topSectionRef.value)
  }

  onMounted(() => {
    if (typeof window !== 'undefined') {
      isDesktopTopLayout.value = window.innerWidth > 992
      window.addEventListener('resize', syncTopSectionHeight, { passive: true })
    }
    void setupTopSectionObserver()
  })

  watch(
    game,
    () => {
      void setupTopSectionObserver()
    },
    { flush: 'post' },
  )

  onUnmounted(() => {
    disconnectTopSectionObserver()
    if (typeof window !== 'undefined') {
      window.removeEventListener('resize', syncTopSectionHeight)
    }
  })

  return {
    canEdit,
    carouselHeight,
    developerNames,
    formatDate: formatGameDetailDate,
    formatSize: formatGameDetailSize,
    game,
    handleDownloadLaunchScript,
    handleDownloadVersion,
    handleEditSuccess,
    handleGoBack,
    handleToggleFavorite,
    hasWikiContent,
    openWikiEditor,
    publisherNames,
    shouldSpanMetadataRow: shouldSpanGameMetadataRow,
    showEditModal,
    topSectionRef,
    versions,
    wiki,
  }
}
