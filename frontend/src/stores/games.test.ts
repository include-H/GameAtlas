import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import type { GameDetail, GameListItem, GameStats } from '@/services/types'

const { setFavoriteMock } = vi.hoisted(() => ({
  setFavoriteMock: vi.fn(),
}))

vi.mock('@/services/games.service', () => ({
  default: {
    setFavorite: setFavoriteMock,
  },
  mapGameVersions: vi.fn(() => []),
}))

import { useGamesStore } from './games'

describe('games store favorite sync', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    setFavoriteMock.mockReset()
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
      popular_games: [
        { id: 1, public_id: 'game-1', title: 'Popular Game', isFavorite: false },
      ],
    } as unknown as GameStats

    const isFavorite = await store.toggleFavorite('game-1')

    expect(isFavorite).toBe(true)
    expect(setFavoriteMock).toHaveBeenCalledWith('game-1', true)
    expect(store.games[0]?.isFavorite).toBe(true)
    expect(store.currentGame?.isFavorite).toBe(true)
    expect(store.stats?.recent_games[0]?.isFavorite).toBe(true)
    expect(store.stats?.popular_games[0]?.isFavorite).toBe(true)
    expect(store.stats?.favorite_count).toBe(1)
  })
})
