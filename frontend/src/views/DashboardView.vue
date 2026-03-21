<template>
  <div class="dashboard" :class="{ 'is-ready': isDashboardReady }">
    <!-- Welcome Section (Slimmer Header) -->
    <div v-show="isDashboardReady" class="dashboard-section-title page-hero">
      <div class="page-hero__content">
        <h1 class="page-hero__title text-gradient">
          发现
        </h1>
        <p class="page-hero__subtitle">
          探索您的游戏库动态
        </p>
      </div>
    </div>

    <!-- Top Hero Section -->
    <a-row v-show="isDashboardReady" :gutter="[16, 16]" class="dashboard-hero-section">
      <a-col :xs="24" :sm="24" :md="24" :lg="17" :xl="17">
        <game-carousel
          v-if="carouselGames.length > 0"
          :games="carouselGames"
          :auto-play="true"
          :interval="5000"
        />
      </a-col>
      <a-col :xs="24" :sm="24" :md="24" :lg="7" :xl="7">
        <div class="dashboard-stats-grid">
          <stat-card
            title="游戏总数"
            :value="totalGames"
            icon="mdi-gamepad-variant"
            color="#1a9fff"
            :height="104"
            @click="router.push('/games')"
          />

          <stat-card
            title="收藏"
            :value="favoriteCount"
            icon="mdi-heart"
            color="#f53f3f"
            :height="104"
            @click="router.push('/games?filter=favorites')"
          />

          <stat-card
            title="新入库"
            :value="recentAdditions.length"
            icon="mdi-new-box"
            color="#00b42a"
            :height="104"
            @click="router.push('/games?sort=newest')"
          />

          <stat-card
            title="待处理"
            :value="pendingReviews"
            icon="mdi-clock"
            color="#ff7d00"
            :height="104"
            @click="router.push('/games/pending')"
          />
        </div>
      </a-col>
    </a-row>

    <!-- Divider between stats and content -->
    <a-divider v-show="isDashboardReady" class="dashboard-divider" />

    <!-- Recently Added -->
    <card-row
      v-show="isDashboardReady"
      v-if="recentAdditions.length > 0"
      title="最近添加"
      icon="mdi-new-box"
      :items="recentAdditions"
      :show-view-all="true"
      view-all-route="/games?sort=newest"
    >
      <template #item="{ item }">
        <game-card
          :game="item"
          @view="viewGame"
          @toggle-favorite="toggleFavorite(item.id)"
        />
      </template>
    </card-row>

    <!-- Most Downloaded -->
    <card-row
      v-show="isDashboardReady"
      v-if="mostPlayed.length > 0"
      title="下载最多"
      icon="mdi-download"
      :items="mostPlayed"
      :show-view-all="true"
      view-all-route="/games?sort=downloads"
    >
      <template #item="{ item }">
        <game-card
          :game="item"
          @view="viewGame"
          @toggle-favorite="toggleFavorite(item.id)"
        />
      </template>
    </card-row>

    <!-- Empty State -->
    <div v-show="isDashboardReady" v-if="isEmpty" class="dashboard-empty">
      <icon-trophy class="dashboard-empty-icon" />
      <h2 class="dashboard-empty-title">还没有游戏</h2>
      <p class="dashboard-empty-text">
        添加一些游戏到您的库中
      </p>
      <a-button
        type="primary"
        size="large"
        class="dashboard-empty-button"
        @click="router.push('/games')"
      >
        浏览游戏
      </a-button>
    </div>

  </div>

  <div
    v-if="showSplash"
    class="dashboard-loading"
    :class="{ 'dashboard-loading--leaving': isSplashLeaving }"
  >
    <div class="dashboard-loading__backdrop" />
    <div class="dashboard-loading__content">
      <p class="dashboard-loading__eyebrow">WELCOME TO</p>
      <h1 class="dashboard-loading__brand">GameAtlas</h1>
      <div class="dashboard-loading__track">
        <span class="dashboard-loading__bar" />
      </div>
    </div>
  </div>

</template>

