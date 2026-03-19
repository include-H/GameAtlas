<template>
  <div class="games-view">
    <!-- Header -->
    <div class="view-header">
      <div class="view-header-title-group">
        <h1 class="view-title text-gradient">{{ pageTitle }}</h1>
      </div>

      <a-space>
        <a-radio-group v-model="viewMode" type="button" size="small">
          <a-radio value="grid">
            <icon-apps />
          </a-radio>
          <a-radio value="list">
            <icon-list />
          </a-radio>
        </a-radio-group>

        <a-button v-if="isAdmin" class="app-primary-cta" type="primary" @click="handleAddGame">
          <template #icon>
            <icon-plus />
          </template>
          添加游戏
        </a-button>
      </a-space>
    </div>

    <!-- Search and Filters -->
    <a-card class="mb-4 search-card glass-panel" :bordered="false">
      <a-row :gutter="12">
        <!-- Search -->
        <a-col :xs="24" :sm="12" :md="6" :lg="6" :xl="6" :xxl="5">
          <a-input-search
            v-model="searchQuery"
            placeholder="搜索游戏"
            allow-clear
            @search="handleSearch"
          />
        </a-col>

        <!-- Platform Filter -->
        <a-col :xs="12" :sm="8" :md="4" :lg="4" :xl="4" :xxl="4">
          <a-select
            v-model="selectedPlatform"
            :options="platformOptions"
            placeholder="平台"
            allow-clear
          />
        </a-col>

        <!-- Sort -->
        <a-col :xs="24" :sm="8" :md="6" :lg="6" :xl="6" :xxl="6">
          <a-select
            v-model="sortBy"
            :options="sortOptions"
            placeholder="排序"
          >
            <template #prefix>
              <icon-sort />
            </template>
          </a-select>
        </a-col>

        <!-- Items Per Page -->
        <a-col :xs="24" :sm="8" :md="3" :lg="3" :xl="3" :xxl="3">
          <a-select
            v-model="itemsPerPage"
            :options="itemsPerPageOptions"
          />
        </a-col>

        <a-col :xs="24" :sm="8" :md="4" :lg="4" :xl="4" :xxl="4">
          <a-button class="games-filter-drawer-btn" type="secondary" long @click="showTagFilters = !showTagFilters">
            <template #icon>
              <icon-filter />
            </template>
            {{ showTagFilters ? '收起标签筛选' : '展开标签筛选' }}
            <span v-if="selectedTagIds.length > 0" class="games-filter-drawer-btn__count">
              {{ selectedTagIds.length }}
            </span>
          </a-button>
        </a-col>
      </a-row>

      <div v-if="showTagFilters" class="tag-filter-section">
        <div class="tag-filter-section__header">
          <span class="tag-filter-section__title">标签筛选</span>
          <span class="tag-filter-section__hint">同组多选为或，不同组之间为且</span>
        </div>

        <a-row v-if="filterableTagGroups.length > 0" :gutter="12" class="mt-3">
          <a-col
            v-for="group in filterableTagGroups"
            :key="group.id"
            :xs="24"
            :sm="12"
            :md="12"
            :lg="12"
            :xl="12"
            :xxl="12"
          >
            <div class="tag-filter-grid-item">
              <div class="tag-filter-drawer__label">{{ group.name }}</div>
              <a-select
                :model-value="getSelectedTagIdsForGroup(group.id)"
                :multiple="group.allow_multiple"
                :allow-search="true"
                allow-clear
                :placeholder="group.name"
                @change="handleTagGroupSelectionChange(group.id, $event)"
              >
                <a-option
                  v-for="tag in getTagsForGroup(group.id)"
                  :key="tag.id"
                  :value="tag.id"
                  :label="tag.name"
                >
                  {{ tag.name }}
                </a-option>
              </a-select>
            </div>
          </a-col>
        </a-row>
        <div v-else class="tag-filter-section__empty">
          暂无可筛选标签。重启后端完成 migration 后，这里会显示标签组。
        </div>
      </div>

      <!-- Active Filters -->
      <a-row v-if="hasActiveFilters" class="mt-3">
        <a-col :span="24">
          <a-space wrap>
            <span class="filter-label">当前筛选:</span>
            <a-tag
              v-if="route.query.search"
              closable
              @close="updateRoute({ search: undefined })"
            >
              搜索: {{ route.query.search }}
            </a-tag>
            <a-tag
              v-if="route.query.platform"
              closable
              @close="updateRoute({ platform: undefined })"
            >
              平台: {{ platformLabelMap[String(route.query.platform)] || route.query.platform }}
            </a-tag>
            <a-tag
              v-if="filterFavorites"
              closable
              @close="updateRoute({ filter: undefined })"
            >
              仅收藏
            </a-tag>
            <a-tag
              v-if="needsFilter"
              closable
              @close="updateRoute({ needs: undefined })"
            >
              待处理: {{ needsFilterLabel }}
            </a-tag>
            <a-tag
              v-for="tagId in selectedTagIds"
              :key="tagId"
              closable
              @close="removeTagFilter(tagId)"
            >
              标签: {{ tagLabelMap[String(tagId)] || tagId }}
            </a-tag>
            <a-button
              size="small"
              type="text"
              @click="clearFilters"
            >
              清除全部
            </a-button>
          </a-space>
        </a-col>
      </a-row>
    </a-card>

    <!-- Results Count -->
    <div class="results-info">
      <span class="results-count">
        显示 {{ games?.length || 0 }} / {{ pagination?.total || 0 }} 个游戏
      </span>
    </div>

    <!-- Loading State -->
    <div v-if="isLoading" class="loading-container">
      <a-spin :size="24" />
      <p class="loading-text">加载中...</p>
    </div>

    <!-- Games Grid/List -->
    <div v-else-if="games && games.length > 0">
      <!-- Grid View -->
      <div v-if="viewMode === 'grid'" class="games-grid">
        <div
          v-for="game in games"
          :key="game.id"
          class="games-grid__item"
        >
          <game-card
            :game="game"
            @view="viewGame"
            @toggle-favorite="toggleFavorite(game.id)"
            @delete="handleDelete(game.id, game.title)"
          />
        </div>
      </div>

      <!-- List View -->
      <a-row v-else :gutter="16">
        <a-col
          v-for="game in games"
          :key="game.id"
          :span="24"
        >
          <game-card
            :game="game"
            is-list
            @view="viewGame"
            @toggle-favorite="toggleFavorite(game.id)"
            @delete="handleDelete(game.id, game.title)"
          />
        </a-col>
      </a-row>

      <!-- Pagination -->
      <div v-if="totalPages > 1" class="pagination-container">
        <a-pagination
          v-model:current="currentPage"
          :total="pagination?.total || 0"
          :page-size="itemsPerPage"
          show-total
          show-jumper
          show-page-size
        />
      </div>
    </div>

    <!-- Empty State -->
    <a-empty v-else class="empty-state">
      <template #image>
        <icon-trophy :style="{ fontSize: '96px', color: 'var(--color-text-3)' }" />
      </template>
      <template #description>
        <div class="empty-description">
          <h3>暂无游戏</h3>
          <p>尝试调整筛选条件或搜索关键词</p>
        </div>
      </template>
      <a-button
        v-if="hasActiveFilters"
        type="secondary"
        @click="clearFilters"
      >
        清除筛选
      </a-button>
    </a-empty>

    <!-- Add Game Modal -->
    <add-game-modal
      v-model:visible="showAddModal"
      @submit="handleAddGameSubmit"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onActivated } from 'vue'
