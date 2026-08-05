<template>
  <div class="dashboard">
    <!-- Welcome Section -->
    <div v-if="isDashboardReady && !loadFailed" class="dashboard-section-title page-hero">
      <div class="page-hero__content">
        <h1 class="page-hero__title text-gradient">
          发现
        </h1>
        <p class="page-hero__subtitle">
          你的游戏库，此刻正在发生什么。
        </p>
      </div>
    </div>

    <a-alert
      v-if="isDashboardReady && !loadFailed && refreshFailedWithStaleData"
      class="dashboard-refresh-warning"
      type="warning"
      show-icon
    >
      仪表盘刷新失败，当前显示的是上次成功加载的数据。
    </a-alert>

    <!-- Initial Skeleton -->
    <div v-if="isLoading && !isDashboardReady" class="dashboard-skeleton">
      <a-skeleton :animation="true" class="dashboard-skeleton__hero">
        <a-skeleton-shape class="dashboard-skeleton__hero-shape" />
        <a-skeleton-line :rows="2" />
      </a-skeleton>
      <a-skeleton :animation="true" class="dashboard-skeleton__stats">
        <a-skeleton-shape class="dashboard-skeleton__stat" />
        <a-skeleton-shape class="dashboard-skeleton__stat" />
        <a-skeleton-shape class="dashboard-skeleton__stat" />
        <a-skeleton-shape class="dashboard-skeleton__stat" />
      </a-skeleton>
      <a-skeleton :animation="true" class="dashboard-skeleton__rows">
        <a-skeleton-line :rows="4" />
      </a-skeleton>
    </div>

    <template v-else-if="isDashboardReady && !loadFailed">
      <dashboard-hero
        class="dashboard-hero-section"
        :games="carouselGames"
        :is-admin="isAdmin"
        :pending-reviews="pendingReviews"
        @enter-store="router.push({ name: 'game-store' })"
        @browse-games="router.push({ name: 'games' })"
        @add-game="showAddGameModal = true"
        @open-pending="router.push({ name: 'pending-center' })"
      />

      <stat-overview
        class="dashboard-stats-section"
        :total-games="totalGames"
        :total-downloads="totalDownloads"
        :favorite-count="favoriteCount"
        :pending-reviews="pendingReviews"
      />

      <a-divider class="dashboard-divider" />

      <game-row-section
        v-if="recentAdditions.length > 0"
        title="最近添加"
        icon="mdi-new-box"
        :items="recentAdditions"
        view-all-route="/games?sort=created_at&order=desc"
        @view="viewGame"
        @view-series="viewSeries"
        @toggle-favorite="toggleFavorite"
      />

      <game-row-section
        v-if="mostPlayed.length > 0"
        title="下载最多"
        icon="mdi-download"
        :items="mostPlayed"
        view-all-route="/games?sort=downloads&order=desc"
        @view="viewGame"
        @view-series="viewSeries"
        @toggle-favorite="toggleFavorite"
      />

      <game-row-section
        v-if="favoriteGames.length > 0"
        title="我的收藏"
        icon="mdi-heart"
        :items="favoriteGames"
        view-all-route="/games?favorite=true"
        @view="viewGame"
        @view-series="viewSeries"
        @toggle-favorite="toggleFavorite"
      />

      <game-row-section
        v-if="recentlyUpdated.length > 0"
        title="最近更新"
        icon="mdi-update"
        :items="recentlyUpdated"
        view-all-route="/games"
        @view="viewGame"
        @view-series="viewSeries"
        @toggle-favorite="toggleFavorite"
      />

      <pending-overview
        v-if="isAdmin"
        :pending-reviews="pendingReviews"
        :groups="pendingGroups"
        @open-pending="router.push({ name: 'pending-center' })"
      />

      <!-- Empty State -->
      <div v-if="isEmpty" class="dashboard-empty">
        <icon-trophy class="dashboard-empty-icon" />
        <h2 class="dashboard-empty-title">还没有游戏</h2>
        <p class="dashboard-empty-text">
          添加一些游戏到您的库中
        </p>
        <a-button
          type="primary"
          size="large"
          @click="router.push('/games')"
        >
          浏览游戏
        </a-button>
      </div>
    </template>

    <a-empty
      v-else-if="isDashboardReady && loadFailed"
      description="仪表盘加载失败，请稍后重试。"
    >
      <a-button type="primary" @click="loadDashboardData">
        重新加载
      </a-button>
    </a-empty>
  </div>

  <add-game-modal
    v-model:visible="showAddGameModal"
    :submitting="addGameSubmitting"
    @submit="handleAddGameSubmit"
  />
</template>

<script setup lang="ts">
import { computed, onActivated, onMounted, ref } from 'vue'
import { storeToRefs } from 'pinia'
import { useRouter } from 'vue-router'
import { IconTrophy } from '@arco-design/web-vue/es/icon'
import AddGameModal from '@/components/AddGameModal.vue'
import DashboardHero from '@/components/dashboard/DashboardHero.vue'
import StatOverview from '@/components/dashboard/StatOverview.vue'
import GameRowSection from '@/components/dashboard/GameRowSection.vue'
import PendingOverview from '@/components/dashboard/PendingOverview.vue'
import gamesService from '@/services/games.service'
import type { GameListItem } from '@/services/types'
import { getHttpErrorMessage } from '@/utils/http-error'
import { useAuthStore } from '@/stores/auth'
import { useGamesStore } from '@/stores/games'
import { useUiStore } from '@/stores/ui'

