<template>
  <div class="media-page">
    <section class="media-card">
      <header class="media-card__header">
        <div class="media-card__heading">
          <span class="media-card__title">封面图</span>
          <span v-if="covers.length > 0" class="media-card__count">{{ covers.length }} 张</span>
        </div>
        <div class="media-card__toolbar">
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
        </div>
      </header>
      <div class="media-card__body">
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
        <div
          v-else
          class="media-card__empty"
          role="button"
          tabindex="0"
          @click="emit('open-cover-selector')"
          @keydown.enter="emit('open-cover-selector')"
          @keydown.space.prevent="emit('open-cover-selector')"
        >
          <icon-image class="media-card__empty-icon" />
          <span class="media-card__empty-title">未设置封面</span>
          <span class="media-card__empty-subtitle">点击添加图片</span>
        </div>
        <div
          v-if="covers.length > 0"
          ref="coversScrollRef"
          class="media-card__thumbs"
          @dragover="onGridDragOver($event, coversScrollRef)"
          @dragleave="stopGridAutoScroll"
          @dragend="stopGridAutoScroll"
        >
          <div
            v-for="(cover, index) in covers"
            :key="cover.asset_uid || cover.path"
            class="media-thumb media-thumb--cover"
            :class="getThumbClass(cover.asset_uid || cover.path, index === 0, draggedCoverKey, dragOverCoverKey)"
            @dragenter.prevent="emit('cover-drag-enter', cover.asset_uid || cover.path)"
            @dragover.prevent
            @drop.prevent="emit('cover-drop', cover.asset_uid || cover.path)"
          >
            <a-image
              :src="cover.path"
              width="100%"
              height="100%"
              fit="contain"
              hide-footer
              :preview="false"
            />
            <div v-if="index === 0" class="media-thumb__badge">主封面</div>
            <div class="media-thumb__actions">
              <a-button
                class="app-text-action-btn media-action-button media-drag-handle"
                type="text"
                shape="circle"
                size="small"
                html-type="button"
                title="拖拽排序"
                draggable="true"
                @dragstart="emit('cover-drag-start', cover.asset_uid || cover.path)"
                @dragend="emit('cover-drag-end')"
              >
                <icon-drag-arrow />
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
                title="删除封面"
                @click.stop="confirmRemoveAsset('这张封面', () => emit('remove-cover', index))"
              >
                <icon-delete />
              </a-button>
            </div>
          </div>
        </div>
      </div>
    </section>

    <section class="media-card">
      <header class="media-card__header">
        <div class="media-card__heading">
          <span class="media-card__title">横幅图</span>
          <span v-if="banners.length > 0" class="media-card__count">{{ banners.length }} 张</span>
        </div>
        <div class="media-card__toolbar">
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
        </div>
      </header>
      <div class="media-card__body">
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
        <div
          v-else
          class="media-card__empty"
          role="button"
          tabindex="0"
          @click="emit('open-banner-selector')"
          @keydown.enter="emit('open-banner-selector')"
          @keydown.space.prevent="emit('open-banner-selector')"
        >
          <icon-image class="media-card__empty-icon" />
          <span class="media-card__empty-title">未设置横幅</span>
          <span class="media-card__empty-subtitle">点击添加图片</span>
        </div>
        <div
          v-if="banners.length > 0"
          ref="bannersScrollRef"
          class="media-card__thumbs"
          @dragover="onGridDragOver($event, bannersScrollRef)"
          @dragleave="stopGridAutoScroll"
          @dragend="stopGridAutoScroll"
        >
          <div
            v-for="(banner, index) in banners"
            :key="banner.asset_uid || banner.path"
            class="media-thumb media-thumb--banner"
            :class="getThumbClass(banner.asset_uid || banner.path, index === 0, draggedBannerKey, dragOverBannerKey)"
            @dragenter.prevent="emit('banner-drag-enter', banner.asset_uid || banner.path)"
            @dragover.prevent
            @drop.prevent="emit('banner-drop', banner.asset_uid || banner.path)"
          >
            <a-image
              :src="banner.path"
              width="100%"
              height="100%"
              fit="contain"
              hide-footer
              :preview="false"
            />
            <div v-if="index === 0" class="media-thumb__badge">主横幅</div>
            <div class="media-thumb__actions">
              <a-button
                class="app-text-action-btn media-action-button media-drag-handle"
                type="text"
                shape="circle"
                size="small"
                html-type="button"
                title="拖拽排序"
                draggable="true"
                @dragstart="emit('banner-drag-start', banner.asset_uid || banner.path)"
                @dragend="emit('banner-drag-end')"
              >
                <icon-drag-arrow />
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
                title="删除横幅"
                @click.stop="confirmRemoveAsset('这张横幅', () => emit('remove-banner', index))"
              >
                <icon-delete />
              </a-button>
            </div>
          </div>
        </div>
      </div>
    </section>

    <section class="media-card media-card--wide">
      <header class="media-card__header">
        <div class="media-card__heading">
          <span class="media-card__title">游戏截图</span>
          <span v-if="screenshots.length > 0" class="media-card__count">{{ screenshots.length }} 张</span>
        </div>
        <div class="media-card__toolbar">
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
        </div>
      </header>
      <div class="media-card__body">
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
        <div
          v-else
          class="media-card__empty"
          role="button"
          tabindex="0"
          @click="emit('open-screenshot-selector')"
          @keydown.enter="emit('open-screenshot-selector')"
          @keydown.space.prevent="emit('open-screenshot-selector')"
        >
          <icon-image class="media-card__empty-icon" />
          <span class="media-card__empty-title">未设置截图</span>
          <span class="media-card__empty-subtitle">点击添加截图</span>
        </div>
        <MediaScreenshotSection
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
      </div>
    </section>

    <section class="media-card media-card--wide">
      <header class="media-card__header">
        <div class="media-card__heading">
          <span class="media-card__title">Logo</span>
          <span v-if="logos.length > 0" class="media-card__count">{{ logos.length }} 个</span>
        </div>
        <div class="media-card__toolbar">
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
        </div>
      </header>
      <div class="media-card__body">
        <div v-if="logos.length > 0" class="media-card__preview media-card__preview--logo">
          <a-image
            :src="logos[0].path"
            width="100%"
            height="100%"
            fit="contain"
            hide-footer
            :preview="false"
          />
          <div v-if="logos.length > 1" class="media-card__preview-badge">主 Logo</div>
        </div>
        <div
          v-else
          class="media-card__empty"
          role="button"
          tabindex="0"
          @click="emit('open-logo-selector')"
          @keydown.enter="emit('open-logo-selector')"
          @keydown.space.prevent="emit('open-logo-selector')"
        >
          <icon-image class="media-card__empty-icon" />
          <span class="media-card__empty-title">未设置 Logo</span>
          <span class="media-card__empty-subtitle">点击添加图片</span>
        </div>
        <div
          v-if="logos.length > 0"
          ref="logosScrollRef"
          class="media-card__thumbs"
          @dragover="onGridDragOver($event, logosScrollRef)"
          @dragleave="stopGridAutoScroll"
          @dragend="stopGridAutoScroll"
        >
          <div
            v-for="(logo, index) in logos"
            :key="logo.asset_uid || logo.path"
            class="media-thumb media-thumb--logo"
            :class="getThumbClass(logo.asset_uid || logo.path, index === 0, draggedLogoKey, dragOverLogoKey)"
            @dragenter.prevent="emit('logo-drag-enter', logo.asset_uid || logo.path)"
            @dragover.prevent
            @drop.prevent="emit('logo-drop', logo.asset_uid || logo.path)"
          >
            <a-image
              :src="logo.path"
              width="100%"
              height="100%"
              fit="contain"
              hide-footer
              :preview="false"
            />
            <div v-if="index === 0" class="media-thumb__badge">主 Logo</div>
            <div class="media-thumb__actions">
              <a-button
                class="app-text-action-btn media-action-button media-drag-handle"
                type="text"
                shape="circle"
                size="small"
                html-type="button"
                title="拖拽排序"
                draggable="true"
                @dragstart="emit('logo-drag-start', logo.asset_uid || logo.path)"
                @dragend="emit('logo-drag-end')"
              >
                <icon-drag-arrow />
              </a-button>
              <a-button
                class="app-text-action-btn media-action-button"
                type="text"
                shape="circle"
                size="small"
                html-type="button"
                title="调整位置"
                @click.stop="emit('open-logo-position-editor', logo.asset_uid || logo.path)"
              >
                <icon-edit />
              </a-button>
              <a-button
                v-if="index !== 0"
                class="app-text-action-btn media-action-button"
                type="text"
                shape="circle"
                size="small"
                html-type="button"
                title="设为主 Logo"
                @click.stop="emit('set-primary-logo', index)"
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
                title="删除 Logo"
                @click.stop="confirmRemoveAsset('这个 Logo', () => emit('remove-logo', index))"
              >
                <icon-delete />
              </a-button>
            </div>
          </div>
        </div>
      </div>
    </section>

    <section class="media-card media-card--wide">
      <header class="media-card__header">
        <div class="media-card__heading">
          <span class="media-card__title">预告片</span>
          <span v-if="previewVideos.length > 0" class="media-card__count">{{ previewVideos.length }} 个</span>
        </div>
        <div class="media-card__toolbar">
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
        </div>
      </header>
      <input
        ref="videoFileInput"
        type="file"
        multiple
        accept="video/mp4,video/webm"
        class="hidden-file-input"
        @change="emit('video-file-change', $event)"
      />
      <div class="media-card__body">
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
        <div
          v-else
          class="media-card__empty media-card__empty--video"
          role="button"
          tabindex="0"
          @click="openVideoFilePicker"
          @keydown.enter="openVideoFilePicker"
          @keydown.space.prevent="openVideoFilePicker"
        >
          <icon-video-camera class="media-card__empty-icon" />
          <span class="media-card__empty-title">未设置预告片</span>
          <span class="media-card__empty-subtitle">上传 MP4 或 WebM 视频</span>
        </div>
        <div
          v-if="previewVideos.length > 0"
          ref="videosScrollRef"
          class="media-card__thumbs"
          @dragover="onGridDragOver($event, videosScrollRef)"
          @dragleave="stopGridAutoScroll"
          @dragend="stopGridAutoScroll"
        >
          <div
            v-for="(video, index) in previewVideos"
            :key="video.asset_uid || video.path"
            class="media-thumb media-thumb--video"
            :class="getThumbClass(video.asset_uid || video.path, index === 0, draggedVideoKey, dragOverVideoKey)"
            @dragenter.prevent="emit('video-drag-enter', video.asset_uid || video.path)"
            @dragover.prevent
            @drop.prevent="emit('video-drop', video.asset_uid || video.path)"
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
            <div v-if="index === 0" class="media-thumb__badge">首个展示</div>
            <div class="media-thumb__actions">
              <a-button
                class="app-text-action-btn media-action-button media-drag-handle"
                type="text"
                shape="circle"
                size="small"
                html-type="button"
                title="拖拽排序"
                draggable="true"
                @dragstart="emit('video-drag-start', video.asset_uid || video.path)"
                @dragend="emit('video-drag-end')"
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
                title="删除预告片"
                @click.stop="confirmRemoveAsset('这个预告片', () => emit('remove-video', video.asset_uid))"
              >
                <icon-delete />
              </a-button>
            </div>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, ref } from 'vue'
