<template>
  <div class="games-view">
    <!-- Header -->
    <div class="view-header">
      <div class="view-header-title-group">
        <h1 class="view-title text-gradient">{{ pageTitle }}</h1>
        <p class="view-subtitle">你收藏的每一款，都在这里。</p>
      </div>

      <a-space>
        <a-radio-group v-model="viewMode" type="button" size="medium">
          <a-radio value="grid">
            <icon-apps />
          </a-radio>
          <a-radio value="list">
            <icon-list />
          </a-radio>
        </a-radio-group>

        <a-button v-if="isAdmin" type="primary" @click="handleAddGame">
          <template #icon>
            <icon-plus />
          </template>
          添加游戏
        </a-button>
      </a-space>
    </div>

    <!-- Search and Filters -->
    <a-card class="mb-4 search-card app-glass-surface" :bordered="false">
      <a-row :gutter="[12, 12]" class="games-filters-row">
        <!-- Search -->
        <a-col :xs="24" :sm="12" :md="6" :lg="6" :xl="6" :xxl="5" class="games-filters-col games-filters-col--search">
          <div class="app-input-action-row">
            <a-input
              v-model="searchQuery"
              class="app-input-action-row__field"
              placeholder="搜索游戏"
              allow-clear
              @press-enter="handleSearch"
            >
              <template #prefix>
                <icon-search />
              </template>
            </a-input>
          </div>
        </a-col>

        <!-- Sort -->
        <a-col :xs="24" :sm="8" :md="6" :lg="6" :xl="6" :xxl="6" class="games-filters-col games-filters-col--sort">
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

        <!-- Favorites Toggle -->
        <a-col :xs="24" :sm="8" :md="1" :lg="1" :xl="1" :xxl="1" class="games-filters-col games-filters-col--favorite">
          <a-tag
            checkable
            :checked="filterFavorites"
            color="red"
            class="favorite-toggle-tag"
            @click="updateRoute({ favorite: filterFavorites ? undefined : 'true' })"
          >
            <icon-heart-fill v-if="filterFavorites" />
            <icon-heart v-else />
            收藏
          </a-tag>
        </a-col>

        <!-- Private Toggle (admin only) -->
        <a-col v-if="isAdmin" :xs="24" :sm="8" :md="1" :lg="1" :xl="1" :xxl="1" class="games-filters-col games-filters-col--favorite">
          <a-tag
            checkable
            :checked="filterPrivate"
            color="orangered"
            class="favorite-toggle-tag"
            @click="updateRoute({ visibility: filterPrivate ? undefined : 'private' })"
          >
            <icon-lock />
            私有
          </a-tag>
        </a-col>
      </a-row>

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
              v-if="filterFavorites"
              closable
              @close="updateRoute({ favorite: undefined })"
            >
              仅收藏
            </a-tag>
            <a-tag
              v-if="filterPrivate && isAdmin"
              closable
              @close="updateRoute({ visibility: undefined })"
            >
              仅私有
            </a-tag>
            <a-button
              class="app-text-action-btn"
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

    <!-- Loading State -->
    <div v-if="isLoading" class="loading-container">
      <a-spin :size="24" />
      <p class="loading-text">加载中...</p>
    </div>

    <!-- Virtualized Games Grid/List -->
    <div v-else-if="games && games.length > 0">
      <div ref="virtualScrollRef" class="games-virtual-scroll">
        <div
          class="games-virtual-scroll__canvas"
          :style="{ height: `${virtualTotalHeight}px` }"
        >
          <div
            v-for="virtualItem in virtualItems"
            :key="virtualItem.key"
            class="games-virtual-scroll__item"
            :style="virtualItem.style"
          >
            <game-card
              :game="virtualItem.game"
              :is-list="viewMode === 'list'"
              can-add-to-start-screen
              can-delete
              :is-on-start-screen="startScreenGameIds.has(virtualItem.game.id)"
              @view="viewGame"
              @view-series="viewSeries"
              @toggle-favorite="toggleFavorite"
              @delete="handleDelete($event, virtualItem.game.title)"
              @add-to-start-screen="handleAddToStartScreen"
              @remove-from-start-screen="handleRemoveFromStartScreen"
            />
          </div>
        </div>

        <div
          ref="loadMoreSentinel"
          class="games-infinite-scroll"
        >
          <a-spin v-if="isLoadingMore" :size="20" />
        </div>
      </div>
    </div>

    <a-empty v-else-if="hasLoadFailure" class="empty-state">
      <template #image>
        <icon-trophy :style="{ fontSize: '96px', color: 'var(--color-text-3)' }" />
      </template>
      <template #description>
        <div class="empty-description">
          <h3>加载游戏失败</h3>
          <p>当前列表请求没有成功返回，请稍后重试。</p>
        </div>
      </template>
    </a-empty>

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
        class="app-text-action-btn"
        type="text"
        @click="clearFilters"
      >
        清除筛选
      </a-button>
    </a-empty>

    <!-- Add Game Modal -->
    <add-game-modal
      v-model:visible="showAddModal"
      :submitting="addGameSubmitting"
      @submit="handleAddGameSubmit"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onActivated, onBeforeUnmount, onDeactivated, onMounted, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useRoute, useRouter } from 'vue-router'
