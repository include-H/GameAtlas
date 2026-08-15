import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import type { GameDetail, GameListItem, GameStats } from '@/services/types'

const { getGameDetailMock, getGamesMock, getStatsMock } = vi.hoisted(() => ({
  getGameDetailMock: vi.fn(),
  getGamesMock: vi.fn(),
  getStatsMock: vi.fn(),
}))

vi.mock('@/services/games.service', () => ({
  default: {
    getGameDetail: getGameDetailMock,
    getGames: getGamesMock,
    getStats: getStatsMock,
  },
  mapGameVersions: vi.fn(() => []),
}))

import { useGamesStore } from './games'

describe('games store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    getGameDetailMock.mockReset()
    getStatsMock.mockReset()
    getGamesMock.mockReset()
  })

  it('appends games when infinite scroll requests the next page', async () => {
    getGamesMock
      .mockResolvedValueOnce({
        data: [{ id: 1, public_id: 'game-1' } as GameListItem],
        pagination: { page: 1, totalPages: 2 },
      })
      .mockResolvedValueOnce({
        data: [{ id: 2, public_id: 'game-2' } as GameListItem],
        pagination: { page: 2, totalPages: 2 },
      })

    const store = useGamesStore()

    await store.fetchGames({ query: { page: 1, limit: 24 } })
    await store.fetchGames({ query: { page: 2, limit: 24 }, append: true })

    expect(store.games.map((game) => game.public_id)).toEqual(['game-1', 'game-2'])
    expect(store.pagination.page).toBe(2)
    expect(store.hasMorePages).toBe(false)
  })

  it('discards stale list responses and keeps the latest loading state', async () => {
    let resolveFirst: ((value: unknown) => void) | undefined
    let resolveSecond: ((value: unknown) => void) | undefined
    const first = new Promise((resolve) => { resolveFirst = resolve })
    const second = new Promise((resolve) => { resolveSecond = resolve })
    getGamesMock.mockReturnValueOnce(first).mockReturnValueOnce(second)

    const store = useGamesStore()
    const firstRequest = store.fetchGames({ query: { page: 1, limit: 24 } })
    const secondRequest = store.fetchGames({ query: { page: 1, limit: 24 } })

    resolveSecond?.({
      data: [{ id: 2, public_id: 'latest-game' }],
      pagination: { page: 1, totalPages: 1 },
    })
    await secondRequest
    resolveFirst?.({
      data: [{ id: 1, public_id: 'stale-game' }],
      pagination: { page: 1, totalPages: 1 },
    })
    await firstRequest

    expect(store.games.map((game) => game.public_id)).toEqual(['latest-game'])
    expect(store.listLoading).toBe(false)
    expect(store.listError).toBeNull()
  })

  it('keeps stats failures out of the list error slot', async () => {
    getStatsMock.mockRejectedValue(new Error('stats failed'))

    const store = useGamesStore()

    await expect(store.fetchStats()).rejects.toThrow('stats failed')

    expect(store.statsError).toBe('stats failed')
    expect(store.listError).toBeNull()
  })

  it('discards stale detail and stats responses', async () => {
    let resolveDetailFirst: ((value: unknown) => void) | undefined
    let resolveDetailSecond: ((value: unknown) => void) | undefined
    let resolveStatsFirst: ((value: unknown) => void) | undefined
    let resolveStatsSecond: ((value: unknown) => void) | undefined
    getGameDetailMock
      .mockReturnValueOnce(new Promise((resolve) => { resolveDetailFirst = resolve }))
      .mockReturnValueOnce(new Promise((resolve) => { resolveDetailSecond = resolve }))
    getStatsMock
      .mockReturnValueOnce(new Promise((resolve) => { resolveStatsFirst = resolve }))
      .mockReturnValueOnce(new Promise((resolve) => { resolveStatsSecond = resolve }))

    const store = useGamesStore()
    const firstDetail = store.fetchGame('stale-game')
    const secondDetail = store.fetchGame('latest-game')
    const firstStats = store.fetchStats()
    const secondStats = store.fetchStats()
    const latestDetail = { id: 2, public_id: 'latest-game', title: 'Latest', files: [] } as unknown as GameDetail
    const staleDetail = { id: 1, public_id: 'stale-game', title: 'Stale', files: [] } as unknown as GameDetail
    const latestStats = { total_games: 2 } as unknown as GameStats
    const staleStats = { total_games: 1 } as unknown as GameStats

    resolveDetailSecond?.(latestDetail)
    resolveStatsSecond?.(latestStats)
    await Promise.all([secondDetail, secondStats])
    resolveDetailFirst?.(staleDetail)
    resolveStatsFirst?.(staleStats)
    await Promise.all([firstDetail, firstStats])

    expect(store.currentGame?.public_id).toBe('latest-game')
    expect(store.stats).toEqual(latestStats)
    expect(store.detailLoading).toBe(false)
    expect(store.statsLoading).toBe(false)
  })
})

describe('games store aggregate list item sync', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    getGameDetailMock.mockReset()
    getStatsMock.mockReset()
    getGamesMock.mockReset()
  })

  it('replaces the matching list item in place with the aggregate save result', () => {
    const store = useGamesStore()
    store.games = [
      { id: 1, public_id: 'game-1', title: 'Old Title', cover_image: null },
      { id: 2, public_id: 'game-2', title: 'Untouched', cover_image: null },
    ] as unknown as GameListItem[]

    store.applyAggregateListItem({
      id: 1,
      public_id: 'game-1',
      title: 'New Title',
      cover_image: '/assets/cover.jpg',
    } as unknown as GameListItem)

    expect(store.games).toHaveLength(2)
    expect(store.games[0]?.title).toBe('New Title')
    expect(store.games[0]?.cover_image).toBe('/assets/cover.jpg')
    expect(store.games[1]?.title).toBe('Untouched')
  })

  it('keeps the list untouched when the saved game is not present', () => {
    const store = useGamesStore()
    store.games = [
      { id: 2, public_id: 'game-2', title: 'Untouched' },
    ] as unknown as GameListItem[]

    store.applyAggregateListItem({
      id: 99,
      public_id: 'missing-game',
      title: 'Nope',
    } as unknown as GameListItem)

    expect(store.games).toHaveLength(1)
    expect(store.games[0]?.title).toBe('Untouched')
  })
})
