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
        <p class="series-detail__subtitle page-hero__subtitle">共 {{ totalGames }} 部作品</p>
      </div>
    </div>

    <div v-if="isLoading" class="series-detail__loading">
      <a-spin :size="24" />
      <p>加载系列作品中...</p>
    </div>

    <template v-else-if="!hasLoadFailure && games.length > 0">
      <div class="series-detail__grid">
        <div
          v-for="game in games"
          :key="game.id"
          class="series-detail__grid-item"
        >
          <game-card
            :game="game"
            @view="openGame"
            @view-series="openSeries"
            @toggle-favorite="toggleFavorite"
          />
        </div>
      </div>

      <div
        ref="loadMoreSentinel"
        class="series-detail__infinite-scroll"
      >
        <a-spin v-if="isLoadingMore" :size="20" />
      </div>
    </template>

    <a-empty v-else-if="hasLoadFailure" description="系列详情加载失败，请稍后重试。" />

    <a-empty v-else description="这个系列下还没有游戏" />
  </div>
</template>

<script setup lang="ts">
import { onActivated, onDeactivated, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { IconLeft } from '@arco-design/web-vue/es/icon'
import { useUiStore } from '@/stores/ui'
import { useGamesStore } from '@/stores/games'
import { seriesService } from '@/services/series.service'
import { useInfiniteScroll } from '@/composables/useInfiniteScroll'
import GameCard from '@/components/GameCard.vue'
import type { GameListItem } from '@/services/types'
import { navigateBackOrFallback } from '@/utils/navigation'
import { getAmbientBackgroundPoolFromGames, hasAmbientBackgroundPoolImages } from '@/utils/ambient-background'

defineOptions({
  name: 'SeriesDetailView',
})

const AMBIENT_BACKGROUND_OWNER = 'series-detail'
const DETAIL_PAGE_SIZE = 24

const route = useRoute()
const router = useRouter()
const uiStore = useUiStore()
const gamesStore = useGamesStore()

const loadMoreSentinel = ref<HTMLElement | null>(null)
const seriesName = ref('系列')

const {
  items: games,
  isLoading,
  isLoadingMore,
  hasLoadFailure,
  total: totalGames,
  loadFirstPage,
} = useInfiniteScroll<GameListItem>({
  pageSize: DETAIL_PAGE_SIZE,
  sentinel: loadMoreSentinel,
  searchQuery: ref(''),
  loadPage: async ({ page, limit }) => {
    const id = Number(route.params.id)
    const detail = await seriesService.getSeriesDetail(id, { page, limit })
    seriesName.value = detail.series.name || `系列 ${id}`
    return {
      data: detail.games,
      pagination: detail.pagination,
    }
  },
  onError: (message) => uiStore.addAlert(message === '加载失败' ? '加载系列详情失败' : '加载更多系列作品失败', 'error'),
})

const syncAmbientBackground = (seriesId: number) => {
  const pool = getAmbientBackgroundPoolFromGames(games.value)
  if (hasAmbientBackgroundPoolImages(pool)) {
    uiStore.setAmbientBackgroundSource({
      owner: AMBIENT_BACKGROUND_OWNER,
      key: String(seriesId),
      pool,
    })
    return
  }

  uiStore.clearAmbientBackgroundSource(AMBIENT_BACKGROUND_OWNER)
}

const handleGoBack = () => {
  navigateBackOrFallback(router, { name: 'series-library' })
}

const loadSeriesDetail = () => {
  const id = Number(route.params.id)
  if (Number.isNaN(id) || id <= 0) {
    uiStore.clearAmbientBackgroundSource(AMBIENT_BACKGROUND_OWNER)
    router.replace({ name: 'series-library' })
    return
  }
  uiStore.clearAmbientBackgroundSource(AMBIENT_BACKGROUND_OWNER)
  void loadFirstPage()
}

const openGame = (publicId: string) => {
  if (!publicId) return
  router.push({
    name: 'game-detail',
    params: { publicId },
  })
}

const openSeries = (id: number) => {
  if (id <= 0) return
  router.push({
    name: 'series-detail',
    params: { id: String(id) },
  })
}

const toggleFavorite = async (gameRef: string) => {
  if (!gameRef) return
  try {
    const isFavorite = await gamesStore.toggleFavorite(gameRef)
    games.value.forEach((game) => {
      if (game.public_id === gameRef) {
        game.isFavorite = isFavorite
      }
    })
  } catch {
    uiStore.addAlert('更新收藏失败', 'error')
  }
}

watch(
  () => route.params.id,
  () => {
    loadSeriesDetail()
  },
  { immediate: true },
)

watch(games, () => {
  const id = Number(route.params.id)
  if (!Number.isNaN(id) && id > 0) {
    syncAmbientBackground(id)
  }
})

onActivated(() => {
  const id = Number(route.params.id)
  if (!Number.isNaN(id) && id > 0) {
    syncAmbientBackground(id)
  }
})

onDeactivated(() => {
  uiStore.clearAmbientBackgroundSource(AMBIENT_BACKGROUND_OWNER)
})

onUnmounted(() => {
  uiStore.clearAmbientBackgroundSource(AMBIENT_BACKGROUND_OWNER)
})
</script>

<style scoped>
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
  content-visibility: auto;
  contain-intrinsic-size: auto 340px;
}

.series-detail__infinite-scroll {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 48px;
  margin-top: 24px;
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
