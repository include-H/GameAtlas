<template>
  <div class="screenshot-carousel" v-if="mediaItems.length > 0" ref="carouselRef">
    <div class="screenshot-carousel__viewport" :style="viewportStyle" ref="viewportRef">
      <a-button
        v-if="mediaItems.length > 1"
        class="screenshot-carousel__arrow screenshot-carousel__arrow--prev app-text-action-btn"
        type="text"
        shape="circle"
        @click="prevImage"
        aria-label="上一张"
      >
        <template #icon>
          <icon-left />
        </template>
      </a-button>

      <div class="screenshot-carousel__main">
        <transition name="screenshot-carousel-fade">
          <div
            v-if="currentMedia"
            :key="currentMedia.key"
            class="screenshot-carousel__media-shell"
          >
            <img
              v-if="currentMedia.type === 'image'"
              :src="currentMedia.url"
              :alt="alt"
              class="screenshot-carousel__image"
              @load="onImageLoad"
              @error="handleImageError(currentMedia.url)"
            />
            <template v-else>
              <video
                ref="videoRef"
                :src="currentMedia.url"
                class="screenshot-carousel__video"
                :class="{ 'screenshot-carousel__video--ready': videoReady }"
                :poster="videoPoster || undefined"
                autoplay
                controls
                muted
                playsinline
                preload="metadata"
                @canplay="tryPlayVideo"
                @loadedmetadata="onVideoLoaded"
                @playing="onVideoPlaying"
                @ended="handleVideoEnded"
                @volumechange="onVideoVolumeChange"
              />
              <transition name="screenshot-carousel-spinner-fade">
                <div v-if="!videoReady" class="screenshot-carousel__video-loader">
                  <div class="screenshot-carousel__loader-ring">
                    <img
                      v-if="videoPoster"
                      :src="resolveAssetUrl(videoPoster)"
                      class="screenshot-carousel__loader-thumb"
                      alt=""
                    />
                    <svg class="screenshot-carousel__loader-arc" viewBox="0 0 100 100">
                      <circle cx="50" cy="50" r="46" fill="none" stroke="currentColor" style="opacity: 0.1" stroke-width="3" />
                      <circle
                        cx="50" cy="50" r="46"
                        fill="none"
                        style="stroke: var(--color-primary-3)"
                        stroke-width="3"
                        stroke-linecap="round"
                        stroke-dasharray="80 210"
                        class="screenshot-carousel__loader-arc-spin"
                      />
                    </svg>
                  </div>
                </div>
              </transition>
            </template>
          </div>
        </transition>
      </div>

      <a-button
        v-if="mediaItems.length > 1"
        class="screenshot-carousel__arrow screenshot-carousel__arrow--next app-text-action-btn"
        type="text"
        shape="circle"
        @click="nextImage"
        aria-label="下一张"
      >
        <template #icon>
          <icon-right />
        </template>
      </a-button>

      <div v-if="mediaItems.length > 1" class="app-glass-surface screenshot-carousel__counter">
        {{ currentIndex + 1 }} / {{ mediaItems.length }}
      </div>
    </div>

    <div v-if="videoEntries.length > 1" class="screenshot-carousel__video-list app-glass-surface">
      <div class="screenshot-carousel__video-list-inner">
        <button
          v-for="(entry, i) in videoEntries"
          :key="entry.item.key"
          :class="['screenshot-carousel__video-item', { active: currentIndex === entry.index }]"
          type="button"
          @click="currentIndex = entry.index"
        >
          <svg viewBox="0 0 24 24" width="16" height="16" class="screenshot-carousel__video-item-icon">
            <path fill="currentColor" d="M8 5v14l11-7z"/>
          </svg>
          <span class="screenshot-carousel__video-item-label">预告片 {{ i + 1 }}</span>
        </button>
      </div>
    </div>
  </div>

  <div v-else class="screenshot-carousel screenshot-carousel--empty">
    <div class="screenshot-carousel__empty">
      <svg viewBox="0 0 24 24" width="48" height="48" class="screenshot-carousel__empty-icon">
        <path fill="currentColor" d="M21 19V5c0-1.1-.9-2-2-2H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2zM8.5 13.5l2.5 3.01L14.5 12l4.5 6H5l3.5-4.5z"/>
      </svg>
      <p class="screenshot-carousel__empty-text">暂无媒体</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { IconLeft, IconRight } from '@arco-design/web-vue/es/icon'
