<template>
  <div
    class="shared-ambient-bg"
    :class="{ 'is-visible': isEnabled && isVisible }"
    :style="backgroundStyle"
    aria-hidden="true"
  >
    <div class="shared-ambient-bg__overlay" />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import gamesService from '@/services/games.service'
import { resolveAssetUrl } from '@/utils/asset-url'

const route = useRoute()

const SUPPORTED_ROUTE_NAMES = new Set([
  'dashboard',
  'games',
  'pending-center',
  'series-library',
  'series-detail',
])

let cachedBackgroundUrl = ''
let inflightBackgroundRequest: Promise<string> | null = null

const isVisible = ref(false)
const backgroundUrl = ref('')

const LIST_CANDIDATE_LIMIT = 24
const DETAIL_CANDIDATE_LIMIT = 8
const MIN_BACKGROUND_WIDTH = 1920
const MIN_BACKGROUND_HEIGHT = 1080
const CUSTOM_BACKGROUND_PATH = '/data/bg.jpg'

const isEnabled = computed(() => SUPPORTED_ROUTE_NAMES.has(String(route.name || '')))

const backgroundStyle = computed(() => {
  if (backgroundUrl.value) {
    return {
      backgroundImage: `url(${backgroundUrl.value})`,
      backgroundSize: 'cover',
      backgroundPosition: 'center',
    }
  }

  return {
    background:
      'radial-gradient(circle at top right, rgba(26, 159, 255, 0.12), transparent 40%), linear-gradient(180deg, rgba(15, 18, 25, 0.96), rgba(15, 18, 25, 0.88))',
  }
})

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
      return `http://127.0.0.1:3000${CUSTOM_BACKGROUND_PATH}`
    }
  }

  if (typeof window !== 'undefined') {
    const protocol = window.location.protocol === 'https:' ? 'https:' : 'http:'
    const host = window.location.hostname || '127.0.0.1'
    const backendPort = import.meta.env.VITE_BACKEND_PORT || '3000'
    return `${protocol}//${host}:${backendPort}${CUSTOM_BACKGROUND_PATH}`
  }

  return `http://127.0.0.1:3000${CUSTOM_BACKGROUND_PATH}`
}

const loadCustomBackground = async () => {
  const customBackgroundUrl = resolveCustomBackgroundUrl()
  const size = await probeImageSize(customBackgroundUrl)
  if (!size) {
    return ''
  }

  return customBackgroundUrl
}

const pickFallbackBackground = (games: Awaited<ReturnType<typeof gamesService.getGames>>['data']) => {
  return shuffleArray(games)
    .flatMap((game) => [game.banner_image, game.cover_image])
    .map((asset) => resolveAssetUrl(asset))
    .filter(Boolean)[0] || ''
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

  const fallbackUrl = pickFallbackBackground(response.data)
  if (fallbackUrl) {
    backgroundUrl.value = fallbackUrl
  }

  const qualifiedScreenshotUrl = await pickQualifiedScreenshotBackground(response.data)
  return qualifiedScreenshotUrl || fallbackUrl
}

const loadBackground = async () => {
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

const refreshVisibility = async () => {
  if (!isEnabled.value) {
    isVisible.value = false
    return
  }

  const nextBackgroundUrl = await loadBackground()
  if (nextBackgroundUrl) {
    backgroundUrl.value = nextBackgroundUrl
  }

  isVisible.value = false
  requestAnimationFrame(() => {
    requestAnimationFrame(() => {
      isVisible.value = true
    })
  })
}

watch(
  () => route.name,
  async () => {
    await refreshVisibility()
  },
)

onMounted(async () => {
  await refreshVisibility()
})
</script>

<style scoped>
.shared-ambient-bg {
  position: fixed;
  inset: 0;
  z-index: 0;
  pointer-events: none;
  opacity: 0;
  filter: saturate(1.04) blur(0.35px);
  transition: opacity 0.6s ease;
}

.shared-ambient-bg.is-visible {
  opacity: 0.58;
}

.shared-ambient-bg__overlay {
  width: 100%;
  height: 100%;
  background:
    radial-gradient(circle at top right, rgba(26, 159, 255, 0.12), transparent 28%),
    linear-gradient(180deg, rgba(15, 18, 25, 0.04), rgba(15, 18, 25, 0.24) 52%, rgba(15, 18, 25, 0.46));
}
</style>
