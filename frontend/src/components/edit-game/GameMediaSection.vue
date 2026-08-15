<template>
  <div class="media-page">
    <media-card-frame
      title="封面图"
      :count="covers.length > 0 ? `${covers.length} 张` : ''"
      card-class="media-card--cover"
    >
      <template #toolbar>
        <a-button
          class="app-text-action-btn media-card__action"
          type="text"
          size="small"
          html-type="button"
          @click="emit('open-cover-selector')"
        >
          <template #icon><icon-plus /></template>
          添加/导入
        </a-button>
      </template>

      <div v-if="covers.length > 0" class="media-card__preview media-card__preview--cover">
        <a-image
          :src="covers[0].path"
          width="100%"
          height="100%"
          fit="contain"
          hide-footer
          :preview="false"
        />
        <div v-if="covers.length > 1" class="media-card__preview-badge">主封面</div>
      </div>
      <media-empty-state
        v-else
        title="未设置封面"
        subtitle="点击添加图片"
        @click="emit('open-cover-selector')"
      />
      <div
        v-if="covers.length > 0"
        ref="coversScrollRef"
        class="media-card__thumbs"
        @dragover="onGridDragOver($event, coversScrollRef)"
        @dragleave="stopGridAutoScroll"
        @dragend="stopGridAutoScroll"
      >
        <media-image-thumb
          v-for="(cover, index) in covers"
          :key="cover.asset_uid || cover.path"
          thumb-class="media-thumb--cover"
          :media-key="cover.asset_uid || cover.path"
          :is-primary="index === 0"
          :dragged-key="draggedCoverKey"
          :drag-over-key="dragOverCoverKey"
          :badge="index === 0 ? '主封面' : undefined"
          delete-title="删除封面"
          confirm-label="这张封面"
          @drag-start="(key) => emit('cover-drag-start', key)"
          @drag-end="emit('cover-drag-end')"
          @drag-enter="(key) => emit('cover-drag-enter', key)"
          @drop="(key) => emit('cover-drop', key)"
          @remove="emit('remove-cover', index)"
        >
          <a-image
            :src="cover.path"
            width="100%"
            height="100%"
            fit="contain"
            hide-footer
            :preview="false"
          />
        </media-image-thumb>
      </div>
    </media-card-frame>

    <media-card-frame
      title="横幅图"
      :count="banners.length > 0 ? `${banners.length} 张` : ''"
      card-class="media-card--banner"
    >
      <template #toolbar>
        <a-button
          class="app-text-action-btn media-card__action"
          type="text"
          size="small"
          html-type="button"
          @click="emit('open-banner-selector')"
        >
          <template #icon><icon-plus /></template>
          添加/导入
        </a-button>
      </template>

      <div v-if="banners.length > 0" class="media-card__preview media-card__preview--banner">
        <a-image
          :src="banners[0].path"
          width="100%"
          height="100%"
          fit="contain"
          hide-footer
          :preview="false"
        />
        <div v-if="banners.length > 1" class="media-card__preview-badge">主横幅</div>
      </div>
      <media-empty-state
        v-else
        title="未设置横幅"
        subtitle="点击添加图片"
        @click="emit('open-banner-selector')"
      />
      <div
        v-if="banners.length > 0"
        ref="bannersScrollRef"
        class="media-card__thumbs"
        @dragover="onGridDragOver($event, bannersScrollRef)"
        @dragleave="stopGridAutoScroll"
        @dragend="stopGridAutoScroll"
      >
        <media-image-thumb
          v-for="(banner, index) in banners"
          :key="banner.asset_uid || banner.path"
          thumb-class="media-thumb--banner"
          :media-key="banner.asset_uid || banner.path"
          :is-primary="index === 0"
          :dragged-key="draggedBannerKey"
          :drag-over-key="dragOverBannerKey"
          :badge="index === 0 ? '主横幅' : undefined"
          delete-title="删除横幅"
          confirm-label="这张横幅"
          @drag-start="(key) => emit('banner-drag-start', key)"
          @drag-end="emit('banner-drag-end')"
          @drag-enter="(key) => emit('banner-drag-enter', key)"
          @drop="(key) => emit('banner-drop', key)"
          @remove="emit('remove-banner', index)"
        >
          <a-image
            :src="banner.path"
            width="100%"
            height="100%"
            fit="contain"
            hide-footer
            :preview="false"
          />
        </media-image-thumb>
      </div>
    </media-card-frame>

    <media-card-frame
      title="游戏截图"
      :count="screenshots.length > 0 ? `${screenshots.length} 张` : ''"
      card-class="media-card--screenshots"
    >
      <template #toolbar>
        <a-button
          class="app-text-action-btn media-card__action"
          type="text"
          size="small"
          html-type="button"
          @click="emit('open-screenshot-selector')"
        >
          <template #icon><icon-plus /></template>
          添加/导入
        </a-button>
      </template>

      <div
        v-if="screenshots.length > 0"
        class="media-card__preview media-card__preview--screenshot"
      >
        <a-image
          :src="screenshots[0].path"
          width="100%"
          height="100%"
          fit="contain"
          hide-footer
          :preview="false"
        />
        <div class="media-card__preview-badge">截图 1</div>
      </div>
      <media-empty-state
        v-else
        title="未设置截图"
        subtitle="点击添加截图"
        @click="emit('open-screenshot-selector')"
      />
      <media-screenshot-section
        v-if="screenshots.length > 0"
        hide-title
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
    </media-card-frame>

    <media-card-frame
      title="Logo"
      :count="logos.length > 0 ? `${logos.length} 个` : ''"
      card-class="media-card--logo"
    >
      <template #toolbar>
        <a-button
          class="app-text-action-btn media-card__action"
          type="text"
          size="small"
          html-type="button"
          @click="emit('open-logo-selector')"
        >
          <template #icon><icon-plus /></template>
          添加/导入
        </a-button>
      </template>

      <logo-position-editor
        v-if="logos.length > 0"
        :logos="logos"
        :banners="banners"
        :covers="covers"
        :logo-visible="logoVisible"
        @logo-position-change="emit('logo-position-change', $event)"
      />
      <media-empty-state
        v-else
        title="未设置 Logo"
        subtitle="点击添加图片"
        @click="emit('open-logo-selector')"
      />
      <div
        v-if="logos.length > 0"
        ref="logosScrollRef"
        class="media-card__thumbs"
        @dragover="onGridDragOver($event, logosScrollRef)"
        @dragleave="stopGridAutoScroll"
        @dragend="stopGridAutoScroll"
      >
        <media-image-thumb
          v-for="(logo, index) in logos"
          :key="logo.asset_uid || logo.path"
          thumb-class="media-thumb--logo"
          :media-key="logo.asset_uid || logo.path"
          :is-primary="index === 0"
          :dragged-key="draggedLogoKey"
          :drag-over-key="dragOverLogoKey"
          delete-title="删除 Logo"
          confirm-label="这个 Logo"
          @drag-start="(key) => emit('logo-drag-start', key)"
          @drag-end="emit('logo-drag-end')"
          @drag-enter="(key) => emit('logo-drag-enter', key)"
          @drop="(key) => emit('logo-drop', key)"
          @remove="emit('remove-logo', index)"
        >
          <a-image
            :src="logo.path"
            width="100%"
            height="100%"
            fit="contain"
            hide-footer
            :preview="false"
          />
        </media-image-thumb>
      </div>
    </media-card-frame>

    <video-section
      :preview-videos="previewVideos"
      :is-uploading-video="isUploadingVideo"
      :video-upload-progress="videoUploadProgress"
      :video-upload-file-name="videoUploadFileName"
      :dragged-video-key="draggedVideoKey"
      :drag-over-video-key="dragOverVideoKey"
      @video-file-change="(event) => emit('video-file-change', event)"
      @select-poster="(video) => emit('select-poster', video)"
      @video-drag-start="(key) => emit('video-drag-start', key)"
      @video-drag-enter="(key) => emit('video-drag-enter', key)"
      @video-drop="(key) => emit('video-drop', key)"
      @video-drag-end="emit('video-drag-end')"
      @remove-video="(assetUid) => emit('remove-video', assetUid)"
    />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { IconPlus } from '@arco-design/web-vue/es/icon'
