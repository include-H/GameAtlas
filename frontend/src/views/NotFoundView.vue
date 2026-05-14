<template>
  <div class="not-found">
    <div class="not-found-fog not-found-fog--1" />
    <div class="not-found-fog not-found-fog--2" />
    <div class="not-found-scene">
      <!-- Ground line -->
      <div class="not-found-ground" />

      <!-- Character -->
      <div class="not-found-character">
        <div class="not-found-character__head" />
        <div class="not-found-character__body" />
        <div class="not-found-character__shadow" />
      </div>

      <!-- Fork signs -->
      <div class="not-found-signpost">
        <div class="not-found-signpost__pole" />
        <div class="not-found-signpost__sign not-found-signpost__sign--left">
          <span>← 403</span>
        </div>
        <div class="not-found-signpost__sign not-found-signpost__sign--right">
          <span>500 →</span>
        </div>
      </div>

      <!-- Scattered items -->
      <div class="not-found-item not-found-item--1" />
      <div class="not-found-item not-found-item--2" />
    </div>

    <div class="not-found-card app-glass-surface">
      <p class="not-found-title">你来到了一片荒芜之地</p>
      <p class="not-found-message">
        这里空无一物，页面可能已被遗弃在废墟之中。
      </p>
      <div class="not-found-actions">
        <a-button type="primary" size="large" @click="goHome">
          返回大厅
        </a-button>
        <a-button size="large" @click="goBack">
          原路返回
        </a-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { navigateBackOrFallback } from '@/utils/navigation'
import { useGamesStore } from '@/stores/games'
import { useUiStore } from '@/stores/ui'
import { getAmbientBackgroundUrlsFromGames } from '@/utils/ambient-background'

const AMBIENT_BACKGROUND_OWNER = 'not-found'

const router = useRouter()
const gamesStore = useGamesStore()
const uiStore = useUiStore()

const goHome = () => {
  router.push('/')
}

const goBack = () => {
  navigateBackOrFallback(router, { name: 'dashboard' })
}

const syncAmbientBackground = () => {
  const games = [
    ...(gamesStore.stats?.recent_games ?? []),
    ...(gamesStore.stats?.popular_games ?? []),
  ]
  const imageUrls = getAmbientBackgroundUrlsFromGames(games)
  if (imageUrls.length > 0) {
    uiStore.setAmbientBackgroundSource({
      owner: AMBIENT_BACKGROUND_OWNER,
      key: '404',
      urls: imageUrls,
    })
    return
  }
  uiStore.clearAmbientBackgroundSource(AMBIENT_BACKGROUND_OWNER)
}

onMounted(async () => {
  syncAmbientBackground()
  if (!gamesStore.stats) {
    try {
      await gamesStore.fetchStats()
      syncAmbientBackground()
    } catch {
      // no stats available — skip ambient background
    }
  }
})

onUnmounted(() => {
  uiStore.clearAmbientBackgroundSource(AMBIENT_BACKGROUND_OWNER)
})
</script>

<style scoped>
.not-found {
  position: relative;
  min-height: 70vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 24px;
  overflow: hidden;
}

/* Fog layers */
.not-found-fog {
  position: absolute;
  inset: 0;
  pointer-events: none;
}

.not-found-fog--1 {
  background: radial-gradient(ellipse at 20% 50%, color-mix(in srgb, var(--color-bg-1) 60%, transparent) 0%, transparent 60%);
  animation: fog-drift-1 12s ease-in-out infinite;
}

.not-found-fog--2 {
  background: radial-gradient(ellipse at 80% 40%, color-mix(in srgb, var(--color-bg-2) 50%, transparent) 0%, transparent 55%);
  animation: fog-drift-2 15s ease-in-out infinite;
}

@keyframes fog-drift-1 {
  0%, 100% { transform: translateX(0) scale(1); opacity: 0.6; }
  50% { transform: translateX(30px) scale(1.05); opacity: 0.8; }
}

