import { computed, onBeforeUnmount, onMounted, ref, watch, type Ref } from 'vue'
import type {
  LocationQuery,
  LocationQueryRaw,
  LocationQueryValue,
  RouteLocationNormalizedLoaded,
  Router,
} from 'vue-router'
import { Modal } from '@arco-design/web-vue'
import { getHttpErrorMessage } from '@/utils/http-error'
import gamesService from '@/services/games.service'
import type { GameListQuery, GameSort, GameSortQuery } from '@/services/types'
import { useGamesStore } from '@/stores/games'
import { useUiStore } from '@/stores/ui'

type GamesViewMode = 'grid' | 'list'
type GamesSortField = Exclude<GameSort['field'], 'pending_issue_count'>
type GamesSortOptionValue =
  | 'updated_at:desc'
  | 'updated_at:asc'
  | 'created_at:desc'
  | 'created_at:asc'
  | 'title:asc'
  | 'title:desc'
  | 'release_date:desc'
  | 'release_date:asc'
  | 'downloads:desc'
  | 'random:desc'

interface UseGamesViewOptions {
  route: RouteLocationNormalizedLoaded
  router: Router
  gamesStore: ReturnType<typeof useGamesStore>
  uiStore: ReturnType<typeof useUiStore>
  isAdmin: Ref<boolean>
  loadMoreSentinel: Ref<HTMLElement | null>
}

interface BuildGamesListRequestOptions {
  routeQuery: LocationQuery
}

// 2026-04-04: keep this UI-only default aligned with the backend list default sort/order.
// Impact: the select shows "最近更新" when route.query omits both fields, but requests still rely
// on the backend native default instead of forcing route-owned sort/order values.
const DEFAULT_SORT = { field: 'updated_at', order: 'desc' } as const satisfies Pick<GameSort, 'field' | 'order'>
const GAMES_PAGE_SIZE = 24

const SORT_VALUES = new Set<GamesSortOptionValue>([
  'updated_at:desc',
  'updated_at:asc',
  'created_at:desc',
  'created_at:asc',
  'title:asc',
  'title:desc',
  'release_date:asc',
  'release_date:desc',
  'downloads:desc',
  'random:desc',
])

const SORT_FIELDS = new Set<GamesSortField>(['updated_at', 'created_at', 'title', 'release_date', 'downloads', 'random'])
const SORT_ORDERS = new Set<GameSort['order']>(['asc', 'desc'])

export const readSingleQueryValue = (
  value: LocationQueryValue | LocationQueryValue[] | undefined,
): string | undefined => {
  if (Array.isArray(value)) {
    return value.find((item): item is string => typeof item === 'string' && item.length > 0)
  }
  return typeof value === 'string' && value.length > 0 ? value : undefined
}

const normalizeCommittedSearchValue = (value: string | undefined): string | undefined => {
  const normalized = value?.trim()
  return normalized ? normalized : undefined
}

const parsePositiveRouteNumber = (
  value: LocationQueryValue | LocationQueryValue[] | undefined,
): number | undefined => {
  const parsed = Number(readSingleQueryValue(value))
  return Number.isInteger(parsed) && parsed > 0 ? parsed : undefined
}

const parseRouteBoolean = (
  value: LocationQueryValue | LocationQueryValue[] | undefined,
): boolean | undefined => {
  const raw = readSingleQueryValue(value)
  if (raw === 'true') return true
  if (raw === 'false') return false
  return undefined
}

export const parseGamesSortField = (value: string | undefined): GamesSortField | undefined => {
  return value && SORT_FIELDS.has(value as GamesSortField) ? value as GamesSortField : undefined
}

export const parseGamesSortOrder = (value: string | undefined): GameSort['order'] | undefined => {
  return value && SORT_ORDERS.has(value as GameSort['order']) ? value as GameSort['order'] : undefined
}

export const buildGamesSortOptionValue = (
  field: GamesSortField,
  order: GameSort['order'],
): GamesSortOptionValue | undefined => {
  const value = `${field}:${order}` as GamesSortOptionValue
  return SORT_VALUES.has(value) ? value : undefined
}

export const buildGamesRouteQuery = (
  currentQuery: LocationQuery,
  newParams: LocationQueryRaw,
): LocationQueryRaw => {
  const query: LocationQueryRaw = { ...currentQuery, ...newParams }

  Object.keys(query).forEach((key) => {
    if (query[key] === undefined || query[key] === null || query[key] === '') {
      delete query[key]
    }
  })

  // Infinite scroll owns page/limit in the store; route URLs only carry filters and sort state.
  delete query.page
  delete query.limit

  return query
}

