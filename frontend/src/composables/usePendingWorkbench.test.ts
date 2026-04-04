import { nextTick } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { usePendingWorkbench, PENDING_WORKBENCH_PAGE_SIZE } from './usePendingWorkbench'

const { getGamesMock, ignoreMock, restoreMock } = vi.hoisted(() => ({
  getGamesMock: vi.fn(),
  ignoreMock: vi.fn(),
  restoreMock: vi.fn(),
}))

vi.mock('@/services/games.service', () => ({
  default: {
    getGames: getGamesMock,
  },
}))

vi.mock('@/services/review-issues.service', () => ({
  default: {
    ignore: ignoreMock,
    restore: restoreMock,
  },
}))

describe('usePendingWorkbench', () => {
  beforeEach(() => {
    getGamesMock.mockReset()
    ignoreMock.mockReset()
    restoreMock.mockReset()
  })

  it('loads pending queue with native games pagination', async () => {
    getGamesMock.mockResolvedValue({
      data: [
        { public_id: 'game-1', title: 'A' },
        { public_id: 'game-2', title: 'B' },
      ],
      pagination: {
        total: 11,
        totalPages: 2,
        page: 1,
        limit: PENDING_WORKBENCH_PAGE_SIZE,
        pending_issue_counts: {
          groups: {
            'missing-assets': 6,
            'missing-wiki': 4,
          },
          ignored_total: 9,
        },
      },
    })

    const addAlert = vi.fn()
    const workbench = usePendingWorkbench({ addAlert })

    await workbench.loadWorkbenchGames()

    expect(getGamesMock).toHaveBeenCalledWith({
      query: {
        page: 1,
        limit: PENDING_WORKBENCH_PAGE_SIZE,
        pending: true,
        search: undefined,
        pending_issue: undefined,
        pending_include_ignored: false,
        pending_severe: false,
        pending_recent_days: undefined,
      },
      sort: {
        field: 'pending_issue_count',
        order: 'desc',
      },
    })
    expect(workbench.pendingGames.value).toEqual([
      { public_id: 'game-1', title: 'A' },
      { public_id: 'game-2', title: 'B' },
    ])
    expect(workbench.pendingIssueCounts.value).toEqual({
      'missing-assets': 6,
      'missing-wiki': 4,
    })
    expect(workbench.pendingIssueIgnoredTotal.value).toBe(9)
    expect(workbench.totalPendingCount.value).toBe(11)
    expect(workbench.totalPages.value).toBe(2)
    expect(workbench.currentPage.value).toBe(1)
    expect(workbench.pageGameCount.value).toBe(2)
    expect(workbench.activeGame.value?.public_id).toBe('game-1')
    expect(addAlert).not.toHaveBeenCalled()
  })

  it('maps workbench filters to the native games query', async () => {
    getGamesMock.mockResolvedValue({
      data: [],
      pagination: {
        total: 0,
        totalPages: 0,
        page: 1,
        limit: PENDING_WORKBENCH_PAGE_SIZE,
        pending_issue_counts: null,
      },
    })

    const workbench = usePendingWorkbench({ addAlert: vi.fn() })
    workbench.searchQuery.value = 'halo'
    workbench.selectedIssue.value = 'missing-assets'
    workbench.onlySevere.value = true
    workbench.onlyRecent.value = true
    workbench.showIgnored.value = true
    workbench.sortBy.value = 'downloads-desc'

    await nextTick()
    getGamesMock.mockClear()
    await workbench.loadWorkbenchGames(1)

    expect(getGamesMock).toHaveBeenCalledWith({
      query: {
        page: 1,
        limit: PENDING_WORKBENCH_PAGE_SIZE,
        pending: true,
        search: 'halo',
        pending_issue: 'missing-assets',
        pending_include_ignored: true,
        pending_severe: true,
        pending_recent_days: 7,
      },
      sort: {
        field: 'downloads',
        order: 'desc',
      },
    })
  })
})
