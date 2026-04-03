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

type GamesViewFilter = 'favorites' | 'recent'
type GamesViewMode = 'grid' | 'list'
type GamesSortKey =
  | 'created_desc'
  | 'created_asc'
  | 'title_asc'
  | 'title_desc'
  | 'release_desc'
  | 'release_asc'
  | 'downloads_desc'
  | 'random_desc'

interface GamesViewProps {
  filter?: GamesViewFilter
}

interface UseGamesViewOptions {
  props: GamesViewProps
  route: RouteLocationNormalizedLoaded
  router: Router
  gamesStore: ReturnType<typeof useGamesStore>
  uiStore: ReturnType<typeof useUiStore>
  isAdmin: Ref<boolean>
}

interface BuildGamesListRequestOptions {
  routeQuery: LocationQuery
  itemsPerPage: number
  filterFavorites: boolean
  sortBy: string
}

const DEFAULT_SORT: GamesSortKey = 'created_desc'
const DEFAULT_ITEMS_PER_PAGE = 24
const AMBIENT_BACKGROUND_OWNER = 'games'

const SORT_ALIASES: Record<string, GamesSortKey> = {
  newest: 'created_desc',
  created_desc: 'created_desc',
  created_asc: 'created_asc',
  title_asc: 'title_asc',
  title_desc: 'title_desc',
  release_asc: 'release_asc',
  release_desc: 'release_desc',
  downloads: 'downloads_desc',
  downloads_desc: 'downloads_desc',
  random: 'random_desc',
  random_desc: 'random_desc',
}

const SORT_FIELD_MAP: Record<'created' | 'title' | 'release' | 'downloads' | 'random', GameSort['field']> = {
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

export const normalizeGamesSortValue = (value: string | undefined): GamesSortKey => {
  return SORT_ALIASES[value || ''] || DEFAULT_SORT
}

export const buildGamesRouteQuery = (
  currentQuery: LocationQuery,
  newParams: LocationQueryRaw,
): LocationQueryRaw => {
  const query: LocationQueryRaw = { ...currentQuery, ...newParams }
  delete query.needs

  Object.keys(query).forEach((key) => {
    if (query[key] === undefined || query[key] === null || query[key] === '') {
      delete query[key]
    }
  })

  if (
    newParams.search !== undefined
    || newParams.platform !== undefined
    || newParams.tag !== undefined
    || newParams.filter !== undefined
  ) {
    query.page = '1'
  }

  return query
}

export const buildGamesListRequest = ({
  routeQuery,
  itemsPerPage,
  filterFavorites,
  sortBy,
}: BuildGamesListRequestOptions): { query: GameListQuery; sort: GameSort } => {
  const page = parsePositiveQueryNumber(readSingleQueryValue(routeQuery.page), 1)
  const normalizedSort = normalizeGamesSortValue(sortBy)
  const [field, order] = normalizedSort.split('_') as [
    keyof typeof SORT_FIELD_MAP,
    GameSort['order'] | undefined,
  ]
  const sortField = SORT_FIELD_MAP[field] || SORT_FIELD_MAP.created
  const resolvedOrder: GameSort['order'] = order === 'asc' ? 'asc' : 'desc'

  return {
    query: {
      page,
      limit: itemsPerPage,
      search: readSingleQueryValue(routeQuery.search),
      platform: parsePositiveRouteNumber(routeQuery.platform),
      tag: parseRouteTagIds(routeQuery.tag),
      favorite: filterFavorites || undefined,
    },
    sort: {
      field: sortField,
      order: resolvedOrder,
      seed: field === 'random'
        ? parsePositiveQueryNumber(readSingleQueryValue(routeQuery.seed), Date.now())
        : undefined,
    },
  }
}

export const useGamesView = ({
  props,
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
    { label: '最新添加', value: 'created_desc' },
    { label: '最早添加', value: 'created_asc' },
    { label: '名称 A-Z', value: 'title_asc' },
    { label: '名称 Z-A', value: 'title_desc' },
    { label: '年份新到旧', value: 'release_desc' },
    { label: '年份旧到新', value: 'release_asc' },
    { label: '下载量最高', value: 'downloads_desc' },
    { label: '随机', value: 'random_desc' },
  ]

  const visibleGames = ref<GameListItem[]>([])
  const visiblePagination = ref({
    total: 0,
    page: 1,
    limit: DEFAULT_ITEMS_PER_PAGE,
    totalPages: 0,
  })

  const games = computed(() => visibleGames.value)
  const pagination = computed(() => visiblePagination.value)
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
    get: () => normalizeGamesSortValue(readSingleQueryValue(route.query.sort)),
    set: (sort: string) => {
      updateRoute({
        sort,
        seed: sort === 'random_desc' ? (readSingleQueryValue(route.query.seed) || String(Date.now())) : undefined,
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
    return props.filter === 'favorites' || readSingleQueryValue(route.query.filter) === 'favorites'
  })

  const selectedTagIds = computed(() => parseRouteTagIds(route.query.tag))

  const filterableTagGroups = computed(() =>
    [...tagGroups.value]
      .filter((group) => group.is_filterable)
      .sort((a, b) => a.sort_order - b.sort_order || a.id - b.id),
  )

  const hasActiveFilters = computed(() => {
    return Boolean(
      searchQuery.value
      || selectedPlatform.value
      || selectedTagIds.value.length > 0
      || filterFavorites.value,
    )
  })

  const pageTitle = computed(() => {
    const currentFilter = props.filter || readSingleQueryValue(route.query.filter)
    if (currentFilter === 'favorites') return '收藏的游戏'
    if (currentFilter === 'recent') return '最近下载'
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
    router.push({ name: 'games', query })
  }

  const loadGames = async () => {
    isLoading.value = true

    const request = buildGamesListRequest({
      routeQuery: route.query,
      itemsPerPage: itemsPerPage.value,
      filterFavorites: filterFavorites.value,
      sortBy: sortBy.value,
    })

    try {
      const response = await gamesStore.fetchGames(request)
      visibleGames.value = response.data
      visiblePagination.value = {
        total: response.pagination.total,
        page: response.pagination.page,
        limit: response.pagination.limit,
        totalPages: response.pagination.totalPages,
      }

      syncAmbientBackground(response.data, response.pagination.page)
    } catch {
      uiStore.addAlert('加载游戏失败', 'error')
    } finally {
      isLoading.value = false
    }
  }

  const syncAmbientBackground = (games: GameListItem[], page = visiblePagination.value.page || 1) => {
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

    searchQuery.value = readSingleQueryValue(route.query.search) || ''
    if (games.value.length === 0 || Object.keys(route.query).length > 0) {
      await loadGames()
    }
  })

  onActivated(async () => {
    await loadFilterOptions()
    syncAmbientBackground(visibleGames.value)
  })

  let searchDebounceTimer: number | undefined
  watch(searchQuery, (newQuery, oldQuery) => {
    if (newQuery === oldQuery) return

    if (searchDebounceTimer) {
      clearTimeout(searchDebounceTimer)
    }

    if (typeof window === 'undefined') return

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
