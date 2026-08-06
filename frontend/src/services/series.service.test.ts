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

  it('normalizes series games without retaining the API favorite field', async () => {
    getMock.mockResolvedValue({
      data: {
        series: { id: 7, name: 'Persona' },
        games: [{
          id: 42,
          public_id: 'persona-5',
          title: 'Persona 5',
          is_favorite: true,
        }],
        pagination: {
          page: 1,
          limit: 24,
          total: 1,
          totalPages: 1,
        },
      },
    })

    const detail = await seriesService.getSeriesDetail(7, { page: 1, limit: 24 })

    expect(detail.games[0]).toMatchObject({
      id: 42,
      public_id: 'persona-5',
      isFavorite: true,
    })
    expect(detail.games[0]).not.toHaveProperty('is_favorite')
    expect(detail.pagination).toEqual({
      page: 1,
      limit: 24,
      total: 1,
      totalPages: 1,
    })
  })
})
