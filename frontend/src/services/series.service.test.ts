import { beforeEach, describe, expect, it, vi } from 'vitest'

const { getMock, postMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
  postMock: vi.fn(),
}))

vi.mock('./api', () => ({
  get: getMock,
  post: postMock,
}))

import { seriesService } from './series.service'

describe('series service', () => {
  beforeEach(() => {
    getMock.mockReset()
    postMock.mockReset()
  })

  it('trims search text before building metadata query params', async () => {
    getMock.mockResolvedValue({
      data: [{ id: 1, name: 'Persona' }],
    })

    await expect(seriesService.searchSeries(' persona ', 5)).resolves.toEqual([
      { id: 1, name: 'Persona' },
    ])

    expect(getMock).toHaveBeenCalledWith('/series', {
      params: expect.any(URLSearchParams),
    })
    const [, config] = getMock.mock.calls[0]
    expect((config.params as URLSearchParams).toString()).toBe('search=persona&limit=5&sort=popular')
  })

  it('omits blank search text instead of sending a fake search filter', async () => {
    getMock.mockResolvedValue({
      data: [{ id: 2, name: 'Final Fantasy' }],
      pagination: {
        page: 1,
        limit: 24,
        total: 1,
        totalPages: 1,
      },
    })

    await expect(seriesService.getSeriesPage({ page: 1, limit: 24, search: '   ', sort: 'name' })).resolves.toEqual({
      data: [{ id: 2, name: 'Final Fantasy' }],
      pagination: {
        page: 1,
        limit: 24,
        total: 1,
        totalPages: 1,
      },
    })

    const [, config] = getMock.mock.calls[0]
    expect((config.params as URLSearchParams).toString()).toBe('page=1&limit=24&sort=name')
  })
})
