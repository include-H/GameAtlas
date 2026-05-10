<template>
  <div class="screenshot-carousel" v-if="mediaItems.length > 0" ref="carouselRef">
    <div class="screenshot-carousel__viewport" :style="viewportStyle" ref="viewportRef">
      <a-button
        v-if="mediaItems.length > 1"
        class="screenshot-carousel__arrow screenshot-carousel__arrow--prev"
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
                      <circle cx="50" cy="50" r="46" fill="none" stroke="rgba(255,255,255,0.1)" stroke-width="3" />
                      <circle
                        cx="50" cy="50" r="46"
                        fill="none"
                        stroke="rgba(170,222,255,0.9)"
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
        class="screenshot-carousel__arrow screenshot-carousel__arrow--next"
        type="text"
        shape="circle"
        @click="nextImage"
        aria-label="下一张"
      >
        <template #icon>
          <icon-right />
        </template>
      </a-button>

      <div v-if="mediaItems.length > 1" class="screenshot-carousel__counter">
        {{ currentIndex + 1 }} / {{ mediaItems.length }}
      </div>
    </div>

    <div v-if="mediaItems.length > 1" class="screenshot-carousel__filmstrip">
      <div class="screenshot-carousel__filmstrip-inner">
        <div
          v-for="(item, index) in mediaItems"
          :key="item.key"
          :class="['screenshot-carousel__film', { active: currentIndex === index }]"
          @click="currentIndex = index"
        >
          <img
            v-if="item.thumbnail"
            :src="item.thumbnail"
            :alt="item.type === 'video' ? 'Video thumbnail' : `Screenshot ${index + 1}`"
            @error="item.type === 'image' ? handleImageError(item.thumbnail) : undefined"
          />
          <div v-else class="screenshot-carousel__film-placeholder">
            <svg viewBox="0 0 24 24" width="24" height="24">
              <path fill="currentColor" d="M8 5v14l11-7z"/>
            </svg>
          </div>
          <div class="screenshot-carousel__film-overlay"></div>
        </div>
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
  height?: number
}

const props = withDefaults(defineProps<Props>(), {
  screenshots: () => [],
  previewVideos: () => [],
  videoPoster: null,
  alt: 'Game screenshot',
  height: undefined,
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
  currentIndex.value = 0
  aspectResolved.value = false
}, { deep: true })

watch(mediaItems, (items) => {
  if (items.length === 0) {
    currentIndex.value = 0
    stopImageAutoplay()
    return
  }
  if (currentIndex.value >= items.length) {
    currentIndex.value = items.length - 1
  }
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

const tryPlayVideo = () => {
  const video = videoRef.value
  if (!video) return
  video.muted = true
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
    radial-gradient(circle at 20% 18%, rgba(40, 52, 84, 0.36), transparent 30%),
    linear-gradient(180deg, #090b10 0%, #05070b 100%);
  border-radius: 18px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  box-shadow: 0 24px 50px rgba(0, 0, 0, 0.34);
}

.screenshot-carousel__empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
}

.screenshot-carousel__empty-icon {
  color: rgba(255, 255, 255, 0.15);
  margin-bottom: 16px;
}

.screenshot-carousel__empty-text {
  color: rgba(255, 255, 255, 0.3);
  font-size: 14px;
  margin: 0;
}

/* Viewport - 固定高度，宽度自适应 */
.screenshot-carousel__viewport {
  position: relative;
  border-radius: 18px;
  overflow: hidden;
  box-shadow: 0 26px 60px rgba(0, 0, 0, 0.38);
  width: 100%;
  display: flex;
  flex-direction: column;
  border: 1px solid rgba(255, 255, 255, 0.08);
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
  background: #000;
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
  background: rgba(5, 8, 14, 0.6);
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
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.4);
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
  background: rgba(5, 10, 18, 0.78);
  padding: 6px 14px;
  border-radius: 20px;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.9);
  font-weight: 500;
  backdrop-filter: blur(8px);
  border: 1px solid rgba(255, 255, 255, 0.1);
  z-index: 10;
}

/* Filmstrip (Thumbnail Navigation) - Steam Style */
.screenshot-carousel__filmstrip {
  padding: 10px 0;
  background: var(--app-card-surface);
  border-radius: 16px;
  border: 1px solid var(--app-card-border);
  backdrop-filter: blur(var(--app-card-backdrop-blur));
  -webkit-backdrop-filter: blur(var(--app-card-backdrop-blur));
  overflow: hidden;
  box-shadow: var(--shadow-soft);
}

.screenshot-carousel__filmstrip-inner {
  display: flex;
  gap: 8px;
  justify-content: flex-start;
  overflow-x: auto;
  padding: 0 10px;
  scrollbar-width: thin;
  scrollbar-color: rgba(255, 255, 255, 0.2) transparent;
}

.screenshot-carousel__filmstrip-inner::-webkit-scrollbar {
  height: 6px;
}

.screenshot-carousel__filmstrip-inner::-webkit-scrollbar-track {
  background: transparent;
}

.screenshot-carousel__filmstrip-inner::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.2);
  border-radius: 3px;
}

.screenshot-carousel__film {
  position: relative;
  width: auto;
  height: 65px;
  aspect-ratio: 16/9;
  border-radius: 12px;
  overflow: hidden;
  cursor: pointer;
  transition: all 0.2s ease;
  border: 1px solid rgba(255, 255, 255, 0.08);
  flex-shrink: 0;
  background: color-mix(in srgb, var(--app-card-surface) 88%, transparent);
  opacity: 1;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.04);
}

.screenshot-carousel__film img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform 0.3s ease;
}

.screenshot-carousel__film-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: rgba(255, 255, 255, 0.8);
  background: linear-gradient(135deg, rgba(20, 20, 30, 0.95) 0%, rgba(35, 35, 50, 0.95) 100%);
}

.screenshot-carousel__film-overlay {
  position: absolute;
  inset: 0;
  background: linear-gradient(
    to bottom,
    rgba(103, 193, 245, 0) 0%,
    rgba(103, 193, 245, 0) 100%
  );
  transition: all 0.2s ease;
}

.screenshot-carousel__film:hover {
  border-color: rgba(147, 204, 255, 0.46);
  opacity: 1;
  transform: translateY(-1px);
}

.screenshot-carousel__film:hover img {
  transform: scale(1.02);
}

.screenshot-carousel__film.active {
  border-color: rgba(170, 222, 255, 0.95);
  opacity: 1;
  box-shadow: 0 0 0 1px rgba(170, 222, 255, 0.22), 0 10px 20px rgba(0, 0, 0, 0.25);
}
</style>