import { useGamesStore } from '@/stores/games'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import GameCard from '@/components/GameCard.vue'
import AddGameModal from '@/components/AddGameModal.vue'
import { useGamesView } from '@/composables/useGamesView'
import startScreenService from '@/services/start-screen.service'
import type { GameListItem, StartScreenLayout } from '@/services/types'
import { getHttpErrorMessage } from '@/utils/http-error'
import { createRequestGeneration } from '@/utils/request-generation'
import { IconApps, IconHeart, IconHeartFill, IconList, IconLock, IconPlus, IconSearch, IconSort, IconTrophy } from '@arco-design/web-vue/es/icon'

defineOptions({
  name: 'GamesView',
})

const route = useRoute()
const router = useRouter()
const gamesStore = useGamesStore()
const authStore = useAuthStore()
const uiStore = useUiStore()
const { isAdmin } = storeToRefs(authStore)

const startScreenGameIds = ref<Set<number>>(new Set())
const loadMoreSentinel = ref<HTMLElement | null>(null)
const virtualScrollRef = ref<HTMLElement | null>(null)
const virtualViewportWidth = ref(0)
const virtualViewportHeight = ref(0)
const virtualScrollTop = ref(0)
let virtualResizeObserver: ResizeObserver | null = null
let virtualScrollFrame = 0
let virtualScrollBound = false
let virtualScrollRoot: HTMLElement | null = null
const startScreenRequests = createRequestGeneration()

const {
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
  pageTitle,
  searchQuery,
  showAddModal,
  sortBy,
  sortOptions,
  toggleFavorite,
  updateRoute,
  viewGame,
  viewSeries,
  viewMode,
} = useGamesView({
  route,
  router,
  gamesStore,
  uiStore,
  isAdmin,
  loadMoreSentinel,
})

const VIRTUAL_GAP = 16
const VIRTUAL_BUFFER_ROWS = 2

interface VirtualGameItem {
  key: string
  game: GameListItem
  style: Record<string, string>
}

const virtualColumns = computed(() => {
  const width = virtualViewportWidth.value || 1200
  if (width < 768) return 2
  if (width < 992) return 3
  if (width < 1200) return 4
  if (width < 1600) return 6
  if (width < 2200) return 8
  return 12
})

const virtualCardWidth = computed(() => {
  if (viewMode.value === 'list') {
    return virtualViewportWidth.value
  }
  const columns = virtualColumns.value
  const width = virtualViewportWidth.value || 1200
  return Math.max(0, (width - VIRTUAL_GAP * (columns - 1)) / columns)
})

const virtualRowHeight = computed(() => {
  if (viewMode.value === 'list') {
    return virtualViewportWidth.value > 0 && virtualViewportWidth.value <= 768 ? 300 : 110
  }
  return virtualCardWidth.value * (4 / 3) + 76
})

