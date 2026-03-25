import { computed, ref, watch } from 'vue'
import pendingWorkbenchService, {
  PENDING_WORKBENCH_WINDOW_SIZE,
} from '@/services/pending-workbench.service'
import reviewIssuesService from '@/services/review-issues.service'
import type { Game, ReviewIssueOverride } from '@/services/types'
import {
  getIgnoredPendingIssueDetails,
  getPendingIssueDetails,
  getPendingIssues,
  isSeverePendingGame,
  pendingIssueDefinitions,
  type PendingIssueDetailKey,
  type PendingIssueKey,
} from '@/utils/pendingIssues'

export { PENDING_WORKBENCH_WINDOW_SIZE }

export type PendingWorkbenchSortBy =
  | 'issue-count'
  | 'created-desc'
  | 'updated-asc'
  | 'downloads-desc'

interface UsePendingWorkbenchOptions {
  addAlert: (message: string, type: 'success' | 'warning' | 'error') => void
}

export const usePendingWorkbench = (options: UsePendingWorkbenchOptions) => {
  const isLoading = ref(false)
  const windowGames = ref<Game[]>([])
  const activeGame = ref<Game | null>(null)
  const reviewIssueOverrides = ref<ReviewIssueOverride[]>([])

  const searchQuery = ref('')
  const selectedIssue = ref<PendingIssueKey | undefined>()
  const sortBy = ref<PendingWorkbenchSortBy>('issue-count')
  const onlySevere = ref(false)
  const onlyRecent = ref(false)
  const showIgnored = ref(false)

  const reviewOverrideMap = computed<Record<string, ReviewIssueOverride[]>>(() => {
    return reviewIssueOverrides.value.reduce<Record<string, ReviewIssueOverride[]>>((acc, item) => {
      const key = String(item.game_id)
      if (!acc[key]) {
        acc[key] = []
      }
      acc[key].push(item)
      return acc
    }, {})
  })

  const ignoredOverridesCount = computed(() => reviewIssueOverrides.value.length)

  const getIgnoredDetails = (gameId: number | string): PendingIssueDetailKey[] => {
    return (reviewOverrideMap.value[String(gameId)] || [])
      .filter((item) => item.status === 'ignored')
      .map((item) => item.issue_key as PendingIssueDetailKey)
  }

  const getVisibleIssueGroups = (game: Game) => getPendingIssues(game, getIgnoredDetails(game.id))
  const getVisibleIssueDetails = (game: Game) => getPendingIssueDetails(game, getIgnoredDetails(game.id))
  const getIgnoredIssueDetails = (game: Game) => getIgnoredPendingIssueDetails(game, getIgnoredDetails(game.id))
  const hasVisibleIssues = (game: Game) => getVisibleIssueDetails(game).length > 0

  const issueCounts = computed(() => {
    const counts = {} as Record<PendingIssueKey, number>
    pendingIssueDefinitions.forEach((definition) => {
      counts[definition.key] = windowGames.value.filter((game) =>
        getVisibleIssueGroups(game).includes(definition.key),
      ).length
    })
    return counts
  })

  const totalWindowCount = computed(() => windowGames.value.length)
  const totalPendingCount = computed(() => windowGames.value.filter((game) => hasVisibleIssues(game)).length)

  const filteredGames = computed(() => {
    const keyword = searchQuery.value.trim().toLowerCase()
    const recentThreshold = Date.now() - 7 * 24 * 60 * 60 * 1000

    const games = windowGames.value.filter((game) => {
      if (!showIgnored.value && !hasVisibleIssues(game)) {
        return false
      }

      if (selectedIssue.value && !getVisibleIssueGroups(game).includes(selectedIssue.value)) {
        return false
      }

      if (onlySevere.value && !isSeverePendingGame(game, getIgnoredDetails(game.id))) {
        return false
      }

      if (onlyRecent.value) {
        const createdAt = new Date(game.created_at).getTime()
        if (Number.isNaN(createdAt) || createdAt < recentThreshold) {
          return false
        }
      }

      if (!keyword) {
        return true
      }

      const metadata = [
        game.title,
        game.summary || '',
        ...(game.developers || []).map((item) => item.name),
        ...(game.publishers || []).map((item) => item.name),
        ...(game.platforms || []),
      ]

      return metadata.join(' ').toLowerCase().includes(keyword)
    })

    return [...games].sort((left, right) => {
      if (sortBy.value === 'created-desc') {
        return new Date(right.created_at).getTime() - new Date(left.created_at).getTime()
      }
      if (sortBy.value === 'updated-asc') {
        return new Date(left.updated_at).getTime() - new Date(right.updated_at).getTime()
      }
      if (sortBy.value === 'downloads-desc') {
        return (right.downloads || 0) - (left.downloads || 0)
      }
      return getVisibleIssueDetails(right).length - getVisibleIssueDetails(left).length
    })
  })

  watch(
    filteredGames,
    (games) => {
      if (games.length === 0) {
        activeGame.value = null
        return
      }

      const currentActiveId = activeGame.value ? String(activeGame.value.id) : null
      const matched = currentActiveId
        ? games.find((game) => String(game.id) === currentActiveId)
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

  const ignoreIssue = async (game: Game, issueKey: PendingIssueDetailKey) => {
    try {
      const override = await reviewIssuesService.ignore(String(game.id), issueKey)
      reviewIssueOverrides.value = [
        ...reviewIssueOverrides.value.filter(
          (item) => !(item.game_id === override.game_id && item.issue_key === override.issue_key),
        ),
        override,
      ]
      options.addAlert('已忽略待处理项', 'success')
    } catch {
      options.addAlert('忽略问题失败', 'error')
    }
  }

  const restoreIssue = async (game: Game, issueKey: PendingIssueDetailKey) => {
    try {
      await reviewIssuesService.restore(String(game.id), issueKey)
      reviewIssueOverrides.value = reviewIssueOverrides.value.filter(
        (item) => !(item.game_id === game.id && item.issue_key === issueKey),
      )
      options.addAlert('已恢复待处理项', 'success')
    } catch {
      options.addAlert('恢复问题失败', 'error')
    }
  }

  const loadWorkbenchGames = async () => {
    isLoading.value = true
    try {
      const snapshot = await pendingWorkbenchService.getSnapshot(PENDING_WORKBENCH_WINDOW_SIZE)
      windowGames.value = snapshot.windowGames
      reviewIssueOverrides.value = snapshot.overrides
    } catch {
      options.addAlert('加载待处理工作台失败', 'error')
    } finally {
      isLoading.value = false
    }
  }

  return {
    isLoading,
    activeGame,
    filteredGames,
    ignoredOverridesCount,
    issueCounts,
    onlyRecent,
    onlySevere,
    searchQuery,
    selectedIssue,
    showIgnored,
    sortBy,
    totalPendingCount,
    totalWindowCount,
    reviewOverrideMap,
    getIgnoredDetails,
    getIgnoredIssueDetails,
    getVisibleIssueDetails,
    getVisibleIssueGroups,
    hasVisibleIssues,
    ignoreIssue,
    loadWorkbenchGames,
    resetFilters,
    restoreIssue,
  }
}
