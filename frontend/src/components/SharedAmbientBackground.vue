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
import { useRoute } from 'vue-router'
import { getRouteParamString, useNamedRouteGuard } from '@/composables/useNamedRouteGuard'
import gamesService from '@/services/games.service'
import { resolveAssetUrl } from '@/utils/asset-url'

const route = useRoute()
const gameDetailRouteGuard = useNamedRouteGuard(route, 'game-detail')

const SUPPORTED_ROUTE_NAMES = new Set([
  'dashboard',
  'games',
  'games-timeline',
  'game-detail',
  'pending-center',
  'series-library',
  'series-detail',
])

const DEFAULT_BACKGROUND =
  'radial-gradient(circle at top right, rgba(26, 159, 255, 0.12), transparent 40%), linear-gradient(180deg, rgba(15, 18, 25, 0.96), rgba(15, 18, 25, 0.88))'

let cachedBackgroundUrl = ''
let inflightBackgroundRequest: Promise<string> | null = null

const layerUrls = ref<string[]>(['', ''])
const activeLayerIndex = ref(0)
const hasAppliedBackground = ref(false)
const applyRequestId = ref(0)

const LIST_CANDIDATE_LIMIT = 24
const DETAIL_CANDIDATE_LIMIT = 8
const MIN_BACKGROUND_WIDTH = 1920
const MIN_BACKGROUND_HEIGHT = 1080
const CUSTOM_BACKGROUND_PATH = '/data/bg.jpg'

const isEnabled = computed(() => SUPPORTED_ROUTE_NAMES.has(String(route.name || '')))

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

const probeImageSize = (url: string) => {
  return new Promise<{ width: number; height: number } | null>((resolve) => {
    const image = new Image()
    image.onload = () => resolve({ width: image.naturalWidth, height: image.naturalHeight })
    image.onerror = () => resolve(null)
    image.src = url
  })
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

const isQualifiedBackground = (size: { width: number; height: number } | null) => {
  return Boolean(size && size.width >= MIN_BACKGROUND_WIDTH && size.height >= MIN_BACKGROUND_HEIGHT)
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
  const size = await probeImageSize(customBackgroundUrl)
  if (!size) {
    return ''
  }

  return customBackgroundUrl
}

const pickFallbackBackground = async (games: Awaited<ReturnType<typeof gamesService.getGames>>['data']) => {
  const candidates = shuffleArray(games)
    .flatMap((game) => [game.banner_image, game.cover_image])
    .map((asset) => resolveAssetUrl(asset))
    .filter(Boolean)

  for (const candidate of candidates) {
    if (await preloadImage(candidate)) {
      return candidate
    }
  }

  return ''
}

const pickGameDetailBackground = async (gameId: string) => {
  try {
    const detail = await gamesService.getGame(gameId)
    const screenshots = shuffleArray(detail.screenshots || [])

    for (const screenshot of screenshots) {
      const resolvedUrl = resolveAssetUrl(screenshot)
      if (!resolvedUrl) continue

      const size = await probeImageSize(resolvedUrl)
      if (isQualifiedBackground(size)) {
        return resolvedUrl
      }
    }

    return resolveAssetUrl(detail.banner_image || detail.cover_image) || ''
  } catch {
    return ''
  }
}

const pickQualifiedScreenshotBackground = async (
  games: Awaited<ReturnType<typeof gamesService.getGames>>['data'],
) => {
  const candidateGames = shuffleArray(games).slice(0, DETAIL_CANDIDATE_LIMIT)

  for (const game of candidateGames) {
    let detail
    try {
      detail = await gamesService.getGame(String(game.id))
    } catch {
      continue
    }

    const screenshots = shuffleArray(detail.screenshots || [])
    for (const screenshot of screenshots) {
      const resolvedUrl = resolveAssetUrl(screenshot)
      if (!resolvedUrl) continue

      const size = await probeImageSize(resolvedUrl)
      if (isQualifiedBackground(size)) {
        return resolvedUrl
      }
    }
  }

  return ''
}

const pickBackgroundFromGames = async () => {
  const response = await gamesService.getGames({
    page: 1,
    pageSize: LIST_CANDIDATE_LIMIT,
    sort: {
      field: 'updated_at',
      order: 'desc',
    },
  })

  const fallbackPromise = pickFallbackBackground(response.data)
  const qualifiedScreenshotUrl = await pickQualifiedScreenshotBackground(response.data)
  const fallbackUrl = await fallbackPromise
  return qualifiedScreenshotUrl || fallbackUrl
}

const loadBackground = async () => {
  const detailBackground = await gameDetailRouteGuard.runWhenActive(async () => {
    const gameId = getRouteParamString(route, 'id')
    return gameId ? pickGameDetailBackground(gameId) : ''
  })
  if (typeof detailBackground === 'string') {
    return detailBackground
  }

  if (cachedBackgroundUrl) {
    return cachedBackgroundUrl
  }

  if (!inflightBackgroundRequest) {
    inflightBackgroundRequest = loadCustomBackground()
      .then((customBackgroundUrl) => customBackgroundUrl || pickBackgroundFromGames())
      .then((url) => {
        cachedBackgroundUrl = url
        return url
      })
      .catch(() => '')
      .finally(() => {
        inflightBackgroundRequest = null
      })
  }

  return inflightBackgroundRequest
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
  [() => route.fullPath, isEnabled],
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
