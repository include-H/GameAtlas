import { beforeEach, describe, expect, it, vi } from 'vitest'

const { getMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
}))

vi.mock('./api', () => ({
  get: getMock,
}))

import hitokotoService from './hitokoto.service'

describe('hitokoto service', () => {
  beforeEach(() => {
    getMock.mockReset()
  })

  it('loads a game sentence with default length bounds', async () => {
    getMock.mockResolvedValue({
      data: {
        id: 1,
        uuid: 'uuid-1',
        hitokoto: '开始游戏',
        type: 'c',
        from: 'Game',
        from_who: null,
        creator: 'creator',
        creator_uid: 1,
        reviewer: 1,
        commit_from: 'web',
        created_at: '2026-01-01T00:00:00Z',
        length: 10,
      },
    })

    const result = await hitokotoService.getGameSentence()

    expect(result.hitokoto).toBe('开始游戏')
    expect(getMock).toHaveBeenCalledWith('/hitokoto', {
      params: {
        c: 'c',
        min_length: 10,
        max_length: 34,
      },
    })
  })

  it('passes custom length bounds through to the api', async () => {
    getMock.mockResolvedValue({
      data: {
        id: 2,
        uuid: 'uuid-2',
        hitokoto: '再来一局',
        type: 'c',
        from: 'Game',
        from_who: null,
        creator: 'creator',
        creator_uid: 1,
        reviewer: 1,
        commit_from: 'web',
        created_at: '2026-01-01T00:00:00Z',
        length: 12,
      },
    })

    await hitokotoService.getGameSentence({
      min_length: 12,
      max_length: 20,
    })

    expect(getMock).toHaveBeenCalledWith('/hitokoto', {
      params: {
        c: 'c',
        min_length: 12,
        max_length: 20,
      },
    })
  })
})
