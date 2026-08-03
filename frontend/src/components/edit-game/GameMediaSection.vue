<template>
  <div class="media-section-list">
      <section class="media-block media-block--cover">
        <div class="media-block__title">封面图</div>
        <div
          v-if="covers.length === 0"
          class="media-frame media-frame--cover"
        >
          <div
            class="app-glass-surface media-empty-action"
            role="button"
            tabindex="0"
            @click="emit('open-cover-selector')"
          >
            <icon-image class="media-empty-icon" />
            <span class="media-empty-title">未设置封面</span>
            <span class="media-empty-subtitle">点击选择图片</span>
          </div>
        </div>
        <div v-else class="media-frame media-frame--cover covers-frame">
          <div class="covers-scroll">
            <a-image-preview-group infinite>
              <div class="covers-grid">
                <div
                  v-for="(cover, index) in covers"
                  :key="cover.asset_uid || cover.path"
                  class="cover-thumb"
                  :class="{ 'is-primary': index === 0 }"
                >
                  <a-image
                    :src="cover.path"
                    width="100%"
                    height="100%"
                    fit="contain"
                    hide-footer
                  />
                  <div v-if="index === 0" class="cover-primary-badge">主封面</div>
                  <div class="cover-overlay">
                    <div class="cover-overlay-actions">
                      <a-button
                        class="app-text-action-btn media-action-button"
                        type="text"
                        shape="circle"
                        size="small"
                        html-type="button"
                        title="管理封面"
                        @click.stop="emit('open-cover-selector')"
                      >
                        <icon-settings />
                      </a-button>
                      <a-button
                        v-if="index !== 0"
                        class="app-text-action-btn media-action-button"
                        type="text"
                        shape="circle"
                        size="small"
                        html-type="button"
                        title="设为主封面"
                        @click.stop="emit('set-primary-cover', index)"
                      >
                        <icon-star />
                      </a-button>
                      <a-button
                        class="app-text-action-btn media-action-button media-action-button--danger"
                        type="text"
                        status="danger"
                        shape="circle"
                        size="small"
                        html-type="button"
                        @click.stop="emit('remove-cover', index)"
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

      <section class="media-block media-block--banner">
        <div class="media-block__title">横幅图</div>
        <div
          v-if="banners.length === 0"
          class="media-frame media-frame--banner"
        >
          <div
            class="app-glass-surface media-empty-action"
            role="button"
            tabindex="0"
            @click="emit('open-banner-selector')"
          >
            <icon-image class="media-empty-icon" />
            <span class="media-empty-title">未设置横幅</span>
            <span class="media-empty-subtitle">点击选择图片</span>
          </div>
        </div>
        <div v-else class="media-frame media-frame--banner banners-frame">
          <div class="banners-scroll">
            <a-image-preview-group infinite>
              <div class="banners-grid">
                <div
                  v-for="(banner, index) in banners"
                  :key="banner.asset_uid || banner.path"
                  class="banner-thumb"
                  :class="{ 'is-primary': index === 0 }"
                >
                  <a-image
                    :src="banner.path"
                    width="100%"
                    height="100%"
                    fit="contain"
                    hide-footer
                  />
                  <div v-if="index === 0" class="banner-primary-badge">主横幅</div>
                  <div class="banner-overlay">
                    <div class="banner-overlay-actions">
                      <a-button
                        class="app-text-action-btn media-action-button"
                        type="text"
                        shape="circle"
                        size="small"
                        html-type="button"
                        title="管理横幅"
                        @click.stop="emit('open-banner-selector')"
                      >
                        <icon-settings />
                      </a-button>
                      <a-button
                        v-if="index !== 0"
                        class="app-text-action-btn media-action-button"
                        type="text"
                        shape="circle"
                        size="small"
                        html-type="button"
                        title="设为主横幅"
                        @click.stop="emit('set-primary-banner', index)"
                      >
                        <icon-star />
                      </a-button>
                      <a-button
                        class="app-text-action-btn media-action-button media-action-button--danger"
                        type="text"
                        status="danger"
                        shape="circle"
                        size="small"
                        html-type="button"
                        @click.stop="emit('remove-banner', index)"
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

      <section class="media-block media-block--logo">
        <div class="media-block__title">Logo</div>
        <div
          v-if="!logoImage"
          class="media-frame media-frame--logo"
        >
          <div
            class="app-glass-surface media-empty-action"
            role="button"
            tabindex="0"
            @click="emit('open-logo-selector')"
          >
            <icon-image class="media-empty-icon" />
            <span class="media-empty-title">未设置 Logo</span>
            <span class="media-empty-subtitle">点击选择图片</span>
          </div>
        </div>
        <div v-else class="media-frame media-frame--logo logo-frame">
          <a-image
            :src="logoImage"
            width="100%"
            height="100%"
            fit="contain"
            hide-footer
          />
          <div class="media-overlay">
            <div class="media-overlay-actions">
              <a-button
                class="app-text-action-btn media-action-button"
                type="text"
                shape="circle"
                size="small"
                html-type="button"
                title="更换 Logo"
                @click.stop="emit('open-logo-selector')"
              >
                <icon-settings />
              </a-button>
              <a-button
                class="app-text-action-btn media-action-button media-action-button--danger"
                type="text"
                status="danger"
                shape="circle"
                size="small"
                html-type="button"
                @click.stop="emit('remove-logo')"
              >
                <icon-delete />
              </a-button>
            </div>
          </div>
        </div>
      </section>

      <MediaScreenshotSection
        :screenshots="screenshots"
        :dragged-screenshot-key="draggedScreenshotKey"
        :drag-over-screenshot-key="dragOverScreenshotKey"
        @open-screenshot-selector="emit('open-screenshot-selector')"
        @remove-screenshot="(key) => emit('remove-screenshot', key)"
        @screenshot-drag-start="(key) => emit('screenshot-drag-start', key)"
        @screenshot-drag-enter="(key) => emit('screenshot-drag-enter', key)"
        @screenshot-drop="(key) => emit('screenshot-drop', key)"
        @screenshot-drag-end="emit('screenshot-drag-end')"
      />

      <section class="media-block media-block--video">
        <div class="media-block__title-row">
          <span class="media-block__title">预告片</span>
          <a-button
            v-if="previewVideos.length > 0"
            type="primary"
            size="mini"
            html-type="button"
            :loading="isUploadingVideo"
            @click="openVideoFilePicker"
          >
            <template #icon><icon-upload /></template>
            {{ isUploadingVideo ? '上传中...' : '上传预告片' }}
          </a-button>
        </div>
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
        <div v-if="previewVideos.length > 0" class="video-library-list">
          <div
            v-for="(video, index) in previewVideos"
            :key="video.asset_uid || video.path"
            class="video-library-item"
            :class="{ 'is-primary': index === 0 }"
          >
            <div class="video-library-item__icon">
              <icon-video-camera />
            </div>
            <div class="video-library-item__info">
              <div class="video-library-item__title-row">
                <span class="video-library-item__title">预告片 {{ index + 1 }}</span>
                <a-tag v-if="index === 0" size="small" color="arcoblue">首个展示</a-tag>
              </div>
              <span class="video-library-item__name">{{ getVideoFileName(video.path) }}</span>
            </div>
            <div class="video-library-item__actions">
              <a-button
                class="app-text-action-btn"
                type="text"
                shape="circle"
                size="small"
                html-type="button"
                title="上移"
                :disabled="index === 0"
                @click="emit('reorder-video', { key: video.asset_uid || video.path, direction: -1 })"
              >
                <icon-arrow-up />
              </a-button>
              <a-button
                class="app-text-action-btn"
                type="text"
                shape="circle"
                size="small"
                html-type="button"
                title="下移"
                :disabled="index === previewVideos.length - 1"
                @click="emit('reorder-video', { key: video.asset_uid || video.path, direction: 1 })"
              >
                <icon-arrow-down />
              </a-button>
              <a-button
                class="app-text-action-btn"
                type="text"
                status="danger"
                shape="circle"
                size="small"
                html-type="button"
                title="删除预告片"
                @click="emit('remove-video', video.asset_uid)"
              >
                <icon-delete />
              </a-button>
            </div>
          </div>
        </div>
        <div
          v-else
          class="app-glass-surface media-empty-action media-empty-action--video"
          role="button"
          tabindex="0"
          @click="openVideoFilePicker"
          @keydown.enter="openVideoFilePicker"
          @keydown.space.prevent="openVideoFilePicker"
        >
          <icon-video-camera class="media-empty-icon" />
          <span class="media-empty-title">未设置预告片</span>
          <span class="media-empty-subtitle">上传 MP4 或 WebM 视频</span>
        </div>
      </section>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import {
  IconArrowDown,
  IconArrowUp,
  IconDelete,
  IconImage,
  IconSettings,
  IconStar,
  IconUpload,
  IconVideoCamera,
} from '@arco-design/web-vue/es/icon'
import MediaScreenshotSection from './MediaScreenshotSection.vue'

