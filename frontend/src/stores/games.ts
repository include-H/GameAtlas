import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import gamesService, { mapGameVersions } from '@/services/games.service'
import type { GameDetail, GameListItem, GameListQuery, GameSortQuery, GameStats, GameVersion } from '@/services/types'
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
    limit: 24,
    totalPages: 0,
  })

  // 2026-04-08: games list/detail/stats/favorite flows keep separate request state.
  // Impact: one read/write failure no longer leaks into another surface's loading/error semantics.
  const listLoading = ref(false)
  const detailLoading = ref(false)
  const statsLoading = ref(false)
  const listError = ref<string | null>(null)
  const detailError = ref<string | null>(null)
  const statsError = ref<string | null>(null)
  const favoriteError = ref<string | null>(null)

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

    games.value.forEach((game) => {
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

  const getFavoriteState = (gameId: string) => {
    const sourceGame =
      games.value.find(game => game.public_id === gameId)
      || (currentGame.value && currentGame.value.public_id === gameId ? currentGame.value : null)
      || stats.value?.recent_games.find(game => game.public_id === gameId)
      || stats.value?.popular_games.find(game => game.public_id === gameId)
      || null

    return Boolean(sourceGame?.isFavorite)
  }

  // Actions
  const fetchGames = async (
    params: {
      query?: GameListQuery
      sort?: GameSortQuery
      append?: boolean
    } = {}
  ) => {
    listLoading.value = true
    listError.value = null

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
      listError.value = getHttpErrorMessage(err, '加载游戏列表失败')
      throw err
    } finally {
      listLoading.value = false
    }
  }

  const fetchGame = async (id: string, signal?: AbortSignal) => {
    detailLoading.value = true
    detailError.value = null

    try {
      const game = await gamesService.getGameDetail(id, signal)
      currentGame.value = game
      currentVersions.value = mapGameVersions(game)
      return game
    } catch (err) {
      detailError.value = getHttpErrorMessage(err, '加载游戏详情失败')
      throw err
    } finally {
      detailLoading.value = false
    }
  }

  const fetchStats = async () => {
    statsLoading.value = true
    statsError.value = null
    try {
      stats.value = await gamesService.getStats()
      return stats.value
    } catch (err) {
      statsError.value = getHttpErrorMessage(err, '加载统计数据失败')
      throw err
    } finally {
      statsLoading.value = false
    }
  }

  const toggleFavorite = async (gameId: string) => {
    favoriteError.value = null
    try {
      const result = await gamesService.setFavorite(gameId, !getFavoriteState(gameId))
      applyFavoriteState(gameId, result.isFavorite)

      return result.isFavorite
    } catch (err) {
      favoriteError.value = getHttpErrorMessage(err, '切换收藏失败')
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
    listLoading,
    detailLoading,
    statsLoading,
    listError,
    detailError,
    statsError,
    favoriteError,
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
