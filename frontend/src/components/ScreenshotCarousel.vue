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
              <!-- 2026-08-07: 无 autoplay/poster。
                   autoplay 会在 src 切换瞬间由浏览器直接播放，绕过下方
                   loader 调度 → "先播后转圈"；poster 在首帧渲染前显示
                   banner，与 loader 的封面展示重复。 -->
              <video
                ref="videoRef"
                v-show="!videoFailed"
                :src="currentMedia.url"
                class="screenshot-carousel__video"
                :class="{ 'screenshot-carousel__video--ready': videoReady }"
                controls
                playsinline
                preload="none"
                @canplay="tryPlayVideo"
                @loadedmetadata="onVideoLoaded"
                @playing="onVideoPlaying"
                @ended="handleVideoEnded"
                @volumechange="onVideoVolumeChange"
                @error="handleVideoError"
              />
              <transition name="screenshot-carousel-spinner-fade">
                <div v-if="!videoReady && !videoFailed" class="screenshot-carousel__video-loader">
                  <div class="screenshot-carousel__loader-ring">
                    <img
                      v-if="currentVideoPoster"
                      :src="currentVideoPoster"
                      class="screenshot-carousel__loader-thumb"
                      alt=""
                      @error="handlePosterError(currentVideoPoster)"
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
              <div v-if="videoFailed" class="screenshot-carousel__video-failed">
                <img
                  v-if="currentVideoPoster"
                  :src="currentVideoPoster"
                  class="screenshot-carousel__video-poster"
                  alt=""
                  @error="handlePosterError(currentVideoPoster)"
                >
                <span class="screenshot-carousel__video-failed-text">视频加载失败</span>
              </div>
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

    <div v-if="mediaItems.length > 1" class="screenshot-carousel__filmstrip app-glass-surface">
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
            @error="item.type === 'image' ? handleImageError(item.thumbnail) : (item.poster ? handlePosterError(item.poster) : undefined)"
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
import { resolveAssetUrl, withAssetWidth } from '@/utils/asset-url'
import { shouldResetVideoPlaybackState } from '@/utils/video-playback'

interface Props {
  screenshots?: string[]
  previewVideos?: Array<{ path: string; poster_path?: string | null }>
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
  poster: string | null
  thumbnail: string | null
}

const currentIndex = ref(0)
const carouselRef = ref<HTMLElement | null>(null)
const viewportRef = ref<HTMLElement | null>(null)
const videoRef = ref<HTMLVideoElement | null>(null)
const viewportAspect = ref<'16 / 9' | '4 / 3'>('16 / 9')
const viewportWidth = ref(0)
const brokenImages = ref<string[]>([])
const brokenPosters = ref<string[]>([])
const aspectResolved = ref(false)
const imageLoaded = ref(false)
const videoReady = ref(false)
const videoFailed = ref(false)
const userUnmuted = ref(false)
let resizeObserver: ResizeObserver | null = null
let imageAutoplayTimer: number | null = null

const visibleScreenshots = computed(() => {
  const brokenSet = new Set(brokenImages.value)
  return props.screenshots.filter((shot) => {
    if (!shot) return false
    return !brokenSet.has(shot)
      && !brokenSet.has(withAssetWidth(shot, 1280))
      && !brokenSet.has(withAssetWidth(shot, 480))
  })
})

const mediaItems = computed<MediaItem[]>(() => {
  const items: MediaItem[] = []
  const brokenSet = new Set(brokenPosters.value)
  const fallbackPoster = props.videoPoster ? resolveAssetUrl(props.videoPoster) : null
  props.previewVideos
    .filter((video) => Boolean(video.path))
    .forEach((video, index) => {
      const videoUrl = resolveAssetUrl(video.path)
      if (!videoUrl) return
      const ownPoster = video.poster_path ? resolveAssetUrl(video.poster_path) : null
      const poster = ownPoster && !brokenSet.has(ownPoster)
        ? ownPoster
        : (fallbackPoster && !brokenSet.has(fallbackPoster) ? fallbackPoster : null)
      items.push({
        key: `video:${index}:${videoUrl}`,
        type: 'video',
        url: videoUrl,
        poster,
        thumbnail: poster || placeholderImage,
      })
    })
  visibleScreenshots.value.forEach((shot, index) => {
    items.push({
      key: `image:${index}:${shot}`,
      type: 'image',
      url: withAssetWidth(shot, 1280),
      poster: null,
      thumbnail: withAssetWidth(shot, 480),
    })
  })
  return items
})

const currentMedia = computed(() => {
  return mediaItems.value[currentIndex.value] || null
})