import { storeToRefs } from 'pinia'
import { useRoute, useRouter } from 'vue-router'
import { useGamesStore } from '@/stores/games'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import gamesService from '@/services/games.service'
import platformService from '@/services/platforms.service'
import tagsService from '@/services/tags.service'
import { getPendingIssueLabel, matchesPendingIssue } from '@/utils/pendingIssues'
import { createDetailRouteQuery } from '@/utils/navigation'
import type { Tag, TagGroup } from '@/services/types'
import GameCard from '@/components/GameCard.vue'
import AddGameModal from '@/components/AddGameModal.vue'
import { Modal, Message } from '@arco-design/web-vue'
import { IconApps, IconList, IconSort, IconTrophy, IconPlus, IconFilter } from '@arco-design/web-vue/es/icon'

defineOptions({
  name: 'GamesView',
})

interface Props {
  filter?: 'favorites' | 'recent'
}

const props = defineProps<Props>()

const route = useRoute()
const router = useRouter()
const gamesStore = useGamesStore()
const authStore = useAuthStore()
const uiStore = useUiStore()
const { isAdmin } = storeToRefs(authStore)

const isLoading = ref(false)
const searchQuery = ref('')
const viewMode = ref<'grid' | 'list'>('grid')
const showAddModal = ref(false)
const showTagFilters = ref(false)
const platformOptions = ref<{ label: string; value: string }[]>([])
const tagGroups = ref<TagGroup[]>([])
const tags = ref<Tag[]>([])