import { Modal } from '@arco-design/web-vue'
import {
  IconDelete,
  IconDragArrow,
  IconEdit,
  IconImage,
  IconPlus,
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
  'set-primary-cover': [index: number]
  'cover-drag-start': [key: string]
  'cover-drag-enter': [key: string]
  'cover-drop': [key: string]
  'cover-drag-end': []
  'open-banner-selector': []
  'remove-banner': [index: number]
  'set-primary-banner': [index: number]
  'banner-drag-start': [key: string]
  'banner-drag-enter': [key: string]
  'banner-drop': [key: string]
  'banner-drag-end': []
  'logo-drag-start': [key: string]
  'logo-drag-enter': [key: string]
  'logo-drop': [key: string]
  'logo-drag-end': []
  'video-file-change': [event: Event]
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
  'open-logo-position-editor': [key: string]
  'remove-logo': [index: number]
  'set-primary-logo': [index: number]
}>()

defineProps<{
  title: string
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
}>()

const videoFileInput = ref<HTMLInputElement | null>(null)
const coversScrollRef = ref<HTMLElement | null>(null)
const bannersScrollRef = ref<HTMLElement | null>(null)
const logosScrollRef = ref<HTMLElement | null>(null)
const videosScrollRef = ref<HTMLElement | null>(null)
let dragScrollRaf = 0

