<template>
  <div
    class="media-card__empty"
    :class="modifierClass"
    role="button"
    tabindex="0"
    @click="emit('click')"
    @keydown.enter="emit('click')"
    @keydown.space.prevent="emit('click')"
  >
    <slot name="icon">
      <icon-image class="media-card__empty-icon" />
    </slot>
    <span class="media-card__empty-title">{{ title }}</span>
    <span class="media-card__empty-subtitle">{{ subtitle }}</span>
  </div>
</template>

<script setup lang="ts">
import { IconImage } from '@arco-design/web-vue/es/icon'

defineProps<{
  title: string
  subtitle: string
  modifierClass?: string
}>()

const emit = defineEmits<{
  click: []
}>()
</script>

<style scoped>
.media-card__empty {
  flex: 1 1 auto;
  width: 100%;
  min-height: 180px;
  border: 1px dashed var(--color-border-2);
  border-radius: 10px;
  background: color-mix(in srgb, var(--app-card-surface) 88%, transparent);
  color: var(--color-text-3);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 24px;
  box-sizing: border-box;
  cursor: pointer;
  transition: color 0.2s ease, background 0.2s ease, border-color 0.2s ease;
}

.media-card__empty:hover {
  color: rgb(var(--primary-6));
  background: rgba(var(--primary-6), 0.06);
  border-color: rgba(var(--primary-6), 0.45);
}

.media-card__empty--video {
  min-height: 150px;
}

.media-card__empty-icon {
  font-size: 30px;
}

.media-card__empty-title {
  font-size: 14px;
  font-weight: 700;
}

.media-card__empty-subtitle {
  font-size: 12px;
  opacity: 0.75;
}
</style>
