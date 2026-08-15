<template>
  <div
    class="shared-ambient-bg"
    :class="{ 'is-enabled': isEnabled, 'shared-ambient-bg--sharp': sharp }"
    aria-hidden="true"
  >
    <div
      v-for="(style, index) in layerStyles"
      :key="index"
      class="shared-ambient-bg__layer"
      :class="{ 'is-active': activeLayerIndex === index }"
      :style="{
        ...style,
        opacity: isEnabled ? (activeLayerIndex === index ? layerOpacity : 0) : 0,
      }"
    >
      <div class="shared-ambient-bg__overlay" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useRoute } from 'vue-router'
import { useUiStore } from '@/stores/ui'
import gamesService from '@/services/games.service'
import { buildApiUrl } from '@/services/api-url'
import {
  createAmbientBackgroundPool,
  getAmbientBackgroundPoolFromGames,
  hasAmbientBackgroundPoolImages,
  mergeAmbientBackgroundPools,
  type AmbientBackgroundPool,
} from '@/utils/ambient-background'

const route = useRoute()
const uiStore = useUiStore()
const { ambientBackgroundSource, sharedBackgroundAvailability } = storeToRefs(uiStore)

const props = withDefaults(defineProps<{
  /** 浮层场景（如开始屏幕）需要背景时强制启用，不受路由白名单限制 */
  forceEnabled?: boolean
  /** 背景源：'page' 吃页面设置的专属池（详情页等）；'global' 只吃自定义背景/全局池 */
  sourceMode?: 'page' | 'global'
  /** 清晰模式：去掉磨砂 blur，提高图层不透明度（浮层类场景用，避免"玻璃"感） */
  sharp?: boolean
}>(), {
  forceEnabled: false,
  sourceMode: 'page',
  sharp: false,
})

const SUPPORTED_ROUTE_NAMES = new Set([
  'login',
  'dashboard',
  'games',
  'games-timeline',
  'game-detail',
  'game-media',
  'pending-center',
  'series-library',
  'series-detail',
  'publisher-library',
  'publisher-detail',
  'wiki-edit',
  'not-found',
  'settings',
])

const GLOBAL_POOL_PAGE_LIMIT = 100

const APPLY_DELAY_MS = 50

const layerUrls = ref<string[]>(['', ''])
const activeLayerIndex = ref(0)
const hasAppliedBackground = ref(false)
const applyRequestId = ref(0)
const globalPool = ref<AmbientBackgroundPool>(createAmbientBackgroundPool())
const globalPoolLoaded = ref(false)
let globalPoolRequest: Promise<void> | null = null

const CUSTOM_BACKGROUND_PATH = buildApiUrl('/data/bg.jpg')

const isEnabled = computed(() =>
  props.forceEnabled || SUPPORTED_ROUTE_NAMES.has(String(route.name || ''))
)
const canUseCustomBackground = computed(() => sharedBackgroundAvailability.value === 'available')

const buildLayerStyle = (url: string) => {
  if (url) {
    return {
      backgroundImage: `url(${url})`,
      backgroundSize: 'cover',
      backgroundPosition: 'center',
      backgroundRepeat: 'no-repeat',
    }
  }

  return {
    background: 'transparent',
  }
}

const layerStyles = computed(() => layerUrls.value.map((url) => buildLayerStyle(url)))

// 清晰模式：不透明度拉高到 0.92（默认磨砂 0.58），配合 CSS 去掉 blur。
const layerOpacity = computed(() => (props.sharp ? 0.92 : 0.58))

const shuffleArray = <T,>(items: T[]) => {
  const copy = [...items]
  for (let index = copy.length - 1; index > 0; index -= 1) {
    const randomIndex = Math.floor(Math.random() * (index + 1))
    ;[copy[index], copy[randomIndex]] = [copy[randomIndex], copy[index]]
  }
  return copy
}

const preloadImage = (url: string) => {
  return new Promise<boolean>((resolve) => {
    if (!url) {
      resolve(false)
      return
    }

    const image = new Image()
    image.onload = () => resolve(true)
    image.onerror = () => resolve(false)
    image.src = url
  })
}

const wait = (ms: number) => new Promise((resolve) => {
  window.setTimeout(resolve, ms)
})

const fetchGlobalPool = async () => {
  const pools: AmbientBackgroundPool[] = []
  let page = 1

  while (true) {
    const result = await gamesService.getGames({
      query: { page, limit: GLOBAL_POOL_PAGE_LIMIT },
      sort: { field: 'created_at', order: 'desc' },
    })
    pools.push(getAmbientBackgroundPoolFromGames(result.data))
    const totalPages = Math.max(1, result.pagination.totalPages || 1)
    if (page >= totalPages) {
      break
    }
    page += 1
  }

  globalPool.value = mergeAmbientBackgroundPools(pools)
  globalPoolLoaded.value = true
}

const ensureGlobalPool = async () => {
  if (globalPoolLoaded.value) return

  if (!globalPoolRequest) {
    globalPoolRequest = fetchGlobalPool().finally(() => {
      globalPoolRequest = null
    })
  }

  try {
    await globalPoolRequest
  } catch {
    // Keep the background non-blocking; another route change can retry the pool load.
  }
}

