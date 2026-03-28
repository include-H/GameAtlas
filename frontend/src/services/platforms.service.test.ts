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

  it('loads all platforms from the api envelope', async () => {
    getMock.mockResolvedValue({
      data: [{ id: 1, name: 'PC' }],
    })

    await expect(platformService.getAllPlatforms()).resolves.toEqual([{ id: 1, name: 'PC' }])
    expect(getMock).toHaveBeenCalledWith('/platforms')
  })

  it('searches platforms case-insensitively and respects the limit', async () => {
    getMock.mockResolvedValue({
      data: [
        { id: 1, name: 'PC' },
        { id: 2, name: 'PC-98' },
        { id: 3, name: 'Switch' },
      ],
    })

    await expect(platformService.searchPlatforms('pc', 1)).resolves.toEqual([
      { id: 1, name: 'PC' },
    ])
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