const SCROLL_ZONE = 60
const SCROLL_SPEED = 12

const stopGridAutoScroll = () => {
  cancelAnimationFrame(dragScrollRaf)
  dragScrollRaf = 0
}

const onGridDragOver = (event: DragEvent, container: HTMLElement | null) => {
  if (!container) return

  const rect = container.getBoundingClientRect()
  let scrollX = 0
  let scrollY = 0
  if (event.clientX < rect.left + SCROLL_ZONE) scrollX = -SCROLL_SPEED
  else if (event.clientX > rect.right - SCROLL_ZONE) scrollX = SCROLL_SPEED
  if (event.clientY < rect.top + SCROLL_ZONE) scrollY = -SCROLL_SPEED
  else if (event.clientY > rect.bottom - SCROLL_ZONE) scrollY = SCROLL_SPEED

  if (scrollX === 0 && scrollY === 0) {
    stopGridAutoScroll()
    return
  }

  if (!dragScrollRaf) {
    const tick = () => {
      container.scrollLeft += scrollX
      container.scrollTop += scrollY
      dragScrollRaf = requestAnimationFrame(tick)
    }
    dragScrollRaf = requestAnimationFrame(tick)
  }
}

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

const openVideoFilePicker = () => {
  videoFileInput.value?.click()
}