const currentVideoPoster = computed(() => {
  const media = currentMedia.value
  return media?.type === 'video' ? (media.poster || null) : null
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
  brokenPosters.value = []
  videoFailed.value = false
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
    videoFailed.value = false
    stopImageAutoplay()
    return
  }
  if (nextMedia.type === 'video') {
    imageLoaded.value = true
    // 2026-08-05: 只有视频 URL 变化才重置播放状态；详情保存后刷新是同一 URL，
    // 直接复用已就绪的 <video>，避免卡在"已加载但不触发事件"的透明态。
    if (shouldResetVideoPlaybackState(previousMedia?.url, nextMedia.url)) {
      videoReady.value = false
      videoFailed.value = false
      userUnmuted.value = false
      markLoaderShown()
    }
    aspectResolved.value = true
    viewportAspect.value = '16 / 9'
    stopImageAutoplay()
    nextTick(() => {
      tryPlayVideo()
    })
    return
  }
  videoReady.value = false
  videoFailed.value = false
  imageLoaded.value = nextMedia.url === previousMedia?.url
  startImageAutoplay()
})

onMounted(() => {
  if (viewportRef.value && typeof ResizeObserver !== 'undefined') {
    resizeObserver = new ResizeObserver((entries) => {
      const entry = entries[0]
      if (!entry) return
      const width = entry.contentRect?.width || viewportRef.value?.clientWidth || 0
      if (width) viewportWidth.value = width
    })
    resizeObserver.observe(viewportRef.value)
  }
  // 2026-08-07: 首次进入时接管原 autoplay 属性的职责。
  // loader 已在首帧挂载（videoReady 初始 false），记录淡入起点并触发加载；
  // watch(currentMedia) 首次求值不触发，只有这里能启动首次拉流。
  loaderShownAt = Date.now()
  tryPlayVideo()
})

