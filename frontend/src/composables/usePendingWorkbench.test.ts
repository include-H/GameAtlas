import { nextTick } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { usePendingWorkbench, PENDING_WORKBENCH_PAGE_SIZE } from './usePendingWorkbench'
import type { PendingIssueEvaluation } from '@/services/types'

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
  const baseEvaluation: PendingIssueEvaluation = {
    groups: ['missing-assets'],
    details: [
      {
        key: 'missing-cover',
        group: 'missing-assets',
        ignored: false,
        reason: null,
      },
    ],
    severe: false,
  }

  beforeEach(() => {
    getGamesMock.mockReset()
    ignoreMock.mockReset()
    restoreMock.mockReset()
  })

  it('loads pending queue with native games pagination', async () => {
    getGamesMock.mockResolvedValue({
      data: [
        { public_id: 'game-1', title: 'A', pending_issues: baseEvaluation },
        { public_id: 'game-2', title: 'B', pending_issues: baseEvaluation },
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
      },
      sort: {
        field: 'pending_issue_count',
        order: 'desc',
      },
    })
    expect(workbench.pendingGames.value).toEqual([
      { public_id: 'game-1', title: 'A', pending_issues: baseEvaluation },
      { public_id: 'game-2', title: 'B', pending_issues: baseEvaluation },
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
    expect(workbench.hasLoadFailure.value).toBe(false)
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
        pending_issue_counts: {
          groups: {},
          ignored_total: 0,
        },
      },
    })

    const workbench = usePendingWorkbench({ addAlert: vi.fn() })
    workbench.searchQuery.value = 'halo'
    workbench.selectedIssue.value = 'missing-assets'
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
      },
      sort: {
        field: 'downloads',
        order: 'desc',
      },
    })
  })

  it('treats missing pending counts as a load error instead of fabricating empty queue metadata', async () => {
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

    const addAlert = vi.fn()
    const workbench = usePendingWorkbench({ addAlert })

    await workbench.loadWorkbenchGames()

    expect(workbench.pendingIssueCounts.value).toEqual({})
    expect(workbench.pendingIssueIgnoredTotal.value).toBe(0)
    expect(workbench.hasLoadFailure.value).toBe(true)
    expect(addAlert).toHaveBeenCalledWith('加载待处理工作台失败', 'error')
  })

  it('treats missing pending evaluations as a load error instead of fabricating empty issue details', async () => {
    getGamesMock.mockResolvedValue({
      data: [
        { public_id: 'game-1', title: 'A', pending_issues: null },
      ],
      pagination: {
        total: 1,
        totalPages: 1,
        page: 1,
        limit: PENDING_WORKBENCH_PAGE_SIZE,
        pending_issue_counts: {
          groups: {
            'missing-assets': 1,
          },
          ignored_total: 0,
        },
      },
    })

    const addAlert = vi.fn()
    const workbench = usePendingWorkbench({ addAlert })

    await workbench.loadWorkbenchGames()

    expect(workbench.pendingGames.value).toEqual([])
    expect(workbench.hasLoadFailure.value).toBe(true)
    expect(addAlert).toHaveBeenCalledWith('加载待处理工作台失败', 'error')
  })

  it('discards stale responses when workbench loads overlap', async () => {
    const stalePage = {
      data: [{ public_id: 'game-old', title: 'Old', pending_issues: baseEvaluation }],
      pagination: {
        total: 1,
        totalPages: 1,
        page: 1,
        limit: PENDING_WORKBENCH_PAGE_SIZE,
        pending_issue_counts: {
          groups: {
            'missing-assets': 1,
          },
          ignored_total: 0,
        },
      },
    }
    const freshPage = {
      data: [{ public_id: 'game-new', title: 'New', pending_issues: baseEvaluation }],
      pagination: {
        total: 8,
        totalPages: 3,
        page: 2,
        limit: PENDING_WORKBENCH_PAGE_SIZE,
        pending_issue_counts: {
          groups: {
            'missing-assets': 2,
          },
          ignored_total: 3,
        },
      },
    }
    let resolveFirst!: (value: unknown) => void
    getGamesMock
      .mockImplementationOnce(() => new Promise((resolve) => {
        resolveFirst = resolve
      }))
      .mockResolvedValueOnce(freshPage)

    const workbench = usePendingWorkbench({ addAlert: vi.fn() })
    const firstLoad = workbench.loadWorkbenchGames()
    const secondLoad = workbench.loadWorkbenchGames()
    resolveFirst(stalePage)

    await Promise.all([firstLoad, secondLoad])

    expect(getGamesMock).toHaveBeenCalledTimes(2)
    expect(workbench.pendingGames.value).toEqual(freshPage.data)
    expect(workbench.pendingIssueCounts.value).toEqual({ 'missing-assets': 2 })
    expect(workbench.pendingIssueIgnoredTotal.value).toBe(3)
    expect(workbench.currentPage.value).toBe(2)
    expect(workbench.totalPages.value).toBe(3)
    expect(workbench.isLoading.value).toBe(false)
  })
})
