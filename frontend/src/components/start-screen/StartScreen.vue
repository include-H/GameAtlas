<template>
  <Teleport to="body">
    <Transition name="start-screen-overlay">
      <div
        v-if="visible"
        ref="wrapperRef"
        class="start-screen-wrapper"
        tabindex="-1"
        @keydown.esc="handleClose"
      >
        <div class="start-screen-scrim" @click="handleClose" />

        <div class="start-screen" @wheel.passive="handleWheel">
          <div class="start-screen__header">
            <h1 class="start-screen__heading">开始</h1>
            <p class="start-screen__subtitle">
              {{ games.length > 0 ? `${games.length} 个收藏` : '你的专属游戏磁贴' }}
            </p>
          </div>

          <div v-if="isLoading && games.length === 0" class="start-screen__state">
            <a-spin :size="28" />
            <p>正在铺开你的收藏...</p>
          </div>

          <div v-else-if="hasLoadFailure && games.length === 0" class="start-screen__state">
            <icon-exclamation-circle />
            <p>开始屏幕加载失败</p>
            <a-button type="primary" @click="emit('retry')">重试</a-button>
          </div>

          <div v-else-if="games.length === 0" class="start-screen__state">
            <icon-star />
            <p>还没有收藏的游戏</p>
            <a-button type="primary" @click="handleBrowseGames">去游戏库逛逛</a-button>
          </div>

          <div v-else ref="metroAreaRef" class="start-screen__metro">
            <TransitionGroup
              name="metro-tile"
              tag="div"
              class="start-screen__metro-grid"
              @enter="onTileEnter"
              @leave="onTileLeave"
            >
              <div
                v-for="(game, index) in games"
                :key="game.public_id"
                :data-tile-index="index"
                class="start-screen__tile-slot"
              >
                <MetroTile
                  :game="game"
                  :color-index="index"
                  @select="handleTileSelect"
                  @unpin="handleUnpin"
                />
              </div>
            </TransitionGroup>
          </div>

          <div class="start-screen__desktop-hint" @click="handleClose">
            <icon-desktop />
            <span>回到桌面</span>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { onUnmounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { IconDesktop, IconExclamationCircle, IconStar } from '@arco-design/web-vue/es/icon'
import MetroTile from './MetroTile.vue'
import type { GameListItem } from '@/services/types'

const props = defineProps<{
  visible: boolean
  games: GameListItem[]
  isLoading: boolean
  hasLoadFailure: boolean
}>()

const emit = defineEmits<{
  close: []
  retry: []
  unpin: [publicId: string]
  select: [publicId: string]
}>()

const router = useRouter()
const wrapperRef = ref<HTMLElement | null>(null)
const metroAreaRef = ref<HTMLElement | null>(null)

const handleClose = () => {
  emit('close')
}

const handleTileSelect = (publicId: string) => {
  emit('close')
  router.push({ name: 'game-detail', params: { publicId } })
}

const handleUnpin = (publicId: string) => {
  emit('unpin', publicId)
}

const handleBrowseGames = () => {
  emit('close')
  router.push({ name: 'games' })
}

const handleWheel = (event: WheelEvent) => {
  const area = metroAreaRef.value
  if (!area) return
  area.scrollLeft += event.deltaY
}

const onTileEnter = (el: Element) => {
  const index = Number((el as HTMLElement).dataset.tileIndex)
  if (Number.isNaN(index)) return
  ;(el as HTMLElement).style.transitionDelay = `${Math.min(index, 24) * 35}ms`
}

const onTileLeave = (el: Element) => {
  ;(el as HTMLElement).style.transitionDelay = '0ms'
}

watch(
  () => props.visible,
  (value) => {
    if (typeof document === 'undefined') return
    document.body.style.overflow = value ? 'hidden' : ''
    if (value) {
      window.setTimeout(() => wrapperRef.value?.focus(), 0)
    }
  },
  { immediate: true },
)

onUnmounted(() => {
  if (typeof document !== 'undefined') {
    document.body.style.overflow = ''
  }
})
</script>

<style>
/* 开始屏幕是沉浸式品牌页特例：全屏场景色与动效留在组件内，不外溢到全局 token。 */
.start-screen-wrapper {
  position: fixed;
  inset: 0;
  z-index: 1500;
  outline: none;
}

.start-screen-scrim {
  position: absolute;
  inset: 0;
  background: rgba(8, 10, 16, 0.72);
  backdrop-filter: blur(6px);
}

.start-screen {
  position: relative;
  height: 100%;
  display: flex;
  flex-direction: column;
  padding: 48px 56px 36px;
  color: #fff;
}

.start-screen__header {
  margin-bottom: 28px;
}

.start-screen__heading {
  margin: 0;
  font-size: 34px;
  font-weight: 700;
  letter-spacing: 0.08em;
}

.start-screen__subtitle {
  margin: 6px 0 0;
  font-size: 14px;
  opacity: 0.65;
}

.start-screen__metro {
  flex: 1;
  min-height: 0;
  overflow-x: auto;
  overflow-y: hidden;
  padding: 8px 4px 16px;
  scrollbar-width: thin;
}

.start-screen__metro-grid {
  display: flex;
  flex-wrap: wrap;
  align-content: flex-start;
  gap: 16px;
}

.start-screen__tile-slot {
  transition: opacity 220ms ease, transform 220ms ease;
}

.metro-tile-enter-from,
.metro-tile-leave-to {
  opacity: 0;
  transform: translateY(18px) scale(0.96);
}

.metro-tile-enter-to {
  opacity: 1;
  transform: translateY(0) scale(1);
}

.metro-tile-leave-active {
  position: absolute;
}

.start-screen__state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 14px;
  color: rgba(255, 255, 255, 0.75);
  font-size: 15px;
}

.start-screen__state .arco-icon {
  font-size: 42px;
  opacity: 0.7;
}

.start-screen__state p {
  margin: 0;
}

.start-screen__desktop-hint {
  display: flex;
  align-items: center;
  gap: 8px;
  align-self: flex-end;
  padding: 8px 12px;
  border-radius: 8px;
  cursor: pointer;
  font-size: 13px;
  opacity: 0.55;
  transition: opacity 120ms ease, background 120ms ease;
}

.start-screen__desktop-hint:hover {
  opacity: 1;
  background: rgba(255, 255, 255, 0.08);
}

.start-screen-overlay-enter-active,
.start-screen-overlay-leave-active {
  transition: opacity 220ms ease;
}

.start-screen-overlay-enter-from,
.start-screen-overlay-leave-to {
  opacity: 0;
}

@media (max-width: 768px) {
  .start-screen {
    padding: 28px 20px 24px;
  }

  .start-screen__heading {
    font-size: 26px;
  }

  .start-screen__metro {
    flex-wrap: nowrap;
    overflow-x: auto;
  }

  .start-screen__metro-grid {
    flex-wrap: nowrap;
  }
}
</style>