interface EditableCover {
  asset_uid?: string
  path: string
}

interface EditableBanner {
  asset_uid?: string
  path: string
}

interface EditableScreenshot {
  asset_uid?: string
  path: string
  client_key: string
}

interface EditableVideo {
  asset_uid?: string
  path: string
}

const emit = defineEmits<{
  'open-cover-selector': []
  'remove-cover': [index: number]
  'set-primary-cover': [index: number]
  'reorder-cover': [payload: { key: string; direction: -1 | 1 }]
  'open-banner-selector': []
  'remove-banner': [index: number]
  'set-primary-banner': [index: number]
  'video-file-change': [event: Event]
  'reorder-video': [payload: { key: string; direction: -1 | 1 }]
  'remove-video': [assetUid?: string]
  'open-screenshot-selector': []
  'remove-screenshot': [clientKey: string]
  'screenshot-drag-start': [clientKey: string]
  'screenshot-drag-enter': [clientKey: string]
  'screenshot-drop': [clientKey: string]
  'screenshot-drag-end': []
  'open-logo-selector': []
  'remove-logo': []
}>()

defineProps<{
  title: string
  covers: EditableCover[]
  banners: EditableBanner[]
  previewVideos: EditableVideo[]
  isUploadingVideo: boolean
  videoUploadProgress: number
  videoUploadFileName: string
  screenshots: EditableScreenshot[]
  draggedScreenshotKey: string | null
  dragOverScreenshotKey: string | null
  logoImage: string
}>()

