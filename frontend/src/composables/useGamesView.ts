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
import platformService from '@/services/platforms.service'
import tagsService from '@/services/tags.service'
import type { GameListItem, GameListQuery, GameSort, Tag, TagGroup } from '@/services/types'
import { useGamesStore } from '@/stores/games'
import { useUiStore } from '@/stores/ui'
import { getAmbientBackgroundUrlsFromGames } from '@/utils/ambient-background'

type GamesViewMode = 'grid' | 'list'
type GamesSortKey =
  | 'updated_desc'
  | 'updated_asc'
  | 'created_desc'
  | 'created_asc'
  | 'title_asc'
  | 'title_desc'
  | 'release_desc'
  | 'release_asc'
  | 'downloads_desc'
  | 'random_desc'

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

// 2026-04-04: keep this UI-only default aligned with the backend list default sort.
// Impact: the select shows "最近更新" when route.query omits sort, but requests still rely on the
// backend native default instead of forcing a front-end sort parameter.
const DEFAULT_SORT: GamesSortKey = 'updated_desc'
const DEFAULT_ITEMS_PER_PAGE = 24
const AMBIENT_BACKGROUND_OWNER = 'games'

const SORT_VALUES = new Set<GamesSortKey>([
  'updated_desc',
  'updated_asc',
  'created_desc',
  'created_asc',
  'title_asc',
  'title_desc',
  'release_asc',
  'release_desc',
  'downloads_desc',
  'random_desc',
])

const SORT_FIELD_MAP: Record<'updated' | 'created' | 'title' | 'release' | 'downloads' | 'random', GameSort['field']> = {
  updated: 'updated_at',
  created: 'created_at',
  title: 'title',
  release: 'release_date',
  downloads: 'downloads',
  random: 'random',
}

export const readSingleQueryValue = (
  value: LocationQueryValue | LocationQueryValue[] | undefined,
): string | undefined => {
  if (Array.isArray(value)) {
    return value.find((item): item is string => typeof item === 'string' && item.length > 0)
  }
  return typeof value === 'string' && value.length > 0 ? value : undefined
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

export const parseRouteBoolean = (
  value: LocationQueryValue | LocationQueryValue[] | undefined,
): boolean | undefined => {
  const raw = readSingleQueryValue(value)
  if (raw === 'true') return true
  if (raw === 'false') return false
  return undefined
}

export const parseGamesSortValue = (value: string | undefined): GamesSortKey | undefined => {
  return value && SORT_VALUES.has(value as GamesSortKey) ? value as GamesSortKey : undefined
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
    || newParams.platform !== undefined
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
}: BuildGamesListRequestOptions): { query: GameListQuery; sort?: GameSort } => {
  const page = parsePositiveQueryNumber(readSingleQueryValue(routeQuery.page), 1)
  const sortValue = parseGamesSortValue(readSingleQueryValue(routeQuery.sort))
  const favorite = parseRouteBoolean(routeQuery.favorite) === true
    ? true
    : undefined

  const request: { query: GameListQuery; sort?: GameSort } = {
    query: {
      page,
      limit: itemsPerPage,
      search: readSingleQueryValue(routeQuery.search),
      platform: parsePositiveRouteNumber(routeQuery.platform),
      tag: parseRouteTagIds(routeQuery.tag),
      favorite,
    },
  }

  if (sortValue) {
    const [field, order] = sortValue.split('_') as [
      keyof typeof SORT_FIELD_MAP,
      GameSort['order'] | undefined,
    ]
    request.sort = {
      field: SORT_FIELD_MAP[field] || SORT_FIELD_MAP.updated,
      order: order === 'asc' ? 'asc' : 'desc',
      seed: field === 'random'
        ? parsePositiveRouteNumber(routeQuery.seed)
        : undefined,
    }
  }

  return request
}

export const hasGamesActiveFilters = (routeQuery: LocationQuery): boolean => {
  return Boolean(
    readSingleQueryValue(routeQuery.search)
    || readSingleQueryValue(routeQuery.platform)
    || parseRouteTagIds(routeQuery.tag).length > 0
    || parseRouteBoolean(routeQuery.favorite) === true,
  )
}

export const normalizeGamesFavoriteRouteQuery = (routeQuery: LocationQuery): LocationQueryRaw | null => {
  const rawFavorite = readSingleQueryValue(routeQuery.favorite)
  if (rawFavorite === undefined) {
    return null
  }
  if (parseRouteBoolean(rawFavorite) === true) {
    return null
  }

  return buildGamesRouteQuery(routeQuery, {
    favorite: undefined,
  })
}

