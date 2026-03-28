import { computed, ref, watch } from 'vue'
import pendingWorkbenchService, {
  PENDING_WORKBENCH_PAGE_SIZE,
} from '@/services/pending-workbench.service'
import reviewIssuesService from '@/services/review-issues.service'
import type { GameListItem, ReviewIssueOverride } from '@/services/types'
import {
  evaluatePendingIssues,
  isSeverePendingEvaluation,
  pendingIssueDefinitions,
  type PendingIssueEvaluation,
  type PendingIssueDetailKey,
  type PendingIssueKey,
} from '@/utils/pendingIssues'

export { PENDING_WORKBENCH_PAGE_SIZE }

type PendingWorkbenchSortBy =
  | 'issue-count'
  | 'created-desc'
  | 'updated-asc'
  | 'downloads-desc'

interface UsePendingWorkbenchOptions {
  addAlert: (message: string, type: 'success' | 'warning' | 'error') => void
}

export const usePendingWorkbench = (options: UsePendingWorkbenchOptions) => {
  const emptyEvaluation: PendingIssueEvaluation = {
    groups: [],
    details: [],
    ignoredDetails: [],
  }

  const isLoading = ref(false)
  const queueGames = ref<GameListItem[]>([])
  const activeGame = ref<GameListItem | null>(null)
  const reviewIssueOverrides = ref<ReviewIssueOverride[]>([])
  const gamePublicIDByInternalID = computed<Record<number, string>>(() => {
    return queueGames.value.reduce<Record<number, string>>((acc, game) => {
      if (game.public_id) {
        acc[game.id] = game.public_id
      }
      return acc
    }, {})
  })

  const currentPage = ref(1)
  const totalPages = ref(0)
  const totalPendingCount = ref(0)

  const searchQuery = ref('')
  const selectedIssue = ref<PendingIssueKey | undefined>()
  const sortBy = ref<PendingWorkbenchSortBy>('issue-count')
  const onlySevere = ref(false)
  const onlyRecent = ref(false)
  const showIgnored = ref(false)

  const reviewOverrideMap = computed<Record<string, ReviewIssueOverride[]>>(() => {
    return reviewIssueOverrides.value.reduce<Record<string, ReviewIssueOverride[]>>((acc, item) => {
      const key = gamePublicIDByInternalID.value[item.game_id]
      if (!key) {
        return acc
      }
      if (!acc[key]) {
        acc[key] = []
      }
      acc[key].push(item)
      return acc
    }, {})
  })

  const ignoredOverridesCount = computed(() => reviewIssueOverrides.value.length)
  const currentBatchCount = computed(() => queueGames.value.length)

  const ignoredDetailMap = computed<Record<string, PendingIssueDetailKey[]>>(() => {
    return Object.entries(reviewOverrideMap.value).reduce<Record<string, PendingIssueDetailKey[]>>((acc, [gameId, items]) => {
      acc[gameId] = items
        .filter((item) => item.status === 'ignored')
        .map((item) => item.issue_key as PendingIssueDetailKey)
      return acc
    }, {})
  })

  const getIgnoredDetails = (game: GameListItem): PendingIssueDetailKey[] => {
    if (!game.public_id) {
      return []
    }
    return ignoredDetailMap.value[game.public_id] || []
  }

  const gameIssueEvaluationMap = computed<Record<string, PendingIssueEvaluation>>(() => {
    return queueGames.value.reduce<Record<string, PendingIssueEvaluation>>((acc, game) => {
      if (!game.public_id) {
        return acc
      }
      acc[game.public_id] = evaluatePendingIssues(game, getIgnoredDetails(game))
      return acc
    }, {})
  })

  const getIssueEvaluation = (game: GameListItem): PendingIssueEvaluation => {
    if (!game.public_id) {
      return emptyEvaluation
    }
    return gameIssueEvaluationMap.value[game.public_id] || emptyEvaluation
  }

  const isSevereGame = (game: GameListItem) => {
    return isSeverePendingEvaluation(getIssueEvaluation(game))
  }

  const getVisibleIssueGroups = (game: GameListItem) => getIssueEvaluation(game).groups
  const getVisibleIssueDetails = (game: GameListItem) => getIssueEvaluation(game).details
  const getIgnoredIssueDetails = (game: GameListItem) => getIssueEvaluation(game).ignoredDetails
  const hasVisibleIssues = (game: GameListItem) => getIssueEvaluation(game).details.length > 0

  const issueCounts = computed(() => {
    const counts = {} as Record<PendingIssueKey, number>
    pendingIssueDefinitions.forEach((definition) => {
      counts[definition.key] = queueGames.value.filter((game) =>
        getVisibleIssueGroups(game).includes(definition.key),
      ).length
    })
    return counts
  })

  const filteredGames = computed(() => {
    const keyword = searchQuery.value.trim().toLowerCase()
    const recentThreshold = Date.now() - 7 * 24 * 60 * 60 * 1000

    const games = queueGames.value.filter((game) => {
      if (!showIgnored.value && !hasVisibleIssues(game)) {
        return false
      }

      if (selectedIssue.value && !getVisibleIssueGroups(game).includes(selectedIssue.value)) {
        return false
      }

      if (onlySevere.value && !isSevereGame(game)) {
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
      return getIssueEvaluation(right).details.length - getIssueEvaluation(left).details.length
    })
  })

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

  const loadWorkbenchGames = async (page = currentPage.value) => {
    isLoading.value = true
    try {
      const snapshot = await pendingWorkbenchService.getSnapshot(page, PENDING_WORKBENCH_PAGE_SIZE)
      queueGames.value = snapshot.queueGames
      reviewIssueOverrides.value = snapshot.overrides
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

  const refreshCurrentPage = async () => {
    await loadWorkbenchGames(currentPage.value)
  }

  const ignoreIssue = async (game: GameListItem, issueKey: PendingIssueDetailKey) => {
    if (!game.public_id) return
    try {
      const override = await reviewIssuesService.ignore(game.public_id, issueKey)
      reviewIssueOverrides.value = [
        ...reviewIssueOverrides.value.filter(
          (item) => !(item.game_id === override.game_id && item.issue_key === override.issue_key),
        ),
        override,
      ]
      options.addAlert('已忽略待处理项', 'success')
      await refreshCurrentPage()
    } catch {
      options.addAlert('忽略问题失败', 'error')
    }
  }

  const restoreIssue = async (game: GameListItem, issueKey: PendingIssueDetailKey) => {
    if (!game.public_id) return
    try {
      await reviewIssuesService.restore(game.public_id, issueKey)
      reviewIssueOverrides.value = reviewIssueOverrides.value.filter(
        (item) => !(
          gamePublicIDByInternalID.value[item.game_id] === game.public_id
          && item.issue_key === issueKey
        ),
      )
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
    reviewOverrideMap,
    getIgnoredDetails,
    getIssueEvaluation,
    isSevereGame,
    getIgnoredIssueDetails,
    getVisibleIssueDetails,
    getVisibleIssueGroups,
    hasVisibleIssues,
    changePage,
    ignoreIssue,
    loadWorkbenchGames,
    resetFilters,
    restoreIssue,
  }
}