const videoFileInput = ref<HTMLInputElement | null>(null)

const openVideoFilePicker = () => {
  videoFileInput.value?.click()
}

const getVideoFileName = (path: string) => {
  return path.split(/[\\/]/).pop() || path
}
</script>

<style scoped>
.media-section-list {
  --media-gap: 22px;
  --media-panel-radius: 14px;
  --media-panel-padding: 14px;
  --media-frame-height: clamp(220px, 24vw, 300px);
  --media-banner-height: clamp(200px, 22vw, 280px);

  display: grid;
  grid-template-columns: repeat(10, minmax(0, 1fr));
  grid-template-areas:
    "cover cover cover cover cover screenshots screenshots screenshots screenshots screenshots"
    "logo logo logo banner banner banner banner banner banner banner"
    "video video video video video video video video video video";
  gap: 10px;
}

.media-block {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
}

.media-block--cover {
  grid-area: cover;
}

.media-block--banner {
  grid-area: banner;
}

.media-block--logo {
  grid-area: logo;
}

.media-block--screenshots {
  grid-area: screenshots;
}

.media-block--video {
  grid-area: video;
}

.media-block__title {
  padding-left: 2px;
  font-size: 14px;
  font-weight: 700;
  color: var(--color-text-2);
}

.media-block__title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
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

.media-frame--cover {
  height: var(--media-frame-height);
  min-height: var(--media-frame-height);
}

.media-frame--banner {
  width: 100%;
  height: 270px;
  min-height: 270px;
}