const itemsPerPageOptions = ref([
  { label: '12', value: 12 },
  { label: '24', value: 24 },
  { label: '48', value: 48 },
  { label: '96', value: 96 },
])

const sortOptions = ref([
  { label: '最新添加', value: 'created_desc' },
  { label: '最早添加', value: 'created_asc' },
  { label: '下载最多', value: 'downloads_desc' },
  { label: '浏览次数', value: 'views_desc' },
])

const visibleGames = ref<any[]>([])
const visiblePagination = ref({
  total: 0,
  page: 1,
  pageSize: 24,
  totalPages: 0,
})
const games = computed(() => visibleGames.value)
const pagination = computed(() => visiblePagination.value)
const totalPages = computed(() => pagination.value?.totalPages || 0)
const currentPage = computed({
  get: () => parseInt(route.query.page as string) || 1,
  set: (page: number) => {
    if (page !== (parseInt(route.query.page as string) || 1)) {
      updateRoute({ page: page.toString() })
    }
  },
})
const selectedPlatform = computed({
  get: () => (route.query.platform as string) || null,
  set: (platform: string | null) => {
    updateRoute({ platform })
  },
})
const sortBy = computed({
  get: () => {
    if (route.query.sort === 'newest' || route.query.sort === 'created_desc') return 'created_desc'
    if (route.query.sort === 'downloads' || route.query.sort === 'downloads_desc') return 'downloads_desc'
    if (route.query.sort === 'views' || route.query.sort === 'views_desc') return 'views_desc'
    return (route.query.sort as string) || 'created_desc'
  },
  set: (sort: string) => {
    updateRoute({ sort })
  },
})
const itemsPerPage = computed({
  get: () => parseInt(route.query.pageSize as string) || 24,
  set: (pageSize: number) => {
    updateRoute({ pageSize: pageSize.toString(), page: '1' })
  },
})

const filterFavorites = computed(() => props.filter === 'favorites' || route.query.filter === 'favorites')
const needsFilter = computed(() => (route.query.needs as string) || '')
const needsFilterLabel = computed(() => getPendingIssueLabel(needsFilter.value))
const selectedTagIds = computed(() => {
  const raw = route.query.tag
  const values = Array.isArray(raw) ? raw : raw ? [raw] : []
  return values
    .map((value) => Number(value))
    .filter((value) => !Number.isNaN(value) && value > 0)
})
const filterableTagGroups = computed(() =>
  [...tagGroups.value]
    .filter((group) => group.is_filterable)
    .sort((a, b) => a.sort_order - b.sort_order || a.id - b.id),
)

const hasActiveFilters = computed(() => {
  return searchQuery.value ||
    selectedPlatform.value ||
    selectedTagIds.value.length > 0 ||
    filterFavorites.value ||
    route.query.status === 'pending-review' ||
    !!needsFilter.value
})

const pageTitle = computed(() => {
  const filter = props.filter || route.query.filter
  if (filter === 'favorites') return '收藏的游戏'
  if (filter === 'recent') return '最近下载'
  if (needsFilter.value) return `${needsFilterLabel.value}`
  if (route.query.status === 'pending-review') return '待处理的游戏'
  return '所有游戏'
})

const platformLabelMap = computed<Record<string, string>>(() => {
  return Object.fromEntries(platformOptions.value.map((item) => [item.value, item.label]))
})

const tagLabelMap = computed<Record<string, string>>(() => {
  return Object.fromEntries(tags.value.map((item) => [String(item.id), item.name]))
})

const updateRoute = (newParams: Record<string, any>) => {
  const query = { ...route.query, ...newParams }
  // Remove undefined or null values
  Object.keys(query).forEach(key => {
    if (query[key] === undefined || query[key] === null || query[key] === '') {
      delete query[key]
    }
  })
  
  // Reset page when filters or search change
  if (newParams.search !== undefined || newParams.platform !== undefined || newParams.tag !== undefined || newParams.filter !== undefined || newParams.needs !== undefined || newParams.status !== undefined) {
    query.page = '1'
  }
  
  router.push({ name: 'games', query })
}

