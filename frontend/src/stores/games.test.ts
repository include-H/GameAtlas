import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import type { GameDetail, GameListItem, GameStats } from '@/services/types'

const { getGamesMock, getStatsMock, setFavoriteMock } = vi.hoisted(() => ({
  getGamesMock: vi.fn(),
  getStatsMock: vi.fn(),
  setFavoriteMock: vi.fn(),
}))

vi.mock('@/services/games.service', () => ({
  default: {
    getGames: getGamesMock,
    getStats: getStatsMock,
    setFavorite: setFavoriteMock,
  },
  mapGameVersions: vi.fn(() => []),
}))

import { useGamesStore } from './games'

describe('games store favorite sync', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    getStatsMock.mockReset()
    setFavoriteMock.mockReset()
    getGamesMock.mockReset()
  })

  it('appends games when infinite scroll requests the next page', async () => {
    getGamesMock
      .mockResolvedValueOnce({
        data: [{ id: 1, public_id: 'game-1', is_favorite: false }],
        pagination: { page: 1, limit: 24, total: 2, totalPages: 2 },
      })
      .mockResolvedValueOnce({
        data: [{ id: 2, public_id: 'game-2', is_favorite: false }],
        pagination: { page: 2, limit: 24, total: 2, totalPages: 2 },
      })

    const store = useGamesStore()

    await store.fetchGames({ query: { page: 1, limit: 24 } })
    await store.fetchGames({ query: { page: 2, limit: 24 }, append: true })

    expect(store.games.map((game) => game.public_id)).toEqual(['game-1', 'game-2'])
    expect(store.pagination.page).toBe(2)
    expect(store.hasMorePages).toBe(false)
  })

  it('syncs favorite state across store-managed surfaces', async () => {
    setFavoriteMock.mockResolvedValue({ isFavorite: true })

    const store = useGamesStore()

    store.games = [
      { id: 1, public_id: 'game-1', title: 'List Game', isFavorite: false },
    ] as unknown as GameListItem[]
    store.currentGame = {
      id: 1,
      public_id: 'game-1',
      title: 'Detail Game',
      isFavorite: false,
      files: [],
    } as unknown as GameDetail
    store.stats = {
      total_games: 1,
      total_downloads: 10,
      favorite_count: 0,
      pending_reviews: 0,
      recent_games: [
        { id: 1, public_id: 'game-1', title: 'Recent Game', isFavorite: false },
      ],
      recently_updated_games: [
        { id: 1, public_id: 'game-1', title: 'Updated Game', isFavorite: false },
      ],
      popular_games: [
        { id: 1, public_id: 'game-1', title: 'Popular Game', isFavorite: false },
      ],
      favorite_games: [],
      pending_issue_counts: {
        groups: {},
        ignored_total: 0,
      },
    } as unknown as GameStats

    const isFavorite = await store.toggleFavorite('game-1')

    expect(isFavorite).toBe(true)
    expect(setFavoriteMock).toHaveBeenCalledWith('game-1', true)
    expect(store.games[0]?.isFavorite).toBe(true)
    expect(store.currentGame?.isFavorite).toBe(true)
    expect(store.stats?.recent_games[0]?.isFavorite).toBe(true)
    expect(store.stats?.recently_updated_games[0]?.isFavorite).toBe(true)
    expect(store.stats?.popular_games[0]?.isFavorite).toBe(true)
    expect(store.stats?.favorite_games[0]?.public_id).toBe('game-1')
    expect(store.stats?.favorite_games[0]?.isFavorite).toBe(true)
    expect(store.stats?.favorite_count).toBe(1)
  })

  it('keeps stats failures out of the list error slot', async () => {
    getStatsMock.mockRejectedValue(new Error('stats failed'))

    const store = useGamesStore()

    await expect(store.fetchStats()).rejects.toThrow('stats failed')

    expect(store.statsError).toBe('stats failed')
    expect(store.listError).toBeNull()
  })
})
