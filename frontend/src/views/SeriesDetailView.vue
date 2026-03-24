<template>
  <div class="series-detail">
    <div class="series-detail__header page-hero">
      <a-button class="app-text-action-btn back-button" type="text" @click="handleGoBack">
        <template #icon>
          <icon-left />
        </template>
        返回
      </a-button>

      <div class="page-hero__content">
        <h1 class="series-detail__title page-hero__title text-gradient">{{ seriesName }}</h1>
        <p class="series-detail__subtitle page-hero__subtitle">共 {{ games.length }} 部作品</p>
      </div>
    </div>

    <div v-if="isLoading" class="series-detail__loading">
      <a-spin :size="24" />
      <p>加载系列作品中...</p>
    </div>

    <div v-else-if="games.length > 0" class="series-detail__grid">
      <div
        v-for="game in games"
        :key="game.id"
        class="series-detail__grid-item"
      >
        <game-card
          :game="game"
          @view="openGame"
          @toggle-favorite="toggleFavorite(game.id)"
        />
      </div>
    </div>

    <a-empty v-else description="这个系列下还没有游戏" />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { IconLeft } from '@arco-design/web-vue/es/icon'
import { useUiStore } from '@/stores/ui'
import gamesService from '@/services/games.service'
import { seriesService } from '@/services/series.service'
import GameCard from '@/components/GameCard.vue'
import { useNamedRouteGuard, watchRouteParamWhenActive } from '@/composables/useNamedRouteGuard'
import type { Game } from '@/services/types'
import { createDetailRouteQuery, resolveReturnRoute } from '@/utils/navigation'

defineOptions({
  name: 'SeriesDetailView',
})

const route = useRoute()
const router = useRouter()
const uiStore = useUiStore()
const { runWhenActive } = useNamedRouteGuard(route, 'series-detail')

const isLoading = ref(false)
const games = ref<Game[]>([])
const seriesName = ref('系列')

const handleGoBack = () => {
  router.push(resolveReturnRoute(route, { name: 'series-library' }))
}

const loadSeriesDetail = async () => {
  await runWhenActive(async () => {
    const id = Number(route.params.id)
    if (Number.isNaN(id) || id <= 0) {
      router.replace({ name: 'series-library' })
      return
    }

    isLoading.value = true
    try {
      const detail = await seriesService.getSeriesDetail(id)
      seriesName.value = detail.series.name || `系列 ${id}`
      games.value = detail.games
    } finally {
      isLoading.value = false
    }
  })
}

const openGame = (id: string | number) => {
  router.push({
    name: 'game-detail',
    params: { id: String(id) },
    query: createDetailRouteQuery(route),
  })
}

const toggleFavorite = async (id: number) => {
  try {
    await gamesService.toggleFavorite(String(id))
    games.value = games.value.map((game) =>
      game.id === id ? { ...game, isFavorite: !game.isFavorite } : game,
    )
  } catch {
    uiStore.addAlert('更新收藏失败', 'error')
  }
}

watchRouteParamWhenActive(
  route,
  'series-detail',
  'id',
  () => {
    void loadSeriesDetail()
  },
)
</script>

<style scoped>
.series-detail {
  animation: fadeInOnly 0.3s ease forwards;
}

.series-detail__header {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 10px;
  margin-bottom: 10px;
}

.series-detail__title {
  margin: 0;
}

.series-detail__subtitle {
  margin: 0;
}

.back-button {
  align-self: flex-start;
}

.series-detail__loading {
  padding: 64px 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
}

.series-detail__grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 16px;
}

.series-detail__grid-item {
  min-width: 0;
}

@keyframes fadeInOnly {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

@media (max-width: 1199px) {
  .series-detail__grid {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }
}

@media (max-width: 991px) {
  .series-detail__grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (max-width: 767px) {
  .series-detail__grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (min-width: 1200px) {
  .series-detail__grid {
    grid-template-columns: repeat(6, minmax(0, 1fr));
  }
}

@media (min-width: 1600px) {
  .series-detail__grid {
    grid-template-columns: repeat(8, minmax(0, 1fr));
  }
}

@media (min-width: 2200px) {
  .series-detail__grid {
    grid-template-columns: repeat(12, minmax(0, 1fr));
  }
}
</style>
