import { beforeEach, describe, expect, it, vi } from 'vitest'

const { delMock, getMock, putMock } = vi.hoisted(() => ({
  delMock: vi.fn(),
  getMock: vi.fn(),
  putMock: vi.fn(),
}))

vi.mock('./api', () => ({
  del: delMock,
  get: getMock,
  put: putMock,
}))

import reviewIssuesService from './review-issues.service'

describe('review issues service', () => {
  beforeEach(() => {
    delMock.mockReset()
    getMock.mockReset()
    putMock.mockReset()
  })

  it('lists overrides with joined game ids when provided', async () => {
    getMock.mockResolvedValue({
      data: [{ game_public_id: 'game-1', issue_key: 'missing-cover' }],
    })

    await expect(reviewIssuesService.list(['game-1', 2])).resolves.toEqual([
      { game_public_id: 'game-1', issue_key: 'missing-cover' },
    ])
    expect(getMock).toHaveBeenCalledWith('/review-issue-overrides', {
      params: { game_ids: 'game-1,2' },
    })
  })

  it('lists overrides without params when no ids are provided', async () => {
    getMock.mockResolvedValue({ data: [] })

    await expect(reviewIssuesService.list()).resolves.toEqual([])
    expect(getMock).toHaveBeenCalledWith('/review-issue-overrides', { params: undefined })
  })

  it('ignores and restores review issues through the expected endpoints', async () => {
    putMock.mockResolvedValue({
      data: { game_public_id: 'game-1', issue_key: 'missing-cover', reason: 'done' },
    })
    delMock.mockResolvedValue({})

    await expect(reviewIssuesService.ignore('game-1', 'missing-cover', 'done')).resolves.toEqual({
      game_public_id: 'game-1',
      issue_key: 'missing-cover',
      reason: 'done',
    })
    expect(putMock).toHaveBeenCalledWith('/games/game-1/review-issues/missing-cover/ignore', {
      reason: 'done',
    })

    await reviewIssuesService.restore('game-1', 'missing-cover')
    expect(delMock).toHaveBeenCalledWith('/games/game-1/review-issues/missing-cover/ignore')
  })
})
