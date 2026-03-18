<template>
  <div class="screenshot-carousel" v-if="mediaItems.length > 0" ref="carouselRef">
    <div class="screenshot-carousel__viewport" :style="viewportStyle" ref="viewportRef">
      <button
        v-if="mediaItems.length > 1"
        class="screenshot-carousel__arrow screenshot-carousel__arrow--prev"
        @click="prevImage"
        @mouseenter="hoverArrow = 'prev'"
        @mouseleave="hoverArrow = null"
      >
        <svg viewBox="0 0 24 24" width="24" height="24">
          <path fill="currentColor" d="M15.41 7.41L14 6l-6 6 6 6 1.41-1.41L10.83 12z"/>
        </svg>
      </button>

      <div class="screenshot-carousel__main">
        <img
          v-if="currentMedia?.type === 'image'"
          :key="currentMedia.url"
          :src="currentMedia.url"
          :alt="alt"
          :class="['screenshot-carousel__image', { 'is-loaded': imageLoaded }]"
          @load="onImageLoad"
          @error="handleImageError(currentMedia.url)"
        />
        <video
          v-else-if="currentMedia?.type === 'video'"
          :key="currentMedia.url"
          ref="videoRef"
          :src="currentMedia.url"
          class="screenshot-carousel__video"
          :poster="videoPoster || undefined"
          autoplay
          controls
          loop
          muted
          playsinline
          preload="metadata"
          @canplay="tryPlayVideo"
          @loadedmetadata="onVideoLoaded"
        />
      </div>

      <button
        v-if="mediaItems.length > 1"
        class="screenshot-carousel__arrow screenshot-carousel__arrow--next"
        @click="nextImage"
        @mouseenter="hoverArrow = 'next'"
        @mouseleave="hoverArrow = null"
      >
        <svg viewBox="0 0 24 24" width="24" height="24">
          <path fill="currentColor" d="M10 6L8.59 7.41 13.17 12l-4.58 4.59L10 18l6-6z"/>
        </svg>
      </button>

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

interface Props {
  screenshots?: string[]
  previewVideo?: string | null
  videoPoster?: string | null
  alt?: string
  height?: number
}

