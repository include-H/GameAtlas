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

  it('forwards explicit empty reasons instead of normalizing them on the client', async () => {
    putMock.mockResolvedValue({
      data: { game_public_id: 'game-1', issue_key: 'missing-cover', reason: null },
    })

    await reviewIssuesService.ignore('game-1', 'missing-cover', '')

    expect(putMock).toHaveBeenCalledWith('/games/game-1/review-issues/missing-cover/ignore', {
      reason: '',
    })
  })
})
