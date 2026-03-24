<template>
  <a-row :gutter="16">
    <a-col :span="8">
      <a-form-item label="封面图">
        <div class="media-section media-section--cover">
          <div class="media-frame media-frame--cover">
            <div v-if="coverImage" class="media-preview">
              <a-image
                :src="coverImage"
                :alt="title"
                width="100%"
                height="100%"
                fit="cover"
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
                    @click.stop="emit('open-cover-selector')"
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
                    @click.stop="emit('remove-cover')"
                  >
                    <icon-delete />
                  </a-button>
                </div>
              </div>
            </div>
            <div
              v-else
              class="media-empty-action"
              role="button"
              tabindex="0"
              @click="emit('open-cover-selector')"
            >
              <icon-image class="media-empty-icon" />
              <span class="media-empty-title">未设置封面</span>
              <span class="media-empty-subtitle">点击选择图片</span>
            </div>
          </div>
        </div>
      </a-form-item>
    </a-col>

    <a-col :span="8">
      <a-form-item label="横幅图">
        <div class="media-section">
          <div class="media-frame media-frame--banner">
            <div v-if="bannerImage" class="media-preview">
              <a-image
                :src="bannerImage"
                :alt="title"
                width="100%"
                height="100%"
                fit="cover"
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
                    @click.stop="emit('open-banner-selector')"
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
                    @click.stop="emit('remove-banner')"
                  >
                    <icon-delete />
                  </a-button>
                </div>
              </div>
            </div>
            <div
              v-else
              class="media-empty-action"
              role="button"
              tabindex="0"
              @click="emit('open-banner-selector')"
            >
              <icon-image class="media-empty-icon" />
              <span class="media-empty-title">未设置横幅</span>
              <span class="media-empty-subtitle">点击选择图片</span>
            </div>
          </div>
        </div>
      </a-form-item>

      <a-form-item label="预告片" class="media-subitem">
        <div class="media-section">
          <div class="media-frame media-frame--video">
            <div v-if="primaryPreviewVideo" class="media-preview">
              <video
                class="media-video"
                controls
                playsinline
                preload="metadata"
              >
                <source
                  v-for="src in previewVideoSources"
                  :key="src"
                  :src="src"
                />
              </video>
              <div class="media-overlay media-overlay--top-right">
                <div class="media-overlay-actions">
                  <a-button
                    class="app-text-action-btn media-action-button"
                    type="text"
                    shape="circle"
                    size="small"
                    html-type="button"
                    @click.stop="emit('open-video-selector')"
                  >
                    <icon-settings />
                  </a-button>
                </div>
              </div>
            </div>
            <div
              v-else
              class="media-empty-action"
              role="button"
              tabindex="0"
              @click="emit('open-video-selector')"
            >
              <icon-upload class="media-empty-icon" />
              <span class="media-empty-title">未设置预告片</span>
              <span class="media-empty-subtitle">点击上传本地视频</span>
            </div>
          </div>
        </div>
      </a-form-item>
    </a-col>

    <a-col :span="8">
      <a-form-item label="截图">
        <div class="media-section media-section--cover">
          <div class="media-frame media-frame--cover screenshots-frame">
            <div v-if="screenshots.length === 0" class="media-empty-action">
              <div
                class="media-empty-action media-empty-action--inner"
                role="button"
                tabindex="0"
                @click="emit('open-screenshot-selector')"
              >
                <icon-image class="media-empty-icon" />
                <span class="media-empty-title">未设置截图</span>
                <span class="media-empty-subtitle">点击添加截图</span>
              </div>
            </div>
            <a-image-preview-group v-else infinite>
              <div class="screenshots-grid-wrapper">
                <div class="screenshots-grid">
                  <div
                    v-for="screenshot in screenshots"
                    :key="screenshot.asset_uid || screenshot.client_key"
                    class="screenshot-thumb"
                    :class="{
                      'is-dragging': draggedScreenshotKey === screenshot.client_key,
                      'is-drop-target': dragOverScreenshotKey === screenshot.client_key,
                    }"
                    draggable="true"
                    @dragstart="emit('screenshot-drag-start', screenshot.client_key)"
                    @dragenter.prevent="emit('screenshot-drag-enter', screenshot.client_key)"
                    @dragover.prevent
                    @drop.prevent="emit('screenshot-drop', screenshot.client_key)"
                    @dragend="emit('screenshot-drag-end')"
                  >
                    <a-image
                      :src="screenshot.path"
                      width="100%"
                      height="100%"
                      fit="cover"
                      hide-footer
                    />
                    <div class="screenshot-overlay">
                      <a-button
                        class="app-text-action-btn media-action-button media-action-button--danger"
                        type="text"
                        status="danger"
                        shape="circle"
                        size="small"
                        html-type="button"
                        @click.stop="emit('remove-screenshot', screenshot.client_key)"
                      >
                        <icon-delete />
                      </a-button>
                    </div>
                  </div>
                  <div
                    class="screenshot-add-tile"
                    role="button"
                    tabindex="0"
                    @click="emit('open-screenshot-selector')"
                  >
                    <span class="screenshot-add-tile__label">添加截图</span>
                  </div>
                </div>
              </div>
            </a-image-preview-group>
          </div>
        </div>
      </a-form-item>
    </a-col>
  </a-row>
</template>

