<template>
  <div
    class="media-thumb"
    :class="[thumbClass, getThumbClass(mediaKey, isPrimary, draggedKey, dragOverKey)]"
    @dragenter.prevent="emit('drag-enter', mediaKey)"
    @dragover.prevent
    @drop.prevent="emit('drop', mediaKey)"
  >
    <slot />
    <div v-if="badge" class="media-thumb__badge">{{ badge }}</div>
    <div class="media-thumb__actions">
      <slot name="extra-actions" />
      <a-button
        class="app-text-action-btn media-action-button media-drag-handle"
        type="text"
        shape="circle"
        size="small"
        html-type="button"
        title="拖拽排序"
        draggable="true"
        @dragstart="emit('drag-start', mediaKey)"
        @dragend="emit('drag-end')"
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
        :title="deleteTitle"
        @click.stop="confirmRemove"
      >
        <icon-delete />
      </a-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { IconDelete, IconDragArrow } from '@arco-design/web-vue/es/icon'
import { Modal } from '@arco-design/web-vue'

const props = defineProps<{
  thumbClass: string
  mediaKey: string
  isPrimary: boolean
  draggedKey: string | null
  dragOverKey: string | null
  badge?: string
  deleteTitle: string
  confirmLabel: string
}>()

const emit = defineEmits<{
  'drag-start': [key: string]
  'drag-end': []
  'drag-enter': [key: string]
  drop: [key: string]
  remove: []
}>()

const getThumbClass = (
  key: string,
  isPrimary: boolean,
  draggedKey: string | null,
  dragOverKey: string | null,
) => ({
  'is-primary': isPrimary,
  'is-dragging': draggedKey === key,
  'is-drop-target': dragOverKey === key,
})

const confirmRemove = () => {
  Modal.confirm({
    title: '删除素材',
    content: `确定要移除${props.confirmLabel}吗？保存后将从磁盘删除。`,
    okText: '删除',
    cancelText: '取消',
    okButtonProps: { status: 'danger' },
    onOk: () => emit('remove'),
  })
}
</script>

<style scoped>
.media-thumb {
  position: relative;
  flex: 0 0 auto;
  align-self: flex-start;
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid var(--app-card-border);
  background:
    linear-gradient(180deg, var(--color-border-1), transparent),
    color-mix(in srgb, var(--app-card-surface) 90%, transparent);
  transition: border-color 0.18s ease, box-shadow 0.18s ease, opacity 0.18s ease, transform 0.18s ease;
}

.media-thumb--cover {
  width: 84px;
  aspect-ratio: 2 / 3;
}

.media-thumb--banner {
  width: 150px;
  aspect-ratio: 16 / 9;
}

.media-thumb--logo {
  width: 160px;
  aspect-ratio: 16 / 9;
}

.media-thumb--video {
  width: 170px;
  aspect-ratio: 16 / 9;
}

.media-thumb.is-primary {
  border-color: rgb(var(--primary-6));
  box-shadow: 0 0 0 1px rgba(var(--primary-6), 0.35);
}

.media-thumb.is-dragging {
  opacity: 0.45;
  transform: scale(0.98);
}

.media-thumb.is-drop-target {
  border-color: rgb(var(--primary-6));
  box-shadow: 0 0 0 1px rgba(var(--primary-6), 0.35);
}

.media-thumb__badge {
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

/* 右上角操作簇（无全遮罩）：拖动/删除等按钮以深色小圆钮常驻右上角，
   桌面悬停浮现、触屏常显；图片内容始终可见，不再被遮罩盖住。 */
.media-thumb__actions {
  position: absolute;
  top: 4px;
  right: 4px;
  z-index: 2;
  display: flex;
  align-items: center;
  gap: 2px;
  padding: 0;
  opacity: 0;
  transition: opacity 0.18s ease;
}

.media-thumb__actions .media-action-button {
  width: 26px;
  height: 26px;
  min-width: 26px;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.92);
  background: rgba(0, 0, 0, 0.42);
  border-radius: 50%;
}

.media-thumb__actions .media-action-button:hover {
  background: rgba(0, 0, 0, 0.7);
}

.media-thumb__actions .media-action-button--danger:hover {
  color: rgb(var(--danger-6));
}

.media-thumb:hover .media-thumb__actions,
.media-thumb:focus-within .media-thumb__actions {
  opacity: 1;
}

/* 触屏：无悬停，按钮常显 */
@media (hover: none) {
  .media-thumb__actions {
    opacity: 1;
  }
}

.media-drag-handle {
  cursor: grab;
}

.media-drag-handle:active {
  cursor: grabbing;
}

.media-thumb :deep(.arco-image) {
  display: flex;
  width: 100%;
  height: 100%;
}

.media-thumb :deep(.arco-image-img) {
  width: 100%;
  height: 100%;
  object-fit: contain;
  object-position: center;
}
</style>
