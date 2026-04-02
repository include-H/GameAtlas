import { beforeEach, describe, expect, it, vi } from 'vitest'

const { getGamesMock } = vi.hoisted(() => ({
  getGamesMock: vi.fn(),
}))

vi.mock('./games.service', () => ({
  default: {
    getGames: getGamesMock,
  },
}))

import pendingWorkbenchService, { PENDING_WORKBENCH_PAGE_SIZE } from './pending-workbench.service'

describe('pending workbench service', () => {
  beforeEach(() => {
    getGamesMock.mockReset()
  })

  it('requests the pending queue with default pagination', async () => {
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
            'missing-files': 3,
            'missing-metadata': 5,
          },
          ignored_total: 9,
        },
      },
    })

    const result = await pendingWorkbenchService.getSnapshot()

    expect(getGamesMock).toHaveBeenCalledWith({
      query: {
        page: 1,
        limit: PENDING_WORKBENCH_PAGE_SIZE,
        pending: true,
        search: undefined,
        pending_issue: undefined,
        pending_include_ignored: undefined,
        pending_severe: undefined,
        pending_recent_days: undefined,
      },
      sort: {
        field: 'pending_issue_count',
        order: 'desc',
      },
    })
    expect(result).toEqual({
      queueGames: [
        { public_id: 'game-1', title: 'A' },
        { public_id: 'game-2', title: 'B' },
      ],
      issueCounts: {
        'missing-assets': 6,
        'missing-wiki': 4,
        'missing-files': 3,
        'missing-metadata': 5,
      },
      ignoredTotal: 9,
      total: 11,
      totalPages: 2,
      page: 1,
      limit: PENDING_WORKBENCH_PAGE_SIZE,
    })
  })

  it('maps native pending filters and sort to the games query', async () => {
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

    await pendingWorkbenchService.getSnapshot(1, PENDING_WORKBENCH_PAGE_SIZE, {
      search: 'halo',
      issue: 'missing-assets',
      onlySevere: true,
      onlyRecent: true,
      showIgnored: true,
      sortBy: 'downloads-desc',
    })

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