import type { LogoPositionChange } from '@/utils/edit-game-form'
import { useGridAutoScroll } from '@/composables/useGridAutoScroll'
import MediaCardFrame from './MediaCardFrame.vue'
import MediaEmptyState from './MediaEmptyState.vue'
import MediaImageThumb from './MediaImageThumb.vue'
import LogoPositionEditor from './LogoPositionEditor.vue'
import MediaScreenshotSection from './MediaScreenshotSection.vue'
import VideoSection from './VideoSection.vue'

interface EditableCover {
  asset_uid?: string
  path: string
}

interface EditableBanner {
  asset_uid?: string
  path: string
}

interface EditableLogo {
  asset_uid?: string
  path: string
  position_x: number | null
  position_y: number | null
  width_pct: number | null
}

interface EditableScreenshot {
  asset_uid?: string
  path: string
  client_key: string
}

interface EditableVideo {
  asset_uid?: string
  path: string
  poster_path?: string | null
}

const emit = defineEmits<{
  'open-cover-selector': []
  'remove-cover': [index: number]
  'cover-drag-start': [key: string]
  'cover-drag-enter': [key: string]
  'cover-drop': [key: string]
  'cover-drag-end': []
  'open-banner-selector': []
  'remove-banner': [index: number]
  'banner-drag-start': [key: string]
  'banner-drag-enter': [key: string]
  'banner-drop': [key: string]
  'banner-drag-end': []
  'logo-drag-start': [key: string]
  'logo-drag-enter': [key: string]
  'logo-drop': [key: string]
  'logo-drag-end': []
  'video-file-change': [event: Event]
  'select-poster': [video: unknown]
  'video-drag-start': [key: string]
  'video-drag-enter': [key: string]
  'video-drop': [key: string]
  'video-drag-end': []
  'remove-video': [assetUid?: string]
  'open-screenshot-selector': []
  'remove-screenshot': [clientKey: string]
  'screenshot-drag-start': [clientKey: string]
  'screenshot-drag-enter': [clientKey: string]
  'screenshot-drop': [clientKey: string]
  'screenshot-drag-end': []
  'open-logo-selector': []
  'logo-position-change': [payload: LogoPositionChange]
  'remove-logo': [index: number]
}>()