watch(() => route.query, () => {
  searchQuery.value = (route.query.search as string) || ''
  loadGames()
})

const viewGame = (id: string | number) => {
  router.push({
    name: 'game-detail',
    params: { id: String(id) },
    query: createDetailRouteQuery(route),
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

    // Refresh game list
    await loadGames()
  } catch (error: any) {
    uiStore.addAlert(`添加游戏失败：${error.message || '未知错误'}`, 'error')
  }
}

const toggleFavorite = async (id: number) => {
  try {
    await gamesStore.toggleFavorite(id.toString())
    uiStore.addAlert('收藏已更新', 'success')
  } catch (error) {
    uiStore.addAlert('更新收藏失败', 'error')
  }
}

const handleDelete = (id: number, title: string) => {
  if (!isAdmin.value) return
  Modal.confirm({
    title: '确认删除',
    content: `确定要删除游戏 "${title}" 吗？此操作不可撤销。`,
    okText: '删除',
    cancelText: '取消',
    okButtonProps: { status: 'danger' },
    onOk: async () => {
      try {
        await gamesService.deleteGame(id.toString())
        Message.success(`游戏 "${title}" 已删除`)
        await loadGames()
      } catch (error: any) {
        Message.error(`删除游戏失败：${error.message || '未知错误'}`)
      }
    }
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

const updateSelectedTagsForGroup = (groupId: number, value: number | number[] | string | string[] | undefined) => {
  const nextGroupValues = (Array.isArray(value) ? value : value === undefined || value === null || value === '' ? [] : [value])
    .map((item) => Number(item))
    .filter((item) => !Number.isNaN(item) && item > 0)

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

const handleTagGroupSelectionChange = (groupId: number, value: number | number[] | string | string[] | undefined) => {
  updateSelectedTagsForGroup(groupId, value)
}

const loadFilterOptions = async () => {
  try {
    const platforms = await platformService.getAllPlatforms()
    platformOptions.value = platforms
      .map((p) => ({ label: p.name, value: String(p.id) }))
      .sort((a, b) => a.label.localeCompare(b.label, 'zh-Hans-CN'))
  } catch (error) {
    console.error('Failed to load platforms:', error)
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
  }
}

const loadGames = async () => {
  isLoading.value = true

  // Use current values which are synced from route query
  const page = parseInt(route.query.page as string) || 1

  // Parse sort value and map to backend fields
  const [field, order] = sortBy.value.split('_')
  const sortFieldMap: Record<string, string> = {
    created: 'created_at',
    downloads: 'downloads',
    views: 'views',
  }
  const filter = {
    search: (route.query.search as string) || undefined,
    platform: (route.query.platform as string) || undefined,
    tag_ids: selectedTagIds.value,
    favorite: filterFavorites.value || undefined,
    status: (route.query.status as string) || undefined,
  }
  const sort = {
    field: (sortFieldMap[field] || 'created_at') as any,
    order: (order || 'desc') as any,
  }

  try {
    if (needsFilter.value) {
      const response = await gamesService.getGames({
        page: 1,
        pageSize: 200,
        filter: {
          ...filter,
          status: 'pending-review',
        },
        sort,
      })
      const detailedGames = await Promise.all(
        response.data.map(async (item) => {
          try {
            return await gamesService.getGame(String(item.id))
          } catch {
            return item
          }
        }),
      )
      const filteredGames = detailedGames.filter((game) => matchesPendingIssue(game, needsFilter.value))
      visibleGames.value = filteredGames
      visiblePagination.value = {
        total: filteredGames.length,
        page: 1,
        pageSize: filteredGames.length || itemsPerPage.value,
        totalPages: 1,
      }
    } else {
      const response = await gamesStore.fetchGames({
        page,
        pageSize: itemsPerPage.value,
        filter,
        sort,
      })
      visibleGames.value = response.data
      visiblePagination.value = {
        total: response.pagination.total,
        page: response.pagination.page,
        pageSize: response.pagination.limit,
        totalPages: response.pagination.totalPages,
      }
    }
  } catch (error) {
    uiStore.addAlert('加载游戏失败', 'error')
  } finally {
    isLoading.value = false
  }
}

onMounted(async () => {
  // Initialize view mode from store
  viewMode.value = uiStore.gamesViewMode

  await loadFilterOptions()

  // Initialize WebSocket for real-time updates
  gamesStore.initializeWebSocket()

  searchQuery.value = (route.query.search as string) || ''
  // Only load if games list is empty or if we have specific queries
  if (games.value.length === 0 || Object.keys(route.query).length > 0) {
    loadGames()
  }
})

// Handle keep-alive activation
onActivated(async () => {
  await loadFilterOptions()
})

// Watch search query for auto-search (with debounce and URL sync)
let searchDebounceTimer: number | undefined
watch(searchQuery, (newQuery, oldQuery) => {
  if (newQuery === oldQuery) return
  
  if (searchDebounceTimer) {
    clearTimeout(searchDebounceTimer)
  }
  
  searchDebounceTimer = window.setTimeout(() => {
    // Check if newQuery is actually different from route.query.search to avoid redundant push
    if (newQuery !== (route.query.search || '')) {
      handleSearch()
    }
  }, 500)
})

// Watch view mode changes
watch(viewMode, (value) => {
  uiStore.setGamesViewMode(value)
})
</script>

<style scoped>
.games-view {
  animation: fadeInUp 0.4s cubic-bezier(0.2, 0.8, 0.2, 1) forwards;
}

@keyframes fadeInUp {
  from {
    opacity: 0;
    transform: translateY(15px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.search-card {
  border-radius: var(--radius-lg);
  margin-bottom: 24px;
}

.search-card :deep(.arco-card-body) {
  padding: 16px 20px;
}

.view-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.view-title {
  font-size: 32px;
  font-weight: 800;
  margin: 0;
  letter-spacing: -0.5px;
}

.text-gradient {
  background: linear-gradient(135deg, var(--color-primary-light-3), var(--color-primary-6));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

.mb-4 {
  margin-bottom: 16px;
}

.mt-3 {
  margin-top: 12px;
}

.games-filter-drawer-btn {
  justify-content: center;
}

.tag-filter-section {
  margin-top: 16px;
  padding-top: 12px;
  border-top: 1px solid var(--color-border-2);
}

.tag-filter-section__header {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  gap: 12px;
  flex-wrap: wrap;
}

.tag-filter-section__title {
  font-size: 14px;
  font-weight: 700;
  color: var(--color-text-1);
}

.tag-filter-section__hint,
.tag-filter-section__empty {
  font-size: 12px;
  color: var(--color-text-3);
}

.games-filter-drawer-btn__count {
  display: inline-flex;
  min-width: 18px;
  height: 18px;
  padding: 0 6px;
  border-radius: 999px;
  align-items: center;
  justify-content: center;
  background: var(--color-primary-light-1);
  color: rgb(var(--primary-6));
  font-size: 12px;
  font-weight: 700;
}

.tag-filter-drawer {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.tag-filter-grid-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.tag-filter-drawer__group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.tag-filter-drawer__label {
  font-size: 13px;
  font-weight: 700;
  color: var(--color-text-2);
}

.results-info {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.results-count {
  color: var(--color-text-3);
  font-size: 14px;
}

.filter-label {
  color: var(--color-text-3);
  font-size: 14px;
}

.loading-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 48px 0;
}

.loading-text {
  color: var(--color-text-3);
  margin-top: 16px;
  margin-bottom: 0;
}

.pagination-container {
  display: flex;
  justify-content: center;
  margin-top: 24px;
}

.games-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 16px;
}

.games-grid__item {
  min-width: 0;
}

.empty-state {
  padding: 48px 0;
}

.empty-description h3 {
  font-size: 16px;
  margin: 16px 0 8px;
  color: var(--color-text-1);
}

.empty-description p {
  color: var(--color-text-3);
  margin: 0;
}

@media (max-width: 1199px) {
  .games-grid {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }
}

@media (max-width: 991px) {
  .games-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (max-width: 767px) {
  .games-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (min-width: 1200px) {
  .games-grid {
    grid-template-columns: repeat(6, minmax(0, 1fr));
  }
}

@media (min-width: 1600px) {
  .games-grid {
    grid-template-columns: repeat(8, minmax(0, 1fr));
  }
}

@media (min-width: 2200px) {
  .games-grid {
    grid-template-columns: repeat(12, minmax(0, 1fr));
  }
}
</style>
