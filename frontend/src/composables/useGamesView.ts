import { computed, onActivated, onBeforeUnmount, onMounted, ref, watch, type Ref } from 'vue'
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
import tagsService from '@/services/tags.service'
import type { GameListItem, GameListQuery, GameSort, GameSortQuery, Tag, TagGroup } from '@/services/types'
import { useGamesStore } from '@/stores/games'
import { useUiStore } from '@/stores/ui'
import { getAmbientBackgroundUrlsFromGames } from '@/utils/ambient-background'

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
}

interface BuildGamesListRequestOptions {
  routeQuery: LocationQuery
  itemsPerPage: number
}

// 2026-04-04: keep this UI-only default aligned with the backend list default sort/order.
// Impact: the select shows "最近更新" when route.query omits both fields, but requests still rely
// on the backend native default instead of forcing route-owned sort/order values.
const DEFAULT_SORT = { field: 'updated_at', order: 'desc' } as const satisfies Pick<GameSort, 'field' | 'order'>
const DEFAULT_ITEMS_PER_PAGE = 24
const AMBIENT_BACKGROUND_OWNER = 'games'
const ITEMS_PER_PAGE_VALUES = new Set([12, 24, 48, 96])

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

export const parsePositiveQueryNumber = (value: string | undefined, fallback: number): number => {
  if (!value) return fallback
  const parsed = Number.parseInt(value, 10)
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback
}

export const parseRouteTagIds = (
  value: LocationQueryValue | LocationQueryValue[] | undefined,
): number[] => {
  const values = Array.isArray(value) ? value : value ? [value] : []
  return values
    .map((item) => Number(item))
    .filter((item) => Number.isInteger(item) && item > 0)
}

export const parsePositiveRouteNumber = (
  value: LocationQueryValue | LocationQueryValue[] | undefined,
): number | undefined => {
  const parsed = Number(readSingleQueryValue(value))
  return Number.isInteger(parsed) && parsed > 0 ? parsed : undefined
}

export const parseGamesItemsPerPage = (
  value: LocationQueryValue | LocationQueryValue[] | undefined,
): number | undefined => {
  const parsed = parsePositiveRouteNumber(value)
  return parsed !== undefined && ITEMS_PER_PAGE_VALUES.has(parsed) ? parsed : undefined
}

export const parseRouteBoolean = (
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

  if (
    newParams.search !== undefined
    || newParams.tag !== undefined
    || newParams.favorite !== undefined
  ) {
    query.page = '1'
  }

  return query
}