@keyframes fog-drift-2 {
  0%, 100% { transform: translateX(0) scale(1); opacity: 0.5; }
  50% { transform: translateX(-25px) scale(1.03); opacity: 0.7; }
}

/* Scene illustration */
.not-found-scene {
  position: relative;
  width: 320px;
  height: 140px;
  margin-bottom: 32px;
}

.not-found-ground {
  position: absolute;
  bottom: 0;
  left: -20px;
  right: -20px;
  height: 2px;
  background: linear-gradient(90deg, transparent, var(--color-border-3), var(--color-border-3), transparent);
}

/* Character */
.not-found-character {
  position: absolute;
  bottom: 2px;
  left: 50%;
  transform: translateX(-50%);
  animation: character-breathe 3s ease-in-out infinite;
}

.not-found-character__head {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: var(--color-text-2);
  margin: 0 auto 2px;
  position: relative;
}

.not-found-character__head::after {
  content: '?';
  position: absolute;
  top: -24px;
  left: 50%;
  transform: translateX(-50%);
  font-size: 16px;
  font-weight: 700;
  color: var(--color-primary-6);
  animation: question-bounce 2s ease-in-out infinite;
}

.not-found-character__body {
  width: 24px;
  height: 32px;
  background: var(--color-text-2);
  border-radius: 6px 6px 3px 3px;
  margin: 0 auto;
}

.not-found-character__shadow {
  width: 30px;
  height: 4px;
  background: var(--color-fill-3);
  border-radius: 50%;
  margin: 4px auto 0;
  opacity: 0.4;
}

@keyframes character-breathe {
  0%, 100% { transform: translateX(-50%) translateY(0); }
  50% { transform: translateX(-50%) translateY(-2px); }
}

@keyframes question-bounce {
  0%, 100% { transform: translateX(-50%) translateY(0); opacity: 1; }
  50% { transform: translateX(-50%) translateY(-6px); opacity: 0.6; }
}

/* Signpost */
.not-found-signpost {
  position: absolute;
  bottom: 2px;
  left: 40px;
}

.not-found-signpost__pole {
  width: 3px;
  height: 80px;
  background: var(--color-border-3);
  margin-left: 2px;
}

.not-found-signpost__sign {
  position: absolute;
  padding: 3px 10px;
  background: var(--color-fill-2);
  border: 1px solid var(--color-border-3);
  border-radius: 3px;
  font-size: 11px;
  color: var(--color-text-3);
  white-space: nowrap;
}

.not-found-signpost__sign--left {
  top: 10px;
  left: -63px;
  transform: rotate(-8deg);
}

.not-found-signpost__sign--right {
  top: 32px;
  left: 8px;
  transform: rotate(5deg);
}

/* Scattered debris */
.not-found-item {
  position: absolute;
  border-radius: 2px;
  background: var(--color-fill-3);
  opacity: 0.3;
}

.not-found-item--1 {
  width: 12px;
  height: 8px;
  bottom: 4px;
  right: 60px;
  transform: rotate(25deg);
}

.not-found-item--2 {
  width: 8px;
  height: 6px;
  bottom: 4px;
  right: 90px;
  transform: rotate(-15deg);
}

/* Card */
.not-found-card {
  position: relative;
  max-width: 440px;
  width: 100%;
  text-align: center;
  padding: 36px 32px;
}

.not-found-title {
  font-size: 22px;
  font-weight: 700;
  color: var(--color-text-1);
  margin: 0 0 10px 0;
}

.not-found-message {
  font-size: 14px;
  color: var(--color-text-3);
  margin: 0 0 28px 0;
  line-height: 1.6;
}

.not-found-actions {
  display: flex;
  gap: 12px;
  justify-content: center;
}

@media (max-width: 576px) {
  .not-found-scene {
    width: 240px;
    height: 110px;
    margin-bottom: 24px;
  }

  .not-found-signpost__sign--left {
    left: -35px;
  }

  .not-found-card {
    padding: 28px 20px;
  }

  .not-found-title {
    font-size: 18px;
  }

  .not-found-actions {
    flex-direction: column;
  }
}
</style>
