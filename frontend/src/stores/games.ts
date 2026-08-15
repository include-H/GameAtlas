import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import gamesService, { mapGameVersions } from '@/services/games.service'
import type { GameDetail, GameListItem, GameListQuery, GameSortQuery, GameStats, GameVersion } from '@/services/types'
import { getHttpErrorMessage } from '@/utils/http-error'
import { createRequestGeneration } from '@/utils/request-generation'

export const useGamesStore = defineStore('games', () => {
  // State
  const games = ref<GameListItem[]>([])
  const currentGame = ref<GameDetail | null>(null)
  const currentVersions = ref<GameVersion[]>([])
  const stats = ref<GameStats | null>(null)

  // 2026-08-06: /games list mode no longer returns total/limit (infinite-scroll
  // contract), so the store keeps only page/totalPages.
  const pagination = ref({
    page: 1,
    totalPages: 0,
  })

  // 2026-04-08: games list/detail/stats flows keep separate request state.
  // Impact: one read/write failure no longer leaks into another surface's loading/error semantics.
  const listLoading = ref(false)
  const detailLoading = ref(false)
  const statsLoading = ref(false)
  const listError = ref<string | null>(null)
  const detailError = ref<string | null>(null)
  const statsError = ref<string | null>(null)

  const listRequests = createRequestGeneration()
  const detailRequests = createRequestGeneration()
  const statsRequests = createRequestGeneration()

  // Computed
  const hasMorePages = computed(() => pagination.value.page < pagination.value.totalPages)
  const totalPages = computed(() => pagination.value.totalPages)

  // Actions
  const fetchGames = async (
    params: {
      query?: GameListQuery
      sort?: GameSortQuery
      append?: boolean
    } = {}
  ) => {
    const request = listRequests.begin()
    listLoading.value = true
    listError.value = null

    const page = params.query?.page ?? 1
    const limit = params.query?.limit ?? 24
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

      if (!request.isCurrent()) {
        return response
      }

      if (append) {
        games.value.push(...response.data)
      } else {
        games.value = response.data
      }

      pagination.value = {
        page: response.pagination.page,
        totalPages: response.pagination.totalPages,
      }

      return response
    } catch (err) {
      if (request.isCurrent()) {
        listError.value = getHttpErrorMessage(err, '加载游戏列表失败')
      }
      throw err
    } finally {
      if (request.isCurrent()) {
        listLoading.value = false
      }
    }
  }

  const fetchGame = async (id: string, signal?: AbortSignal) => {
    const request = detailRequests.begin()
    detailLoading.value = true
    detailError.value = null

    try {
      const game = await gamesService.getGameDetail(id, signal)
      if (request.isCurrent()) {
        currentGame.value = game
        currentVersions.value = mapGameVersions(game)
      }
      return game
    } catch (err) {
      if (request.isCurrent()) {
        detailError.value = getHttpErrorMessage(err, '加载游戏详情失败')
      }
      throw err
    } finally {
      if (request.isCurrent()) {
        detailLoading.value = false
      }
    }
  }

  const fetchStats = async () => {
    const request = statsRequests.begin()
    statsLoading.value = true
    statsError.value = null
    try {
      const nextStats = await gamesService.getStats()
      if (request.isCurrent()) {
        stats.value = nextStats
      }
      return stats.value
    } catch (err) {
      if (request.isCurrent()) {
        statsError.value = getHttpErrorMessage(err, '加载统计数据失败')
      }
      throw err
    } finally {
      if (request.isCurrent()) {
        statsLoading.value = false
      }
    }
  }

  // 2026-08-08: 编辑保存（updateGameAggregate）响应里已携带最新状态的 GameListItem，
  // 原地替换列表项即可让 keep-alive 恢复后的游戏库立即显示新素材状态，无需重拉列表。
  // 数组长度不变 → 虚拟滚动 canvas 高度与滚动位置不受影响。
  const applyAggregateListItem = (item: GameListItem) => {
    const index = games.value.findIndex((game) => game.public_id === item.public_id)
    if (index >= 0) {
      games.value[index] = item
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
    // Computed
    hasMorePages,
    totalPages,
    // Actions
    fetchGames,
    fetchGame,
    fetchStats,
    applyAggregateListItem,
  }
})
