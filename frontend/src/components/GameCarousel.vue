<template>
  <div class="game-carousel">
    <div class="carousel-background">
      <div
        v-for="(game, index) in games"
        :key="game.id"
        class="carousel-slide"
        :class="{ active: currentIndex === index }"
      >
        <div 
          class="slide-foreground" 
          :style="{ backgroundImage: `url(${getBackgroundImage(game)})` }"
        />
      </div>
      <div class="carousel-overlay" />
    </div>

    <!-- Content -->
    <div class="carousel-content">
      <div
        v-for="(game, index) in games"
        :key="game.id"
        class="carousel-info"
        :class="{ active: currentIndex === index }"
      >
        <h2 class="carousel-title">{{ game.title }}</h2>
        <p class="carousel-meta">{{ getMetaInfo(game) }}</p>
        <p class="carousel-description">{{ getDescription(game) }}</p>
        <a-button
          type="primary"
          size="large"
          @click="viewGame(game.id)"
        >
          查看详情
        </a-button>
      </div>
    </div>

    <!-- Indicators -->
    <div class="carousel-indicators">
      <a-button
        v-for="(game, index) in games"
        :key="game.id"
        class="indicator"
        :class="{ active: currentIndex === index }"
        type="text"
        shape="circle"
        @click="goToSlide(index)"
      />
    </div>

  </div>
</template>

<script setup lang="ts">
import { ref, computed, onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import type { Game } from '@/services/types'
import { resolveAssetUrl } from '@/utils/asset-url'
import { createDetailRouteQuery } from '@/utils/navigation'

interface Props {
  games: Game[]
  autoPlay?: boolean
  interval?: number
}

const props = withDefaults(defineProps<Props>(), {
  autoPlay: true,
  interval: 5000,
})

const route = useRoute()
const router = useRouter()
const currentIndex = ref(0)
let autoPlayTimer: number | null = null

const games = computed(() => props.games.slice(0, 5))

const getBackgroundImage = (game: Game) => {
  return resolveAssetUrl(
    game.primary_screenshot
      || game.banner_image
      || game.cover_image
      || '/placeholder-game.jpg',
  )
}

const getDescription = (game: Game) => {
  const desc = game.summary || ''
  if (desc) {
    return desc.length > 200
      ? desc.slice(0, 200) + '...'
      : desc
  }
  return ''
}

const getMetaInfo = (game: Game) => {
  const parts: string[] = []
  if (game.developers && game.developers.length > 0) {
    parts.push(game.developers[0].name)
  }
  if (game.release_date) {
    parts.push(new Date(game.release_date).getFullYear() + '年')
  }
  return parts.join(' · ')
}

const viewGame = (id: number) => {
  router.push({
    name: 'game-detail',
    params: { id: String(id) },
    query: createDetailRouteQuery(route),
  })
}

const nextSlide = () => {
  currentIndex.value = (currentIndex.value + 1) % games.value.length
}

const goToSlide = (index: number) => {
  currentIndex.value = index
}

const stopAutoPlay = () => {
  if (autoPlayTimer) {
    clearInterval(autoPlayTimer)
    autoPlayTimer = null
  }
}

const startAutoPlay = () => {
  stopAutoPlay()
  if (props.autoPlay && games.value.length > 1) {
    autoPlayTimer = window.setInterval(nextSlide, props.interval)
  }
}

watch(
  () => games.value.map((game) => game.id).join(','),
  () => {
    if (currentIndex.value >= games.value.length) {
      currentIndex.value = 0
    }
    stopAutoPlay()
    startAutoPlay()
  },
  { immediate: true },
)

watch(
  () => [props.autoPlay, props.interval] as const,
  () => {
    stopAutoPlay()
    startAutoPlay()
  },
)

onUnmounted(() => {
  stopAutoPlay()
})
</script>

<style scoped>
.game-carousel {
  position: relative;
  width: 100%;
  height: 45vh;
  min-height: 320px;
  max-height: 480px;
  border-radius: var(--radius-xl);
  overflow: hidden;
  box-shadow: var(--shadow-soft);
}

.carousel-background {
  position: absolute;
  inset: 0;
}

.carousel-slide {
  position: absolute;
  inset: 0;
  opacity: 0;
  transition: opacity 0.8s ease-in-out;
  overflow: hidden;
}

.carousel-slide.active {
  opacity: 1;
}

.slide-foreground {
  position: absolute;
  inset: 0;
  background-size: cover;
  background-position: center;
  z-index: 2;
  filter: drop-shadow(0 24px 42px rgba(0, 0, 0, 0.48));
  will-change: transform, opacity;
}

.carousel-overlay {
  position: absolute;
  inset: 0;
  background:
    linear-gradient(90deg, rgba(6, 10, 18, 0.82) 0%, rgba(6, 10, 18, 0.46) 34%, rgba(6, 10, 18, 0.18) 58%, rgba(6, 10, 18, 0.56) 100%),
    radial-gradient(circle at center, rgba(0, 0, 0, 0) 34%, rgba(0, 0, 0, 0.2) 72%, rgba(0, 0, 0, 0.58) 100%);
  z-index: 3;
}

.carousel-content {
  position: relative;
  z-index: 10; /* Above the overlay to keep text bright */
  height: 100%;
  display: flex;
  align-items: center;
  padding: 0 48px;
  padding-left: 98px;
}

.carousel-info {
  position: absolute;
  max-width: 500px;
  opacity: 0;
  transform: translateY(20px);
  transition: all 0.5s ease-in-out;
  pointer-events: none;
}

.carousel-info.active {
  opacity: 1;
  transform: translateY(0);
  pointer-events: auto;
}

.carousel-title {
  font-size: 32px;
  font-weight: 800;
  color: white;
  margin: 0 0 12px 0;
  text-shadow: 0 4px 12px rgba(0, 0, 0, 0.8);
  letter-spacing: -0.5px;
}

.carousel-meta {
  font-size: 14px;
  color: rgba(255, 255, 255, 0.7);
  margin: 0 0 12px 0;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.5);
}

