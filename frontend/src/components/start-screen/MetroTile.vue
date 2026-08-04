<template>
  <button
    type="button"
    class="metro-tile"
    :class="[
      `metro-tile--${tile.tile_size}`,
      { 'metro-tile--editing': editing },
    ]"
    :style="tileStyle"
    :title="tile.title"
    @click="handleClick"
  >
    <img
      v-if="coverUrl"
      :src="coverUrl"
      :alt="tile.title"
      class="metro-tile__cover"
      loading="lazy"
      draggable="false"
    >
    <span v-else class="metro-tile__fallback">{{ initial }}</span>
    <span class="metro-tile__shade" />
    <span class="metro-tile__label">{{ tile.title }}</span>

    <template v-if="editing">
      <span
        class="metro-tile__action metro-tile__resize"
        role="button"
        tabindex="-1"
        :title="resizeHint"
        @click.stop="emit('resize', tile.game_id)"
      >
        <icon-expand />
      </span>
      <span
        class="metro-tile__action metro-tile__remove"
        role="button"
        tabindex="-1"
        title="从开始屏幕移除"
        @click.stop="emit('remove', tile.game_id)"
      >
        <icon-close />
      </span>
    </template>
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { IconClose, IconExpand } from '@arco-design/web-vue/es/icon'
import type { StartScreenTile, StartScreenTileSize } from '@/services/types'

const props = defineProps<{
  tile: StartScreenTile
  colorIndex: number
  editing: boolean
}>()

const emit = defineEmits<{
  select: [publicId: string]
  resize: [gameId: number]
  remove: [gameId: number]
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

const SIZE_HINTS: Record<StartScreenTileSize, string> = {
  small: '当前：小磁贴',
  wide: '当前：宽磁贴',
  large: '当前：大磁贴',
}

const coverUrl = computed(() => props.tile.cover_image || '')
const initial = computed(() => props.tile.title.trim().charAt(0).toUpperCase() || '?')
const resizeHint = computed(() => `${SIZE_HINTS[props.tile.tile_size]}，点击切换`)
const tileStyle = computed(() => ({
  '--metro-tile-color': metroColors[props.colorIndex % metroColors.length],
}))

const handleClick = () => {
  if (props.editing) return
  emit('select', props.tile.public_id)
}
</script>

<style scoped>
.metro-tile {
  position: relative;
  display: flex;
  align-items: flex-end;
  width: 100%;
  height: 100%;
  padding: 10px;
  border: none;
  border-radius: 6px;
  overflow: hidden;
  cursor: pointer;
  background: var(--metro-tile-color, #2980b9);
  color: #fff;
  font-family: 'LXGW WenKai GB Screen', 'Microsoft YaHei', 'PingFang SC', sans-serif;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.28);
  transition: transform 150ms ease, box-shadow 150ms ease;
}

.metro-tile:hover {
  transform: translateY(-2px) scale(1.02);
  box-shadow: 0 8px 22px rgba(0, 0, 0, 0.4);
}

.metro-tile--editing {
  cursor: grab;
}

.metro-tile--editing:active {
  cursor: grabbing;
}

.metro-tile--editing:hover {
  transform: none;
  box-shadow: 0 0 0 2px rgba(255, 255, 255, 0.65);
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
  font-size: 44px;
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

.metro-tile__action {
  position: absolute;
  top: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.42);
  color: rgba(255, 255, 255, 0.9);
}

.metro-tile__action:hover {
  background: rgba(0, 0, 0, 0.68);
}

.metro-tile__resize {
  right: 38px;
}

.metro-tile__remove {
  right: 6px;
}

@media (hover: none) {
  .metro-tile__action {
    background: rgba(0, 0, 0, 0.55);
  }
}
</style>
