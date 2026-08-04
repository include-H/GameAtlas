<template>
  <div class="dashboard-hero">
    <div class="dashboard-hero__carousel">
      <game-carousel
        v-if="games.length > 0"
        :games="games"
        :auto-play="true"
        :interval="6000"
      />
      <div v-else class="dashboard-hero__carousel-empty app-glass-surface">
        <icon-trophy class="dashboard-hero__empty-icon" />
        <p class="dashboard-hero__empty-text">游戏库还是空的，先去添加几款游戏吧。</p>
      </div>
    </div>

    <aside class="dashboard-hero__actions app-glass-surface">
      <h3 class="dashboard-hero__title">快速入口</h3>
      <a-button
        class="dashboard-hero__action"
        type="primary"
        long
        @click="emit('enter-store')"
      >
        <template #icon>
          <icon-fire />
        </template>
        进入游戏店
      </a-button>
      <a-button
        class="dashboard-hero__action"
        long
        @click="emit('browse-games')"
      >
        <template #icon>
          <icon-trophy />
        </template>
        浏览游戏库
      </a-button>
      <a-button
        v-if="isAdmin"
        class="dashboard-hero__action"
        long
        @click="emit('add-game')"
      >
        <template #icon>
          <icon-plus />
        </template>
        添加游戏
      </a-button>
      <a-button
        v-if="isAdmin"
        class="dashboard-hero__action"
        long
        @click="emit('open-pending')"
      >
        <template #icon>
          <icon-exclamation-circle />
        </template>
        待处理工作台
        <span v-if="pendingReviews > 0" class="dashboard-hero__badge">
          {{ pendingReviews }}
        </span>
      </a-button>
    </aside>
  </div>
</template>

<script setup lang="ts">
import GameCarousel from '@/components/GameCarousel.vue'
import {
  IconExclamationCircle,
  IconFire,
  IconPlus,
  IconTrophy,
} from '@arco-design/web-vue/es/icon'
import type { GameListItem } from '@/services/types'

defineProps<{
  games: GameListItem[]
  isAdmin: boolean
  pendingReviews: number
}>()

const emit = defineEmits<{
  'enter-store': []
  'browse-games': []
  'add-game': []
  'open-pending': []
}>()
</script>

<style scoped>
.dashboard-hero {
  display: grid;
  grid-template-columns: minmax(0, 2fr) minmax(260px, 1fr);
  gap: 16px;
  align-items: stretch;
}

.dashboard-hero__carousel {
  min-width: 0;
}

.dashboard-hero__carousel-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  height: 45vh;
  min-height: 320px;
  max-height: 480px;
  border-radius: var(--radius-xl);
}

.dashboard-hero__empty-icon {
  font-size: 64px;
  color: var(--color-text-3);
}

.dashboard-hero__empty-text {
  margin: 0;
  color: var(--color-text-3);
}

.dashboard-hero__actions {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 20px;
  border-radius: var(--radius-lg);
}

.dashboard-hero__title {
  margin: 0 0 4px;
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-1);
}

.dashboard-hero__action {
  position: relative;
}

.dashboard-hero__badge {
  position: absolute;
  top: 6px;
  right: 8px;
  min-width: 20px;
  height: 20px;
  padding: 0 6px;
  border-radius: 10px;
  background: rgb(var(--danger-6));
  color: #fff;
  font-size: 12px;
  line-height: 20px;
  text-align: center;
}

@media (max-width: 992px) {
  .dashboard-hero {
    grid-template-columns: 1fr;
  }
}
</style>
