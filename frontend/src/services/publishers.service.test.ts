import { beforeEach, describe, expect, it, vi } from 'vitest'

const { getMock, postMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
  postMock: vi.fn(),
}))

vi.mock('./api', () => ({
  get: getMock,
  post: postMock,
}))

import { publishersService } from './publishers.service'

describe('publishers service', () => {
  beforeEach(() => {
    getMock.mockReset()
    postMock.mockReset()
  })

  it('passes search and limit to the api', async () => {
    getMock.mockResolvedValue({
      data: [{ id: 1, name: 'SEGA' }],
    })

    await expect(publishersService.listPublishers({ query: ' sega ', limit: 1 })).resolves.toEqual([
      { id: 1, name: 'SEGA' },
    ])

    expect(getMock).toHaveBeenCalledWith('/publishers', {
      params: expect.any(URLSearchParams),
    })
    const [, config] = getMock.mock.calls[0]
    expect((config.params as URLSearchParams).toString()).toBe('search=sega&limit=1')
  })

  it('creates a publisher via post', async () => {
    postMock.mockResolvedValue({
      data: { id: 7, name: 'Atlus' },
    })

    await expect(
      publishersService.createPublisher({
        name: 'Atlus',
      }),
    ).resolves.toEqual({ id: 7, name: 'Atlus' })

    expect(postMock).toHaveBeenCalledWith('/publishers', { name: 'Atlus' })
  })
})
