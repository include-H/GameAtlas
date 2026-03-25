import gamesService from './games.service'
import reviewIssuesService from './review-issues.service'
import type { Game, ReviewIssueOverride } from './types'

export const PENDING_WORKBENCH_WINDOW_SIZE = 50

export interface PendingWorkbenchSnapshot {
  windowGames: Game[]
  overrides: ReviewIssueOverride[]
}

const pendingWorkbenchService = {
  async getSnapshot(windowSize = PENDING_WORKBENCH_WINDOW_SIZE): Promise<PendingWorkbenchSnapshot> {
    const response = await gamesService.getGames({
      page: 1,
      pageSize: windowSize,
      sort: {
        field: 'updated_at',
        order: 'desc',
      },
    })

    const windowGames = response.data
    const overrides = await reviewIssuesService.list(windowGames.map((game) => game.id))

    return {
      windowGames,
      overrides,
    }
  },
}

export default pendingWorkbenchService