.media-frame--logo {
  height: 270px;
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

.media-empty-action--video {
  min-height: 150px;
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

.video-library-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.video-library-item {
  min-width: 0;
  padding: 10px 12px;
  border: 1px solid var(--color-border-2);
  border-radius: 8px;
  background: var(--color-fill-1);
  display: flex;
  align-items: center;
  gap: 10px;
}

.video-library-item.is-primary {
  border-color: rgba(var(--primary-6), 0.6);
  box-shadow: 0 0 0 1px rgba(var(--primary-6), 0.25);
}

.video-library-item__icon {
  width: 32px;
  height: 32px;
  flex: 0 0 32px;
  border-radius: 6px;
  color: rgb(var(--primary-6));
  background: rgba(var(--primary-6), 0.12);
  display: flex;
  align-items: center;
  justify-content: center;
}

.video-library-item__info {
  min-width: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.video-library-item__title-row {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 6px;
}

.video-library-item__title {
  color: var(--color-text-2);
  font-size: 13px;
  font-weight: 700;
}

.video-library-item__name {
  min-width: 0;
  color: var(--color-text-3);
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.video-library-item__actions {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 2px;
}

.media-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--app-scrim);
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.2s ease;
}


.media-overlay-actions,
.cover-overlay-actions {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  pointer-events: auto;
}

.cover-overlay-actions .media-action-button {
  width: 28px;
  height: 28px;
  min-width: 28px;
  font-size: 12px;
}

.media-frame:hover .media-overlay,
.cover-thumb:hover .cover-overlay {
  opacity: 1;
}

.media-frame :deep(.arco-image),
.cover-thumb :deep(.arco-image),
.logo-frame :deep(.arco-image) {
  display: flex;
  width: 100%;
  height: 100%;
}

.media-frame :deep(.arco-image-img),
.cover-thumb :deep(.arco-image-img),
.logo-frame :deep(.arco-image-img) {
  width: 100%;
  height: 100%;
  object-fit: contain;
  object-position: center;
}

.covers-frame,
.banners-frame {
  align-items: stretch;
  justify-content: stretch;
  padding: var(--media-panel-padding);
  box-sizing: border-box;
}

.covers-scroll,
.banners-scroll {
  width: 100%;
  height: 100%;
  min-height: 0;
  overflow-x: hidden;
  overflow-y: auto;
}

.covers-scroll :deep(.arco-image-preview-group) {
  display: block;
  width: 100%;
}

.covers-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  align-items: start;
  gap: 12px;
}

.banners-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  align-items: start;
  gap: 12px;
}

.cover-thumb {
  width: 100%;
  aspect-ratio: 2 / 3;
  border-radius: 10px;
  overflow: hidden;
  background:
    linear-gradient(180deg, var(--color-border-1), transparent),
    color-mix(in srgb, var(--app-card-surface) 90%, transparent);
  position: relative;
  border: 1px solid var(--app-card-border);
  transition: border-color 0.18s ease, box-shadow 0.18s ease;
}

.cover-thumb.is-primary {
  border-color: rgb(var(--primary-6));
  box-shadow: 0 0 0 1px rgba(var(--primary-6), 0.35);
}

.banner-thumb {
  width: 100%;
  aspect-ratio: 16 / 9;
  border-radius: 10px;
  overflow: hidden;
  background:
    linear-gradient(180deg, var(--color-border-1), transparent),
    color-mix(in srgb, var(--app-card-surface) 90%, transparent);
  position: relative;
  border: 1px solid var(--app-card-border);
  transition: border-color 0.18s ease, box-shadow 0.18s ease;
}

.banner-thumb.is-primary {
  border-color: rgb(var(--primary-6));
  box-shadow: 0 0 0 1px rgba(var(--primary-6), 0.35);
}

.banner-primary-badge {
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

.banner-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--app-scrim);
  opacity: 0;
  transition: opacity 0.2s ease;
}

.banner-overlay-actions {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  pointer-events: auto;
}

.banner-thumb:hover .banner-overlay {
  opacity: 1;
}

.cover-primary-badge {
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

.cover-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--app-scrim);
  opacity: 0;
  transition: opacity 0.2s ease;
}

.logo-frame {
  display: flex;
  align-items: center;
  justify-content: center;
}

@media (max-width: 768px) {
  .media-section-list {
    --media-gap: 16px;
    --media-frame-height: 220px;
    --media-banner-height: 220px;
  }

  .media-block--logo {
    width: 100%;
  }

  .covers-grid,
  .banners-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 576px) {
  .media-section-list {
    --media-frame-height: 190px;
    --media-banner-height: 190px;
    --media-panel-padding: 10px;
  }

  .media-frame--banner {
    aspect-ratio: 16 / 9;
  }

  .video-library-item {
    align-items: flex-start;
  }

  .video-library-item__actions {
    margin-left: -4px;
  }
}
</style>
