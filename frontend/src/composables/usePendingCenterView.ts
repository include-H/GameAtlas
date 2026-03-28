import { computed, onActivated, onBeforeUnmount, onDeactivated, onMounted, ref, watch } from 'vue'
import type { RouteLocationNormalizedLoaded, Router } from 'vue-router'
import gamesService from '@/services/games.service'
import type { GameDetail, GameListItem } from '@/services/types'
import {
  getPendingIssueDetailLabel,
  getPendingIssueLabel,
  pendingIssueDetailDefinitions,
  type PendingIssueDetailKey,
  type PendingIssueKey,
} from '@/utils/pendingIssues'
import { formatDisplayDate } from '@/utils/date'
import { createDetailRouteQuery } from '@/utils/navigation'
import { usePendingWorkbench } from '@/composables/usePendingWorkbench'
import { useUiStore } from '@/stores/ui'

const PLACEHOLDER_IMAGE = 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"%3E%3Cpath fill="%23424242" d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z"/%3E%3C/svg%3E'

type PendingIssueDetailDefinition = (typeof pendingIssueDetailDefinitions)[number]

interface UsePendingCenterViewOptions {
  route: RouteLocationNormalizedLoaded
  router: Router
  uiStore: ReturnType<typeof useUiStore>
}

const pendingIssueDetailDefinitionMap = pendingIssueDetailDefinitions.reduce<
  Record<PendingIssueDetailKey, PendingIssueDetailDefinition>
>((acc, item) => {
  acc[item.key] = item
  return acc
}, {} as Record<PendingIssueDetailKey, PendingIssueDetailDefinition>)

export const getPendingCenterVisualImage = (game: GameListItem) => {
  return game.banner_image || game.primary_screenshot || game.cover_image || PLACEHOLDER_IMAGE
}

export const getPendingCenterDisplayImage = (game: GameListItem) => {
  return game.cover_image || game.banner_image || game.primary_screenshot || PLACEHOLDER_IMAGE
}

export const formatPendingCenterDate = (value?: string | null) => {
  if (!value) return '未知时间'
  return formatDisplayDate(value) || '未知时间'
}