onBeforeUnmount(() => {
  stopImageAutoplay()
  if (playTimer !== undefined) {
    window.clearTimeout(playTimer)
    playTimer = undefined
  }
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

const handleVideoError = () => {
  videoReady.value = false
  videoFailed.value = true
}

const onVideoVolumeChange = () => {
  const video = videoRef.value
  if (!video) return
  if (!video.muted) {
    userUnmuted.value = true
  }
}

// ===== 2026-08-07 视频播放时序设计（回归必读） =====
// 背景：轮播切换预告片时曾出现"视频先播、loader 转圈后淡入盖在画面上"的
// 错乱观感。控制台取证（拦截 play/playing/canplay + loader opacity 采样）确认：
// 1) video 元素原带 autoplay 属性——src 切换瞬间由浏览器直接触发加载播放，
//    完全绕过代码调度（play 事件先于任何代码调用出现）；
// 2) loader 背景是半透明 scrim，视频在 loader 底下播放时画面透出，
//    视觉上"先播后转圈"；
// 3) loader 的 1.5s 淡入 CSS animation 会压制 leave transition，导致
//    loader 必须等淡入动画播完才消失（曾表现为"播放后 1.5s 转圈才消失"）。
// 修复决策：
// - 移除 autoplay 属性，播放完全由代码调度（见模板 video 元素注释）；
// - 视频就绪后不立即 play()：等 loader 淡入（LOADER_FADE_IN_MS）+
//   淡出（LOADER_FADE_OUT_MS）动画完整播完再播放，保证"先动画后播放"；
// - 首次进入由 onMounted 触发加载（watch(currentMedia) 首次求值不触发，
//   autoplay 移除后必须显式接管）；
// - 移除 video poster：首帧渲染前会显示 banner，与 loader 的封面展示重复。
// 动画时长权衡：1.5s → 0.8s → 0.5s（用户迭代：入场等待过长），
// 淡出 0.3s → 0.1s（用户迭代：淡出拖慢播放）。改动必须 CSS 与常量同步。
// ==================================================
const LOADER_FADE_IN_MS = 500
const LOADER_FADE_OUT_MS = 100
// 2026-08-14: 默认播放音量 10%，见 tryPlayVideo。
const DEFAULT_VIDEO_VOLUME = 0.1
let loaderShownAt = 0
let playScheduled = false
let playTimer: number | undefined

const markLoaderShown = () => {
  loaderShownAt = Date.now()
  playScheduled = false
  if (playTimer !== undefined) {
    window.clearTimeout(playTimer)
    playTimer = undefined
  }
}

const schedulePlayAfterLoader = () => {
  const video = videoRef.value
  if (!video || playScheduled) return
  playScheduled = true
  const elapsed = Date.now() - loaderShownAt
  const remainingFadeIn = Math.max(0, LOADER_FADE_IN_MS - elapsed)
  playTimer = window.setTimeout(() => {
    // loader 淡入完成：隐藏 loader（淡出），彻底消失后再开始播放
    videoReady.value = true
    playTimer = window.setTimeout(() => {
      playTimer = undefined
      playScheduled = false
      const playPromise = video.play()
      if (playPromise && typeof playPromise.catch === 'function') {
        playPromise.catch(() => {
          // 2026-08-14: 移除 muted 后，无用户手势的带声 play() 可能被浏览器
          // 自动播放策略拒绝（NotAllowedError）。静音重试保证"先动画后播放"
          // 时序不回归；用户手动取消静音后即以默认 10% 音量播放。
          // 注意不能依赖 userUnmuted 判断：tryPlayVideo 设置 volume/muted 本身
          // 就会触发 volumechange → userUnmuted 被置 true，静音回退会失效。
          video.muted = true
          const retry = video.play()
          if (retry && typeof retry.catch === 'function') {
            retry.catch(() => {
              // Ignore autoplay rejections; controls remain available for manual play.
            })
          }
        })
      }
    }, LOADER_FADE_OUT_MS)
  }, remainingFadeIn)
}

const tryPlayVideo = () => {
  const video = videoRef.value
  if (!video) return
  if (!userUnmuted.value) {
    // 2026-08-14: 默认以 10% 音量非静音播放，不再强制 muted。
    // 若浏览器自动播放策略拒绝带声 play()，schedulePlayAfterLoader 内静音重试；
    // 用户手动取消静音/调音量后（userUnmuted）不再覆盖用户选择。
    video.muted = false
    video.volume = DEFAULT_VIDEO_VOLUME
  }
  if (video.readyState >= 2) {
    // 已就绪（缓存/已加载完成）：等 loader 动画播完再播放
    schedulePlayAfterLoader()
    return
  }
  // 未就绪：触发加载但不播放（preload=none 下只有 load() 会启动拉流），
  // 期间 loader 转圈；canplay 后走 readyState >= 2 分支调度播放
  if (video.networkState === 0 || video.readyState === 0) {
    video.load()
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

const handlePosterError = (url: string) => {
  if (!url || brokenPosters.value.includes(url)) return
  brokenPosters.value = [...brokenPosters.value, url]
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
  /* 短入场动画（0.8s）：期间视频已开始加载，动画结束即播放，
     切视频时没有"突然出现"的跳变 */
  animation: screenshot-carousel-loader-in 0.5s ease;
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
  animation: screenshot-carousel-thumb-in 0.5s ease;
}

@keyframes screenshot-carousel-loader-in {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

@keyframes screenshot-carousel-thumb-in {
  from {
    opacity: 0;
    transform: translate(-50%, -50%) scale(0.8);
  }
  to {
    opacity: 1;
    transform: translate(-50%, -50%) scale(1);
  }
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
  to {
    transform: rotate(360deg);
  }
}

/* 视频加载失败态：整幅封面 + 错误提示（21:9 banner 兜底时 contain 不裁切） */
.screenshot-carousel__video-poster {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.screenshot-carousel__video-failed {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  background: var(--color-carousel-empty-start);
  color: var(--color-text-3);
}

.screenshot-carousel__video-failed-text {
  font-size: 14px;
}

.screenshot-carousel-spinner-fade-enter-active,
.screenshot-carousel-spinner-fade-leave-active {
  transition: opacity 0.1s ease;
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

/* Filmstrip (Thumbnail Navigation) - Steam Style */
.screenshot-carousel__filmstrip {
  padding: 10px 0 0;
  border-radius: 16px;
  overflow: hidden;
}

.screenshot-carousel__filmstrip-inner {
  display: flex;
  gap: 8px;
  justify-content: flex-start;
  overflow-x: auto;
  padding: 0 10px 6px;
  scrollbar-width: thin;
  scrollbar-color: var(--color-border-3) transparent;
}

.screenshot-carousel__filmstrip-inner::-webkit-scrollbar {
  height: 6px;
}

.screenshot-carousel__filmstrip-inner::-webkit-scrollbar-track {
  background: transparent;
}

.screenshot-carousel__filmstrip-inner::-webkit-scrollbar-thumb {
  background: var(--color-border-3);
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
  border: 1px solid var(--color-border-2);
  flex-shrink: 0;
  background: color-mix(in srgb, var(--app-card-surface) 88%, transparent);
  opacity: 1;
  box-shadow: inset 0 1px 0 var(--color-border-1);
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
  color: var(--color-text-on-dark);
  background: var(--color-bg-3);
}

.screenshot-carousel__film-overlay {
  position: absolute;
  inset: 0;
  background: transparent;
  transition: all 0.2s ease;
}

.screenshot-carousel__film:hover {
  border-color: var(--app-glass-border-hover);
  opacity: 1;
  transform: translateY(-1px);
}

.screenshot-carousel__film:hover img {
  transform: scale(1.02);
}

.screenshot-carousel__film.active {
  border-color: var(--color-primary-4);
  opacity: 1;
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--color-primary-4) 22%, transparent), var(--shadow-hover);
}
</style>