defineOptions({
  name: 'DashboardView',
})

const router = useRouter()
const gamesStore = useGamesStore()
const uiStore = useUiStore()
const authStore = useAuthStore()
const { isAdmin } = storeToRefs(authStore)

const isLoading = ref(false)
const isDashboardReady = ref(false)
const loadFailed = ref(false)
const refreshFailedWithStaleData = ref(false)
const lastLoadedAt = ref(0)

const showAddGameModal = ref(false)
const addGameSubmitting = ref(false)

const totalGames = computed(() => gamesStore.stats?.total_games ?? 0)
const totalDownloads = computed(() => gamesStore.stats?.total_downloads ?? 0)
const recentAdditions = computed(() => gamesStore.stats?.recent_games ?? [])
const mostPlayed = computed(() => gamesStore.stats?.popular_games ?? [])
const favoriteGames = computed(() => gamesStore.stats?.favorite_games ?? [])
const favoriteCount = computed(() => gamesStore.stats?.favorite_count ?? 0)
const pendingReviews = computed(() => gamesStore.stats?.pending_reviews ?? 0)
const pendingGroups = computed(() => gamesStore.stats?.pending_issue_counts ?? null)

// “最近完善”与“最近添加”去重：同一款游戏优先出现在最近添加，避免两行重复。
const recentlyUpdated = computed(() => {
  const recentIds = new Set(recentAdditions.value.map(game => game.id))
  return (gamesStore.stats?.recently_updated_games ?? []).filter(game => !recentIds.has(game.id))
})

const isEmpty = computed(() => totalGames.value === 0)

// 轮播保持稳定：优先下载最多，不足 5 个时用最近添加补齐，不再随机打乱。
const carouselGames = computed(() => {
  const seen = new Set<number>()
  const games: GameListItem[] = []
  for (const game of [...mostPlayed.value, ...recentAdditions.value]) {
    if (seen.has(game.id)) {
      continue
    }
    seen.add(game.id)
    games.push(game)
    if (games.length >= 5) {
      break
    }
  }
  return games
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

const toggleFavorite = async (gameRef: string) => {
  if (!gameRef) return
  try {
    await gamesStore.toggleFavorite(gameRef)
    uiStore.addAlert('收藏已更新', 'success')
  } catch {
    uiStore.addAlert('更新收藏失败', 'error')
  }
}

const loadDashboardData = async () => {
  isLoading.value = true
  isDashboardReady.value = false
  loadFailed.value = false
  refreshFailedWithStaleData.value = false
  try {
    await gamesStore.fetchStats()
    isDashboardReady.value = true
    lastLoadedAt.value = Date.now()
  } catch {
    uiStore.addAlert('加载数据失败', 'error')
    // 2026-04-08: dashboard refresh failures keep last good stats visible, but must not
    // pretend the current refresh succeeded. Distinguish stale-data rendering from both
    // first-load failure and successful refresh.
    refreshFailedWithStaleData.value = gamesStore.stats !== null
    loadFailed.value = gamesStore.stats === null
    isDashboardReady.value = true
  } finally {
    isLoading.value = false
  }
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
    showAddGameModal.value = false
    await loadDashboardData()
  } catch (error) {
    uiStore.addAlert(`添加游戏失败：${getHttpErrorMessage(error)}`, 'error')
  } finally {
    addGameSubmitting.value = false
  }
}

onMounted(async () => {
  await loadDashboardData()
})

onActivated(async () => {
  if (Date.now() - lastLoadedAt.value > 30000) {
    await loadDashboardData()
    return
  }
})
</script>

<style scoped>
.dashboard {
  position: relative;
  z-index: 2;
  padding-bottom: 24px;
}

.dashboard-section-title {
  margin-bottom: 24px;
}

.dashboard-hero-section {
  margin-bottom: 24px;
}

.dashboard-stats-section {
  margin-bottom: 8px;
}

.dashboard-skeleton {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.dashboard-skeleton__hero {
  display: block;
}

.dashboard-skeleton__hero-shape {
  display: block;
  width: 100%;
  height: 320px;
  margin-bottom: 16px;
}

.dashboard-skeleton__stats {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 16px;
}

.dashboard-skeleton__stat {
  width: 100%;
  height: 104px;
}

.dashboard-skeleton__rows {
  display: block;
}

.dashboard-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 48px 24px;
  text-align: center;
}

.dashboard-empty-icon {
  font-size: 96px;
  color: var(--color-text-3);
}

.dashboard-empty-title {
  font-size: 20px;
  font-weight: 600;
  margin: 16px 0 8px;
  color: var(--color-text-1);
}

.dashboard-empty-text {
  color: var(--color-text-3);
  margin: 0 0 24px;
}

@media (max-width: 992px) {
  .dashboard-skeleton__stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 576px) {
  .dashboard-skeleton__stats {
    grid-template-columns: 1fr;
  }
}
</style>