export const usePendingCenterView = ({
  route,
  router,
  uiStore,
}: UsePendingCenterViewOptions) => {
  const editingGame = ref<GameDetail | null>(null)
  const showEditModal = ref(false)
  const detailHeroFit = ref<'cover' | 'contain'>('cover')
  const detailHeroSrc = ref('')
  const detailHeroRequestId = ref(0)

  const {
    activeGame,
    changePage,
    currentBatchCount,
    currentPage,
    filteredGames,
    getIssueEvaluation,
    getIgnoredIssueDetails,
    getVisibleIssueGroups,
    getVisibleIssueDetails,
    ignoredOverridesCount,
    ignoreIssue,
    isSevereGame,
    isLoading,
    issueCounts,
    loadWorkbenchGames,
    onlyRecent,
    onlySevere,
    resetFilters,
    restoreIssue,
    reviewOverrideMap,
    searchQuery,
    selectedIssue,
    showIgnored,
    sortBy,
    totalPages,
    totalPendingCount,
  } = usePendingWorkbench({
    addAlert: (message, type) => uiStore.addAlert(message, type),
  })

  const syncAmbientBackground = () => {
    if (!activeGame.value?.public_id) {
      uiStore.clearAmbientBackgroundOverride()
      return
    }

    const imageUrl = getPendingCenterVisualImage(activeGame.value)
    if (!imageUrl || imageUrl === PLACEHOLDER_IMAGE) {
      uiStore.clearAmbientBackgroundOverride()
      return
    }

    uiStore.setAmbientBackgroundOverride({
      key: activeGame.value.public_id,
      url: imageUrl,
    })
  }

  const preloadDetailHero = (src: string) => {
    return new Promise<{ width: number; height: number } | null>((resolve) => {
      if (!src || src === PLACEHOLDER_IMAGE) {
        resolve(null)
        return
      }

      const image = new Image()
      image.onload = () => {
        resolve({
          width: image.naturalWidth,
          height: image.naturalHeight,
        })
      }
      image.onerror = () => resolve(null)
      image.src = src
    })
  }

  const updateDetailHero = async () => {
    const requestId = detailHeroRequestId.value + 1
    detailHeroRequestId.value = requestId

    const nextSrc = activeGame.value ? getPendingCenterVisualImage(activeGame.value) : PLACEHOLDER_IMAGE

    if (!nextSrc || nextSrc === PLACEHOLDER_IMAGE) {
      detailHeroFit.value = 'contain'
      detailHeroSrc.value = PLACEHOLDER_IMAGE
      return
    }

    const meta = await preloadDetailHero(nextSrc)
    if (requestId !== detailHeroRequestId.value) {
      return
    }

    const aspectRatio = meta?.width && meta?.height ? meta.width / meta.height : 1
    detailHeroFit.value = aspectRatio >= 1.5 ? 'cover' : 'contain'
    detailHeroSrc.value = nextSrc
  }

  watch(
    activeGame,
    () => {
      syncAmbientBackground()
      void updateDetailHero()
    },
    { immediate: true },
  )

  const activeGameDetails = computed(() => {
    if (!activeGame.value) return []
    const activeOverrides = reviewOverrideMap.value[String(activeGame.value.id)] || []
    const activeOverrideReasonMap = Object.fromEntries(activeOverrides.map((item) => [item.issue_key, item.reason || '']))
    const evaluation = getIssueEvaluation(activeGame.value)

    return [
      ...evaluation.details.map((key) => {
        const definition = pendingIssueDetailDefinitionMap[key]
        return definition ? { ...definition, ignored: false, reason: '' } : null
      }),
      ...evaluation.ignoredDetails.map((key) => {
        const definition = pendingIssueDetailDefinitionMap[key]
        return definition ? { ...definition, ignored: true, reason: activeOverrideReasonMap[key] || '' } : null
      }),
    ].filter((item): item is NonNullable<typeof item> => Boolean(item))
  })

  const selectGame = (game: GameListItem) => {
    activeGame.value = game
  }

  const toggleIssueFilter = (key: PendingIssueKey) => {
    selectedIssue.value = selectedIssue.value === key ? undefined : key
  }

  const openEdit = async (game: GameListItem) => {
    if (!game.public_id) return
    try {
      editingGame.value = await gamesService.getGame(game.public_id)
      showEditModal.value = true
    } catch {
      uiStore.addAlert('加载游戏详情失败', 'error')
    }
  }

  const openWiki = (game: GameListItem) => {
    if (!game.public_id) return
    router.push({
      name: 'wiki-edit',
      params: { publicId: game.public_id },
      query: createDetailRouteQuery(route),
    })
  }

  const viewGame = (game: GameListItem) => {
    if (!game.public_id) return
    router.push({
      name: 'game-detail',
      params: { publicId: game.public_id },
      query: createDetailRouteQuery(route),
    })
  }

  const refreshWorkbench = async () => {
    await loadWorkbenchGames()
  }

  const handleEditSuccess = async () => {
    showEditModal.value = false
    await loadWorkbenchGames()
  }

  onMounted(async () => {
    await loadWorkbenchGames()
  })

  onActivated(() => {
    syncAmbientBackground()
  })

  onDeactivated(() => {
    uiStore.clearAmbientBackgroundOverride()
  })

  onBeforeUnmount(() => {
    uiStore.clearAmbientBackgroundOverride()
  })

  return {
    activeGame,
    activeGameDetails,
    changePage,
    currentBatchCount,
    currentPage,
    detailHeroFit,
    detailHeroSrc,
    editingGame,
    filteredGames,
    formatDate: formatPendingCenterDate,
    getDisplayImage: getPendingCenterDisplayImage,
    getIgnoredIssueDetails,
    getPendingIssueDetailLabel,
    getPendingIssueLabel,
    getVisibleIssueDetails,
    getVisibleIssueGroups,
    handleEditSuccess,
    ignoreIssue,
    ignoredOverridesCount,
    isLoading,
    isSevereGame,
    issueCounts,
    onlyRecent,
    onlySevere,
    openEdit,
    openWiki,
    refreshWorkbench,
    resetFilters,
    restoreIssue,
    searchQuery,
    selectGame,
    selectedIssue,
    showEditModal,
    showIgnored,
    sortBy,
    toggleIssueFilter,
    totalPages,
    totalPendingCount,
    viewGame,
  }
}
