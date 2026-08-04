<template>
  <div class="publisher-detail">
    <div class="publisher-detail__header page-hero">
      <a-button class="app-text-action-btn back-button" type="text" @click="handleGoBack">
        <template #icon>
          <icon-left />
        </template>
        返回
      </a-button>

      <div class="page-hero__content">
        <h1 class="publisher-detail__title page-hero__title text-gradient">{{ publisherName }}</h1>
        <p class="publisher-detail__subtitle page-hero__subtitle">共 {{ games.length }} 部作品</p>
      </div>
    </div>

    <div v-if="isLoading" class="publisher-detail__loading">
      <a-spin :size="24" />
      <p>加载发行商作品中...</p>
    </div>

    <div v-else-if="!hasLoadFailure && games.length > 0" class="publisher-detail__grid">
      <div
        v-for="game in games"
        :key="game.id"
        class="publisher-detail__grid-item"
      >
        <game-card
          :game="game"
          @view="openGame"
          @view-series="openSeries"
          @toggle-favorite="toggleFavorite"
        />
      </div>
    </div>

    <a-empty v-else-if="hasLoadFailure" description="发行商详情加载失败，请稍后重试。" />

    <a-empty v-else description="这个发行商下还没有游戏" />
  </div>
</template>

<script setup lang="ts">
import { onActivated, onDeactivated, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { IconLeft } from '@arco-design/web-vue/es/icon'
import { useUiStore } from '@/stores/ui'
import { useGamesStore } from '@/stores/games'
import { publishersService } from '@/services/publishers.service'
import GameCard from '@/components/GameCard.vue'
import type { GameListItem } from '@/services/types'
import { navigateBackOrFallback } from '@/utils/navigation'
import { getAmbientBackgroundPoolFromGames, hasAmbientBackgroundPoolImages } from '@/utils/ambient-background'

defineOptions({
  name: 'PublisherDetailView',
})

const AMBIENT_BACKGROUND_OWNER = 'publisher-detail'

const route = useRoute()
const router = useRouter()
const uiStore = useUiStore()
const gamesStore = useGamesStore()

const isLoading = ref(false)
const hasLoadFailure = ref(false)
const games = ref<GameListItem[]>([])
const publisherName = ref('发行商')
let loadRequestId = 0

const syncAmbientBackground = (publisherId: number) => {
  const pool = getAmbientBackgroundPoolFromGames(games.value)
  if (hasAmbientBackgroundPoolImages(pool)) {
    uiStore.setAmbientBackgroundSource({
      owner: AMBIENT_BACKGROUND_OWNER,
      key: String(publisherId),
      pool,
    })
    return
  }

  uiStore.clearAmbientBackgroundSource(AMBIENT_BACKGROUND_OWNER)
}

const handleGoBack = () => {
  navigateBackOrFallback(router, { name: 'publisher-library' })
}

const loadPublisherDetail = async () => {
  const requestId = loadRequestId + 1
  loadRequestId = requestId
  const id = Number(route.params.id)
  if (Number.isNaN(id) || id <= 0) {
    uiStore.clearAmbientBackgroundSource(AMBIENT_BACKGROUND_OWNER)
    router.replace({ name: 'publisher-library' })
    return
  }

  isLoading.value = true
  hasLoadFailure.value = false
  uiStore.clearAmbientBackgroundSource(AMBIENT_BACKGROUND_OWNER)
  try {
    const detail = await publishersService.getPublisherDetail(id)
    if (requestId !== loadRequestId) {
      return
    }
    publisherName.value = detail.publisher.name || `发行商 ${id}`
    games.value = detail.games
    syncAmbientBackground(detail.publisher.id || id)
  } catch {
    if (requestId !== loadRequestId) {
      return
    }
    hasLoadFailure.value = true
    games.value = []
    publisherName.value = `发行商 ${id}`
    uiStore.clearAmbientBackgroundSource(AMBIENT_BACKGROUND_OWNER)
    uiStore.addAlert('加载发行商详情失败', 'error')
  } finally {
    if (requestId === loadRequestId) {
      isLoading.value = false
    }
  }
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
    void loadPublisherDetail()
  },
  { immediate: true },
)

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
  loadRequestId += 1
  uiStore.clearAmbientBackgroundSource(AMBIENT_BACKGROUND_OWNER)
})
</script>

<style scoped>
.publisher-detail {
  animation: fadeInOnly 0.3s ease forwards;
}

.publisher-detail__header {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 10px;
  margin-bottom: 10px;
}

.publisher-detail__title,
.publisher-detail__subtitle {
  margin: 0;
}

.back-button {
  align-self: flex-start;
}

.publisher-detail__loading {
  padding: 64px 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
}

.publisher-detail__grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 16px;
}

.publisher-detail__grid-item {
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
  .publisher-detail__grid {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }
}

@media (max-width: 991px) {
  .publisher-detail__grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (max-width: 767px) {
  .publisher-detail__grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (min-width: 1200px) {
  .publisher-detail__grid {
    grid-template-columns: repeat(6, minmax(0, 1fr));
  }
}

@media (min-width: 1600px) {
  .publisher-detail__grid {
    grid-template-columns: repeat(8, minmax(0, 1fr));
  }
}

@media (min-width: 2200px) {
  .publisher-detail__grid {
    grid-template-columns: repeat(12, minmax(0, 1fr));
  }
}
</style>