import { resolveAssetUrl } from '@/utils/asset-url'

interface Props {
  screenshots?: string[]
  previewVideos?: string[]
  videoPoster?: string | null
  alt?: string
}

const props = withDefaults(defineProps<Props>(), {
  screenshots: () => [],
  previewVideos: () => [],
  videoPoster: null,
  alt: 'Game screenshot',
})

interface MediaItem {
  key: string
  type: 'image' | 'video'
  url: string
  thumbnail: string | null
}

const currentIndex = ref(0)
const carouselRef = ref<HTMLElement | null>(null)
const viewportRef = ref<HTMLElement | null>(null)
const videoRef = ref<HTMLVideoElement | null>(null)
const viewportAspect = ref<'16 / 9' | '4 / 3'>('16 / 9')
const viewportWidth = ref(0)
const brokenImages = ref<string[]>([])
const aspectResolved = ref(false)
const imageLoaded = ref(false)
const videoReady = ref(false)
const userUnmuted = ref(false)
let resizeObserver: ResizeObserver | null = null
let imageAutoplayTimer: number | null = null

const visibleScreenshots = computed(() => {
  const brokenSet = new Set(brokenImages.value)
  return props.screenshots.filter((shot) => !!shot && !brokenSet.has(shot))
})

const mediaItems = computed<MediaItem[]>(() => {
  const items: MediaItem[] = []
  const resolvedVideoList = props.previewVideos
    .filter(Boolean)
    .map((video) => resolveAssetUrl(video))
  resolvedVideoList.forEach((videoUrl, index) => {
    if (!videoUrl) return
    items.push({
      key: `video:${index}:${videoUrl}`,
      type: 'video',
      url: videoUrl,
      thumbnail: props.videoPoster ? resolveAssetUrl(props.videoPoster) : placeholderImage,
    })
  })
  visibleScreenshots.value.forEach((shot, index) => {
    items.push({
      key: `image:${index}:${shot}`,
      type: 'image',
      url: shot,
      thumbnail: shot,
    })
  })
  return items
})

const videoEntries = computed(() => {
  return mediaItems.value
    .map((item, index) => ({ item, index }))
    .filter(({ item }) => item.type === 'video')
})

const currentMedia = computed(() => {
  return mediaItems.value[currentIndex.value] || null
})

const placeholderImage = 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 800 450"%3E%3Crect fill="%231a1a1a" width="800" height="450"/%3E%3Ctext x="50%25" y="50%25" text-anchor="middle" dy=".3em" fill="%23666" font-size="24"%3E暂无截图%3C/text%3E%3C/svg%3E'

const viewportStyle = computed(() => {
  if (viewportWidth.value > 0) {
    const ratio = currentMedia.value?.type === 'video' ? 16 / 9 : (viewportAspect.value === '16 / 9' ? 16 / 9 : 4 / 3)
    const height = Math.round(viewportWidth.value / ratio)
    return { height: `${height}px`, minHeight: '0' }
  }
  if (currentMedia.value?.type === 'video') {
    return { aspectRatio: '16 / 9', minHeight: '0' }
  }
  return { aspectRatio: viewportAspect.value, minHeight: '0' }
})


watch(() => [props.screenshots, props.previewVideos], () => {
  brokenImages.value = []
  aspectResolved.value = false
  const items = mediaItems.value
  if (items.length === 0) {
    currentIndex.value = 0
    stopImageAutoplay()
    return
  }
  currentIndex.value = 0
})

watch(currentMedia, (nextMedia, previousMedia) => {
  if (!nextMedia) {
    imageLoaded.value = false
    videoReady.value = false
    stopImageAutoplay()
    return
  }
  if (nextMedia.type === 'video') {
    imageLoaded.value = true
    videoReady.value = false
    userUnmuted.value = false
    aspectResolved.value = true
    viewportAspect.value = '16 / 9'
    stopImageAutoplay()
    nextTick(() => {
      tryPlayVideo()
    })
    return
  }
  videoReady.value = false
  imageLoaded.value = nextMedia.url === previousMedia?.url
  startImageAutoplay()
})