const virtualRowCount = computed(() => {
  if (viewMode.value === 'list') {
    return games.value.length
  }
  return Math.ceil(games.value.length / virtualColumns.value)
})

const virtualTotalHeight = computed(() => {
  if (virtualRowCount.value === 0) {
    return 0
  }
  return virtualRowCount.value * (virtualRowHeight.value + VIRTUAL_GAP) + VIRTUAL_GAP
})

const virtualStartRow = computed(() => {
  const stride = virtualRowHeight.value + VIRTUAL_GAP
  return Math.max(0, Math.floor(virtualScrollTop.value / stride) - VIRTUAL_BUFFER_ROWS)
})

const virtualEndRow = computed(() => {
  const stride = virtualRowHeight.value + VIRTUAL_GAP
  const viewportHeight = virtualViewportHeight.value || 800
  const visibleEnd = Math.ceil((virtualScrollTop.value + viewportHeight) / stride) + VIRTUAL_BUFFER_ROWS
  return Math.min(virtualRowCount.value, visibleEnd)
})

const virtualItems = computed<VirtualGameItem[]>(() => {
  const stride = virtualRowHeight.value + VIRTUAL_GAP
  const items: VirtualGameItem[] = []

  // 防御：keep-alive 恢复瞬间测量未就绪时 startRow 可能越过 endRow，
  // 直接回退到第一屏，避免整个列表空白。
  const startRow = Math.min(virtualStartRow.value, Math.max(0, virtualEndRow.value - 1))

  if (viewMode.value === 'list') {
    for (let index = startRow; index < virtualEndRow.value; index += 1) {
      const game = games.value[index]
      if (!game) continue
      items.push({
        key: `list-${game.id}`,
        game,
        style: {
          width: `${virtualCardWidth.value}px`,
          height: `${virtualRowHeight.value}px`,
          transform: `translate3d(0, ${index * stride}px, 0)`,
        },
      })
    }
    return items
  }

  const columns = virtualColumns.value
  for (let row = startRow; row < virtualEndRow.value; row += 1) {
    for (let column = 0; column < columns; column += 1) {
      const gameIndex = row * columns + column
      const game = games.value[gameIndex]
      if (!game) break
      items.push({
        key: `grid-${game.id}`,
        game,
        style: {
          width: `${virtualCardWidth.value}px`,
          height: `${virtualRowHeight.value}px`,
          transform: `translate3d(${column * (virtualCardWidth.value + VIRTUAL_GAP)}px, ${row * stride}px, 0)`,
        },
      })
    }
  }

  return items
})

const syncStartScreenLayout = (layout: StartScreenLayout) => {
  startScreenGameIds.value = new Set(layout.tiles.map((tile) => tile.game_id))
}

const refreshStartScreenTiles = async () => {
  if (!isAdmin.value) {
    startScreenRequests.invalidate()
    return
  }
  const request = startScreenRequests.begin()
  try {
    const layout = await startScreenService.getTiles()
    if (request.isCurrent() && isAdmin.value) {
      syncStartScreenLayout(layout)
    }
  } catch {
    // 开始屏幕状态加载失败时保持当前状态，不打断游戏库浏览
  }
}

const handleAddToStartScreen = async (gameId: number) => {
  if (!isAdmin.value) return
  const request = startScreenRequests.begin()
  try {
    const layout = await startScreenService.addTile(gameId, 'small')
    if (request.isCurrent() && isAdmin.value) {
      syncStartScreenLayout(layout)
      uiStore.addAlert('已添加到开始屏幕', 'success')
    }
  } catch (error) {
    if (request.isCurrent()) {
      uiStore.addAlert(`添加到开始屏幕失败：${getHttpErrorMessage(error, '未知错误')}`, 'error')
    }
  }
}

const handleRemoveFromStartScreen = async (gameId: number) => {
  if (!isAdmin.value) return
  const request = startScreenRequests.begin()
  try {
    const layout = await startScreenService.removeTile(gameId)
    if (request.isCurrent() && isAdmin.value) {
      syncStartScreenLayout(layout)
      uiStore.addAlert('已从开始菜单移除', 'success')
    }
  } catch (error) {
    if (request.isCurrent()) {
      uiStore.addAlert(`从开始菜单移除失败：${getHttpErrorMessage(error, '未知错误')}`, 'error')
    }
  }
}

