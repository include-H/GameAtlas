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
      :style="{
        ...style,
        opacity: isEnabled ? (activeLayerIndex === index ? 0.36 : 0) : 0,
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
import { getRouteParamString, useNamedRouteGuard } from '@/composables/useNamedRouteGuard'
import gamesService from '@/services/games.service'
import { useUiStore } from '@/stores/ui'
import { resolveAssetUrl } from '@/utils/asset-url'

const route = useRoute()
const uiStore = useUiStore()
const { ambientBackgroundOverride } = storeToRefs(uiStore)
const gameDetailRouteGuard = useNamedRouteGuard(route, 'game-detail')
const wikiEditRouteGuard = useNamedRouteGuard(route, 'wiki-edit')

const SUPPORTED_ROUTE_NAMES = new Set([
  'dashboard',
  'games',
  'games-timeline',
  'game-detail',
  'pending-center',
  'series-library',
  'series-detail',
  'wiki-edit',
])

const DEFAULT_BACKGROUND =
  'radial-gradient(circle at top right, rgba(26, 159, 255, 0.12), transparent 40%), linear-gradient(180deg, rgba(15, 18, 25, 0.96), rgba(15, 18, 25, 0.88))'

const layerUrls = ref<string[]>(['', ''])
const activeLayerIndex = ref(0)
const hasAppliedBackground = ref(false)
const applyRequestId = ref(0)

const LIST_CANDIDATE_LIMIT = 24
const CUSTOM_BACKGROUND_PATH = '/data/bg.jpg'

const isEnabled = computed(() => SUPPORTED_ROUTE_NAMES.has(String(route.name || '')))
const pendingCenterOverrideUrl = computed(() => {
  if (route.name !== 'pending-center') {
    return ''
  }

  return resolveAssetUrl(ambientBackgroundOverride.value?.url || '')
})

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
    background: DEFAULT_BACKGROUND,
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

const resolveCustomBackgroundUrl = () => {
  if (!import.meta.env.DEV) {
    return CUSTOM_BACKGROUND_PATH
  }

  const apiBase = import.meta.env.VITE_API_BASE_URL || '/api'
  if (apiBase.startsWith('http://') || apiBase.startsWith('https://')) {
    try {
      const url = new URL(apiBase)
      return `${url.origin}${CUSTOM_BACKGROUND_PATH}`
    } catch {
      return 'http://127.0.0.1:3000/data/bg.jpg'
    }
  }

  if (typeof window !== 'undefined') {
    const protocol = window.location.protocol === 'https:' ? 'https:' : 'http:'
    const host = window.location.hostname || '127.0.0.1'
    const backendPort = import.meta.env.VITE_BACKEND_PORT || '3000'
    return `${protocol}//${host}:${backendPort}${CUSTOM_BACKGROUND_PATH}`
  }

  return 'http://127.0.0.1:3000/data/bg.jpg'
}

const loadCustomBackground = async () => {
  const customBackgroundUrl = resolveCustomBackgroundUrl()
  if (!(await preloadImage(customBackgroundUrl))) {
    return ''
  }

  return customBackgroundUrl
}

const pickRandomFromUrls = async (urls: Array<string | null | undefined>) => {
  for (const url of shuffleArray(urls)) {
    const resolvedUrl = resolveAssetUrl(url)
    if (!resolvedUrl) continue

    if (await preloadImage(resolvedUrl)) {
      return resolvedUrl
    }
  }

  return ''
}

const pickGameScopedBackground = async (gameId: string) => {
  try {
    const detail = await gamesService.getGame(gameId)
    const screenshotUrl = await pickRandomFromUrls(detail.screenshots.map((item) => item.path))
    if (screenshotUrl) {
      return screenshotUrl
    }

    return pickRandomFromUrls([detail.banner_image, detail.cover_image])
  } catch {
    return ''
  }
}

const pickBackgroundFromGames = async () => {
  const response = await gamesService.getGames({
    query: {
      page: 1,
      limit: LIST_CANDIDATE_LIMIT,
    },
    sort: {
      field: 'updated_at',
      order: 'desc',
    },
  })

  for (const game of shuffleArray(response.data)) {
    if (!game.public_id) {
      continue
    }

    const backgroundUrl = await pickGameScopedBackground(game.public_id)
    if (backgroundUrl) {
      return backgroundUrl
    }
  }

  return ''
}

const loadBackground = async () => {
  if (pendingCenterOverrideUrl.value) {
    return pendingCenterOverrideUrl.value
  }

  const detailBackground = await gameDetailRouteGuard.runWhenActive(async () => {
    const gameId = getRouteParamString(route, 'publicId')
    return gameId ? pickGameScopedBackground(gameId) : ''
  })
  if (typeof detailBackground === 'string') {
    return detailBackground
  }

  const wikiBackground = await wikiEditRouteGuard.runWhenActive(async () => {
    const gameId = getRouteParamString(route, 'publicId')
    return gameId ? pickGameScopedBackground(gameId) : ''
  })
  if (typeof wikiBackground === 'string') {
    return wikiBackground
  }

  const customBackgroundUrl = await loadCustomBackground()
  if (customBackgroundUrl) {
    return customBackgroundUrl
  }

  return pickBackgroundFromGames()
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
    if (!currentUrl) {
      layerUrls.value[activeLayerIndex.value] = ''
    }
    return
  }

  const nextLayerIndex = activeLayerIndex.value === 0 ? 1 : 0
  layerUrls.value[nextLayerIndex] = nextUrl

  requestAnimationFrame(() => {
    if (requestId !== applyRequestId.value) {
      return
    }
    activeLayerIndex.value = nextLayerIndex
  })
}

watch(
  [() => route.fullPath, isEnabled, () => pendingCenterOverrideUrl.value, () => ambientBackgroundOverride.value?.key || ''],
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
  filter: saturate(1.06) blur(18px) brightness(1.16);
  transform: scale(1.06);
  transform-origin: center center;
  transition: opacity 0.85s ease;
}

.shared-ambient-bg__overlay {
  width: 100%;
  height: 100%;
  background:
    radial-gradient(circle at 18% 20%, rgba(255, 255, 255, 0.16), transparent 24%),
    radial-gradient(circle at 82% 16%, rgba(255, 255, 255, 0.12), transparent 22%),
    linear-gradient(180deg, rgba(255, 255, 255, 0.035) 0%, rgba(255, 255, 255, 0.03) 42%, rgba(255, 255, 255, 0.055) 100%);
}
</style>