const props = withDefaults(defineProps<Props>(), {
  screenshots: () => [],
  previewVideo: null,
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
const hoverArrow = ref<'prev' | 'next' | null>(null)
const carouselRef = ref<HTMLElement | null>(null)
const viewportRef = ref<HTMLElement | null>(null)
const videoRef = ref<HTMLVideoElement | null>(null)
const viewportAspect = ref<'16 / 9' | '4 / 3'>('16 / 9')
const viewportWidth = ref(0)
const brokenImages = ref<string[]>([])
const aspectResolved = ref(false)
const imageLoaded = ref(false)
let resizeObserver: ResizeObserver | null = null

const visibleScreenshots = computed(() => {
  const brokenSet = new Set(brokenImages.value)
  return props.screenshots.filter((shot) => !!shot && !brokenSet.has(shot))
})

const mediaItems = computed<MediaItem[]>(() => {
  const items: MediaItem[] = []
  if (props.previewVideo) {
    items.push({
      key: `video:${props.previewVideo}`,
      type: 'video',
      url: props.previewVideo,
      thumbnail: props.videoPoster || placeholderImage,
    })
  }
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
  if (props.height) {
    return { height: `${props.height}px` }
  }
  if (currentMedia.value?.type === 'video') {
    return { aspectRatio: '16 / 9' }
  }
  if (viewportWidth.value > 0) {
    const ratio = viewportAspect.value === '16 / 9' ? 16 / 9 : 4 / 3
    const height = Math.round(viewportWidth.value / ratio)
    return { height: `${height}px` }
  }
  return { aspectRatio: viewportAspect.value }
})

// 应用高度的函数
const applyHeight = (height: number | undefined) => {
  if (carouselRef.value) {
    if (height) {
      carouselRef.value.style.height = `${height}px`
    } else {
      carouselRef.value.style.height = ''
    }
  }
}

// 监听高度变化
watch(() => props.height, (newHeight) => {
  applyHeight(newHeight)
}, { immediate: true })

watch(() => [props.screenshots, props.previewVideo], () => {
  brokenImages.value = []
  currentIndex.value = 0
  aspectResolved.value = false
}, { deep: true })

watch(mediaItems, (items) => {
  if (items.length === 0) {
    currentIndex.value = 0
    return
  }
  if (currentIndex.value >= items.length) {
    currentIndex.value = items.length - 1
  }
})

watch(currentMedia, (nextMedia, previousMedia) => {
  if (!nextMedia) {
    imageLoaded.value = false
    return
  }
  if (nextMedia.type === 'video') {
    imageLoaded.value = true
    aspectResolved.value = true
    viewportAspect.value = '16 / 9'
    nextTick(() => {
      tryPlayVideo()
    })
    return
  }
  imageLoaded.value = nextMedia.url === previousMedia?.url
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
  if (resizeObserver) {
    resizeObserver.disconnect()
    resizeObserver = null
  }
})

const onImageLoad = (event: Event) => {
  imageLoaded.value = true
  if (aspectResolved.value || props.height) return

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

const prevImage = () => {
  currentIndex.value = currentIndex.value > 0 ? currentIndex.value - 1 : mediaItems.value.length - 1
}

const nextImage = () => {
  currentIndex.value = currentIndex.value < mediaItems.value.length - 1 ? currentIndex.value + 1 : 0
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
}

.screenshot-carousel--empty {
  background: linear-gradient(135deg, #1a1a2e 0%, #0f0f1a 100%);
  border-radius: 4px;
  border: 1px solid rgba(255, 255, 255, 0.05);
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
  border-radius: 4px;
  overflow: hidden;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
  width: 100%;
  display: flex;
  flex-direction: column;
}

/* Main Image Area */
.screenshot-carousel__main {
  position: relative;
  width: 100%;
  height: 100%;
  overflow: hidden;
}

.screenshot-carousel__image {
  width: 100%;
  height: 100%;
  object-fit: contain;
  display: block;
  opacity: 0;
  transition: opacity 0.16s ease;
}

.screenshot-carousel__video {
  width: 100%;
  height: 100%;
  display: block;
  background: #000;
}

.screenshot-carousel__image.is-loaded {
  opacity: 1;
}

/* Navigation Arrows */
.screenshot-carousel__arrow {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  width: 48px;
  height: 48px;
  background: linear-gradient(90deg, rgba(0, 0, 0, 0.8) 0%, rgba(0, 0, 0, 0.4) 100%);
  border: none;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  color: rgba(255, 255, 255, 0.6);
  transition: all 0.2s ease;
  z-index: 10;
  border-radius: 0;
}

.screenshot-carousel__arrow--prev {
  left: 0;
  background: linear-gradient(90deg, rgba(0, 0, 0, 0.8) 0%, rgba(0, 0, 0, 0) 100%);
}

.screenshot-carousel__arrow--next {
  right: 0;
  background: linear-gradient(90deg, rgba(0, 0, 0, 0) 0%, rgba(0, 0, 0, 0.8) 100%);
}

.screenshot-carousel__arrow:hover {
  color: #fff;
  background: rgba(0, 0, 0, 0.6);
}

.screenshot-carousel__arrow--prev:hover {
  background: linear-gradient(90deg, rgba(0, 0, 0, 0.9) 0%, rgba(0, 0, 0, 0.3) 100%);
}

.screenshot-carousel__arrow--next:hover {
  background: linear-gradient(90deg, rgba(0, 0, 0, 0.3) 0%, rgba(0, 0, 0, 0.9) 100%);
}

/* Counter */
.screenshot-carousel__counter {
  position: absolute;
  bottom: 12px;
  right: 16px;
  background: rgba(0, 0, 0, 0.75);
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
  margin-top: 4px;
  padding: 8px 0;
  background: rgba(0, 0, 0, 0.3);
  border-radius: 0 0 4px 4px;
}

.screenshot-carousel__filmstrip-inner {
  display: flex;
  gap: 4px;
  justify-content: flex-start;
  overflow-x: auto;
  padding: 0 8px;
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
  border-radius: 2px;
  overflow: hidden;
  cursor: pointer;
  transition: all 0.2s ease;
  border: 2px solid transparent;
  flex-shrink: 0;
  background: #1a1a1a;
  opacity: 0.6;
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
  border-color: rgba(255, 255, 255, 0.5);
  opacity: 1;
}

.screenshot-carousel__film:hover img {
  transform: scale(1.02);
}

.screenshot-carousel__film.active {
  border-color: #fff;
  opacity: 1;
}
</style>
