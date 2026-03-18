<template>
  <div class="dashboard">
    <!-- Welcome Section (Slimmer Header) -->
    <div class="dashboard-section-title">
      <h1 class="text-gradient">
        发现
      </h1>
      <p class="subtitle">
        探索您的游戏库动态
      </p>
    </div>

    <!-- Top Game Carousel (Full Width) -->
    <a-row class="dashboard-carousel-section">
      <a-col :span="24">
        <game-carousel
          v-if="carouselGames.length > 0"
          :games="carouselGames"
          :auto-play="true"
          :interval="5000"
        />
      </a-col>
    </a-row>

    <!-- Statistics Cards Row -->
    <a-row :gutter="[16, 16]" class="dashboard-stats-row">
      <a-col :xs="12" :sm="12" :md="6" :lg="6" :xl="6">
        <stat-card
          title="游戏总数"
          :value="totalGames"
          icon="mdi-gamepad-variant"
          color="#1a9fff"
          @click="router.push('/games')"
        />
      </a-col>

      <a-col :xs="12" :sm="12" :md="6" :lg="6" :xl="6">
        <stat-card
          title="收藏"
          :value="favorites.length"
          icon="mdi-heart"
          color="#f53f3f"
          @click="router.push('/games?filter=favorites')"
        />
      </a-col>

      <a-col :xs="12" :sm="12" :md="6" :lg="6" :xl="6">
        <stat-card
          title="新入库"
          :value="recentAdditions.length"
          icon="mdi-new-box"
          color="#00b42a"
          @click="router.push('/games?sort=newest')"
        />
      </a-col>

      <a-col :xs="12" :sm="12" :md="6" :lg="6" :xl="6">
        <stat-card
          title="待处理"
          :value="pendingReviews"
          icon="mdi-clock"
          color="#ff7d00"
          @click="router.push('/games/pending')"
        />
      </a-col>
    </a-row>

    <!-- Divider between stats and content -->
    <a-divider class="dashboard-divider" />

    <!-- Recently Added -->
    <card-row
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
          @click="viewGame(item.id)"
          @toggle-favorite="toggleFavorite(item.id)"
        />
      </template>
    </card-row>

    <!-- Most Downloaded -->
    <card-row
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
          @click="viewGame(item.id)"
          @toggle-favorite="toggleFavorite(item.id)"
        />
      </template>
    </card-row>

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
        class="dashboard-empty-button"
        @click="router.push('/games')"
      >
        浏览游戏
      </a-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onActivated } from 'vue'
import { useRouter } from 'vue-router'
import { useGamesStore } from '@/stores/games'
import { useUiStore } from '@/stores/ui'
import gamesService from '@/services/games.service'
import { getPendingIssues } from '@/utils/pendingIssues'
import {
  IconTrophy
} from '@arco-design/web-vue/es/icon'
import StatCard from '@/components/StatCard.vue'
import CardRow from '@/components/CardRow.vue'
import GameCard from '@/components/GameCard.vue'
import GameCarousel from '@/components/GameCarousel.vue'

defineOptions({
  name: 'DashboardView',
})

const router = useRouter()
const gamesStore = useGamesStore()
const uiStore = useUiStore()

const isLoading = ref(false)
const detailedGames = ref<Record<number, import('@/services/types').Game>>({})

// Directly use gamesStore.stats (it's already a ref)
const totalGames = computed(() => gamesStore.stats?.total_games || 0)
const recentAdditions = computed(() => gamesStore.stats?.recent_games || [])
const mostPlayed = computed(() => gamesStore.stats?.popular_games || [])
const favorites = computed(() => gamesStore.stats?.favorite_games || [])

const isEmpty = computed(() => {
  return recentAdditions.value.length === 0
})

// Get games for carousel (combine recent and most played, shuffle them)
const carouselGames = computed(() => {
  const orderedIds = [...recentAdditions.value, ...mostPlayed.value]
    .map((game) => game.id)
    .filter((id, index, self) => self.indexOf(id) === index)

  return orderedIds
    .map((id) => detailedGames.value[id])
    .filter((game): game is import('@/services/types').Game => !!game)
    .sort(() => Math.random() - 0.5)
})

const pendingReviewGameCount = ref(0)
const lastLoadedAt = ref(0)

const pendingReviews = computed(() => pendingReviewGameCount.value)

const viewGame = (id: string) => {
  router.push({ name: 'game-detail', params: { id } })
}

const toggleFavorite = async (id: string) => {
  try {
    await gamesStore.toggleFavorite(id)
    uiStore.addAlert('收藏已更新', 'success')
  } catch (error) {
    uiStore.addAlert('更新收藏失败', 'error')
  }
}

const loadDashboardData = async () => {
  isLoading.value = true
  try {
    await gamesStore.fetchStats()
    const allGames = await gamesService.getGames({
      page: 1,
      pageSize: 500,
      sort: { field: 'updated_at', order: 'desc' },
    })
    const details = await Promise.all(
      allGames.data.map(async (game) => {
        try {
          return await gamesService.getGame(String(game.id))
        } catch {
          return null
        }
      }),
    )
    detailedGames.value = details
      .filter((game): game is import('@/services/types').Game => !!game)
      .reduce<Record<number, import('@/services/types').Game>>((acc, game) => {
        acc[game.id] = game
        return acc
      }, {})
    pendingReviewGameCount.value = 0
    for (const game of details) {
      if (!game) continue
      const issues = getPendingIssues(game)
      if (issues.length > 0) {
        pendingReviewGameCount.value += 1
      }
    }
    lastLoadedAt.value = Date.now()
  } catch (error) {
    uiStore.addAlert('加载数据失败', 'error')
  } finally {
    isLoading.value = false
  }
}

onMounted(loadDashboardData)

onActivated(async () => {
  if (Date.now() - lastLoadedAt.value > 30000) {
    await loadDashboardData()
  }
})
</script>

<style scoped>
.dashboard {
  animation: fadeInUp 0.6s cubic-bezier(0.2, 0.8, 0.2, 1) forwards;
  padding-bottom: 24px;
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
  display: flex;
  align-items: flex-end;
  gap: 16px;
}

.dashboard-section-title .text-gradient {
  font-size: 32px;
  font-weight: 800;
  margin: 0;
  letter-spacing: -0.5px;
  background: linear-gradient(135deg, var(--color-primary-light-3), var(--color-primary-6));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

.dashboard-section-title .subtitle {
  margin: 0;
  padding-bottom: 6px;
  color: var(--color-text-3);
  font-size: 16px;
  font-weight: 500;
}

.dashboard-stats-row {
  margin-bottom: 32px;
  margin-top: 16px;
}

.dashboard-carousel-section {
  margin-bottom: 16px;
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

.dashboard-empty-button {
  margin-top: 8px;
}

/* Responsive - Arco Design Breakpoints */
/* md: 768px */
@media (max-width: 768px) {
  .dashboard-section-title {
    flex-direction: column;
    align-items: flex-start;
    gap: 4px;
  }
}
</style>
