import gamesService from './games.service'
import type { GameListItem } from './types'

export const PENDING_WORKBENCH_PAGE_SIZE = 10

export type PendingWorkbenchSortBy =
  | 'issue-count'
  | 'created-desc'
  | 'updated-asc'
  | 'downloads-desc'

interface PendingWorkbenchQuery {
  search?: string
  issue?: string
  onlySevere?: boolean
  onlyRecent?: boolean
  showIgnored?: boolean
  sortBy?: PendingWorkbenchSortBy
}

interface PendingWorkbenchSnapshot {
  queueGames: GameListItem[]
  issueCounts: Record<string, number>
  ignoredTotal: number
  total: number
  totalPages: number
  page: number
  limit: number
}

const pendingWorkbenchService = {
  async getSnapshot(
    page = 1,
    limit = PENDING_WORKBENCH_PAGE_SIZE,
    query: PendingWorkbenchQuery = {},
  ): Promise<PendingWorkbenchSnapshot> {
    const sort = resolvePendingWorkbenchSort(query.sortBy)
    const response = await gamesService.getGames({
      query: {
        page,
        limit,
        pending: true,
        search: query.search,
        pending_issue: query.issue,
        pending_include_ignored: query.showIgnored,
        pending_severe: query.onlySevere,
        pending_recent_days: query.onlyRecent ? 7 : undefined,
      },
      sort,
    })

    const queueGames = response.data

    return {
      queueGames,
      issueCounts: response.pagination.pending_issue_counts?.groups || {},
      ignoredTotal: response.pagination.pending_issue_counts?.ignored_total || 0,
      total: response.pagination.total,
      totalPages: response.pagination.totalPages,
      page: response.pagination.page,
      limit: response.pagination.limit,
    }
  },
}

function resolvePendingWorkbenchSort(sortBy: PendingWorkbenchSortBy | undefined) {
  if (sortBy === 'created-desc') {
    return { field: 'created_at' as const, order: 'desc' as const }
  }
  if (sortBy === 'downloads-desc') {
    return { field: 'downloads' as const, order: 'desc' as const }
  }
  if (sortBy === 'updated-asc') {
    return { field: 'updated_at' as const, order: 'asc' as const }
  }

  return { field: 'pending_issue_count' as const, order: 'desc' as const }
}

export default pendingWorkbenchService