<script setup lang="ts">
import { IconDelete, IconImage, IconSettings, IconUpload } from '@arco-design/web-vue/es/icon'

interface EditableScreenshot {
  asset_uid?: string
  path: string
  client_key: string
}

interface EditableVideo {
  asset_uid?: string
  path: string
}

defineProps<{
  title: string
  coverImage: string
  bannerImage: string
  primaryPreviewVideo: EditableVideo | null
  previewVideoSources: string[]
  screenshots: EditableScreenshot[]
  draggedScreenshotKey: string | null
  dragOverScreenshotKey: string | null
}>()

const emit = defineEmits<{
  'open-cover-selector': []
  'remove-cover': []
  'open-banner-selector': []
  'remove-banner': []
  'open-video-selector': []
  'open-screenshot-selector': []
  'remove-screenshot': [clientKey: string]
  'screenshot-drag-start': [clientKey: string]
  'screenshot-drag-enter': [clientKey: string]
  'screenshot-drop': [clientKey: string]
  'screenshot-drag-end': []
}>()
</script>

<style scoped>
.media-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
}

.media-section--cover {
  max-width: 88%;
  margin: 0 auto;
}

.media-subitem {
  margin-top: 8px;
}

.media-frame {
  width: 100%;
  overflow: hidden;
  border-radius: 8px;
  border: 1px solid var(--app-card-border);
  background: color-mix(in srgb, var(--app-card-surface) 90%, transparent);
  backdrop-filter: blur(var(--app-card-backdrop-blur));
  -webkit-backdrop-filter: blur(var(--app-card-backdrop-blur));
  display: flex;
  align-items: center;
  justify-content: center;
}

.media-preview {
  position: relative;
  width: 100%;
  height: 100%;
}

.media-empty-action {
  width: 100%;
  height: 100%;
  border: 1px dashed rgba(255, 255, 255, 0.1);
  border-radius: 8px;
  background: color-mix(in srgb, var(--app-card-surface) 88%, transparent);
  backdrop-filter: blur(var(--app-card-backdrop-blur));
  -webkit-backdrop-filter: blur(var(--app-card-backdrop-blur));
  color: var(--color-text-3);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  cursor: pointer;
  transition: color 0.2s ease, background 0.2s ease, border-color 0.2s ease;
}

.media-empty-action:hover {
  color: rgb(var(--primary-6));
  background: rgba(var(--primary-6), 0.06);
  border-color: rgba(var(--primary-6), 0.45);
}

.media-empty-action--inner {
  border: none;
  border-radius: 0;
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

.media-overlay {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(8, 10, 16, 0.5);
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.2s ease;
}

.media-overlay--top-right {
  align-items: flex-start;
  justify-content: flex-end;
  padding: 14px;
}

.media-overlay-actions {
  display: inline-flex;
  align-items: center;
  gap: 12px;
  pointer-events: auto;
}

.media-action-button {
  width: 40px;
  height: 40px;
  min-width: 40px;
  padding: 0;
  border-radius: 999px;
  backdrop-filter: blur(8px);
  box-shadow: 0 10px 24px rgba(0, 0, 0, 0.24);
  transition: transform 0.2s ease;
}

.media-action-button:hover {
  transform: scale(1.06);
}

.media-preview:hover .media-overlay,
.screenshot-thumb:hover .screenshot-overlay {
  opacity: 1;
}

.media-frame--cover {
  aspect-ratio: 2 / 3;
}

.media-frame--banner {
  aspect-ratio: 16 / 9;
}

.media-frame--video {
  aspect-ratio: 16 / 9;
}

.media-video {
  width: 100%;
  height: 100%;
  display: block;
  background: #000;
}

.screenshots-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}

.screenshots-frame {
  align-items: stretch;
  justify-content: stretch;
}

.screenshots-grid-wrapper {
  width: 100%;
  height: 100%;
  padding: 10px;
  overflow-y: auto;
}

.screenshot-thumb {
  aspect-ratio: 16/9;
  border-radius: 6px;
  overflow: hidden;
  background: color-mix(in srgb, var(--app-card-surface) 86%, transparent);
  cursor: grab;
  position: relative;
  border: 1px solid var(--app-card-border);
  transition: transform 0.18s ease, border-color 0.18s ease, box-shadow 0.18s ease, opacity 0.18s ease;
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

.screenshot-add-tile {
  width: 100%;
  aspect-ratio: 16 / 9;
  border-radius: 6px;
  overflow: hidden;
  position: relative;
  border: 1px dashed rgba(255, 255, 255, 0.1);
  background: color-mix(in srgb, var(--app-card-surface) 88%, transparent);
  backdrop-filter: blur(var(--app-card-backdrop-blur));
  -webkit-backdrop-filter: blur(var(--app-card-backdrop-blur));
  color: var(--color-text-3);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 10px;
  box-sizing: border-box;
  cursor: pointer;
  transition: border-color 0.2s ease, color 0.2s ease, background 0.2s ease;
}

.screenshot-add-tile__label {
  font-size: 12px;
  font-weight: 700;
}

.screenshot-add-tile:hover {
  border-color: rgb(var(--primary-6));
  color: rgb(var(--primary-6));
  background: rgba(var(--primary-6), 0.06);
}

.screenshot-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(8, 10, 16, 0.5);
  opacity: 0;
  transition: opacity 0.2s ease;
}
</style>
