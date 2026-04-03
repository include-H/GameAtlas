import { beforeEach, describe, expect, it, vi } from 'vitest'

const { getMock, postMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
  postMock: vi.fn(),
}))

vi.mock('./api', () => ({
  get: getMock,
  post: postMock,
}))

import { developersService } from './developers.service'

describe('developers service', () => {
  beforeEach(() => {
    getMock.mockReset()
    postMock.mockReset()
  })

  it('passes search and limit to the api', async () => {
    getMock.mockResolvedValue({
      data: [{ id: 1, name: 'Capcom' }],
    })

    await expect(developersService.listDevelopers({ query: ' cap ', limit: 1 })).resolves.toEqual([
      { id: 1, name: 'Capcom' },
    ])

    expect(getMock).toHaveBeenCalledWith('/developers', {
      params: expect.any(URLSearchParams),
    })
    const [, config] = getMock.mock.calls[0]
    expect((config.params as URLSearchParams).toString()).toBe('search=cap&limit=1')
  })

  it('creates a developer via post', async () => {
    postMock.mockResolvedValue({
      data: { id: 7, name: 'Ryu Ga Gotoku Studio' },
    })

    await expect(
      developersService.createDeveloper({
        name: 'Ryu Ga Gotoku Studio',
      }),
    ).resolves.toEqual({ id: 7, name: 'Ryu Ga Gotoku Studio' })

    expect(postMock).toHaveBeenCalledWith('/developers', { name: 'Ryu Ga Gotoku Studio' })
  })
})
