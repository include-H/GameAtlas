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

  it('passes search, limit, and sort to the api', async () => {
    getMock.mockResolvedValue({
      data: [{ id: 1, name: 'SEGA' }],
    })

    await expect(publishersService.listPublishers({ query: ' sega ', limit: 1, sort: 'popular' })).resolves.toEqual([
      { id: 1, name: 'SEGA' },
    ])

    expect(getMock).toHaveBeenCalledWith('/publishers', {
      params: expect.any(URLSearchParams),
    })
    const [, config] = getMock.mock.calls[0]
    expect((config.params as URLSearchParams).toString()).toBe('search=sega&limit=1&sort=popular')
  })

  it('loads a publisher page with page and limit', async () => {
    getMock.mockResolvedValue({
      data: [{ id: 1, name: 'SEGA' }],
      pagination: {
        page: 2,
        limit: 24,
        total: 25,
        totalPages: 2,
      },
    })

    await expect(
      publishersService.getPublishersPage({
        page: 2,
        limit: 24,
        search: ' sega ',
        sort: 'name',
      }),
    ).resolves.toEqual({
      data: [{ id: 1, name: 'SEGA' }],
      pagination: {
        page: 2,
        limit: 24,
        total: 25,
        totalPages: 2,
      },
    })

    const [, config] = getMock.mock.calls[0]
    expect((config.params as URLSearchParams).toString()).toBe('page=2&limit=24&search=sega&sort=name')
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

  it('normalizes publisher games without retaining the api favorite field', async () => {
    getMock.mockResolvedValue({
      data: {
        publisher: { id: 7, name: 'Atlus' },
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

    const detail = await publishersService.getPublisherDetail(7, { page: 1, limit: 24 })

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