<script setup lang="ts">
import { ref, computed, onMounted, onActivated } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useGamesStore } from '@/stores/games'
import { useUiStore } from '@/stores/ui'
import { createDetailRouteQuery } from '@/utils/navigation'
import {
  IconTrophy
} from '@arco-design/web-vue/es/icon'
import StatCard from '@/components/StatCard.vue'
import CardRow from '@/components/CardRow.vue'
import GameCard from '@/components/GameCard.vue'
import GameCarousel from '@/components/GameCarousel.vue'
import type { Game } from '@/services/types'

defineOptions({
  name: 'DashboardView',
})

const DASHBOARD_SPLASH_STORAGE_KEY = 'dashboard-splash-played'

const hasPlayedDashboardSplash = () => {
  if (typeof window === 'undefined') return false
  return window.sessionStorage.getItem(DASHBOARD_SPLASH_STORAGE_KEY) === '1'
}

const markDashboardSplashPlayed = () => {
  if (typeof window === 'undefined') return
  window.sessionStorage.setItem(DASHBOARD_SPLASH_STORAGE_KEY, '1')
}

const route = useRoute()
const router = useRouter()
const gamesStore = useGamesStore()
const uiStore = useUiStore()

const isLoading = ref(false)
const isDashboardReady = ref(false)
const showSplash = ref(!hasPlayedDashboardSplash())
const isSplashLeaving = ref(false)
const splashStartedAt = ref(0)

const MIN_SPLASH_DURATION = 1200
const SPLASH_FADE_DURATION = 400

// Directly use gamesStore.stats (it's already a ref)
const totalGames = computed(() => gamesStore.stats?.total_games || 0)
const recentAdditions = computed(() => gamesStore.stats?.recent_games || [])
const mostPlayed = computed(() => gamesStore.stats?.popular_games || [])
const favoriteCount = computed(() => gamesStore.stats?.favorite_count || 0)

const isEmpty = computed(() => {
  return recentAdditions.value.length === 0
})

// Get games for carousel (combine recent and most played, shuffle them)
const carouselGames = computed(() => {
  const seen = new Set<number>()

  return [...recentAdditions.value, ...mostPlayed.value]
    .filter((game): game is Game => {
      if (!game || seen.has(game.id)) {
        return false
      }
      seen.add(game.id)
      return true
    })
    .sort(() => Math.random() - 0.5)
})

const pendingReviewGameCount = ref(0)
const lastLoadedAt = ref(0)

const pendingReviews = computed(() => pendingReviewGameCount.value)

const viewGame = (id: string | number) => {
  router.push({
    name: 'game-detail',
    params: { id: String(id) },
    query: createDetailRouteQuery(route),
  })
}

const toggleFavorite = async (id: string) => {
  try {
    await gamesStore.toggleFavorite(id)
    uiStore.addAlert('收藏已更新', 'success')
  } catch {
    uiStore.addAlert('更新收藏失败', 'error')
  }
}

const loadDashboardData = async () => {
  isLoading.value = true
  isDashboardReady.value = false
  try {
    const stats = await gamesStore.fetchStats()
    pendingReviewGameCount.value = stats.pending_reviews || 0
    isDashboardReady.value = true
    lastLoadedAt.value = Date.now()
  } catch {
    uiStore.addAlert('加载数据失败', 'error')
    isDashboardReady.value = true
  } finally {
    isLoading.value = false
  }
}

const finishSplash = async () => {
  const elapsed = Date.now() - splashStartedAt.value
  const remaining = Math.max(0, MIN_SPLASH_DURATION - elapsed)

  if (remaining > 0) {
    await new Promise((resolve) => window.setTimeout(resolve, remaining))
  }

  isSplashLeaving.value = true
  await new Promise((resolve) => window.setTimeout(resolve, SPLASH_FADE_DURATION))
  showSplash.value = false
}

onMounted(async () => {
  if (hasPlayedDashboardSplash()) {
    showSplash.value = false
    isSplashLeaving.value = false
    await loadDashboardData()
    return
  }

  markDashboardSplashPlayed()
  splashStartedAt.value = Date.now()
  await Promise.all([
    loadDashboardData(),
    finishSplash(),
  ])
})

onActivated(async () => {
  if (Date.now() - lastLoadedAt.value > 30000) {
    await loadDashboardData()
  }
})
</script>