onMounted(() => {
  if (!viewportRef.value || typeof ResizeObserver === 'undefined') return
  resizeObserver = new ResizeObserver((entries) => {
    const entry = entries[0]
    if (!entry) return
    const width = entry.contentRect?.width || viewportRef.value?.clientWidth || 0
    if (width) viewportWidth.value = width
  })
  resizeObserver.observe(viewportRef.value)
})

onBeforeUnmount(() => {
  stopImageAutoplay()
  if (resizeObserver) {
    resizeObserver.disconnect()
    resizeObserver = null
  }
})

const onImageLoad = (event: Event) => {
  imageLoaded.value = true
  if (aspectResolved.value) return

  const img = event.target as HTMLImageElement | null
  if (!img || !img.naturalWidth || !img.naturalHeight) return

  const ratio = img.naturalWidth / img.naturalHeight
  const diff169 = Math.abs(ratio - 16 / 9)
  const diff43 = Math.abs(ratio - 4 / 3)

  viewportAspect.value = diff169 <= diff43 ? '16 / 9' : '4 / 3'
  aspectResolved.value = true
}

const onVideoLoaded = () => {
  imageLoaded.value = true
  viewportAspect.value = '16 / 9'
  aspectResolved.value = true
}

const onVideoPlaying = () => {
  videoReady.value = true
}

const onVideoVolumeChange = () => {
  const video = videoRef.value
  if (!video) return
  if (!video.muted) {
    userUnmuted.value = true
  }
}

const tryPlayVideo = () => {
  const video = videoRef.value
  if (!video) return
  if (!userUnmuted.value) {
    video.muted = true
  }
  const playPromise = video.play()
  if (playPromise && typeof playPromise.catch === 'function') {
    playPromise.catch(() => {
      // Ignore autoplay rejections; controls remain available for manual play.
    })
  }
}

const stopImageAutoplay = () => {
  if (imageAutoplayTimer !== null) {
    window.clearTimeout(imageAutoplayTimer)
    imageAutoplayTimer = null
  }
}

const startImageAutoplay = () => {
  stopImageAutoplay()
  if (mediaItems.value.length <= 1 || currentMedia.value?.type !== 'image') return

  imageAutoplayTimer = window.setTimeout(() => {
    nextImage()
  }, 5000)
}

const prevImage = () => {
  currentIndex.value = currentIndex.value > 0 ? currentIndex.value - 1 : mediaItems.value.length - 1
}

const nextImage = () => {
  currentIndex.value = currentIndex.value < mediaItems.value.length - 1 ? currentIndex.value + 1 : 0
}

const handleVideoEnded = () => {
  if (mediaItems.value.length <= 1) return
  nextImage()
}

const handleImageError = (url: string) => {
  if (!url || brokenImages.value.includes(url)) return
  brokenImages.value = [...brokenImages.value, url]
}
</script>

<style scoped>
/* Main Carousel Container */
.screenshot-carousel {
  position: relative;
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.screenshot-carousel--empty {
  width: 100%;
  min-height: 420px;
  aspect-ratio: 16 / 9;
  background:
    radial-gradient(circle at 20% 18%, color-mix(in srgb, var(--color-primary-6) 12%, transparent), transparent 30%),
    linear-gradient(180deg, var(--color-carousel-empty-start) 0%, var(--color-carousel-empty-end) 100%);
  border-radius: 18px;
  border: 1px solid var(--color-border-2);
  box-shadow: var(--shadow-float);
}

.screenshot-carousel__empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
}

.screenshot-carousel__empty-icon {
  color: var(--color-border-3);
  margin-bottom: 16px;
}

.screenshot-carousel__empty-text {
  color: var(--color-text-3);
  font-size: 14px;
  margin: 0;
}

/* Viewport - 固定高度，宽度自适应 */
.screenshot-carousel__viewport {
  position: relative;
  border-radius: 18px;
  overflow: hidden;
  box-shadow: var(--shadow-float);
  width: 100%;
  display: flex;
  flex-direction: column;
  border: 1px solid var(--color-border-2);
}

/* Main Image Area */
.screenshot-carousel__main {
  position: relative;
  width: 100%;
  height: 100%;
  min-height: 0;
  overflow: hidden;
  z-index: 1;
}

