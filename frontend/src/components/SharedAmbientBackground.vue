<template>
  <div
    class="shared-ambient-bg"
    :class="{ 'is-enabled': isEnabled }"
    aria-hidden="true"
  >
    <div
      v-for="(style, index) in layerStyles"
      :key="index"
      class="shared-ambient-bg__layer"
      :class="{ 'is-active': activeLayerIndex === index }"
      :style="{
        ...style,
        opacity: isEnabled ? (activeLayerIndex === index ? 0.58 : 0) : 0,
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
import { getAmbientBackgroundUrlsFromGameListItem } from '@/utils/ambient-background'

const route = useRoute()
const uiStore = useUiStore()
const { ambientBackgroundSource, sharedBackgroundAvailability } = storeToRefs(uiStore)

const SUPPORTED_ROUTE_NAMES = new Set([
  'login',
  'dashboard',
  'games',
  'games-timeline',
  'game-detail',
  'pending-center',
  'series-library',
  'series-detail',
  'wiki-edit',
  'not-found',
  'settings',
])

const GLOBAL_POOL_PAGE_LIMIT = 50

const APPLY_DELAY_MS = 50

const layerUrls = ref<string[]>(['', ''])
const activeLayerIndex = ref(0)
const hasAppliedBackground = ref(false)
const applyRequestId = ref(0)
const globalPoolUrls = ref<string[]>([])
const globalPoolLoading = ref(false)

const CUSTOM_BACKGROUND_PATH = '/data/bg.jpg'

const isEnabled = computed(() => SUPPORTED_ROUTE_NAMES.has(String(route.name || '')))
const pageSpecificUrls = computed(() => ambientBackgroundSource.value?.urls || [])
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

const ensureGlobalPool = async () => {
  if (globalPoolUrls.value.length > 0 || globalPoolLoading.value) return

  globalPoolLoading.value = true
  try {
    const result = await gamesService.getGames({
      query: { page: 1, limit: GLOBAL_POOL_PAGE_LIMIT },
      sort: { field: 'created_at', order: 'desc' },
    })
    const urls = result.data.flatMap((game) => getAmbientBackgroundUrlsFromGameListItem(game))
    globalPoolUrls.value = urls.filter((url, index) => url && urls.indexOf(url) === index)
  } catch {
    // ignore
  } finally {
    globalPoolLoading.value = false
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

const loadBackground = async () => {
  // 如果有页面特定的背景源（game-detail、series-detail、wiki-edit），优先使用
  if (ambientBackgroundSource.value) {
    if (pageSpecificUrls.value.length > 0) {
      return pickRandomBackground(pageSpecificUrls.value, layerUrls.value[activeLayerIndex.value] || '')
    }
    // 有背景源但暂时没有 URL（比如正在加载中），保持当前背景
    return layerUrls.value[activeLayerIndex.value] || ''
  }

  // 无页面特定背景源时，检查自定义背景
  if (canUseCustomBackground.value) {
    return CUSTOM_BACKGROUND_PATH
  }

  await ensureGlobalPool()

  if (globalPoolUrls.value.length > 0) {
    return pickRandomBackground(globalPoolUrls.value, layerUrls.value[activeLayerIndex.value] || '')
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

  if (!nextUrl && currentUrl) {
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
    () => route.name,
    () => ambientBackgroundSource.value?.key || '',
    () => (ambientBackgroundSource.value?.urls || []).join('|'),
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
  transition: opacity 0.85s ease;
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
