import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import gamesService from '@/services/games.service'
import { getWebSocketService, type WebSocketEvent } from '@/services/websocket'
import type { Game, GameVersion, GameFilter, GameSort, GameStats } from '@/services/types'
import { getHttpErrorMessage } from '@/utils/http-error'

export const useGamesStore = defineStore('games', () => {
  // State
  const games = ref<Game[]>([])
  const currentGame = ref<Game | null>(null)
  const currentVersions = ref<GameVersion[]>([])
  const stats = ref<GameStats | null>(null)

  const pagination = ref({
    total: 0,
    page: 1,
    pageSize: 20,
    totalPages: 0,
  })

  const isLoading = ref(false)
  const error = ref<string | null>(null)
  let lastListQuery: { page: number; pageSize: number; filter?: GameFilter; sort?: GameSort } | null = null

  // Computed
  const hasMorePages = computed(() => pagination.value.page < pagination.value.totalPages)
  const totalPages = computed(() => pagination.value.totalPages)

  const applyFavoriteState = (gameId: string, isFavorite: boolean) => {
    const updateGame = (game: Game) => {
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
      page?: number
      pageSize?: number
      filter?: GameFilter
      sort?: GameSort
      append?: boolean
    } = {}
  ) => {
    isLoading.value = true
    error.value = null

    const page = params.page ?? 1
    const pageSize = params.pageSize ?? pagination.value.pageSize
    const append = params.append ?? false

    lastListQuery = {
      page,
      pageSize,
      filter: params.filter,
      sort: params.sort,
    }

    try {
      const response = await gamesService.getGames({
        page,
        pageSize,
        filter: params.filter,
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
        pageSize: response.pagination.limit,
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
      return game
    } catch (err) {
      error.value = getHttpErrorMessage(err, 'Failed to fetch game')
      throw err
    } finally {
      isLoading.value = false
    }
  }

  const fetchGameVersions = async (gameId: string) => {
    try {
      const versions = await gamesService.getGameVersions(gameId)
      currentVersions.value = versions
      return versions
    } catch (err) {
      error.value = getHttpErrorMessage(err, 'Failed to fetch versions')
      throw err
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

  // 初始化 WebSocket 监听
  let unsubscribeWebSocket: (() => void) | null = null
  
  const initializeWebSocket = () => {
    if (unsubscribeWebSocket) return // 避免重复初始化

    const wsService = getWebSocketService()

    unsubscribeWebSocket = wsService.subscribe((event: WebSocketEvent) => {
      switch (event.type) {
        case 'game:created':
        case 'game:updated':
        case 'game:deleted':
          // 自动刷新游戏列表
          if (lastListQuery) {
            fetchGames(lastListQuery)
          }
          break
        case 'game:wiki_updated':
          // 如果当前正在查看该游戏，刷新详情
          if (currentGame.value && event.gameId === currentGame.value.id) {
            if (currentGame.value.public_id) {
              fetchGame(currentGame.value.public_id)
            }
          }
          break
      }
    })
    
    // 连接 WebSocket
    wsService.connect()
  }

  // 取消订阅 WebSocket
  const unsubscribeWebSocketEvents = () => {
    if (unsubscribeWebSocket) {
      unsubscribeWebSocket()
      unsubscribeWebSocket = null
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
    fetchGameVersions,
    fetchStats,
    toggleFavorite,
    initializeWebSocket,
    unsubscribeWebSocketEvents,
  }
})