const confirmRemoveAsset = (label: string, onConfirm: () => void) => {
  Modal.confirm({
    title: '删除素材',
    content: `确定要移除${label}吗？保存后将从磁盘删除。`,
    okText: '删除',
    cancelText: '取消',
    okButtonProps: { status: 'danger' },
    onOk: () => onConfirm(),
  })
}

onBeforeUnmount(stopGridAutoScroll)
</script>

<style scoped>
.media-page {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
  padding: 2px;
}

.media-card {
  min-width: 0;
  display: flex;
  flex-direction: column;
  border: 1px solid var(--app-card-border);
  border-radius: 14px;
  background: color-mix(in srgb, var(--app-card-surface) 92%, transparent);
  overflow: hidden;
}

.media-card--wide {
  grid-column: 1 / -1;
}

.media-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 12px 14px 0;
}

.media-card__heading {
  display: flex;
  align-items: baseline;
  gap: 8px;
  min-width: 0;
}

.media-card__title {
  font-size: 14px;
  font-weight: 700;
  color: var(--color-text-1);
}

.media-card__count {
  font-size: 12px;
  color: var(--color-text-3);
}

.media-card__toolbar {
  display: flex;
  align-items: center;
  gap: 2px;
  flex-shrink: 0;
}

.media-card__action {
  color: var(--color-text-2);
}

.media-card__body {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 10px 14px 14px;
  min-width: 0;
}

.media-card__preview {
  position: relative;
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

.media-card__preview--cover {
  height: clamp(260px, 30vh, 420px);
}

.media-card__preview--banner {
  height: clamp(220px, 26vh, 380px);
}

.media-card__preview--screenshot {
  height: clamp(260px, 30vh, 420px);
}

.media-card__preview--logo {
  height: clamp(180px, 22vh, 320px);
}

.media-card__preview--video {
  height: clamp(240px, 28vh, 400px);
}

.video-card__player {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: contain;
  background: var(--color-media-bg);
}

.media-card__preview :deep(.arco-image),
.media-thumb :deep(.arco-image) {
  display: flex;
  width: 100%;
  height: 100%;
}

.media-card__preview :deep(.arco-image-img),
.media-thumb :deep(.arco-image-img) {
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

.media-card__empty {
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

.media-card__thumbs {
  display: flex;
  gap: 8px;
  overflow-x: auto;
  padding: 4px 2px;
  scrollbar-width: thin;
}

.media-thumb {
  position: relative;
  flex: 0 0 auto;
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid var(--app-card-border);
  background:
    linear-gradient(180deg, var(--color-border-1), transparent),
    color-mix(in srgb, var(--app-card-surface) 90%, transparent);
  transition: border-color 0.18s ease, box-shadow 0.18s ease, opacity 0.18s ease, transform 0.18s ease;
}

.media-thumb--cover {
  width: 132px;
  aspect-ratio: 2 / 3;
}

.media-thumb--banner {
  width: 180px;
  aspect-ratio: 16 / 9;
}

.media-thumb--logo {
  width: 200px;
  aspect-ratio: 16 / 9;
}

.media-thumb--video {
  width: 220px;
  aspect-ratio: 16 / 9;
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

.media-thumb__actions {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  background: var(--app-scrim);
  opacity: 0;
  transition: opacity 0.18s ease;
}

.media-thumb__actions .media-action-button {
  width: 26px;
  height: 26px;
  min-width: 26px;
  font-size: 12px;
}

.media-thumb:hover .media-thumb__actions,
.media-thumb:focus-within .media-thumb__actions {
  opacity: 1;
}

.media-drag-handle {
  cursor: grab;
}

.media-drag-handle:active {
  cursor: grabbing;
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

@media (hover: none) {
  .media-thumb__actions {
    opacity: 1;
    align-items: flex-end;
    background: linear-gradient(180deg, transparent 0%, var(--app-scrim) 60%);
    padding: 6px;
  }
}

@media (max-width: 768px) {
  .media-page {
    grid-template-columns: 1fr;
  }

  .media-card--wide {
    grid-column: auto;
  }

  .video-library-item {
    align-items: flex-start;
  }

  .video-library-item__actions {
    margin-left: -4px;
  }
}
</style>