<style scoped>
.dashboard {
  position: relative;
  z-index: 2;
  animation: fadeInUp 0.6s cubic-bezier(0.2, 0.8, 0.2, 1) forwards;
  padding-bottom: 24px;
  transition: opacity 0.28s ease;
}

.dashboard:not(.is-ready) {
  opacity: 0;
}

.dashboard.is-ready {
  opacity: 1;
}

.dashboard-loading {
  position: fixed;
  inset: 0;
  z-index: 30;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  pointer-events: auto;
  opacity: 1;
  transition: opacity 0.4s ease;
}

.dashboard-loading--leaving {
  opacity: 0;
}

.dashboard-loading__backdrop {
  position: absolute;
  inset: 0;
  background:
    radial-gradient(circle at top, rgba(62, 123, 250, 0.28), transparent 45%),
    radial-gradient(circle at bottom, rgba(44, 212, 191, 0.18), transparent 40%),
    linear-gradient(180deg, #07111f 0%, #04070d 100%);
}

.dashboard-loading__content {
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 18px;
  width: min(520px, calc(100vw - 48px));
  animation: splashReveal 1.4s cubic-bezier(0.22, 1, 0.36, 1) both;
}

.dashboard-loading__eyebrow {
  margin: 0;
  color: rgba(255, 255, 255, 0.56);
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.42em;
}

.dashboard-loading__brand {
  margin: 0;
  color: rgba(255, 255, 255, 0.96);
  font-size: clamp(44px, 9vw, 84px);
  font-weight: 900;
  letter-spacing: -0.03em;
  line-height: 1;
  text-transform: uppercase;
}

.dashboard-loading__track {
  position: relative;
  width: 100%;
  height: 3px;
  overflow: hidden;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.12);
  box-shadow: 0 0 30px rgba(26, 159, 255, 0.18);
}

.dashboard-loading__bar {
  display: block;
  width: 100%;
  height: 100%;
  transform-origin: left center;
  background: linear-gradient(90deg, #7ce7d4 0%, #4b8dff 52%, #f0f7ff 100%);
  animation: splashLoad 1.2s cubic-bezier(0.22, 1, 0.36, 1) forwards;
}

@keyframes splashReveal {
  0% {
    opacity: 0;
    transform: translateY(0);
  }
  22% {
    opacity: 1;
    transform: translateY(0);
  }
  74% {
    opacity: 1;
    transform: translateY(0);
  }
  100% {
    opacity: 0;
    transform: translateY(0);
  }
}

@keyframes splashLoad {
  0% {
    transform: scaleX(0);
    opacity: 0.7;
  }
  20% {
    opacity: 1;
  }
  100% {
    transform: scaleX(1);
    opacity: 1;
  }
}

@keyframes fadeInUp {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.dashboard-section-title {
  margin-bottom: 24px;
}

.dashboard-hero-section {
  margin-bottom: 32px;
  align-items: stretch;
}

.dashboard-stats-grid {
  display: grid;
  grid-template-columns: 1fr;
  grid-template-rows: repeat(4, minmax(0, 1fr));
  gap: 10px;
  height: 100%;
}

.dashboard-stats-grid :deep(.stat-card) {
  height: 100%;
}

.dashboard-stats-grid :deep(.arco-card-body) {
  padding: 14px 18px;
  height: 100%;
}

.dashboard-stats-grid :deep(.stat-card-main) {
  height: 100%;
}

.dashboard-stats-grid :deep(.stat-card-title) {
  font-size: 12px;
  margin-bottom: 4px;
}

.dashboard-stats-grid :deep(.stat-card-value) {
  font-size: 24px;
  line-height: 1;
}

.dashboard-stats-grid :deep(.stat-icon-wrapper) {
  padding: 8px;
}

.dashboard-stats-grid :deep(.stat-card-icon) {
  font-size: 22px !important;
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

/* Responsive - Arco Design Breakpoints */
@media (max-width: 992px) {
  .dashboard-stats-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    grid-template-rows: none;
  }
}

/* md: 768px */
@media (max-width: 768px) {
  .dashboard-section-title {
    flex-direction: column;
    align-items: flex-start;
    gap: 4px;
  }
}

@media (max-width: 576px) {
  .dashboard-stats-grid {
    grid-template-columns: 1fr;
  }
}
</style>
