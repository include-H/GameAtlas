import { computed, nextTick, onActivated, onMounted, onUnmounted, ref, watch, type Ref } from 'vue'

import type { RouteLocationNormalizedLoaded, Router } from 'vue-router'
import downloadService from '@/services/download.service'
import { isAdminGameDetail, type AdminGameDetail, type GameVersion } from '@/services/types'
import { formatDisplayDate } from '@/utils/date'
import { GAME_DETAIL_RETURN_QUERY, navigateToExplicitReturnOrFallback } from '@/utils/navigation'
import { useGamesStore } from '@/stores/games'
import { useUiStore } from '@/stores/ui'
import { getAmbientBackgroundPoolFromGameDetail, hasAmbientBackgroundPoolImages } from '@/utils/ambient-background'
import { createRequestGeneration } from '@/utils/request-generation'

interface UseGameDetailViewOptions {
  route: RouteLocationNormalizedLoaded
  router: Router
  gamesStore: ReturnType<typeof useGamesStore>
  uiStore: ReturnType<typeof useUiStore>
  isAdmin: Ref<boolean>
}

// 2026-05-01: these helpers are intentionally consumed only by GameDetailView.vue template
// bindings after being returned from this composable. They have no local call sites inside
// useGameDetailView.ts, so "unused in this file" does not mean dead code; do not delete them
// unless the corresponding template bindings are removed in GameDetailView.vue as well.
const formatGameDetailDate = (dateStr: string) => {
  return formatDisplayDate(dateStr)
}

const AMBIENT_BACKGROUND_OWNER = 'game-detail'

const formatGameDetailSize = (bytes: number) => {
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let size = bytes
  let unitIndex = 0

  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024
    unitIndex++
  }

  return `${size.toFixed(1)} ${units[unitIndex]}`
}

interface StartVersionDownloadOptions {
  gameId: string
  versionId: string
  versionLabel: string
  downloadUrl: string
  recordDownload: (gameId: string, fileId: string) => Promise<{ recorded: boolean }>
  navigateToUrl: (url?: string) => void
  addAlert: (message: string, type: 'success' | 'warning' | 'error') => void
}

export const startVersionDownload = async ({
  gameId,
  versionId,
  versionLabel,
  downloadUrl,
  recordDownload,
  navigateToUrl,
  addAlert,
}: StartVersionDownloadOptions) => {
  let recordFailed = false

  try {
    await recordDownload(gameId, versionId)
  } catch {
    recordFailed = true
  }

  try {
    // 2026-05-01: keep the file download as the primary action on this chain.
    // Impact: download-stat recording is ancillary. A record failure must surface as a warning,
    // not be upgraded into a fake "download failed" result that blocks the real file transfer.
    navigateToUrl(downloadUrl)
    addAlert(
      recordFailed ? `已开始下载 ${versionLabel}，但下载记录失败` : `已开始下载 ${versionLabel}`,
      recordFailed ? 'warning' : 'success',
    )
  } catch {
    addAlert('下载启动失败', 'error')
  }
}

