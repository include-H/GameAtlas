<template>
  <media-card-frame title="预告片" :count="countText" card-class="media-card--video">
    <template #toolbar>
      <a-button
        v-if="previewVideos.length > 0"
        class="app-text-action-btn media-card__action"
        type="text"
        size="small"
        html-type="button"
        :loading="isUploadingVideo"
        @click="openVideoFilePicker"
      >
        <template #icon><icon-upload /></template>
        {{ isUploadingVideo ? '上传中...' : '上传预告片' }}
      </a-button>
    </template>

    <input
      ref="videoFileInput"
      type="file"
      multiple
      accept="video/mp4,video/webm"
      class="hidden-file-input"
      @change="emit('video-file-change', $event)"
    />
    <div v-if="isUploadingVideo" class="video-upload-progress">
      <div class="video-upload-progress__meta">
        <span>{{ videoUploadFileName || '预告片上传中' }}</span>
        <span>{{ videoUploadProgress }}%</span>
      </div>
      <a-progress :percent="videoUploadProgress" :show-text="false" size="small" />
    </div>
    <div v-if="previewVideos.length > 0" class="media-card__preview media-card__preview--video">
      <video
        :src="previewVideos[0].path"
        :poster="previewVideos[0].poster_path || undefined"
        class="video-card__player"
        controls
        preload="metadata"
      />
      <div class="media-card__preview-badge">首个展示</div>
    </div>
    <media-empty-state
      v-else
      title="未设置预告片"
      subtitle="上传 MP4 或 WebM 视频"
      modifier-class="media-card__empty--video"
      @click="openVideoFilePicker"
    >
      <icon-video-camera class="media-card__empty-icon" />
    </media-empty-state>
    <div
      v-if="previewVideos.length > 0"
      ref="videosScrollRef"
      class="media-card__thumbs"
      @dragover="onGridDragOver($event, videosScrollRef)"
      @dragleave="stopGridAutoScroll"
      @dragend="stopGridAutoScroll"
    >
      <media-image-thumb
        v-for="(video, index) in previewVideos"
        :key="video.asset_uid || video.path"
        thumb-class="media-thumb--video"
        :media-key="video.asset_uid || video.path"
        :is-primary="index === 0"
        :dragged-key="draggedVideoKey"
        :drag-over-key="dragOverVideoKey"
        :badge="index === 0 ? '首个展示' : undefined"
        delete-title="删除预告片"
        confirm-label="这个预告片"
        @drag-start="(key) => emit('video-drag-start', key)"
        @drag-end="emit('video-drag-end')"
        @drag-enter="(key) => emit('video-drag-enter', key)"
        @drop="(key) => emit('video-drop', key)"
        @remove="emit('remove-video', video.asset_uid)"
      >
        <img
          v-if="video.poster_path"
          :src="video.poster_path"
          class="media-thumb--video__poster"
          alt=""
        />
        <div v-else class="media-thumb--video__placeholder">
          <icon-video-camera />
        </div>
      </media-image-thumb>
    </div>
  </media-card-frame>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { IconUpload, IconVideoCamera } from '@arco-design/web-vue/es/icon'
import { useGridAutoScroll } from '@/composables/useGridAutoScroll'
import MediaCardFrame from './MediaCardFrame.vue'
import MediaEmptyState from './MediaEmptyState.vue'
import MediaImageThumb from './MediaImageThumb.vue'

interface EditableVideo {
  asset_uid?: string
  path: string
  poster_path?: string | null
}

const props = defineProps<{
  previewVideos: EditableVideo[]
  isUploadingVideo: boolean
  videoUploadProgress: number
  videoUploadFileName: string
  draggedVideoKey: string | null
  dragOverVideoKey: string | null
}>()

const emit = defineEmits<{
  'video-file-change': [event: Event]
  'video-drag-start': [key: string]
  'video-drag-enter': [key: string]
  'video-drop': [key: string]
  'video-drag-end': []
  'remove-video': [assetUid?: string]
}>()

const countText = props.previewVideos.length > 0 ? `${props.previewVideos.length} 个` : ''

const videoFileInput = ref<HTMLInputElement | null>(null)
const videosScrollRef = ref<HTMLElement | null>(null)
const { onGridDragOver, stopGridAutoScroll } = useGridAutoScroll()

const openVideoFilePicker = () => {
  videoFileInput.value?.click()
}
</script>

<style scoped>
.media-card__preview {
  position: relative;
  flex: 1 1 auto;
  min-height: 0;
  width: 100%;
  border: 1px solid var(--app-card-border);
  border-radius: 10px;
  overflow: hidden;
  background:
    linear-gradient(180deg, var(--color-border-1), transparent),
    color-mix(in srgb, var(--app-card-surface) 90%, transparent);
  display: flex;
  align-items: center;
  justify-content: center;
}

.media-card__preview--video {
  height: clamp(180px, 22vh, 300px);
}

.media-card__preview-badge {
  position: absolute;
  top: 8px;
  left: 8px;
  background: rgb(var(--primary-6));
  color: var(--color-text-on-dark);
  font-size: 10px;
  font-weight: 700;
  padding: 2px 7px;
  border-radius: 4px;
  line-height: 1.4;
  z-index: 1;
}

.media-card__thumbs {
  display: flex;
  gap: 8px;
  flex: 0 0 auto;
  align-items: flex-start;
  min-height: 0;
  overflow-x: auto;
  padding: 4px 2px;
  scrollbar-width: thin;
}

.media-card__empty-icon {
  font-size: 30px;
}

.video-card__player {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: contain;
  background: var(--color-media-bg);
}

.hidden-file-input {
  display: none;
}

.video-upload-progress {
  padding: 10px 12px;
  border: 1px solid var(--color-border-2);
  border-radius: 8px;
  background: var(--color-fill-2);
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.video-upload-progress__meta {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  color: var(--color-text-2);
  font-size: 12px;
}

.media-thumb--video__poster {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.media-thumb--video__placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text-3);
  background: color-mix(in srgb, var(--app-card-surface) 90%, transparent);
  font-size: 22px;
}
</style>
