import { computed, ref, watch } from 'vue'
import pendingWorkbenchService, {
  PENDING_WORKBENCH_PAGE_SIZE,
  type PendingWorkbenchSortBy,
} from '@/services/pending-workbench.service'
import reviewIssuesService from '@/services/review-issues.service'
import type { GameListItem, PendingIssueDetailState, PendingIssueEvaluation } from '@/services/types'

export { PENDING_WORKBENCH_PAGE_SIZE }

interface UsePendingWorkbenchOptions {
  addAlert: (message: string, type: 'success' | 'warning' | 'error') => void
}

export const usePendingWorkbench = (options: UsePendingWorkbenchOptions) => {
  const emptyEvaluation: PendingIssueEvaluation = {
    groups: [],
    details: [],
    severe: false,
  }

  const isLoading = ref(false)
  const queueGames = ref<GameListItem[]>([])
  const activeGame = ref<GameListItem | null>(null)

  const currentPage = ref(1)
  const totalPages = ref(0)
  const totalPendingCount = ref(0)

  const searchQuery = ref('')
  const selectedIssue = ref<string | undefined>()
  const sortBy = ref<PendingWorkbenchSortBy>('issue-count')
  const onlySevere = ref(false)
  const onlyRecent = ref(false)
  const showIgnored = ref(false)
  const backendIgnoredOverridesCount = ref(0)
  const backendIssueCounts = ref<Record<string, number>>({})

  const ignoredOverridesCount = computed(() => backendIgnoredOverridesCount.value)
  const currentBatchCount = computed(() => queueGames.value.length)

  const getIssueEvaluation = (game: GameListItem): PendingIssueEvaluation => {
    return game.pending_issues || emptyEvaluation
  }

  const isSevereGame = (game: GameListItem) => {
    return getIssueEvaluation(game).severe
  }

  const getVisibleIssueGroups = (game: GameListItem) => getIssueEvaluation(game).groups
  const getVisibleIssueDetails = (game: GameListItem): PendingIssueDetailState[] => (
    getIssueEvaluation(game).details.filter((detail) => !detail.ignored)
  )
  const getIgnoredIssueDetails = (game: GameListItem): PendingIssueDetailState[] => (
    getIssueEvaluation(game).details.filter((detail) => detail.ignored)
  )

  const issueCounts = computed(() => backendIssueCounts.value)
  const filteredGames = computed(() => queueGames.value)

  watch(
    filteredGames,
    (games) => {
      if (games.length === 0) {
        activeGame.value = null
        return
      }

      const currentActiveId = activeGame.value?.public_id || null
      const matched = currentActiveId
        ? games.find((game) => game.public_id === currentActiveId)
        : null

      activeGame.value = matched || games[0]
    },
    { immediate: true },
  )

  const resetFilters = () => {
    searchQuery.value = ''
    selectedIssue.value = undefined
    sortBy.value = 'issue-count'
    onlySevere.value = false
    onlyRecent.value = false
    showIgnored.value = false
  }

  const buildWorkbenchQuery = () => ({
    search: searchQuery.value.trim() || undefined,
    issue: selectedIssue.value,
    onlySevere: onlySevere.value,
    onlyRecent: onlyRecent.value,
    showIgnored: showIgnored.value,
    sortBy: sortBy.value,
  })

  const loadWorkbenchGames = async (page = currentPage.value) => {
    isLoading.value = true
    try {
      const snapshot = await pendingWorkbenchService.getSnapshot(
        page,
        PENDING_WORKBENCH_PAGE_SIZE,
        buildWorkbenchQuery(),
      )
      queueGames.value = snapshot.queueGames
      backendIgnoredOverridesCount.value = snapshot.ignoredTotal
      backendIssueCounts.value = snapshot.issueCounts
      currentPage.value = snapshot.page
      totalPages.value = snapshot.totalPages
      totalPendingCount.value = snapshot.total

      if (snapshot.page > snapshot.totalPages && snapshot.totalPages > 0) {
        await loadWorkbenchGames(snapshot.totalPages)
      }
    } catch {
      options.addAlert('加载待处理工作台失败', 'error')
    } finally {
      isLoading.value = false
    }
  }

  watch(
    [searchQuery, selectedIssue, sortBy, onlySevere, onlyRecent, showIgnored],
    async () => {
      await loadWorkbenchGames(1)
    },
  )

  const refreshCurrentPage = async () => {
    await loadWorkbenchGames(currentPage.value)
  }

  const ignoreIssue = async (game: GameListItem, issueKey: string) => {
    if (!game.public_id) return
    try {
      await reviewIssuesService.ignore(game.public_id, issueKey)
      options.addAlert('已忽略待处理项', 'success')
      await refreshCurrentPage()
    } catch {
      options.addAlert('忽略问题失败', 'error')
    }
  }

  const restoreIssue = async (game: GameListItem, issueKey: string) => {
    if (!game.public_id) return
    try {
      await reviewIssuesService.restore(game.public_id, issueKey)
      options.addAlert('已恢复待处理项', 'success')
      await refreshCurrentPage()
    } catch {
      options.addAlert('恢复问题失败', 'error')
    }
  }

  const changePage = async (page: number) => {
    const safePage = Math.max(1, page)
    if (safePage === currentPage.value && queueGames.value.length > 0) {
      return
    }
    await loadWorkbenchGames(safePage)
  }

  return {
    isLoading,
    activeGame,
    currentBatchCount,
    currentPage,
    filteredGames,
    ignoredOverridesCount,
    issueCounts,
    onlyRecent,
    onlySevere,
    searchQuery,
    selectedIssue,
    showIgnored,
    sortBy,
    totalPages,
    totalPendingCount,
    getIssueEvaluation,
    isSevereGame,
    getIgnoredIssueDetails,
    getVisibleIssueDetails,
    getVisibleIssueGroups,
    loadWorkbenchGames,
    ignoreIssue,
    restoreIssue,
    changePage,
    resetFilters,
  }
}
