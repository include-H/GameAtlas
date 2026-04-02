import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import gamesService, { mapGameVersions } from '@/services/games.service'
import type { GameDetail, GameListItem, GameListQuery, GameSort, GameStats, GameVersion } from '@/services/types'
import { getHttpErrorMessage } from '@/utils/http-error'

export const useGamesStore = defineStore('games', () => {
  // State
  const games = ref<GameListItem[]>([])
  const currentGame = ref<GameDetail | null>(null)
  const currentVersions = ref<GameVersion[]>([])
  const stats = ref<GameStats | null>(null)

  const pagination = ref({
    total: 0,
    page: 1,
    limit: 20,
    totalPages: 0,
  })

  const isLoading = ref(false)
  const error = ref<string | null>(null)

  // Computed
  const hasMorePages = computed(() => pagination.value.page < pagination.value.totalPages)
  const totalPages = computed(() => pagination.value.totalPages)

  const applyFavoriteState = (gameId: string, isFavorite: boolean) => {
    const updateGame = (game: { isFavorite?: boolean }) => {
      game.isFavorite = isFavorite
    }

    const sourceGame =
      games.value.find(game => game.public_id === gameId)
      || (currentGame.value && currentGame.value.public_id === gameId ? currentGame.value : null)
      || stats.value?.recent_games.find(game => game.public_id === gameId)
      || stats.value?.popular_games.find(game => game.public_id === gameId)
      || null

    games.value.forEach(game => {
      if (game.public_id === gameId) {
        updateGame(game)
      }
    })

    if (currentGame.value && currentGame.value.public_id === gameId) {
      updateGame(currentGame.value)
    }

    if (!stats.value) {
      return
    }

    stats.value.recent_games.forEach(game => {
      if (game.public_id === gameId) {
        updateGame(game)
      }
    })

    stats.value.popular_games.forEach(game => {
      if (game.public_id === gameId) {
        updateGame(game)
      }
    })

    if (typeof stats.value.favorite_count === 'number') {
      if (isFavorite) {
        stats.value.favorite_count += 1
      } else {
        stats.value.favorite_count = Math.max(0, stats.value.favorite_count - 1)
      }
    } else if (isFavorite && sourceGame) {
      stats.value.favorite_count = 1
    }
  }

  // Actions
  const fetchGames = async (
    params: {
      query?: GameListQuery
      sort?: GameSort
      append?: boolean
    } = {}
  ) => {
    isLoading.value = true
    error.value = null

    const page = params.query?.page ?? 1
    const limit = params.query?.limit ?? pagination.value.limit
    const append = params.append ?? false

    try {
      const response = await gamesService.getGames({
        query: {
          ...params.query,
          page,
          limit,
        },
        sort: params.sort,
      })

      if (append) {
        games.value.push(...response.data)
      } else {
        games.value = response.data
      }

      pagination.value = {
        total: response.pagination.total,
        page: response.pagination.page,
        limit: response.pagination.limit,
        totalPages: response.pagination.totalPages,
      }

      return response
    } catch (err) {
      error.value = getHttpErrorMessage(err, 'Failed to fetch games')
      throw err
    } finally {
      isLoading.value = false
    }
  }

  const fetchGame = async (id: string) => {
    isLoading.value = true
    error.value = null

    try {
      const game = await gamesService.getGame(id)
      currentGame.value = game
      currentVersions.value = mapGameVersions(game)
      return game
    } catch (err) {
      error.value = getHttpErrorMessage(err, 'Failed to fetch game')
      throw err
    } finally {
      isLoading.value = false
    }
  }

  const fetchStats = async () => {
    try {
      stats.value = await gamesService.getStats()
      return stats.value
    } catch (err) {
      error.value = getHttpErrorMessage(err, 'Failed to fetch stats')
      throw err
    }
  }

  const toggleFavorite = async (gameId: string) => {
    try {
      const result = await gamesService.toggleFavorite(gameId)
      applyFavoriteState(gameId, result.isFavorite)

      return result.isFavorite
    } catch (err) {
      error.value = getHttpErrorMessage(err, 'Failed to toggle favorite')
      throw err
    }
  }

  return {
    // State
    games,
    currentGame,
    currentVersions,
    stats,
    pagination,
    isLoading,
    error,
    // Computed
    hasMorePages,
    totalPages,
    // Actions
    fetchGames,
    fetchGame,
    fetchStats,
    toggleFavorite,
  }
})