.carousel-description {
  font-size: 16px;
  color: rgba(255, 255, 255, 0.85);
  margin: 0 0 24px 0;
  line-height: 1.6;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.5);
  max-width: 600px;
}

.carousel-indicators {
  position: absolute;
  bottom: 24px;
  left: 48px;
  display: flex;
  gap: 8px;
  z-index: 11;
}

.indicator {
  width: 8px;
  height: 8px;
  background: rgba(255, 255, 255, 0.4);
  transition: all 0.3s ease;
  padding: 0;
  min-width: 8px;
  color: transparent;
}

.indicator:deep(.arco-btn-content) {
  display: none;
}

.indicator.active {
  background: white;
  width: 24px;
  border-radius: 4px;
}

.indicator:hover {
  background: rgba(255, 255, 255, 0.8);
}

/* Responsive - Arco Design Breakpoints */
/* md: 768px */
@media (max-width: 768px) {
  .game-carousel {
    height: 300px;
  }

  .carousel-content {
    padding: 0 24px;
    padding-left: 64px;
  }

  .carousel-title {
    font-size: 24px;
  }

  .carousel-description {
    font-size: 14px;
  }

  .carousel-indicators {
    left: 24px;
  }
}

@media (max-width: 576px) {
  .game-carousel {
    min-height: 280px;
    height: 280px;
  }

  .carousel-content {
    padding: 0 16px;
    padding-left: 16px;
    align-items: flex-end;
    padding-bottom: 44px;
  }

  .carousel-info {
    max-width: 100%;
  }

  .carousel-title {
    font-size: 22px;
    margin-bottom: 8px;
  }

  .carousel-meta {
    margin-bottom: 8px;
  }

  .carousel-description {
    margin-bottom: 16px;
    -webkit-line-clamp: 3;
    -webkit-box-orient: vertical;
    display: -webkit-box;
    overflow: hidden;
  }

  .carousel-indicators {
    left: 16px;
    bottom: 16px;
  }
}
</style>