const updateVirtualMetrics = () => {
  const root = virtualScrollRoot || virtualScrollRef.value
  if (!root) return
  virtualViewportWidth.value = virtualScrollRef.value?.clientWidth || root.clientWidth
  virtualViewportHeight.value = root.clientHeight
}

const handleVirtualScroll = () => {
  if (virtualScrollFrame) return
  virtualScrollFrame = window.requestAnimationFrame(() => {
    virtualScrollFrame = 0
    virtualScrollTop.value = virtualScrollRoot?.scrollTop || 0
  })
}

const setupVirtualScroll = () => {
  const element = virtualScrollRef.value
  if (!element) {
    return
  }
  if (virtualScrollBound) {
    updateVirtualMetrics()
    return
  }
  virtualScrollRoot = element.closest<HTMLElement>('.content') || element
  virtualScrollRoot.addEventListener('scroll', handleVirtualScroll, { passive: true })
  virtualScrollBound = true
  updateVirtualMetrics()
  if (typeof ResizeObserver === 'undefined') {
    return
  }
  virtualResizeObserver = new ResizeObserver(() => {
    updateVirtualMetrics()
  })
  virtualResizeObserver.observe(virtualScrollRoot)
}

const teardownVirtualScroll = () => {
  virtualScrollBound = false
  virtualScrollRoot?.removeEventListener('scroll', handleVirtualScroll)
  virtualScrollRoot = null
  if (virtualResizeObserver) {
    virtualResizeObserver.disconnect()
    virtualResizeObserver = null
  }
  if (virtualScrollFrame) {
    window.cancelAnimationFrame(virtualScrollFrame)
    virtualScrollFrame = 0
  }
}

onActivated(() => {
  void refreshStartScreenTiles()
  // keep-alive 恢复：重绑滚动监听。用双 rAF 等布局完全展开（列表总高就绪）后再
  // 量测并恢复滚动位置——单 nextTick 时 canvas 高度未定型，scrollTo 会被夹在矮高度上。
  // 恢复后再读回容器实际 scrollTop 作为渲染基准：若被外部（浏览器历史恢复等）覆盖，
  // 虚拟列表与可视区仍保持对齐，不会出现顶部白屏。
  setupVirtualScroll()
  const restore = () => {
    updateVirtualMetrics()
    const requested = virtualScrollTop.value
    virtualScrollRoot?.scrollTo({ top: requested })
    requestAnimationFrame(() => {
      virtualScrollTop.value = virtualScrollRoot?.scrollTop ?? 0
    })
  }
  requestAnimationFrame(() => {
    requestAnimationFrame(restore)
  })
})

onDeactivated(() => {
  teardownVirtualScroll()
})

onMounted(() => {
  setupVirtualScroll()
})

watch(
  () => JSON.stringify(route.query),
  (current, previous) => {
    // 仅在筛选/排序等 query 实际变化时重置滚动；keep-alive 返回时 query 未变，需保留浏览位置。
    if (current === previous) return
    virtualScrollTop.value = 0
    virtualScrollRoot?.scrollTo({ top: 0 })
    updateVirtualMetrics()
  },
)

watch(viewMode, () => {
  virtualScrollTop.value = 0
  updateVirtualMetrics()
  virtualScrollRoot?.scrollTo({ top: 0 })
})

watch(virtualScrollRef, (element) => {
  virtualScrollBound = false
  virtualScrollRoot = null
  if (virtualResizeObserver) {
    virtualResizeObserver.disconnect()
    virtualResizeObserver = null
  }
  if (element) {
    setupVirtualScroll()
  }
})

onBeforeUnmount(teardownVirtualScroll)
onBeforeUnmount(() => {
  startScreenRequests.invalidate()
})
</script>

<style scoped src="./GamesView.css"></style>