export const buildGamesListRequest = ({
  routeQuery,
}: BuildGamesListRequestOptions): { query: GameListQuery; sort?: GameSortQuery } => {
  const rawSort = readSingleQueryValue(routeQuery.sort)
  const rawOrder = readSingleQueryValue(routeQuery.order)
  const sortField = parseGamesSortField(rawSort)
  const rawFavorite = readSingleQueryValue(routeQuery.favorite)
  const favorite = parseRouteBoolean(routeQuery.favorite)
  const rawVisibility = readSingleQueryValue(routeQuery.visibility)
  const visibility = rawVisibility === 'private' ? 'private' as const : rawVisibility === 'public' ? 'public' as const : undefined

  const request: { query: GameListQuery; sort?: GameSortQuery } = {
    query: {
      page: 1,
      limit: GAMES_PAGE_SIZE,
      search: normalizeCommittedSearchValue(readSingleQueryValue(routeQuery.search)),
      visibility,
      favorite,
    },
  }

  if (rawFavorite !== undefined && favorite === undefined) {
    request.query.favorite_raw = rawFavorite
  }

  // 2026-05-01: forward route-owned sort/order values verbatim whenever they are present.
  // GamesView may interpret only a supported subset for local UI state, but it must not
  // silently coerce malformed sort/order query params back to backend defaults.
  if (rawSort !== undefined || rawOrder !== undefined) {
    request.sort = {
      field: rawSort,
      order: rawOrder,
      seed: sortField === 'random'
        ? parsePositiveRouteNumber(routeQuery.seed)
        : undefined,
    }
  }

  return request
}

export const hasGamesActiveFilters = (routeQuery: LocationQuery): boolean => {
  return Boolean(
    normalizeCommittedSearchValue(readSingleQueryValue(routeQuery.search))
    || parseRouteBoolean(routeQuery.favorite) === true
    || readSingleQueryValue(routeQuery.visibility) === 'private',
  )
}

export const normalizeGamesFavoriteRouteQuery = (routeQuery: LocationQuery): LocationQueryRaw | null => {
  const rawFavorite = readSingleQueryValue(routeQuery.favorite)
  if (rawFavorite === undefined) {
    return null
  }
  // 2026-05-01: keep malformed favorite values in the route so shared URLs and bad links
  // fail at the backend transport boundary. GamesView only treats favorite=true as an active
  // UI filter, but it must not silently rewrite favorite=false or arbitrary strings away.
  return null
}

export const normalizeGamesSortRouteQuery = (routeQuery: LocationQuery): LocationQueryRaw | null => {
  const rawSort = readSingleQueryValue(routeQuery.sort)
  const rawOrder = readSingleQueryValue(routeQuery.order)
  const sortField = parseGamesSortField(rawSort)
  const sortOrder = parseGamesSortOrder(rawOrder)

  if (!rawSort && !rawOrder) return null

  // 2026-05-01: keep malformed sort/order in the route so the backend transport decoder can reject them.
  // This avoids client-side silent correction masking invalid shared URLs or bad internal links.
  if (!rawSort || !sortField) {
    return null
  }

  if (rawOrder !== undefined && !sortOrder) {
    return null
  }

  if (sortField !== 'random') {
    if (readSingleQueryValue(routeQuery.seed) === undefined) {
      return null
    }
    return buildGamesRouteQuery(routeQuery, {
      seed: undefined,
    })
  }

  if (parsePositiveRouteNumber(routeQuery.seed) !== undefined) {
    return null
  }

  // 2026-04-04: keep the random seed in route state because backend pagination only stays stable
  // when every page request reuses the same native random seed. Impact: the URL owns that state,
  // while request building no longer invents hidden fallback seeds.
  return buildGamesRouteQuery(routeQuery, {
    seed: String(Date.now()),
  })
}

export const normalizeGamesPaginationRouteQuery = (routeQuery: LocationQuery): LocationQueryRaw | null => {
  const rawPage = readSingleQueryValue(routeQuery.page)
  const rawLimit = readSingleQueryValue(routeQuery.limit)

  if (rawPage === undefined && rawLimit === undefined) {
    return null
  }

  // 2026-08-06: games list uses infinite scroll with a fixed page size, so page/limit
  // are owned by the client instead of the URL. Impact: shared links always start from
  // the first 24 games and no longer depend on stale pagination params.
  return buildGamesRouteQuery(routeQuery, {
    page: undefined,
    limit: undefined,
  })
}

