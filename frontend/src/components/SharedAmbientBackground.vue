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
import { getRouteParamString, useNamedRouteGuard } from '@/composables/useNamedRouteGuard'
import gamesService from '@/services/games.service'
import { useUiStore } from '@/stores/ui'
import { resolveAssetUrl } from '@/utils/asset-url'

const route = useRoute()
const uiStore = useUiStore()
const { ambientBackgroundOverride, sharedBackgroundAvailability } = storeToRefs(uiStore)
const gameDetailRouteGuard = useNamedRouteGuard(route, 'game-detail')
const wikiEditRouteGuard = useNamedRouteGuard(route, 'wiki-edit')

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
])

const DEFAULT_BACKGROUND =
  'radial-gradient(circle at 18% 18%, rgba(122, 162, 199, 0.16), transparent 34%), radial-gradient(circle at 82% 12%, rgba(70, 98, 128, 0.18), transparent 28%), linear-gradient(180deg, rgba(10, 14, 21, 0.98), rgba(16, 22, 31, 0.92))'

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

const canUseCustomBackground = computed(() => sharedBackgroundAvailability.value === 'available')

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

  if (canUseCustomBackground.value) {
    return CUSTOM_BACKGROUND_PATH
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
  [
    () => route.fullPath,
    isEnabled,
    () => pendingCenterOverrideUrl.value,
    () => ambientBackgroundOverride.value?.key || '',
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
