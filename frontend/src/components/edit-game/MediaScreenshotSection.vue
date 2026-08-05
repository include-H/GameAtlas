<template>
  <section class="media-block media-block--screenshots">
    <div v-if="!hideTitle" class="media-block__title">游戏截图</div>
    <div
      v-if="screenshots.length === 0"
      class="media-frame media-frame--screenshots"
    >
      <div
        class="app-glass-surface media-empty-action"
        role="button"
        tabindex="0"
        @click="emit('open-screenshot-selector')"
      >
        <icon-image class="media-empty-icon" />
        <span class="media-empty-title">未设置截图</span>
        <span class="media-empty-subtitle">点击添加截图</span>
      </div>
    </div>
    <div v-else class="media-frame media-frame--screenshots screenshots-frame">
      <div ref="screenshotsScrollRef" class="screenshots-scroll" @dragover="onScreenshotsDragOver" @dragend="onScreenshotsDragEnd">
        <a-image-preview-group infinite>
          <div class="screenshots-grid">
            <div
              v-for="(screenshot, index) in screenshots"
              :key="screenshot.asset_uid || screenshot.client_key"
              class="screenshot-thumb"
              :class="{
                'is-primary': index === 0,
                'is-dragging': draggedScreenshotKey === screenshot.client_key,
                'is-drop-target': dragOverScreenshotKey === screenshot.client_key,
              }"
              @dragenter.prevent="emit('screenshot-drag-enter', screenshot.client_key)"
              @dragover.prevent
              @drop.prevent="emit('screenshot-drop', screenshot.client_key)"
            >
              <ThumbImage :src="screenshot.path" />
              <div v-if="index === 0" class="screenshot-primary-badge">主图</div>
              <div class="screenshot-overlay">
                <div class="screenshot-overlay-actions">
                  <a-button
                    class="app-text-action-btn media-action-button screenshot-drag-handle"
                    type="text"
                    shape="circle"
                    size="small"
                    html-type="button"
                    title="拖拽排序"
                    draggable="true"
                    @dragstart="emit('screenshot-drag-start', screenshot.client_key)"
                    @dragend="emit('screenshot-drag-end')"
                  >
                    <icon-drag-arrow />
                  </a-button>
                  <a-button
                    class="app-text-action-btn media-action-button media-action-button--danger"
                    type="text"
                    status="danger"
                    shape="circle"
                    size="small"
                    html-type="button"
                    title="删除截图"
                    @click.stop="confirmRemoveScreenshot(screenshot.client_key)"
                  >
                    <icon-delete />
                  </a-button>
                </div>
              </div>
            </div>
          </div>
        </a-image-preview-group>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { IconDelete, IconDragArrow, IconImage } from '@arco-design/web-vue/es/icon'
import { Modal } from '@arco-design/web-vue'
import { ref, onBeforeUnmount } from 'vue'
import ThumbImage from './ThumbImage.vue'

interface EditableScreenshot {
  asset_uid?: string
  path: string
  client_key: string
}

defineProps<{
  screenshots: EditableScreenshot[]
  draggedScreenshotKey: string | null
  dragOverScreenshotKey: string | null
  hideTitle?: boolean
}>()

const emit = defineEmits<{
  'open-screenshot-selector': []
  'remove-screenshot': [clientKey: string]
  'screenshot-drag-start': [clientKey: string]
  'screenshot-drag-enter': [clientKey: string]
  'screenshot-drop': [clientKey: string]
  'screenshot-drag-end': []
}>()

const confirmRemoveScreenshot = (clientKey: string) => {
  Modal.confirm({
    title: '删除素材',
    content: '确定要移除这张截图吗？保存后将从磁盘删除。',
    okText: '删除',
    cancelText: '取消',
    okButtonProps: { status: 'danger' },
    onOk: () => emit('remove-screenshot', clientKey),
  })
}

const screenshotsScrollRef = ref<HTMLElement | null>(null)
let autoScrollRaf = 0

const SCROLL_ZONE = 60
const SCROLL_SPEED = 12

const onScreenshotsDragOver = (e: DragEvent) => {
  const container = screenshotsScrollRef.value
  if (!container) return

  const rect = container.getBoundingClientRect()
  const x = e.clientX

  if (x < rect.left || x > rect.right) {
    cancelAnimationFrame(autoScrollRaf)
    autoScrollRaf = 0
    return
  }

  if (x < rect.left + SCROLL_ZONE) {
    if (!autoScrollRaf) {
      const tick = () => {
        container.scrollLeft -= SCROLL_SPEED
        autoScrollRaf = requestAnimationFrame(tick)
      }
      autoScrollRaf = requestAnimationFrame(tick)
    }
  } else if (x > rect.right - SCROLL_ZONE) {
    if (!autoScrollRaf) {
      const tick = () => {
        container.scrollLeft += SCROLL_SPEED
        autoScrollRaf = requestAnimationFrame(tick)
      }
      autoScrollRaf = requestAnimationFrame(tick)
    }
  } else {
    cancelAnimationFrame(autoScrollRaf)
    autoScrollRaf = 0
  }
}