export const buildGamesListRequest = ({
  routeQuery,
  itemsPerPage,
}: BuildGamesListRequestOptions): { query: GameListQuery; sort?: GameSortQuery } => {
  const page = parsePositiveQueryNumber(readSingleQueryValue(routeQuery.page), 1)
  const rawSort = readSingleQueryValue(routeQuery.sort)
  const rawOrder = readSingleQueryValue(routeQuery.order)
  const sortField = parseGamesSortField(rawSort)
  const rawFavorite = readSingleQueryValue(routeQuery.favorite)
  const favorite = parseRouteBoolean(routeQuery.favorite)

  const request: { query: GameListQuery; sort?: GameSortQuery } = {
    query: {
      page,
      limit: itemsPerPage,
      search: normalizeCommittedSearchValue(readSingleQueryValue(routeQuery.search)),
      tag: parseRouteTagIds(routeQuery.tag),
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
    || parseRouteTagIds(routeQuery.tag).length > 0
    || parseRouteBoolean(routeQuery.favorite) === true,
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

  const normalizedPage = parsePositiveRouteNumber(routeQuery.page)
  const normalizedLimit = parseGamesItemsPerPage(routeQuery.limit)

  if ((rawPage === undefined || normalizedPage !== undefined) && (rawLimit === undefined || normalizedLimit !== undefined)) {
    return null
  }

  // 2026-04-06: games route paging stays inside the UI-supported page sizes and valid positive pages.
  // Impact: shared URLs no longer depend on backend coercion to recover malformed limit/page inputs.
  return buildGamesRouteQuery(routeQuery, {
    page: normalizedPage !== undefined ? String(normalizedPage) : undefined,
    limit: normalizedLimit !== undefined ? String(normalizedLimit) : undefined,
  })
}

export const normalizeGamesPaginationResponseQuery = (
  routeQuery: LocationQuery,
  pagination: Pick<GameListQuery, 'page' | 'limit'> & { totalPages: number },
): LocationQueryRaw | null => {
  const requestedPage = parsePositiveQueryNumber(readSingleQueryValue(routeQuery.page), 1)

  if (pagination.totalPages === 0) {
    if (requestedPage === 1) {
      return null
    }
    return buildGamesRouteQuery(routeQuery, {
      page: undefined,
    })
  }

  if (requestedPage <= pagination.totalPages) {
    return null
  }

  return buildGamesRouteQuery(routeQuery, {
    page: String(pagination.totalPages),
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
}: UseGamesViewOptions) => {
  const isLoading = ref(false)
  const searchQuery = ref('')
  const viewMode = ref<GamesViewMode>('grid')
  const showAddModal = ref(false)
  const addGameSubmitting = ref(false)
  const showTagFilters = ref(false)
  const tagGroups = ref<TagGroup[]>([])
  const tags = ref<Tag[]>([])
  const hasFilterOptionsLoadFailure = ref(false)
  const filterOptionsLoadFailedWithStaleData = ref(false)

  const itemsPerPageOptions = [
    { label: '12', value: 12 },
    { label: '24', value: 24 },
    { label: '48', value: 48 },
    { label: '96', value: 96 },
  ]

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
  const totalPages = computed(() => pagination.value.totalPages || 0)
  const hasLoadFailure = computed(() => {
    // 2026-04-07: a failed catalog read must not fall through to the empty-state copy.
    // Impact: "暂无游戏" now only means the backend successfully returned an empty page.
    return Boolean(gamesStore.listError) && games.value.length === 0
  })

  const currentPage = computed({
    get: () => parsePositiveQueryNumber(readSingleQueryValue(route.query.page), 1),
    set: (page: number) => {
      if (page !== parsePositiveQueryNumber(readSingleQueryValue(route.query.page), 1)) {
        updateRoute({ page: String(page) })
      }
    },
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
        page: '1',
      })
    },
  })

  const itemsPerPage = computed({
    get: () => parseGamesItemsPerPage(route.query.limit) ?? DEFAULT_ITEMS_PER_PAGE,
    set: (limit: number) => {
      updateRoute({ limit: String(limit), page: '1' })
    },
  })

  const filterFavorites = computed(() => {
    return parseRouteBoolean(route.query.favorite) === true
  })

  const selectedTagIds = computed(() => parseRouteTagIds(route.query.tag))

  const filterableTagGroups = computed(() =>
    [...tagGroups.value]
      .filter((group) => group.is_filterable)
      .sort((a, b) => a.sort_order - b.sort_order || a.id - b.id),
  )

  const hasActiveFilters = computed(() => hasGamesActiveFilters(route.query))

  const pageTitle = computed(() => {
    if (filterFavorites.value) return '收藏的游戏'
    return '所有游戏'
  })

  const tagLabelMap = computed<Record<string, string>>(() => {
    return Object.fromEntries(tags.value.map((item) => [String(item.id), item.name]))
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
    isLoading.value = true

    const request = buildGamesListRequest({
      routeQuery: route.query,
      itemsPerPage: itemsPerPage.value,
    })

    try {
      const response = await gamesStore.fetchGames(request)
      const normalizedPaginationQuery = normalizeGamesPaginationResponseQuery(route.query, response.pagination)
      if (normalizedPaginationQuery && !isSameRouteQuery(route.query, normalizedPaginationQuery)) {
        // 2026-04-06: when filters or deletes invalidate the current page, route state must follow
        // the backend pagination contract instead of rendering an out-of-range empty page as "no data".
        router.replace({
          name: 'games',
          query: normalizedPaginationQuery,
        })
        return
      }
      syncAmbientBackground(response.data, response.pagination.page)
    } catch {
      uiStore.addAlert('加载游戏失败', 'error')
    } finally {
      isLoading.value = false
    }
  }

  const syncAmbientBackground = (games: GameListItem[], page = pagination.value.page || 1) => {
    const imageUrls = getAmbientBackgroundUrlsFromGames(games)
    if (imageUrls.length > 0) {
      uiStore.setAmbientBackgroundSource({
        owner: AMBIENT_BACKGROUND_OWNER,
        key: `${pageTitle.value}:${page}:${games.length}`,
        urls: imageUrls,
      })
      return
    }

    uiStore.clearAmbientBackgroundSource(AMBIENT_BACKGROUND_OWNER)
  }

  // 2026-04-04: keep route.query as the only active-filter source of truth.
  // Impact: the debounced search input remains a local draft, while badges/empty states/pagination
  // only reflect filters that have actually reached the backend request.
  watch(() => route.query, () => {
    searchQuery.value = readSingleQueryValue(route.query.search) || ''
    void loadGames()
  })

  const viewGame = (publicId: string) => {
    if (!publicId) return
    router.push({
      name: 'game-detail',
      params: { publicId },
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
    try {
      await gamesStore.toggleFavorite(gameRef)
      uiStore.addAlert('收藏已更新', 'success')
    } catch {
      uiStore.addAlert('更新收藏失败', 'error')
    }
  }

  const deleteGame = async (gameRef: string, title: string) => {
    const result = await gamesService.deleteGame(gameRef)
    uiStore.addAlert(`游戏 "${title}" 已删除`, 'success')
    if (result.warnings.length > 0) {
      uiStore.addAlert(
        `游戏 "${title}" 已删除，但仍有 ${result.warnings.length} 个残留素材等待清理，系统会在下次后端启动时自动重试删除`,
        'warning'
      )
    }
    try {
      // 2026-04-10: deletion and catalog refresh are separate outcomes.
      // Impact: once the delete request succeeds, a later list refresh failure must not
      // be reported as "delete failed".
      await loadGames()
    } catch {
      uiStore.addAlert('删除已生效，但列表刷新失败，请稍后重试', 'warning')
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

  const getTagsForGroup = (groupId: number) => {
    return tags.value
      .filter((item) => item.group_id === groupId && item.is_active)
      .sort((a, b) => a.sort_order - b.sort_order || a.id - b.id)
  }

  const getSelectedTagIdsForGroup = (groupId: number) => {
    const values = selectedTagIds.value.filter((tagId) => {
      const tag = tags.value.find((item) => item.id === tagId)
      return tag?.group_id === groupId
    })
    const group = tagGroups.value.find((item) => item.id === groupId)
    return group?.allow_multiple ? values : (values[0] ?? undefined)
  }

  const updateSelectedTagsForGroup = (
    groupId: number,
    value: number | number[] | string | string[] | undefined,
  ) => {
    const nextGroupValues = (
      Array.isArray(value)
        ? value
        : value === undefined || value === null || value === ''
          ? []
          : [value]
    )
      .map((item) => Number(item))
      .filter((item) => Number.isInteger(item) && item > 0)

    const nextTagIds = selectedTagIds.value.filter((tagId) => {
      const tag = tags.value.find((item) => item.id === tagId)
      return tag?.group_id !== groupId
    })

    nextTagIds.push(...nextGroupValues)
    updateRoute({ tag: nextTagIds.length > 0 ? nextTagIds.map(String) : undefined })
  }

  const removeTagFilter = (tagId: number) => {
    const nextTagIds = selectedTagIds.value.filter((value) => value !== tagId)
    updateRoute({ tag: nextTagIds.length > 0 ? nextTagIds.map(String) : undefined })
  }

  const handleTagGroupSelectionChange = (
    groupId: number,
    value: number | number[] | string | string[] | undefined,
  ) => {
    updateSelectedTagsForGroup(groupId, value)
  }

  const loadFilterOptions = async () => {
    hasFilterOptionsLoadFailure.value = false
    filterOptionsLoadFailedWithStaleData.value = false
    try {
      const [loadedGroups, loadedTags] = await Promise.all([
        tagsService.getTagGroups(),
        tagsService.getTags({ active: true }),
      ])
      tagGroups.value = loadedGroups
      tags.value = loadedTags
    } catch (error) {
      // 2026-04-09: tag filter metadata failures must not masquerade as either
      // "no filterable tags" or freshly loaded filter options from the current request.
      if (tagGroups.value.length > 0 || tags.value.length > 0) {
        filterOptionsLoadFailedWithStaleData.value = true
      } else {
        hasFilterOptionsLoadFailure.value = true
      }
      console.error('Failed to load tags:', error)
      uiStore.addAlert('加载标签筛选失败', 'error')
    }
  }

  onMounted(async () => {
    viewMode.value = uiStore.gamesViewMode

    await loadFilterOptions()
    const routeWasNormalized = normalizeRouteSortQuery()
    if (routeWasNormalized) {
      return
    }

    searchQuery.value = readSingleQueryValue(route.query.search) || ''
    if (games.value.length === 0 || Object.keys(route.query).length > 0) {
      await loadGames()
      return
    }

    // Re-entering the catalog can reuse the cached store payload, which skips loadGames().
    // Keep the ambient background in sync even when the page comes entirely from client cache.
    syncAmbientBackground(games.value)
  })

  onActivated(async () => {
    await loadFilterOptions()
    syncAmbientBackground(games.value)
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
    if (searchDebounceTimer) {
      clearTimeout(searchDebounceTimer)
    }
  })

  return {
    clearFilters,
    currentPage,
    addGameSubmitting,
    filterFavorites,
    filterableTagGroups,
    games,
    getSelectedTagIdsForGroup,
    getTagsForGroup,
    handleAddGame,
    handleAddGameSubmit,
    handleDelete,
    handleSearch,
    handleTagGroupSelectionChange,
    hasActiveFilters,
    hasLoadFailure,
    hasFilterOptionsLoadFailure,
    isLoading,
    itemsPerPage,
    itemsPerPageOptions,
    loadGames,
    pageTitle,
    pagination,
    filterOptionsLoadFailedWithStaleData,
    removeTagFilter,
    searchQuery,
    selectedTagIds,
    showAddModal,
    showTagFilters,
    sortBy,
    sortOptions,
    tagLabelMap,
    toggleFavorite,
    totalPages,
    updateRoute,
    viewGame,
    viewMode,
  }
}