const pickRandomBackground = async (urls: string[], currentUrl: string) => {
  const uniqueUrls = urls.filter((url, index) => url && urls.indexOf(url) === index)
  const preferredUrls = uniqueUrls.length > 1
    ? uniqueUrls.filter((url) => url !== currentUrl)
    : uniqueUrls
  const candidateUrls = preferredUrls.length > 0 ? preferredUrls : uniqueUrls

  for (const url of shuffleArray(candidateUrls)) {
    if (await preloadImage(url)) {
      return url
    }
  }

  return ''
}

const pickFromAmbientBackgroundPool = async (pool: AmbientBackgroundPool, currentUrl: string) => {
  for (const urls of [pool.screenshots, pool.banners]) {
    if (urls.length === 0) {
      continue
    }

    const pickedUrl = await pickRandomBackground(urls, currentUrl)
    if (pickedUrl) {
      return pickedUrl
    }
  }

  return ''
}

const loadBackground = async () => {
  const currentUrl = layerUrls.value[activeLayerIndex.value] || ''
  const sourcePool = ambientBackgroundSource.value?.pool
  // 浮层（开始屏幕）用 sourceMode='global'：忽略页面专属背景池，
  // 只吃自定义背景或全局池，避免从详情页打开时继承该游戏的背景。
  if (props.sourceMode === 'page' && sourcePool && hasAmbientBackgroundPoolImages(sourcePool)) {
    const sourceUrl = await pickFromAmbientBackgroundPool(sourcePool, currentUrl)
    if (sourceUrl) {
      return sourceUrl
    }
  }

  if (canUseCustomBackground.value) {
    return CUSTOM_BACKGROUND_PATH
  }

  await ensureGlobalPool()

  if (hasAmbientBackgroundPoolImages(globalPool.value)) {
    return pickFromAmbientBackgroundPool(globalPool.value, currentUrl)
  }

  return ''
}

const applyBackground = async () => {
  const requestId = applyRequestId.value + 1
  applyRequestId.value = requestId

  if (!isEnabled.value) {
    layerUrls.value = ['', '']
    hasAppliedBackground.value = false
    return
  }

  const nextBackgroundUrl = await loadBackground()
  if (requestId !== applyRequestId.value) {
    return
  }

  const nextUrl = nextBackgroundUrl || ''
  const currentUrl = layerUrls.value[activeLayerIndex.value]

  if (!hasAppliedBackground.value) {
    layerUrls.value = [nextUrl, nextUrl]
    activeLayerIndex.value = 0
    hasAppliedBackground.value = true
    await nextTick()
    return
  }

  if (nextUrl === currentUrl) {
    return
  }

  const nextLayerIndex = activeLayerIndex.value === 0 ? 1 : 0
  layerUrls.value[nextLayerIndex] = nextUrl

  await nextTick()
  await wait(APPLY_DELAY_MS)

  if (requestId !== applyRequestId.value) {
    return
  }

  requestAnimationFrame(() => {
    if (requestId !== applyRequestId.value) {
      return
    }
    activeLayerIndex.value = nextLayerIndex
  })
}

watch(
  [
    isEnabled,
    () => props.sourceMode,
    () => route.name,
    () => ambientBackgroundSource.value?.key || '',
    () => [
      ...(ambientBackgroundSource.value?.pool.screenshots || []),
      '::',
      ...(ambientBackgroundSource.value?.pool.banners || []),
    ].join('|'),
    () => sharedBackgroundAvailability.value,
  ],
  async () => {
    await applyBackground()
  },
  { immediate: true },
)
</script>

<style scoped>
.shared-ambient-bg {
  position: fixed;
  inset: 0;
  z-index: 0;
  pointer-events: none;
  overflow: hidden;
  contain: paint;
}

.shared-ambient-bg__layer {
  position: absolute;
  inset: 0;
  filter: saturate(1.02) blur(8px) brightness(1.1);
  transform: scale(1.015);
  transform-origin: center center;
  /* 非活动层在过渡结束后 visibility:hidden，释放全屏 blur 合成层（GPU 纹理）
     切换时由 .is-active 立即恢复可见，避免隐藏后切回闪烁 */
  visibility: hidden;
  transition:
    opacity 0.85s ease,
    visibility 0s linear 0.85s;
}

/* 清晰模式（浮层类场景）：去掉磨砂 blur，只保留轻微增亮 */
.shared-ambient-bg--sharp .shared-ambient-bg__layer {
  filter: saturate(1.02) brightness(1.06);
  transform: scale(1.01);
}

.shared-ambient-bg__layer.is-active {
  visibility: visible;
  transition:
    opacity 0.85s ease,
    visibility 0s;
}

.shared-ambient-bg__overlay {
  width: 100%;
  height: 100%;
  background:
    radial-gradient(circle at 18% 20%, rgba(196, 214, 230, 0.06), transparent 24%),
    radial-gradient(circle at 82% 16%, rgba(173, 196, 219, 0.05), transparent 22%),
    linear-gradient(180deg, rgba(7, 11, 18, 0.05) 0%, rgba(10, 14, 21, 0.08) 44%, rgba(17, 23, 32, 0.14) 100%);
}
</style>
