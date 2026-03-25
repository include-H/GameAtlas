import { beforeEach, describe, expect, it, vi } from 'vitest'

const { getGamesMock, listMock } = vi.hoisted(() => ({
  getGamesMock: vi.fn(),
  listMock: vi.fn(),
}))

vi.mock('./games.service', () => ({
  default: {
    getGames: getGamesMock,
  },
}))

vi.mock('./review-issues.service', () => ({
  default: {
    list: listMock,
  },
}))

import pendingWorkbenchService, { PENDING_WORKBENCH_PAGE_SIZE } from './pending-workbench.service'

describe('pending workbench service', () => {
  beforeEach(() => {
    getGamesMock.mockReset()
    listMock.mockReset()
  })

  it('requests the pending queue with default pagination and loads overrides', async () => {
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
      },
    })
    listMock.mockResolvedValue([{ game_public_id: 'game-1', ignored_details: [] }])

    const result = await pendingWorkbenchService.getSnapshot()

    expect(getGamesMock).toHaveBeenCalledWith({
      query: {
        page: 1,
        limit: PENDING_WORKBENCH_PAGE_SIZE,
        pending: true,
      },
      sort: {
        field: 'updated_at',
        order: 'asc',
      },
    })
    expect(listMock).toHaveBeenCalledWith(['game-1', 'game-2'])
    expect(result).toEqual({
      queueGames: [
        { public_id: 'game-1', title: 'A' },
        { public_id: 'game-2', title: 'B' },
      ],
      overrides: [{ game_public_id: 'game-1', ignored_details: [] }],
      total: 11,
      totalPages: 2,
      page: 1,
      limit: PENDING_WORKBENCH_PAGE_SIZE,
    })
  })
})