defineProps<{
  covers: EditableCover[]
  banners: EditableBanner[]
  logos: EditableLogo[]
  previewVideos: EditableVideo[]
  isUploadingVideo: boolean
  videoUploadProgress: number
  videoUploadFileName: string
  screenshots: EditableScreenshot[]
  draggedScreenshotKey: string | null
  dragOverScreenshotKey: string | null
  draggedCoverKey: string | null
  dragOverCoverKey: string | null
  draggedBannerKey: string | null
  dragOverBannerKey: string | null
  draggedLogoKey: string | null
  dragOverLogoKey: string | null
  draggedVideoKey: string | null
  dragOverVideoKey: string | null
  logoVisible: boolean
}>()

const coversScrollRef = ref<HTMLElement | null>(null)
const bannersScrollRef = ref<HTMLElement | null>(null)
const logosScrollRef = ref<HTMLElement | null>(null)
const { onGridDragOver, stopGridAutoScroll } = useGridAutoScroll()
</script>

<style scoped>
.media-page {
  display: grid;
  grid-template-columns: repeat(12, minmax(0, 1fr));
  gap: 12px;
  align-items: stretch;
  padding: 2px;
}

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

.media-card__preview--banner,
.media-card__preview--screenshot {
  height: auto;
  aspect-ratio: 16 / 9;
}

.media-card__preview--cover {
  height: clamp(240px, 30vh, 380px);
}

.media-card__preview :deep(.arco-image) {
  display: flex;
  width: 100%;
  height: 100%;
}

.media-card__preview :deep(.arco-image-img) {
  width: 100%;
  height: 100%;
  object-fit: contain;
  object-position: center;
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

@media (max-width: 768px) {
  .media-page {
    grid-template-columns: 1fr;
  }
}
</style>
