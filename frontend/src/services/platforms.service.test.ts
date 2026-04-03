import { beforeEach, describe, expect, it, vi } from 'vitest'

const { getMock, postMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
  postMock: vi.fn(),
}))

vi.mock('./api', () => ({
  get: getMock,
  post: postMock,
}))

import platformService from './platforms.service'

describe('platform service', () => {
  beforeEach(() => {
    getMock.mockReset()
    postMock.mockReset()
  })

  it('lists all platforms from the api envelope', async () => {
    getMock.mockResolvedValue({
      data: [{ id: 1, name: 'PC' }],
    })

    await expect(platformService.listPlatforms()).resolves.toEqual([{ id: 1, name: 'PC' }])
    expect(getMock).toHaveBeenCalledWith('/platforms', {
      params: expect.any(URLSearchParams),
    })
  })

  it('passes search and limit to the api', async () => {
    getMock.mockResolvedValue({
      data: [{ id: 1, name: 'PC' }],
    })

    await expect(platformService.listPlatforms({ query: ' pc ', limit: 1 })).resolves.toEqual([{ id: 1, name: 'PC' }])

    expect(getMock).toHaveBeenCalledWith('/platforms', {
      params: expect.any(URLSearchParams),
    })
    const [, config] = getMock.mock.calls[0]
    expect((config.params as URLSearchParams).toString()).toBe('search=pc&limit=1')
  })

  it('creates a platform via post', async () => {
    postMock.mockResolvedValue({
      data: { id: 7, name: 'Steam Deck' },
    })

    await expect(
      platformService.createPlatform({
        name: 'Steam Deck',
      }),
    ).resolves.toEqual({ id: 7, name: 'Steam Deck' })

    expect(postMock).toHaveBeenCalledWith('/platforms', { name: 'Steam Deck' })
  })
})
