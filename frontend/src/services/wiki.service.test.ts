import { beforeEach, describe, expect, it, vi } from 'vitest'

const { getMock, putMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
  putMock: vi.fn(),
}))

vi.mock('./api', () => ({
  get: getMock,
  put: putMock,
}))

import wikiService from './wiki.service'

describe('wiki service', () => {
  beforeEach(() => {
    getMock.mockReset()
    putMock.mockReset()
  })

  it('loads wiki history without sending an unused limit parameter', async () => {
    getMock.mockResolvedValue({
      data: [{ id: 1, content: 'v1', created_at: '2026-04-06T00:00:00Z' }],
    })

    const result = await wikiService.getWikiHistory('game-1')

    expect(result).toEqual([{ id: 1, content: 'v1', created_at: '2026-04-06T00:00:00Z' }])
    expect(getMock).toHaveBeenCalledWith('/games/game-1/wiki/history')
  })
})