const onScreenshotsDragEnd = () => {
  cancelAnimationFrame(autoScrollRaf)
  autoScrollRaf = 0
}

onBeforeUnmount(() => {
  cancelAnimationFrame(autoScrollRaf)
})
</script>

<style scoped>
.media-block {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
}

.media-block__title {
  padding-left: 2px;
  font-size: 14px;
  font-weight: 700;
  color: var(--color-text-2);
}

.media-frame {
  position: relative;
  width: 100%;
  min-width: 0;
  overflow: hidden;
  border-radius: var(--media-panel-radius);
  border: 1px solid var(--app-card-border);
  background: transparent;
  display: flex;
  align-items: center;
  justify-content: center;
  box-sizing: border-box;
}

.media-frame--screenshots {
  height: 128px;
  min-height: 128px;
}

.media-empty-action {
  width: 100%;
  height: 100%;
  border: 1px dashed var(--color-border-2);
  border-radius: calc(var(--media-panel-radius) - 2px);
  background: color-mix(in srgb, var(--app-card-surface) 88%, transparent);
  color: var(--color-text-3);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 24px;
  box-sizing: border-box;
  cursor: pointer;
  transition: color 0.2s ease, background 0.2s ease, border-color 0.2s ease;
}

.media-empty-action:hover {
  color: rgb(var(--primary-6));
  background: rgba(var(--primary-6), 0.06);
  border-color: rgba(var(--primary-6), 0.45);
}

.media-empty-icon {
  font-size: 30px;
}

.media-empty-title {
  font-size: 14px;
  font-weight: 700;
}

.media-empty-subtitle {
  font-size: 12px;
  opacity: 0.75;
}

.screenshots-frame {
  align-items: stretch;
  justify-content: stretch;
  padding: var(--media-panel-padding);
  box-sizing: border-box;
}

.screenshots-scroll {
  width: 100%;
  height: 100%;
  min-height: 0;
  overflow-x: auto;
  overflow-y: hidden;
  scrollbar-width: thin;
  scrollbar-color: var(--color-border-3) transparent;
}

.screenshots-scroll::-webkit-scrollbar {
  height: 6px;
}

.screenshots-scroll::-webkit-scrollbar-track {
  background: transparent;
}

.screenshots-scroll::-webkit-scrollbar-thumb {
  background: var(--color-border-3);
  border-radius: 3px;
}

.screenshots-scroll :deep(.arco-image-preview-group) {
  display: block;
  width: max-content;
  min-width: 100%;
}

.screenshots-grid {
  display: flex;
  align-items: stretch;
  gap: 10px;
}

.screenshot-thumb {
  width: 220px;
  flex: 0 0 auto;
  aspect-ratio: 16 / 9;
  border-radius: 10px;
  overflow: hidden;
  background:
    linear-gradient(180deg, var(--color-border-1), transparent),
    color-mix(in srgb, var(--app-card-surface) 90%, transparent);
  cursor: grab;
  position: relative;
  border: 1px solid var(--app-card-border);
  transition: transform 0.18s ease, border-color 0.18s ease, box-shadow 0.18s ease, opacity 0.18s ease;
}

.screenshot-thumb.is-primary {
  border-color: rgb(var(--primary-6));
  box-shadow: 0 0 0 1px rgba(var(--primary-6), 0.35);
}

.screenshot-thumb.is-dragging {
  opacity: 0.45;
  transform: scale(0.98);
  cursor: grabbing;
}

.screenshot-thumb.is-drop-target {
  border-color: rgb(var(--primary-6));
  box-shadow: 0 0 0 1px rgba(var(--primary-6), 0.35);
}

.screenshot-primary-badge {
  position: absolute;
  top: 3px;
  left: 3px;
  background: rgb(var(--primary-6));
  color: var(--color-text-on-dark);
  font-size: 9px;
  font-weight: 700;
  padding: 1px 5px;
  border-radius: 3px;
  line-height: 1.3;
  z-index: 1;
}

.screenshot-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--app-scrim);
  opacity: 0;
  transition: opacity 0.2s ease;
}

.screenshot-overlay-actions {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  pointer-events: auto;
}

.screenshot-overlay-actions .media-action-button {
  width: 28px;
  height: 28px;
  min-width: 28px;
  font-size: 12px;
}

.screenshot-drag-handle {
  cursor: grab;
}

.screenshot-drag-handle:active {
  cursor: grabbing;
}

.screenshot-thumb:hover .screenshot-overlay {
  opacity: 1;
}

.screenshot-thumb:focus-within .screenshot-overlay {
  opacity: 1;
}

.screenshot-thumb :deep(.arco-image) {
  display: flex;
  width: 100%;
  height: 100%;
}

.screenshot-thumb :deep(.arco-image-img) {
  width: 100%;
  height: 100%;
  object-fit: contain;
  object-position: center;
}

@media (max-width: 768px) {
  .screenshot-thumb {
    width: 180px;
  }
}

@media (hover: none) {
  .screenshot-overlay {
    opacity: 1;
    align-items: flex-end;
    background: linear-gradient(180deg, transparent 0%, var(--app-scrim) 60%);
    padding: 6px;
  }
}
</style>
