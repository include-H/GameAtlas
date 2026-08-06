<template>
  <div ref="storeRootRef" class="game-store">
    <div
      ref="stageRef"
      class="store-stage"
      :class="{ 'store-stage--dim': stageDim }"
      :style="{ '--stage-scale': String(stageScale) }"
    >
      <StoreStage :posters="storePosters" />

      <StoreShelf
        :rows="shelfRows"
        :picked-id="pickedGame?.publicId ?? null"
        @pick="onPickGame"
      />

      <CrtTv :playlist="crtPlaylist" />
    </div>

    <!-- 拿出来的游戏盒 -->
    <GameInspect
      ref="inspectRef"
      :game="pickedGame"
      :pickup-source="pickupSource"
      :is-opening="isOpening"
      :hint="launchHint"
      :hint-success="launchHintSuccess"
      @open-case="onOpenCase"
      @put-back="onPutBack"
      @dim="stageDim = $event"
    />

    <!-- 多版本选择：开盒后存在多个可启动版本时弹出 -->
    <LaunchVersionModal
      :visible="launchModalVisible"
      :title="launchTitle"
      :options="launchOptions"
      @select="handleLaunchVersion"
      @cancel="launchModalVisible = false"
    />
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { useLaunchGame } from '@/composables/useLaunchGame'
import { useShelfLayout } from '@/composables/useShelfLayout'
import { useStoreSession, type StoreShelfGame } from '@/composables/useStoreSession'
import { useStoreWaifu } from '@/composables/useStoreWaifu'
import type GameInspect from '@/components/game-store/GameInspect.vue'
import type { PickupSource } from '@/components/game-store/GameInspect.vue'
import { useUiStore } from '@/stores/ui'
import '@/assets/game-store-waifu.css'

const uiStore = useUiStore()

const storeRootRef = ref<HTMLElement | null>(null)
const stageRef = ref<HTMLElement | null>(null)
const inspectRef = ref<InstanceType<typeof GameInspect> | null>(null)

// 编排状态：抓取起点（供 GameInspect 做飞行动画）、舞台变暗、开盒中
const pickedGame = ref<StoreShelfGame | null>(null)
const pickupSource = ref<PickupSource | null>(null)
const stageDim = ref(false)
const isOpening = ref(false)
const stageScale = ref(1)

const session = useStoreSession()
const {
  gameStoreSessionGames,
  storePosters,
  crtPlaylist,
  start: startSession,
  dispose: disposeSession,
} = session
const { shelfRows } = useShelfLayout(gameStoreSessionGames)
const waifu = useStoreWaifu({ stageRef })
const {
  launchModalVisible,
  launchTitle,
  launchOptions,
  launchHint,
  launchHintSuccess,
  handleOpenCase: handleLaunchOpenCase,
  handleLaunchVersion,
  cancelPending,
  resetHint,
  dispose: disposeLaunch,
} = useLaunchGame({
  isOpening,
  requestPutBack: () => inspectRef.value?.putBack(),
})

/**
 * 固定 1280×720 设计稿，按内容区尺寸（框架内可用区域）缩放，
 * 保证任何分辨率下货架/封面/电视的相对位置都不错位。
 */
const DESIGN_WIDTH = 1280
const DESIGN_HEIGHT = 720

const updateStageScale = () => {
  const container = storeRootRef.value
  if (!container) {
    stageScale.value = 1
    return
  }
  stageScale.value = Math.min(
    container.clientWidth / DESIGN_WIDTH,
    container.clientHeight / DESIGN_HEIGHT,
  )
}

let storeResizeObserver: ResizeObserver | null = null

const onPickGame = (game: StoreShelfGame, event: MouseEvent) => {
  const button = event.currentTarget as HTMLElement | null
  const buttonRect = button?.getBoundingClientRect()
  resetHint()
  pickupSource.value = buttonRect
    ? {
        left: buttonRect.left,
        top: buttonRect.top,
        width: buttonRect.width,
        height: buttonRect.height,
        rot: Number.parseFloat(button?.style.getPropertyValue('--rot') ?? '') || 0,
      }
    : null
  pickedGame.value = game
}

const onOpenCase = () => {
  if (!pickedGame.value) return
  handleLaunchOpenCase(pickedGame.value.publicId)
}

// GameInspect 放回动画完成：清掉开盒标记、选中游戏与挂起的启动流程
const onPutBack = () => {
  isOpening.value = false
  pickedGame.value = null
  cancelPending()
}

const handleKeydown = (event: KeyboardEvent) => {
  if (event.key === 'Escape') {
    cancelPending()
    inspectRef.value?.putBack()
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleKeydown)
  updateStageScale()
  // 侧边栏收起/展开不会触发 window.resize，监听容器尺寸变化才能让场景跟随内容区缩放。
  if (storeRootRef.value) {
    storeResizeObserver = new ResizeObserver(() => updateStageScale())
    storeResizeObserver.observe(storeRootRef.value)
  }
  startSession()
  waifu.reset()
  void waifu.init().catch(() => {
    waifu.cleanup()
    if (!waifu.disposed.value) {
      uiStore.addAlert('看板娘加载失败，已停用', 'warning')
    }
  })
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
  storeResizeObserver?.disconnect()
  storeResizeObserver = null
  disposeSession()
  waifu.dispose()
  waifu.cleanup()
  disposeLaunch()
})
</script>

<style scoped>
/* 游戏店是沉浸式品牌页面特例：本页场景色直接在局部样式中定义，不外溢到全局 token。 */
.game-store {
  position: absolute;
  inset: 0;
  overflow: hidden;
  background: #17110d;
  user-select: none;
  -webkit-user-select: none;
  font-family: 'LXGW WenKai GB Screen', 'Microsoft YaHei', 'PingFang SC', sans-serif;
}

/* ---------- 场景透视 ---------- */
.store-stage {
  position: absolute;
  left: 50%;
  top: 50%;
  width: 1280px;
  height: 720px;
  transform: translate(-50%, -50%) scale(var(--stage-scale, 1));
  transform-origin: center;
  perspective: 800px;
}

.store-stage--dim .store-backwall,
.store-stage--dim .store-shelf,
.store-stage--dim .store-crt,
.store-stage--dim .store-counter,
.store-stage--dim .store-floor {
  filter: brightness(0.45) saturate(0.8);
  transition: filter 0.35s ease;
}
</style>
