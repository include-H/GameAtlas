import gamesService from './games.service'
import reviewIssuesService from './review-issues.service'
import type { Game, ReviewIssueOverride } from './types'

export const PENDING_WORKBENCH_PAGE_SIZE = 10

export interface PendingWorkbenchSnapshot {
  queueGames: Game[]
  overrides: ReviewIssueOverride[]
  total: number
  totalPages: number
  page: number
  pageSize: number
}

const pendingWorkbenchService = {
  async getSnapshot(page = 1, pageSize = PENDING_WORKBENCH_PAGE_SIZE): Promise<PendingWorkbenchSnapshot> {
    const response = await gamesService.getGames({
      page,
      pageSize,
      filter: {
        pending_queue: true,
      },
      sort: {
        field: 'updated_at',
        order: 'asc',
      },
    })

    const queueGames = response.data
    const overrides = await reviewIssuesService.list(
      queueGames
        .map((game) => game.public_id)
        .filter((value): value is string => Boolean(value)),
    )

    return {
      queueGames,
      overrides,
      total: response.pagination.total,
      totalPages: response.pagination.totalPages,
      page: response.pagination.page,
      pageSize: response.pagination.limit,
    }
  },
}

export default pendingWorkbenchService
