<template>
  <a-modal
    :visible="visible"
    title="设置预告片"
    :width="720"
    :footer="false"
    @update:visible="emit('update:visible', $event)"
  >
    <div class="cover-selector-content">
      <input
        ref="videoFileInput"
        type="file"
        accept="video/mp4,video/webm"
        class="hidden-file-input"
        @change="emit('video-file-change', $event)"
      />
      <a-button class="app-text-action-btn" type="text" long html-type="button" :loading="isUploadingVideo" @click="openVideoFilePicker">
        <template #icon>
          <icon-upload />
        </template>
        {{ isUploadingVideo ? '上传中...' : '上传 MP4 / WebM' }}
      </a-button>
      <div v-if="isUploadingVideo || videoUploadProgress > 0" class="video-upload-progress">
        <div class="video-upload-progress__meta">
          <span>{{ videoUploadFileName || '预告片上传中' }}</span>
          <span>{{ videoUploadProgress }}%</span>
        </div>
        <a-progress :percent="videoUploadProgress" :show-text="false" size="small" />
      </div>
      <div v-if="previewVideos.length > 0" class="video-library-card">
        <div class="video-library-card__header">
          <span>当前预告片</span>
          <span>{{ previewVideos.length }} 个</span>
        </div>
        <div class="video-library-list">
          <div
            v-for="(video, index) in previewVideos"
            :key="video.asset_uid || video.path"
            class="video-library-item"
            :class="{ 'is-primary': index === 0 }"
          >
            <div class="video-library-item__preview">
              <div class="video-library-item__thumb">
                <img
                  v-if="bannerImage || coverImage"
                  :src="bannerImage || coverImage || ''"
                  :alt="`预告片 ${index + 1}`"
                />
                <div v-else class="video-library-item__thumb-placeholder">
                  <icon-video-camera />
                </div>
              </div>
              <div class="video-library-item__info">
                <div class="video-library-item__meta-row">
                  <span class="video-library-item__title">预告片 {{ index + 1 }}</span>
                  <a-tag v-if="index === 0" size="small" color="arcoblue">首个展示</a-tag>
                  <span class="video-library-item__path">{{ video.asset_uid || video.path }}</span>
                </div>
              </div>
            </div>
            <div class="video-library-item__actions">
              <a-button
                class="app-text-action-btn"
                size="mini"
                type="text"
                html-type="button"
                :disabled="index === 0"
                @click="emit('reorder-video', { key: video.asset_uid || video.path, direction: -1 })"
              >
                上移
              </a-button>
              <a-button
                class="app-text-action-btn"
                size="mini"
                type="text"
                html-type="button"
                :disabled="index === previewVideos.length - 1"
                @click="emit('reorder-video', { key: video.asset_uid || video.path, direction: 1 })"
              >
                下移
              </a-button>
              <a-button class="app-text-action-btn" size="mini" type="text" status="danger" html-type="button" @click="emit('remove-video', video.asset_uid)">
                删除
              </a-button>
            </div>
          </div>
        </div>
      </div>
      <a-empty
        v-else
        description="还没有添加预告片，可上传本地视频"
        class="video-library-empty"
      />
      <div class="cover-selector-actions">
        <a-button class="app-text-action-btn" type="text" html-type="button" @click="emit('update:visible', false)">完成</a-button>
      </div>
    </div>
  </a-modal>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { IconUpload, IconVideoCamera } from '@arco-design/web-vue/es/icon'

interface EditableVideo {
  id?: number
  asset_uid?: string
  path: string
}

defineProps<{
  visible: boolean
  isUploadingVideo: boolean
  videoUploadProgress: number
  videoUploadFileName: string
  previewVideos: EditableVideo[]
  bannerImage: string
  coverImage: string
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'video-file-change': [event: Event]
  'reorder-video': [payload: { key: string; direction: -1 | 1 }]
  'remove-video': [assetUid?: string]
}>()

const videoFileInput = ref<HTMLInputElement | null>(null)

const openVideoFilePicker = () => {
  videoFileInput.value?.click()
}
</script>

<style scoped>
.cover-selector-content {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.cover-selector-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 4px;
}

.hidden-file-input {
  display: none;
}

.video-upload-progress {
  margin-top: 8px;
  padding: 10px;
  border-radius: 8px;
  border: 1px solid var(--color-border-2);
  background: var(--color-fill-2);
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.video-upload-progress__meta {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  color: var(--color-text-2);
}

.video-library-card {
  margin-top: 10px;
  border: 1px solid var(--color-border-2);
  border-radius: 10px;
  background: var(--color-fill-2);
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.video-library-card__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 13px;
  font-weight: 700;
  color: var(--color-text-2);
}

.video-library-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  max-height: 360px;
  overflow-y: auto;
}

.video-library-item {
  border: 1px solid var(--color-border-2);
  border-radius: 10px;
  background: var(--color-fill-1);
  padding: 10px 12px;
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 10px;
  overflow: hidden;
}

.video-library-item.is-primary {
  border-color: rgba(var(--primary-6), 0.6);
  box-shadow: 0 0 0 1px rgba(var(--primary-6), 0.35);
}

.video-library-item__preview {
  display: flex;
  gap: 10px;
  align-items: center;
  min-width: 0;
  border-radius: 8px;
  padding: 4px;
}

.video-library-item__thumb {
  width: 132px;
  height: 74px;
  border-radius: 8px;
  overflow: hidden;
  flex-shrink: 0;
  border: 1px solid var(--color-border-2);
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-fill-2);
}

.video-library-item__thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.video-library-item__thumb-placeholder {
  color: var(--color-text-3);
  font-size: 20px;
}

.video-library-item__info {
  min-width: 0;
  flex: 1;
  display: flex;
  align-items: center;
}

.video-library-item__meta-row {
  display: flex;
  align-items: center;
  flex-wrap: nowrap;
  gap: 8px;
  min-width: 0;
  width: 100%;
  overflow: hidden;
}

.video-library-item__title {
  font-size: 13px;
  font-weight: 700;
  color: var(--color-text-2);
  white-space: nowrap;
}

.video-library-item__path {
  min-width: 0;
  color: var(--color-text-3);
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.video-library-item__actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 4px;
  flex-shrink: 0;
  min-width: fit-content;
}

.video-library-empty {
  margin-top: 16px;
}

@media (max-width: 768px) {
  .video-library-item {
    padding: 8px;
  }

  .video-library-item__actions {
    justify-content: flex-end;
  }
}

@media (max-width: 560px) {
  .video-library-item {
    grid-template-columns: minmax(0, 1fr);
    align-items: stretch;
  }

  .video-library-item__actions {
    width: 100%;
    justify-content: flex-start;
    flex-wrap: wrap;
  }
}
</style>