export const useGameDetailView = ({
  route,
  router,
  gamesStore,
  uiStore,
  isAdmin,
}: UseGameDetailViewOptions) => {
  let currentAbortController: AbortController | null = null
  const detailViewRequests = createRequestGeneration()
  const requestedGameId = computed(() => {
    const rawValue = route.params.publicId
    return typeof rawValue === 'string' ? rawValue.trim() : Array.isArray(rawValue) ? String(rawValue[0] || '').trim() : ''
  })
  const hasLoadFailure = ref(false)
  const game = computed(() => {
    if (!requestedGameId.value) {
      return null
    }
    return gamesStore.currentGame?.public_id === requestedGameId.value
      ? gamesStore.currentGame
      : null
  })
  const versions = computed(() => gamesStore.currentVersions)
  const showEditModal = ref(false)
  const topSectionRef = ref<HTMLElement | null>(null)
  const topSectionHeight = ref<number | undefined>(undefined)
  const isDesktopTopLayout = ref(false)
  let topSectionObserver: ResizeObserver | null = null

  const developerNames = computed(() => (game.value?.developers || []).map((item) => item.name).join(' / '))
  const publisherNames = computed(() => (game.value?.publishers || []).map((item) => item.name).join(' / '))
  // 2026-04-06: the detail page reads wiki content from the native /games detail
  // payload only. Do not compose a second wiki source here and invent split-brain
  // semantics between game.wiki_content and /games/:publicId/wiki.
  const hasWikiContent = computed(() => Boolean(game.value?.wiki_content?.trim()))
  const canEdit = computed(() => isAdmin.value)
  // The detail page can render either public or admin payloads, but the edit modal
  // must never receive the public shape because missing file_path means broken edit state,
  // not a valid degraded experience.
  const editableGame = computed<AdminGameDetail | null>(() => {
    if (!canEdit.value || !isAdminGameDetail(game.value)) {
      return null
    }
    return game.value
  })

  const navigateToUrl = (url?: string) => {
    if (!url || typeof window === 'undefined') {
      throw new Error('缺少下载地址')
    }
    window.location.assign(url)
  }

  const handleDownloadVersion = async (version: GameVersion) => {
    if (!game.value?.public_id || !version.downloadUrl) return
    await startVersionDownload({
      gameId: game.value.public_id,
      versionId: version.id,
      versionLabel: version.version,
      downloadUrl: version.downloadUrl,
      recordDownload: downloadService.recordDownload,
      navigateToUrl,
      addAlert: uiStore.addAlert,
    })
  }

  const handleDownloadLaunchScript = (version: GameVersion) => {
    if (!version.launchScriptUrl) return

    try {
      navigateToUrl(version.launchScriptUrl)
      uiStore.addAlert(`已为 ${version.version} 生成启动脚本`, 'success')
    } catch {
      uiStore.addAlert('开始游玩失败', 'error')
    }
  }

  const refreshDetailAfterEdit = async () => {
    const gameId = requestedGameId.value
    if (!gameId) {
      return
    }
    const request = detailViewRequests.begin()

    try {
      // 2026-04-07: edit saves can succeed even if the follow-up detail refresh fails.
      // Impact: surface that sync failure explicitly instead of silently leaving stale detail data on screen.
      await gamesStore.fetchGame(gameId)
      if (!request.isCurrent() || requestedGameId.value !== gameId) return
      hasLoadFailure.value = false
    } catch {
      if (!request.isCurrent() || requestedGameId.value !== gameId) return
      uiStore.addAlert('保存已生效，但详情刷新失败，请稍后重试', 'warning')
    }
  }

  const handleEditSuccess = refreshDetailAfterEdit

  const handleEditSync = refreshDetailAfterEdit

  const handleGoBack = () => {
    navigateToExplicitReturnOrFallback(router, route.query[GAME_DETAIL_RETURN_QUERY], { name: 'games' })
  }

  const openWikiEditor = () => {
    if (!game.value?.public_id) return
    router.push({
      name: 'wiki-edit',
      params: { publicId: game.value.public_id },
    })
  }

  const openEditModal = () => {
    if (!canEdit.value) return
    if (!editableGame.value) {
      uiStore.addAlert('编辑数据缺少管理员文件路径，无法打开编辑器', 'error')
      return
    }
    showEditModal.value = true
  }

  const disconnectTopSectionObserver = () => {
    if (topSectionObserver) {
      topSectionObserver.disconnect()
      topSectionObserver = null
    }
  }

  const syncAmbientBackground = () => {
    const pool = getAmbientBackgroundPoolFromGameDetail(game.value)
    if (!game.value?.public_id || !hasAmbientBackgroundPoolImages(pool)) {
      uiStore.clearAmbientBackgroundSource(AMBIENT_BACKGROUND_OWNER)
      return
    }

    uiStore.setAmbientBackgroundSource({
      owner: AMBIENT_BACKGROUND_OWNER,
      key: game.value.public_id,
      pool,
    })
  }

  const loadGameDetail = async (gameId: string) => {
    const request = detailViewRequests.begin()
    // Abort previous request if still in flight
    if (currentAbortController) {
      currentAbortController.abort()
    }
    currentAbortController = new AbortController()
    const { signal } = currentAbortController

    hasLoadFailure.value = false
    try {
      await gamesStore.fetchGame(gameId, signal)
      if (!request.isCurrent() || requestedGameId.value !== gameId) return
    } catch {
      // Ignore aborted requests
      if (signal.aborted || !request.isCurrent() || requestedGameId.value !== gameId) return
      // 2026-04-07: detail routing must never keep rendering the previous game when
      // the current publicId fails to load. A failed read is not the same as "show stale detail".
      hasLoadFailure.value = true
      uiStore.clearAmbientBackgroundSource(AMBIENT_BACKGROUND_OWNER)
      uiStore.addAlert('加载游戏详情失败', 'error')
    }
  }

  watch(
    requestedGameId,
    async (gameId) => {
      if (!gameId) {
        detailViewRequests.invalidate()
        uiStore.clearAmbientBackgroundSource(AMBIENT_BACKGROUND_OWNER)
        return
      }
      showEditModal.value = false
      uiStore.clearAmbientBackgroundSource(AMBIENT_BACKGROUND_OWNER)
      await loadGameDetail(gameId)
      if (requestedGameId.value !== gameId) return
      syncAmbientBackground()
      void setupTopSectionObserver()
    },
    { immediate: true },
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

  onActivated(() => {
    syncAmbientBackground()
    void setupTopSectionObserver()
  })

  // Sync on non-ID-triggered game changes (e.g., edit modal aggregate update).
  watch(
    game,
    () => {
      syncAmbientBackground()
      void setupTopSectionObserver()
    },
    { flush: 'post' },
  )

  onUnmounted(() => {
    detailViewRequests.invalidate()
    if (currentAbortController) {
      currentAbortController.abort()
      currentAbortController = null
    }
    uiStore.clearAmbientBackgroundSource(AMBIENT_BACKGROUND_OWNER)
    disconnectTopSectionObserver()
    if (typeof window !== 'undefined') {
      window.removeEventListener('resize', syncTopSectionHeight)
    }
  })

  return {
    canEdit,
    developerNames,
    editableGame,
    formatDate: formatGameDetailDate,
    formatSize: formatGameDetailSize,
    game,
    hasLoadFailure,
    handleDownloadLaunchScript,
    handleDownloadVersion,
    handleEditSuccess,
    handleEditSync,
    handleGoBack,
    hasWikiContent,
    openEditModal,
    openWikiEditor,
    publisherNames,
    showEditModal,
    topSectionRef,
    versions,
  }
}