const normalizeRouteQueryValue = (
  value: LocationQueryValue | LocationQueryValue[] | undefined,
): string[] => {
  if (Array.isArray(value)) {
    return value.map((item) => String(item))
  }
  if (value === undefined) {
    return []
  }
  return [String(value)]
}

const isSameRouteQuery = (left: LocationQuery, right: LocationQueryRaw): boolean => {
  const leftKeys = Object.keys(left).sort()
  const rightKeys = Object.keys(right).sort()
  if (leftKeys.length != rightKeys.length) {
    return false
  }

  for (let index = 0; index < leftKeys.length; index += 1) {
    if (leftKeys[index] !== rightKeys[index]) {
      return false
    }
  }

  for (const key of leftKeys) {
    const leftValue = normalizeRouteQueryValue(left[key])
    const rightValue = normalizeRouteQueryValue(right[key] as LocationQueryValue | LocationQueryValue[] | undefined)
    if (leftValue.length !== rightValue.length) {
      return false
    }
    for (let index = 0; index < leftValue.length; index += 1) {
      if (leftValue[index] !== rightValue[index]) {
        return false
      }
    }
  }

  return true
}

export const useGamesView = ({
  route,
  router,
  gamesStore,
  uiStore,
  isAdmin,
  loadMoreSentinel,
}: UseGamesViewOptions) => {
  const isLoading = ref(false)
  const isLoadingMore = ref(false)
  const searchQuery = ref('')
  const viewMode = ref<GamesViewMode>('grid')
  const showAddModal = ref(false)
  const addGameSubmitting = ref(false)
  const isTogglingFavorite = ref(false)
  let listRequestId = 0
  let loadMoreObserver: IntersectionObserver | null = null
  // 记录最后一次真正发起加载时的 query 内容快照。keep-alive 返回时 route.query 引用
  // 必然变化（每次导航重建对象），但内容相同；若直接重拉会清空已加载的深列表。
  let lastLoadedQueryKey: string | null = null

  const sortOptions = [
    { label: '最近更新', value: 'updated_at:desc' },
    { label: '最早更新', value: 'updated_at:asc' },
    { label: '最新添加', value: 'created_at:desc' },
    { label: '最早添加', value: 'created_at:asc' },
    { label: '名称 A-Z', value: 'title:asc' },
    { label: '名称 Z-A', value: 'title:desc' },
    { label: '年份新到旧', value: 'release_date:desc' },
    { label: '年份旧到新', value: 'release_date:asc' },
    { label: '下载量最高', value: 'downloads:desc' },
    { label: '随机', value: 'random:desc' },
  ]

  const games = computed(() => gamesStore.games)
  const pagination = computed(() => gamesStore.pagination)
  const hasLoadFailure = computed(() => {
    // 2026-04-07: a failed catalog read must not fall through to the empty-state copy.
    // Impact: "暂无游戏" now only means the backend successfully returned an empty page.
    return Boolean(gamesStore.listError) && games.value.length === 0
  })

  const sortBy = computed({
    get: () => {
      const field = parseGamesSortField(readSingleQueryValue(route.query.sort)) || DEFAULT_SORT.field
      const order = parseGamesSortOrder(readSingleQueryValue(route.query.order)) || DEFAULT_SORT.order
      return buildGamesSortOptionValue(field, order) || buildGamesSortOptionValue(DEFAULT_SORT.field, DEFAULT_SORT.order)!
    },
    set: (sort: string) => {
      const [nextFieldRaw, nextOrderRaw] = String(sort).split(':')
      const nextField = parseGamesSortField(nextFieldRaw)
      const nextOrder = parseGamesSortOrder(nextOrderRaw)
      if (!nextField || !nextOrder) {
        return
      }
      const isDefaultSort = nextField === DEFAULT_SORT.field && nextOrder === DEFAULT_SORT.order
      updateRoute({
        sort: isDefaultSort ? undefined : nextField,
        order: isDefaultSort ? undefined : nextOrder,
        seed: nextField === 'random'
          ? (readSingleQueryValue(route.query.seed) || String(Date.now()))
          : undefined,
        page: undefined,
      })
    },
  })

  const filterFavorites = computed(() => {
    return parseRouteBoolean(route.query.favorite) === true
  })

  const filterPrivate = computed(() => {
    return readSingleQueryValue(route.query.visibility) === 'private'
  })

  const hasActiveFilters = computed(() => hasGamesActiveFilters(route.query))

  const pageTitle = computed(() => {
    if (filterFavorites.value) return '收藏的游戏'
    if (filterPrivate.value) return '私有游戏'
    return '所有游戏'
  })

  const updateRoute = (newParams: LocationQueryRaw) => {
    const query = buildGamesRouteQuery(route.query, newParams)
    if (isSameRouteQuery(route.query, query)) {
      return
    }
    router.push({ name: 'games', query })
  }

  const normalizeRouteSortQuery = () => {
    const query = normalizeGamesFavoriteRouteQuery(route.query)
      || normalizeGamesPaginationRouteQuery(route.query)
      || normalizeGamesSortRouteQuery(route.query)
    if (!query || isSameRouteQuery(route.query, query)) return false

    router.replace({
      name: 'games',
      query,
    })
    return true
  }

  const loadGames = async () => {
    lastLoadedQueryKey = JSON.stringify(route.query)
    const requestId = ++listRequestId
    isLoading.value = true

    const request = buildGamesListRequest({
      routeQuery: route.query,
    })

    try {
      await gamesStore.fetchGames(request)
    } catch {
      uiStore.addAlert('加载游戏失败', 'error')
    } finally {
      if (requestId === listRequestId) {
        isLoading.value = false
      }
    }
  }

  const loadMoreGames = async () => {
    if (isLoading.value || isLoadingMore.value || !gamesStore.hasMorePages) {
      return
    }

    const requestId = listRequestId
    isLoadingMore.value = true
    try {
      const request = buildGamesListRequest({
        routeQuery: route.query,
      })
      await gamesStore.fetchGames({
        query: {
          ...request.query,
          page: gamesStore.pagination.page + 1,
        },
        sort: request.sort,
        append: true,
      })
      if (requestId !== listRequestId) {
        await loadGames()
      }
    } catch {
      uiStore.addAlert('加载更多游戏失败', 'error')
    } finally {
      isLoadingMore.value = false
    }
  }

  const setupLoadMoreObserver = () => {
    if (loadMoreObserver) {
      loadMoreObserver.disconnect()
      loadMoreObserver = null
    }
    if (!loadMoreSentinel.value || typeof IntersectionObserver === 'undefined') {
      return
    }

    loadMoreObserver = new IntersectionObserver((entries) => {
      if (entries.some((entry) => entry.isIntersecting)) {
        void loadMoreGames()
      }
    }, {
      rootMargin: '320px 0px',
    })
    loadMoreObserver.observe(loadMoreSentinel.value)
  }

  watch(loadMoreSentinel, () => {
    setupLoadMoreObserver()
  })

  // 2026-04-04: keep route.query as the only active-filter source of truth.
  // Impact: the debounced search input remains a local draft, while badges/empty states/pagination
  // only reflect filters that have actually reached the backend request.
  watch(() => route.query, () => {
    searchQuery.value = readSingleQueryValue(route.query.search) || ''
    if (route.name !== 'games') {
      return
    }
    const queryKey = JSON.stringify(route.query)
    if (queryKey === lastLoadedQueryKey) {
      // keep-alive 恢复：query 内容未变，store 已有该查询的完整数据，跳过重拉
      return
    }
    void loadGames()
  })

  const viewGame = (publicId: string) => {
    if (!publicId) return
    router.push({
      name: 'game-detail',
      params: { publicId },
    })
  }

  const viewSeries = (id: number) => {
    if (id <= 0) return
    router.push({
      name: 'series-detail',
      params: { id: String(id) },
    })
  }

  const handleAddGame = () => {
    if (!isAdmin.value) return
    showAddModal.value = true
  }

  const handleAddGameSubmit = async (data: { title: string; visibility: 'public' | 'private' }) => {
    if (addGameSubmitting.value) return
    addGameSubmitting.value = true
    try {
      await gamesService.createGame({
        title: data.title,
        visibility: data.visibility,
      })

      uiStore.addAlert(`游戏 "${data.title}" 添加成功`, 'success')
      showAddModal.value = false
      try {
        // 2026-04-10: creating the game and refreshing the catalog are separate outcomes.
        // Impact: a successful write must stay visible even if the follow-up list refresh fails.
        await loadGames()
      } catch {
        uiStore.addAlert('添加已生效，但列表刷新失败，请稍后重试', 'warning')
      }
    } catch (error) {
      uiStore.addAlert(`添加游戏失败：${getHttpErrorMessage(error)}`, 'error')
    } finally {
      addGameSubmitting.value = false
    }
  }

  const toggleFavorite = async (gameRef: string) => {
    if (!gameRef) return
    isTogglingFavorite.value = true
    try {
      await gamesStore.toggleFavorite(gameRef)
      uiStore.addAlert('收藏已更新', 'success')
    } catch {
      uiStore.addAlert('更新收藏失败', 'error')
    } finally {
      isTogglingFavorite.value = false
    }
  }

  const deleteGame = async (gameRef: string, title: string) => {
    try {
      const result = await gamesService.deleteGame(gameRef)
      uiStore.addAlert(`游戏 "${title}" 已删除`, 'success')
      if (result.warnings.length > 0) {
        uiStore.addAlert(
          `游戏 "${title}" 已删除，但仍有 ${result.warnings.length} 个残留素材等待清理，系统会在下次后端启动时自动重试删除`,
          'warning'
        )
      }
      try {
        await loadGames()
      } catch {
        uiStore.addAlert('删除已生效，但列表刷新失败，请稍后重试', 'warning')
      }
    } catch (error) {
      uiStore.addAlert(`删除游戏失败：${getHttpErrorMessage(error)}`, 'error')
    }
  }

  const handleDelete = (gameRef: string, title: string) => {
    if (!gameRef || !isAdmin.value) return

    // Destructive actions still require an explicit blocking confirmation; only the result toast is unified via uiStore.
    Modal.confirm({
      title: '确认删除',
      content: `确定要删除游戏 "${title}" 吗？此操作不可撤销。`,
      okText: '删除',
      cancelText: '取消',
      okButtonProps: { status: 'danger' },
      onOk: async () => {
        try {
          await deleteGame(gameRef, title)
        } catch (error) {
          uiStore.addAlert(`删除游戏失败：${getHttpErrorMessage(error)}`, 'error')
        }
      },
    })
  }

  const handleSearch = () => {
    // 2026-04-06: empty and whitespace-only input should mean "no backend search filter".
    // Impact: route state, active-filter badges, and request params now share one native search semantic.
    updateRoute({ search: normalizeCommittedSearchValue(searchQuery.value) })
  }

  const clearFilters = () => {
    searchQuery.value = ''
    if (Object.keys(route.query).length === 0) {
      return
    }
    router.push({ name: 'games' })
  }

  onMounted(async () => {
    viewMode.value = uiStore.gamesViewMode

    const routeWasNormalized = normalizeRouteSortQuery()
    if (routeWasNormalized) {
      return
    }

    searchQuery.value = readSingleQueryValue(route.query.search) || ''
    await loadGames()
    setupLoadMoreObserver()
  })

  let searchDebounceTimer: number | undefined
  watch(searchQuery, (newQuery, oldQuery) => {
    if (newQuery === oldQuery) return

    if (searchDebounceTimer) {
      clearTimeout(searchDebounceTimer)
    }

    if (typeof window === 'undefined') return

    // 2026-04-04: keep debounce at the input edge only.
    // Impact: typing does not spam route updates, but the request state still flips only when the
    // debounced value is committed into route.query.
    searchDebounceTimer = window.setTimeout(() => {
      if (newQuery !== (readSingleQueryValue(route.query.search) || '')) {
        handleSearch()
      }
    }, 500)
  })

  watch(viewMode, (value) => {
    uiStore.setGamesViewMode(value)
  })

  onBeforeUnmount(() => {
    if (loadMoreObserver) {
      loadMoreObserver.disconnect()
      loadMoreObserver = null
    }
    if (searchDebounceTimer) {
      clearTimeout(searchDebounceTimer)
    }
  })

  return {
    clearFilters,
    addGameSubmitting,
    filterFavorites,
    filterPrivate,
    games,
    handleAddGame,
    handleAddGameSubmit,
    handleDelete,
    handleSearch,
    hasActiveFilters,
    hasLoadFailure,
    isLoading,
    isLoadingMore,
    isTogglingFavorite,
    loadGames,
    pageTitle,
    pagination,
    searchQuery,
    showAddModal,
    sortBy,
    sortOptions,
    toggleFavorite,
    updateRoute,
    viewGame,
    viewSeries,
    viewMode,
  }
}
