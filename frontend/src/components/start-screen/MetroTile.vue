<template>
  <button
    type="button"
    class="metro-tile"
    :style="tileStyle"
    :title="game.title"
    @click="emit('select', game.public_id)"
  >
    <img
      v-if="coverUrl"
      :src="coverUrl"
      :alt="game.title"
      class="metro-tile__cover"
      loading="lazy"
      draggable="false"
    >
    <span v-else class="metro-tile__fallback">{{ initial }}</span>
    <span class="metro-tile__shade" />
    <span class="metro-tile__label">{{ game.title }}</span>
    <span
      class="metro-tile__unpin"
      role="button"
      tabindex="-1"
      title="取消收藏"
      @click.stop="emit('unpin', game.public_id)"
    >
      <icon-close />
    </span>
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { IconClose } from '@arco-design/web-vue/es/icon'
import type { GameListItem } from '@/services/types'

const props = defineProps<{
  game: GameListItem
  colorIndex: number
}>()

const emit = defineEmits<{
  select: [publicId: string]
  unpin: [publicId: string]
}>()

// Win8 Metro 24 色磁贴配色：开始屏幕是沉浸式品牌页特例，色板留在组件内部不外溢。
const metroColors = [
  '#16a085', '#27ae60', '#2980b9', '#8e44ad',
  '#2c3e50', '#f39c12', '#d35400', '#c0392b',
  '#7f8c8d', '#1abc9c', '#3498db', '#9b59b6',
  '#e74c3c', '#f1c40f', '#e67e22', '#00bcd4',
  '#009688', '#4caf50', '#ff9800', '#795548',
  '#607d8b', '#ff5722', '#673ab7', '#3f51b5',
]

const coverUrl = computed(() => props.game.cover_image || '')
const initial = computed(() => props.game.title.trim().charAt(0).toUpperCase() || '?')
const tileStyle = computed(() => ({
  '--metro-tile-color': metroColors[props.colorIndex % metroColors.length],
}))
</script>

<style scoped>
.metro-tile {
  position: relative;
  display: flex;
  align-items: flex-end;
  width: min(240px, 34vw);
  height: 120px;
  padding: 12px;
  border: none;
  border-radius: 6px;
  overflow: hidden;
  cursor: pointer;
  background: var(--metro-tile-color, #2980b9);
  color: #fff;
  font-family: 'LXGW WenKai GB Screen', 'Microsoft YaHei', 'PingFang SC', sans-serif;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.28);
  transition: transform 150ms ease, box-shadow 150ms ease;
  flex-shrink: 0;
}

.metro-tile:hover {
  transform: translateY(-2px) scale(1.02);
  box-shadow: 0 8px 22px rgba(0, 0, 0, 0.4);
}

.metro-tile__cover,
.metro-tile__fallback {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.metro-tile__fallback {
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 56px;
  font-weight: 700;
  opacity: 0.55;
}

.metro-tile__shade {
  position: absolute;
  inset: 0;
  background: linear-gradient(180deg, rgba(0, 0, 0, 0) 45%, rgba(0, 0, 0, 0.55) 100%);
  pointer-events: none;
}

.metro-tile__label {
  position: relative;
  max-width: 100%;
  font-size: 15px;
  font-weight: 600;
  line-height: 1.25;
  text-align: left;
  text-shadow: 0 1px 3px rgba(0, 0, 0, 0.6);
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.metro-tile__unpin {
  position: absolute;
  top: 6px;
  right: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.42);
  color: rgba(255, 255, 255, 0.88);
  opacity: 0;
  transition: opacity 120ms ease, background 120ms ease;
}

.metro-tile:hover .metro-tile__unpin,
.metro-tile__unpin:focus-visible {
  opacity: 1;
}

.metro-tile__unpin:hover {
  background: rgba(0, 0, 0, 0.68);
}

@media (hover: none) {
  .metro-tile__unpin {
    opacity: 1;
  }
}
</style>
