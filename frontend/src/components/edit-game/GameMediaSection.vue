<template>
  <div class="media-section-list">
      <section class="media-block media-block--cover">
        <div class="media-block__title">封面图</div>
        <div
          v-if="covers.length === 0"
          class="media-frame media-frame--cover"
        >
          <div
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
        <div class="media-frame media-frame--banner">
          <template v-if="bannerImage">
            <a-image
              :src="bannerImage"
              :alt="title"
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
          </template>
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
      </section>

      <section class="media-block media-block--logo">
        <div class="media-block__title">Logo</div>
        <div
          v-if="!logoImage"
          class="media-frame media-frame--logo"
        >
          <div
            class="media-empty-action"
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

      <section class="media-block media-block--screenshots">
        <div class="media-block__title">游戏截图</div>
        <div
          v-if="screenshots.length === 0"
          class="media-frame media-frame--screenshots"
        >
          <div
            class="media-empty-action"
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
          <div class="screenshots-scroll">
            <a-image-preview-group infinite>
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
                    fit="contain"
                    hide-footer
                  />
                  <div class="screenshot-overlay">
                    <div class="screenshot-overlay-actions">
                      <a-button
                        class="app-text-action-btn media-action-button"
                        type="text"
                        shape="circle"
                        size="small"
                        html-type="button"
                        title="管理截图"
                        @click.stop="emit('open-screenshot-selector')"
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
                        @click.stop="emit('remove-screenshot', screenshot.client_key)"
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

      <section class="media-block media-block--video">
        <div class="media-block__title-row">
          <span class="media-block__title">预告片</span>
          <a-button
            v-if="primaryPreviewVideo"
            class="app-text-action-btn"
            type="text"
            size="mini"
            html-type="button"
            @click.stop="emit('open-video-selector')"
          >
            <template #icon><icon-settings /></template>
            管理预告片
          </a-button>
        </div>
        <div class="media-frame media-frame--video">
          <template v-if="primaryPreviewVideo">
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
          </template>
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
      </section>
  </div>
</template>

<script setup lang="ts">
import { IconDelete, IconImage, IconSettings, IconStar, IconUpload } from '@arco-design/web-vue/es/icon'

interface EditableCover {
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
  'remove-banner': []
  'open-video-selector': []
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
  bannerImage: string
  primaryPreviewVideo: EditableVideo | null
  previewVideoSources: string[]
  screenshots: EditableScreenshot[]
  draggedScreenshotKey: string | null
  dragOverScreenshotKey: string | null
  logoImage: string
}>()

</script>

<style scoped>
.media-section-list {
  --media-gap: 22px;
  --media-panel-radius: 14px;
  --media-panel-padding: 14px;
  --media-frame-height: clamp(220px, 24vw, 300px);
  --media-video-height: clamp(300px, 36vw, 420px);
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

.media-frame--cover,
.media-frame--screenshots {
  height: var(--media-frame-height);
  min-height: var(--media-frame-height);
}

.media-frame--video {
  height: var(--media-video-height);
  min-height: var(--media-video-height);
}

.media-frame--banner {
  width: 100%;
  height: 270px;
}

.media-frame--logo {
  height: 270px;
}

.media-empty-action {
  width: 100%;
  height: 100%;
  border: 1px dashed rgba(255, 255, 255, 0.1);
  border-radius: calc(var(--media-panel-radius) - 2px);
  background: color-mix(in srgb, var(--app-card-surface) 88%, transparent);
  backdrop-filter: blur(var(--app-card-backdrop-blur));
  -webkit-backdrop-filter: blur(var(--app-card-backdrop-blur));
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

.media-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(8, 10, 16, 0.5);
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.2s ease;
}


.media-overlay-actions,
.cover-overlay-actions,
.screenshot-overlay-actions {
  display: inline-flex;
  align-items: center;
  gap: 6px;
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

.cover-overlay-actions .media-action-button,
.screenshot-overlay-actions .media-action-button {
  width: 28px;
  height: 28px;
  min-width: 28px;
  font-size: 12px;
}

.media-frame:hover .media-overlay,
.cover-thumb:hover .cover-overlay,
.screenshot-thumb:hover .screenshot-overlay {
  opacity: 1;
}

.media-video {
  width: 100%;
  height: 100%;
  display: block;
  background: #000;
  object-fit: contain;
}


.media-frame :deep(.arco-image),
.cover-thumb :deep(.arco-image),
.logo-frame :deep(.arco-image),
.screenshot-thumb :deep(.arco-image) {
  display: flex;
  width: 100%;
  height: 100%;
}

.media-frame :deep(.arco-image-img),
.cover-thumb :deep(.arco-image-img),
.logo-frame :deep(.arco-image-img),
.screenshot-thumb :deep(.arco-image-img) {
  width: 100%;
  height: 100%;
  object-fit: contain;
  object-position: center;
}

.covers-frame,
.screenshots-frame {
  align-items: stretch;
  justify-content: stretch;
  padding: var(--media-panel-padding);
  box-sizing: border-box;
}

.covers-scroll,
.screenshots-scroll {
  width: 100%;
  height: 100%;
  min-height: 0;
  overflow-x: hidden;
  overflow-y: auto;
}

.covers-scroll :deep(.arco-image-preview-group),
.screenshots-scroll :deep(.arco-image-preview-group) {
  display: block;
  width: 100%;
}

.covers-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  align-items: start;
  gap: 12px;
}

.screenshots-grid {
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
    linear-gradient(180deg, rgba(255, 255, 255, 0.04), transparent),
    color-mix(in srgb, var(--app-card-surface) 90%, transparent);
  position: relative;
  border: 1px solid var(--app-card-border);
  transition: border-color 0.18s ease, box-shadow 0.18s ease;
}

.cover-thumb.is-primary {
  border-color: rgb(var(--primary-6));
  box-shadow: 0 0 0 1px rgba(var(--primary-6), 0.35);
}

.cover-primary-badge {
  position: absolute;
  top: 3px;
  left: 3px;
  background: rgb(var(--primary-6));
  color: #fff;
  font-size: 9px;
  font-weight: 700;
  padding: 1px 5px;
  border-radius: 3px;
  line-height: 1.3;
  z-index: 1;
}

.cover-overlay,
.screenshot-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(8, 10, 16, 0.5);
  opacity: 0;
  transition: opacity 0.2s ease;
}

.logo-frame {
  display: flex;
  align-items: center;
  justify-content: center;
}

.screenshot-thumb {
  width: 100%;
  aspect-ratio: 16 / 9;
  border-radius: 10px;
  overflow: hidden;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.04), transparent),
    color-mix(in srgb, var(--app-card-surface) 90%, transparent);
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
  .screenshots-grid {
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
}
</style>
