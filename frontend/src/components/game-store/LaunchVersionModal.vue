<template>
  <a-modal
    :visible="visible"
    :footer="false"
    :width="480"
    :title="`开始游戏：${title}`"
    @cancel="emit('cancel')"
  >
    <div class="store-launch-list">
      <button
        v-for="option in options"
        :key="option.id"
        type="button"
        class="store-launch-item"
        @click="emit('select', option)"
      >
        <icon-play-arrow />
        <span class="store-launch-item__name">{{ option.version }}</span>
        <span class="store-launch-item__action">开始游戏</span>
      </button>
    </div>
  </a-modal>
</template>

<script setup lang="ts">
import { IconPlayArrow } from '@arco-design/web-vue/es/icon'
import type { LaunchOption } from '@/composables/useLaunchGame'

defineProps<{
  visible: boolean
  title: string
  options: LaunchOption[]
}>()

const emit = defineEmits<{
  (e: 'select', option: LaunchOption): void
  (e: 'cancel'): void
}>()
</script>

<style scoped>
/* ---------- 多版本启动选择 ---------- */
.store-launch-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.store-launch-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border: 1px solid var(--app-card-border);
  border-radius: 8px;
  background: var(--color-fill-2);
  color: var(--color-text-1);
  cursor: pointer;
  text-align: left;
  transition: background 120ms ease, border-color 120ms ease;
}

.store-launch-item:hover {
  background: var(--color-fill-3);
  border-color: var(--app-glass-border-hover);
}

.store-launch-item__name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.store-launch-item__action {
  font-size: 13px;
  color: var(--color-primary-6);
}
</style>