.screenshot-carousel__media-shell {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  will-change: opacity;
}

.screenshot-carousel__image {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
  object-position: center center;
}

.screenshot-carousel__video {
  width: 100%;
  height: 100%;
  display: block;
  object-fit: contain;
  background: var(--color-media-bg);
  object-position: center center;
  opacity: 0;
  transform: scale(1.02);
  transition: opacity 0.5s ease, transform 0.5s ease;
}

.screenshot-carousel__video--ready {
  opacity: 1;
  transform: scale(1);
}

.screenshot-carousel__video-loader {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--app-scrim);
  z-index: 5;
}

.screenshot-carousel__loader-ring {
  position: relative;
  width: 80px;
  height: 80px;
}

.screenshot-carousel__loader-thumb {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 56px;
  height: 56px;
  border-radius: 50%;
  object-fit: cover;
  box-shadow: var(--shadow-soft);
}

.screenshot-carousel__loader-arc {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
}

.screenshot-carousel__loader-arc-spin {
  transform-origin: center;
  animation: screenshot-carousel-arc-spin 1s linear infinite;
}

@keyframes screenshot-carousel-arc-spin {
  to { transform: rotate(360deg); }
}

.screenshot-carousel-spinner-fade-enter-active,
.screenshot-carousel-spinner-fade-leave-active {
  transition: opacity 0.3s ease;
}

.screenshot-carousel-spinner-fade-enter-from,
.screenshot-carousel-spinner-fade-leave-to {
  opacity: 0;
}

.screenshot-carousel-fade-enter-active,
.screenshot-carousel-fade-leave-active {
  transition: opacity 0.32s ease;
}

.screenshot-carousel-fade-enter-active,
.screenshot-carousel-fade-leave-active {
  position: absolute;
  inset: 0;
}

.screenshot-carousel-fade-enter-from,
.screenshot-carousel-fade-leave-to {
  opacity: 0;
}

/* Navigation Arrows */
.screenshot-carousel__arrow {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  z-index: 10;
}

.screenshot-carousel__arrow--prev {
  left: 16px;
}

.screenshot-carousel__arrow--next {
  right: 16px;
}

/* Counter */
.screenshot-carousel__counter {
  position: absolute;
  bottom: 12px;
  right: 16px;
  padding: 6px 14px;
  border-radius: 20px;
  font-size: 12px;
  color: var(--color-text-1);
  font-weight: 500;
  z-index: 10;
}

/* Video List (File-Manager Style) */
.screenshot-carousel__video-list {
  border-radius: 16px;
  overflow: hidden;
}

.screenshot-carousel__video-list-inner {
  display: flex;
  flex-direction: column;
  gap: 2px;
  max-height: 180px;
  overflow-y: auto;
  padding: 6px;
  scrollbar-width: thin;
  scrollbar-color: var(--color-border-3) transparent;
}

.screenshot-carousel__video-list-inner::-webkit-scrollbar {
  width: 6px;
}

.screenshot-carousel__video-list-inner::-webkit-scrollbar-track {
  background: transparent;
}

.screenshot-carousel__video-list-inner::-webkit-scrollbar-thumb {
  background: var(--color-border-3);
  border-radius: 3px;
}

.screenshot-carousel__video-item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 8px 12px;
  border: none;
  border-radius: 10px;
  background: transparent;
  cursor: pointer;
  transition: all 0.15s ease;
  color: var(--color-text-2);
  font-size: 13px;
  text-align: left;
  font-family: inherit;
}

.screenshot-carousel__video-item:hover {
  background: color-mix(in srgb, var(--color-fill-2) 60%, transparent);
  color: var(--color-text-1);
}

.screenshot-carousel__video-item.active {
  background: color-mix(in srgb, var(--color-primary-6) 14%, transparent);
  color: var(--color-primary-3);
}

.screenshot-carousel__video-item-icon {
  flex-shrink: 0;
  opacity: 0.6;
}

.screenshot-carousel__video-item.active .screenshot-carousel__video-item-icon {
  opacity: 1;
}

.screenshot-carousel__video-item-label {
  font-weight: 500;
  line-height: 1;
}
</style>