export const normalizeGamesSortRouteQuery = (routeQuery: LocationQuery): LocationQueryRaw | null => {
  const rawSort = readSingleQueryValue(routeQuery.sort)
  if (!rawSort) return null

  const sortValue = parseGamesSortValue(rawSort)
  if (!sortValue) {
    return buildGamesRouteQuery(routeQuery, {
      sort: undefined,
      seed: undefined,
    })
  }

  if (sortValue !== 'random_desc') {
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
  const showTagFilters = ref(false)
  const platformOptions = ref<Array<{ label: string; value: string }>>([])
  const tagGroups = ref<TagGroup[]>([])
  const tags = ref<Tag[]>([])

  const itemsPerPageOptions = [
    { label: '12', value: 12 },
    { label: '24', value: 24 },
    { label: '48', value: 48 },
    { label: '96', value: 96 },
  ]

  const sortOptions = [
    { label: '最近更新', value: 'updated_desc' },
    { label: '最早更新', value: 'updated_asc' },
    { label: '最新添加', value: 'created_desc' },
    { label: '最早添加', value: 'created_asc' },
    { label: '名称 A-Z', value: 'title_asc' },
    { label: '名称 Z-A', value: 'title_desc' },
    { label: '年份新到旧', value: 'release_desc' },
    { label: '年份旧到新', value: 'release_asc' },
    { label: '下载量最高', value: 'downloads_desc' },
    { label: '随机', value: 'random_desc' },
  ]

  const games = computed(() => gamesStore.games)
  const pagination = computed(() => gamesStore.pagination)
  const totalPages = computed(() => pagination.value.totalPages || 0)

  const currentPage = computed({
    get: () => parsePositiveQueryNumber(readSingleQueryValue(route.query.page), 1),
    set: (page: number) => {
      if (page !== parsePositiveQueryNumber(readSingleQueryValue(route.query.page), 1)) {
        updateRoute({ page: String(page) })
      }
    },
  })

  const selectedPlatform = computed({
    get: () => readSingleQueryValue(route.query.platform) || null,
    set: (platform: string | null) => {
      updateRoute({ platform })
    },
  })

  const sortBy = computed({
    get: () => parseGamesSortValue(readSingleQueryValue(route.query.sort)) || DEFAULT_SORT,
    set: (sort: string) => {
      const normalizedSort = parseGamesSortValue(sort) || DEFAULT_SORT
      updateRoute({
        sort: normalizedSort === DEFAULT_SORT ? undefined : normalizedSort,
        seed: normalizedSort === 'random_desc'
          ? (readSingleQueryValue(route.query.seed) || String(Date.now()))
          : undefined,
        page: '1',
      })
    },
  })

  const itemsPerPage = computed({
    get: () => parsePositiveQueryNumber(readSingleQueryValue(route.query.limit), DEFAULT_ITEMS_PER_PAGE),
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

  const platformLabelMap = computed<Record<string, string>>(() => {
    return Object.fromEntries(platformOptions.value.map((item) => [item.value, item.label]))
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
    try {
      await gamesService.createGame({
        title: data.title,
        visibility: data.visibility,
      })

      uiStore.addAlert(`游戏 "${data.title}" 添加成功`, 'success')
      await loadGames()
    } catch (error) {
      uiStore.addAlert(`添加游戏失败：${getHttpErrorMessage(error)}`, 'error')
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
    await loadGames()
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
    updateRoute({ search: searchQuery.value })
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
    try {
      const platforms = await platformService.listPlatforms()
      platformOptions.value = platforms
        .map((item) => ({ label: item.name, value: String(item.id) }))
        .sort((a, b) => a.label.localeCompare(b.label, 'zh-Hans-CN'))
    } catch (error) {
      console.error('Failed to load platforms:', error)
      uiStore.addAlert('加载平台筛选失败', 'error')
    }

    try {
      const [loadedGroups, loadedTags] = await Promise.all([
        tagsService.getTagGroups(),
        tagsService.getTags({ active: true }),
      ])
      tagGroups.value = loadedGroups
      tags.value = loadedTags
    } catch (error) {
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
    }
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
    isLoading,
    itemsPerPage,
    itemsPerPageOptions,
    loadGames,
    pageTitle,
    pagination,
    platformLabelMap,
    platformOptions,
    removeTagFilter,
    searchQuery,
    selectedPlatform,
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
