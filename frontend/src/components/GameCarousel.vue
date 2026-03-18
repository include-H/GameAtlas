<template>
  <div class="game-carousel">
    <!-- Background Images -->
    <div class="carousel-background">
      <div
        v-for="(game, index) in games"
        :key="game.id"
        class="carousel-slide"
        :class="{ active: currentIndex === index }"
      >
        <!-- Layer 1: Blurred Backdrop (fills 21:9) -->
        <div 
          class="slide-backdrop" 
          :style="{ backgroundImage: `url(${getBackgroundImage(game)})` }"
        />
        <!-- Layer 2: Clear Foreground (centered, preserves aspect ratio) -->
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
          class="carousel-btn"
          @click="viewGame(game.id)"
        >
          查看详情
        </a-button>
      </div>
    </div>

    <!-- Indicators -->
    <div class="carousel-indicators">
      <button
        v-for="(game, index) in games"
        :key="game.id"
        class="indicator"
        :class="{ active: currentIndex === index }"
        @click="goToSlide(index)"
      />
    </div>

    <!-- Navigation Arrows -->
    <button class="carousel-arrow carousel-arrow-prev" @click="prevSlide">
      <icon-left />
    </button>
    <button class="carousel-arrow carousel-arrow-next" @click="nextSlide">
      <icon-right />
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onUnmounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { IconLeft, IconRight } from '@arco-design/web-vue/es/icon'
import type { Game } from '@/services/types'

interface Props {
  games: Game[]
  autoPlay?: boolean
  interval?: number
}

const props = withDefaults(defineProps<Props>(), {
  autoPlay: true,
  interval: 5000,
})

const router = useRouter()
const currentIndex = ref(0)
let autoPlayTimer: number | null = null

// 缓存每个游戏的截图索引，确保同一游戏每次显示相同的截图
const screenshotIndexCache = new Map<number, number>()

const games = computed(() => props.games.slice(0, 5))

const getBackgroundImage = (game: Game) => {
  // 优先使用截图，从截图中随机选择一张
  if (game.screenshots && game.screenshots.length > 0) {
    // 如果还没有缓存索引，就随机选择一个并缓存
    if (!screenshotIndexCache.has(game.id)) {
      const randomIndex = Math.floor(Math.random() * game.screenshots.length)
      screenshotIndexCache.set(game.id, randomIndex)
    }
    const index = screenshotIndexCache.get(game.id)!
    return game.screenshots[index]
  }
  // 其次使用横幅图，最后使用封面
  return game.banner_image || game.cover_image || '/placeholder-game.jpg'
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
  router.push({ name: 'game-detail', params: { id: String(id) } })
}

const nextSlide = () => {
  currentIndex.value = (currentIndex.value + 1) % games.value.length
}

const prevSlide = () => {
  currentIndex.value = (currentIndex.value - 1 + games.value.length) % games.value.length
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
    screenshotIndexCache.clear()
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

.slide-backdrop {
  position: absolute;
  inset: 0;
  background-size: cover;
  background-position: center;
  /* Performance-friendly atmospheric effect instead of expensive blur */
  opacity: 0.35;
  filter: grayscale(0.5) brightness(0.7);
  z-index: 1;
  will-change: transform, opacity;
}

.slide-foreground {
  position: absolute;
  inset: 0;
  background-size: contain;
  background-position: center;
  background-repeat: no-repeat;
  z-index: 2;
  /* Subtly darkened for better contrast with backdrop */
  filter: drop-shadow(0 0 30px rgba(0, 0, 0, 0.6));
  will-change: transform, opacity;
}

.carousel-overlay {
  position: absolute;
  inset: 0;
  /* Stronger radial vignette to blend foreground into atmospheric backdrop */
  background: radial-gradient(
    circle at center,
    rgba(0, 0, 0, 0) 30%,
    rgba(0, 0, 0, 0.4) 70%,
    rgba(0, 0, 0, 0.8) 100%
  );
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

.carousel-btn {
  background: rgba(255, 255, 255, 0.2);
  backdrop-filter: blur(10px);
  border: 1px solid rgba(255, 255, 255, 0.3);
  color: white;
  font-weight: 500;
}

.carousel-btn:hover {
  background: rgba(255, 255, 255, 0.3);
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
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.4);
  border: none;
  cursor: pointer;
  transition: all 0.3s ease;
}

.indicator.active {
  background: white;
  width: 24px;
  border-radius: 4px;
}

.indicator:hover {
  background: rgba(255, 255, 255, 0.8);
}

.carousel-arrow {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  width: 44px;
  height: 44px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.1);
  backdrop-filter: blur(10px);
  border: 1px solid rgba(255, 255, 255, 0.2);
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  z-index: 11;
  transition: all 0.3s ease;
}

.carousel-arrow:hover {
  background: rgba(255, 255, 255, 0.2);
}

.carousel-arrow-prev {
  left: 16px;
}

.carousel-arrow-next {
  right: 16px;
}

/* Responsive - Arco Design Breakpoints */
/* md: 768px */
@media (max-width: 768px) {
  .game-carousel {
    height: 300px;
  }

  .carousel-content {
    padding: 0 24px;
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
</style>
