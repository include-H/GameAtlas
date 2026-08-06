import { computed, ref, watch } from 'vue'
import gamesService from '@/services/games.service'
import reviewIssuesService from '@/services/review-issues.service'
import type {
  GameListItem,
  GameSort,
  PendingIssueDetailState,
  PendingIssueEvaluation,
  PendingWorkbenchPagination,
} from '@/services/types'

export const PENDING_WORKBENCH_PAGE_SIZE = 10

type PendingWorkbenchSortBy =
  | 'issue-count'
  | 'created-desc'
  | 'updated-asc'
  | 'downloads-desc'

interface UsePendingWorkbenchOptions {
  addAlert: (message: string, type: 'success' | 'warning' | 'error') => void
}

export const usePendingWorkbench = (options: UsePendingWorkbenchOptions) => {
  const isLoading = ref(false)
  const hasLoadFailure = ref(false)
  const pendingGames = ref<GameListItem[]>([])
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
  const pendingIssueIgnoredTotal = ref(0)
  const pendingIssueCounts = ref<Record<string, number>>({})

  let workbenchRequestId = 0

  const pageGameCount = computed(() => pendingGames.value.length)

  const getIssueEvaluation = (game: GameListItem): PendingIssueEvaluation => {
    if (game.pending_issues) {
      return game.pending_issues
    }
    throw new Error(`pending workbench game ${game.public_id || game.id} is missing pending_issues`)
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

  watch(
    pendingGames,
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
    const requestId = ++workbenchRequestId
    isLoading.value = true
    hasLoadFailure.value = false
    try {
      const query = buildWorkbenchQuery()
      const response = await gamesService.getGames<PendingWorkbenchPagination>({
        query: {
          page,
          limit: PENDING_WORKBENCH_PAGE_SIZE,
          pending: true,
          search: query.search,
          pending_issue: query.issue,
          pending_include_ignored: query.showIgnored,
          pending_severe: query.onlySevere,
          pending_recent_days: query.onlyRecent ? 7 : undefined,
        },
        sort: resolvePendingWorkbenchSort(query.sortBy),
      })
      // 2026-04-07: pending workbench reads the backend-native pending queue contract directly.
      // Impact: this screen no longer fabricates empty counts/evaluations when the API stops
      // attaching pending_issue_counts or pending_issues to a pending=true response.
      const countsSummary = response.pagination.pending_issue_counts
      if (!countsSummary) {
        throw new Error('pending workbench response missing pending_issue_counts')
      }
      if (response.data.some((game) => !game.pending_issues)) {
        throw new Error('pending workbench response missing pending_issues')
      }
      if (requestId !== workbenchRequestId) {
        return
      }
      pendingGames.value = response.data
      pendingIssueIgnoredTotal.value = countsSummary.ignored_total
      pendingIssueCounts.value = countsSummary.groups
      currentPage.value = response.pagination.page
      totalPages.value = response.pagination.totalPages
      totalPendingCount.value = response.pagination.total

      if (response.pagination.page > response.pagination.totalPages && response.pagination.totalPages > 0) {
        await loadWorkbenchGames(response.pagination.totalPages)
        return
      }
    } catch {
      if (requestId !== workbenchRequestId) {
        return
      }
      hasLoadFailure.value = true
      options.addAlert('加载待处理工作台失败', 'error')
    } finally {
      if (requestId === workbenchRequestId) {
        isLoading.value = false
      }
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
      try {
        // 2026-04-10: pending override writes and queue refresh are separate outcomes.
        // Impact: ignore success must stay visible even if the follow-up queue refresh fails.
        await refreshCurrentPage()
      } catch {
        options.addAlert('忽略已生效，但待处理列表刷新失败，请稍后重试', 'warning')
      }
    } catch {
      options.addAlert('忽略问题失败', 'error')
    }
  }

  const restoreIssue = async (game: GameListItem, issueKey: string) => {
    if (!game.public_id) return
    try {
      await reviewIssuesService.restore(game.public_id, issueKey)
      options.addAlert('已恢复待处理项', 'success')
      try {
        // 2026-04-10: restore success must not be rewritten into a generic failure
        // by a later refreshCurrentPage() error.
        await refreshCurrentPage()
      } catch {
        options.addAlert('恢复已生效，但待处理列表刷新失败，请稍后重试', 'warning')
      }
    } catch {
      options.addAlert('恢复问题失败', 'error')
    }
  }

  const changePage = async (page: number) => {
    const safePage = Math.max(1, page)
    if (safePage === currentPage.value && pendingGames.value.length > 0) {
      return
    }
    await loadWorkbenchGames(safePage)
  }

  return {
    isLoading,
    hasLoadFailure,
    activeGame,
    pageGameCount,
    currentPage,
    pendingGames,
    pendingIssueCounts,
    pendingIssueIgnoredTotal,
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

function resolvePendingWorkbenchSort(sortBy: PendingWorkbenchSortBy | undefined): GameSort {
  if (sortBy === 'created-desc') {
    return { field: 'created_at', order: 'desc' }
  }
  if (sortBy === 'downloads-desc') {
    return { field: 'downloads', order: 'desc' }
  }
  if (sortBy === 'updated-asc') {
    return { field: 'updated_at', order: 'asc' }
  }

  return { field: 'pending_issue_count', order: 'desc' }
}
