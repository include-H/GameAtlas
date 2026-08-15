<template>
  <button
    type="button"
    class="metro-tile"
    :class="[
      `metro-tile--${tile.tile_size}`,
      { 'metro-tile--editing': editing },
      ...(pressDir ? [`metro-tile--press-${pressDir}`] : []),
    ]"
    :style="tileStyle"
    :title="tile.title"
    @pointerdown="onPointerDown"
    @click="handleClick"
  >
  <img
    v-if="canAnimate"
    ref="frontImgEl"
    :src="frontSrc"
    :style="focusStyle"
    :alt="tile.title"
    class="metro-tile__cover"
    draggable="false"
  >
  <img
    v-else-if="imageSrc"
    :src="imageSrc"
    :style="focusStyle"
    :alt="tile.title"
    class="metro-tile__cover"
    draggable="false"
  >
    <span v-else class="metro-tile__fallback">{{ initial }}</span>
    <span class="metro-tile__shade" />
    <span class="metro-tile__label">{{ tile.title }}</span>

    <template v-if="editing">
      <span
        class="metro-tile__action metro-tile__crop"
        role="button"
        tabindex="-1"
        title="选择磁贴图片"
        @click.stop="emit('select-image', tile.game_id)"
      >
        <icon-image />
      </span>
      <span
        class="metro-tile__action metro-tile__resize"
        role="button"
        tabindex="-1"
        :title="resizeHint"
        @click.stop="emit('resize', tile.game_id)"
      >
        <icon-expand />
      </span>
      <span
        class="metro-tile__action metro-tile__remove"
        role="button"
        tabindex="-1"
        title="从开始屏幕移除"
        @click.stop="emit('remove', tile.game_id)"
      >
        <icon-close />
      </span>
    </template>
  </button>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { IconClose, IconExpand, IconImage } from '@arco-design/web-vue/es/icon'
import { withAssetWidth } from '@/utils/asset-url'
import type { StartScreenTile, StartScreenTileSize } from '@/services/types'

// Win8 磁贴按压：按哪里哪里微微下沉——transform-origin 移到按下位置的对角，
// 小角度单轴组合旋转，按下侧位移最大、对角几乎不动。仅按压期间生效。
type PressDirection = 'top-left' | 'top-right' | 'bottom-left' | 'bottom-right'

const props = withDefaults(defineProps<{
  tile: StartScreenTile
  colorIndex: number
  editing: boolean
  /** 是否允许活磁贴动画（开始屏幕按进入次数随机抽取限流）。 */
  animate?: boolean
}>(), {
  animate: true,
})

const emit = defineEmits<{
  select: [publicId: string, rect: DOMRect]
  'select-image': [gameId: number]
  resize: [gameId: number]
  remove: [gameId: number]
}>()

const pressDir = ref<PressDirection | null>(null)
let pressClearTimer: number | null = null

const clearPressNow = () => {
  if (pressClearTimer !== null) {
    clearTimeout(pressClearTimer)
    pressClearTimer = null
  }
  pressDir.value = null
  window.removeEventListener('pointerup', onPointerUpRelease)
  window.removeEventListener('pointercancel', onPointerCancel)
}

// 松开后保持按压姿态 180ms：让 150ms 的倾斜过渡完整走完，避免快速点击时一闪而过。
const onPointerUpRelease = () => {
  window.removeEventListener('pointerup', onPointerUpRelease)
  window.removeEventListener('pointercancel', onPointerCancel)
  if (pressClearTimer !== null) {
    clearTimeout(pressClearTimer)
  }
  pressClearTimer = window.setTimeout(() => {
    pressDir.value = null
    pressClearTimer = null
  }, 180)
}

const onPointerCancel = () => {
  clearPressNow()
}

const onPointerDown = (event: PointerEvent) => {
  if (props.editing) return
  const rect = (event.currentTarget as HTMLElement).getBoundingClientRect()
  const dx = event.clientX - rect.left - rect.width / 2
  const dy = event.clientY - rect.top - rect.height / 2
  const row = dy >= 0 ? 'bottom' : 'top'
  const col = dx >= 0 ? 'right' : 'left'
  pressDir.value = `${row}-${col}` as PressDirection
  window.addEventListener('pointerup', onPointerUpRelease)
  window.addEventListener('pointercancel', onPointerCancel)
}

onUnmounted(clearPressNow)

// Win8 Metro 24 色磁贴配色：开始屏幕是沉浸式品牌页特例，色板留在组件内部不外溢。
const metroColors = [
  '#16a085', '#27ae60', '#2980b9', '#8e44ad',
  '#2c3e50', '#f39c12', '#d35400', '#c0392b',
  '#7f8c8d', '#1abc9c', '#3498db', '#9b59b6',
  '#e74c3c', '#f1c40f', '#e67e22', '#00bcd4',
  '#009688', '#4caf50', '#ff9800', '#795548',
  '#607d8b', '#ff5722', '#673ab7', '#3f51b5',
]

const SIZE_HINTS: Record<StartScreenTileSize, string> = {
  small: '当前：小磁贴',
  wide: '当前：宽磁贴',
  large: '当前：大磁贴',
}

// 磁贴只用选定的原图 + 焦点（object-position），不再有封面/banner 兜底链。
// 磁贴显示尺寸远小于原图：请求 640 宽 WebP 变体（懒生成+永久缓存），
// 避免几十张 1080P 原图同时解码卡顿；全屏展开动画才用原图（StartScreen 侧）。
const TILE_IMAGE_WIDTH = 640
const imageSrc = computed(() => withAssetWidth(props.tile.image_path, TILE_IMAGE_WIDTH))
const focusStyle = computed(() => ({
  objectPosition: `${props.tile.focus_x}% ${props.tile.focus_y}%`,
}))
const initial = computed(() => props.tile.title.trim().charAt(0).toUpperCase() || '?')
const resizeHint = computed(() => `${SIZE_HINTS[props.tile.tile_size]}，点击切换`)
const tileStyle = computed(() => ({
  '--metro-tile-color': metroColors[props.colorIndex % metroColors.length],
}))

const handleClick = (event: MouseEvent) => {
  if (props.editing) return
  const rect = (event.currentTarget as HTMLElement).getBoundingClientRect()
  emit('select', props.tile.public_id, rect)
}

// ---------- 活磁贴（仅 2x4 宽磁贴）----------
// 轮播帧 = image_path（首帧）+ flip_images（追加帧）；每轮随机选择翻面或交叉淡入淡出，
// Win8 宽磁贴节奏：间隔停顿后再动。系统要求减少动效时保持静态。
const prefersReducedMotion = () =>
  typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches

const liveFrames = computed(() => {
  const frames = [props.tile.image_path, ...(props.tile.flip_images ?? [])].filter(Boolean) as string[]
  return frames.map((path) => withAssetWidth(path, TILE_IMAGE_WIDTH))
})
const canAnimate = computed(() =>
  props.tile.tile_size === 'wide'
    && props.animate
    && liveFrames.value.length >= 2
    && !prefersReducedMotion()
    && !props.editing
    && typeof window !== 'undefined',
)

const frontSrc = ref('')
let liveIndex = 0
let liveTimer: number | null = null
let liveAnim: Animation | null = null
let liveFadeAnim: Animation | null = null
const frontImgEl = ref<HTMLImageElement | null>(null)

const LIVE_INTERVAL_MS = 5000
const FLIP_DURATION_MS = 900
const FADE_DURATION_MS = 260

const stopLiveAnimation = () => {
  if (liveTimer !== null) {
    clearInterval(liveTimer)
    liveTimer = null
  }
  liveAnim?.cancel()
  liveAnim = null
  liveFadeAnim?.cancel()
  liveFadeAnim = null
}

const startLiveAnimation = () => {
  if (!canAnimate.value || liveTimer !== null) return
  liveIndex = 0
  frontSrc.value = liveFrames.value[0] ?? ''
  liveTimer = window.setInterval(() => {
    if (document.hidden || !canAnimate.value) return
    const next = (liveIndex + 1) % liveFrames.value.length
    const nextPath = liveFrames.value[next]
    if (Math.random() < 0.5) {
      runFlip(nextPath)
    } else {
      runFade(nextPath)
    }
    liveIndex = next
  }, LIVE_INTERVAL_MS)
}

// 压扁式翻面：scaleX 收到 0（侧面）时换图再展开，无镜像、不依赖 3D 上下文
// （磁贴 overflow:hidden 会压平 preserve-3d，双面结构会两面同显）。
const runFlip = (nextPath: string) => {
  const img = frontImgEl.value
  if (!img || frontSrc.value === nextPath) return
  liveFadeAnim?.cancel()
  liveAnim?.cancel()
  const half = FLIP_DURATION_MS / 2
  liveAnim = img.animate(
    [
      { transform: 'scaleX(1)' },
      { transform: 'scaleX(0)' },
    ],
    { duration: half, easing: 'cubic-bezier(0.4, 0, 1, 1)' },
  )
  liveAnim.onfinish = () => {
    frontSrc.value = nextPath
    liveAnim = img.animate(
      [
        { transform: 'scaleX(0)' },
        { transform: 'scaleX(1)' },
      ],
      { duration: half, easing: 'cubic-bezier(0, 0, 0.2, 1)' },
    )
  }
}

const runFade = (nextPath: string) => {
  const img = frontImgEl.value
  if (!img || frontSrc.value === nextPath) return
  liveAnim?.cancel()
  liveFadeAnim?.cancel()
  const out = img.animate(
    [{ opacity: 1 }, { opacity: 0 }],
    { duration: FADE_DURATION_MS, easing: 'ease-in' },
  )
  out.onfinish = () => {
    frontSrc.value = nextPath
    liveFadeAnim = img.animate(
      [{ opacity: 0 }, { opacity: 1 }],
      { duration: FADE_DURATION_MS, easing: 'ease-out' },
    )
  }
}

watch(liveFrames, () => {
  stopLiveAnimation()
  if (canAnimate.value) {
    startLiveAnimation()
  }
}, { deep: true })

watch(() => props.editing, (editing) => {
  if (editing) {
    stopLiveAnimation()
    return
  }
  startLiveAnimation()
})

onMounted(() => {
  startLiveAnimation()
})

onUnmounted(() => {
  stopLiveAnimation()
})
</script>

<style scoped>
.metro-tile {
  position: relative;
  display: flex;
  align-items: flex-end;
  width: 100%;
  height: 100%;
  padding: 10px;
  border: none;
  border-radius: 6px;
  overflow: hidden;
  cursor: pointer;
  background: var(--metro-tile-color, #2980b9);
  color: #fff;
  font-family: 'LXGW WenKai GB Screen', 'Microsoft YaHei', 'PingFang SC', sans-serif;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.28);
  transition: transform 150ms ease, box-shadow 150ms ease;
}

.metro-tile:hover {
  transform: translateY(-2px) scale(1.02);
  box-shadow: 0 8px 22px rgba(0, 0, 0, 0.4);
}

/* Win8 按压：transform-origin 在按下位置的对角，按下侧局部下沉、对角几乎不动 */
.metro-tile--press-top-left:active {
  transform: perspective(1000px) rotateX(5deg) rotateY(-5deg);
  transform-origin: 100% 100%;
}

.metro-tile--press-top-right:active {
  transform: perspective(1000px) rotateX(5deg) rotateY(5deg);
  transform-origin: 0% 100%;
}

.metro-tile--press-bottom-left:active {
  transform: perspective(1000px) rotateX(-5deg) rotateY(-5deg);
  transform-origin: 100% 0%;
}

.metro-tile--press-bottom-right:active {
  transform: perspective(1000px) rotateX(-5deg) rotateY(5deg);
  transform-origin: 0% 0%;
}

.metro-tile--editing {
  cursor: grab;
}

.metro-tile--editing:active {
  cursor: grabbing;
}

.metro-tile--editing:hover {
  transform: none;
  box-shadow: 0 0 0 2px rgba(255, 255, 255, 0.65);
}

.metro-tile__cover,
.metro-tile__fallback {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.metro-tile__fallback {
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 44px;
  font-weight: 700;
  opacity: 0.55;
}

.metro-tile__shade {
  position: absolute;
  inset: 0;
  background: linear-gradient(180deg, rgba(0, 0, 0, 0) 45%, rgba(0, 0, 0, 0.55) 100%);
  pointer-events: none;
}

.metro-tile__label {
  position: relative;
  max-width: 100%;
  font-size: 15px;
  font-weight: 600;
  line-height: 1.25;
  text-align: left;
  text-shadow: 0 1px 3px rgba(0, 0, 0, 0.6);
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.metro-tile__action {
  position: absolute;
  top: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.42);
  color: rgba(255, 255, 255, 0.9);
}

.metro-tile__action:hover {
  background: rgba(0, 0, 0, 0.68);
}

/* Win10 编辑布局：调整尺寸手柄固定在右上角，裁剪/移除依次左移 */
.metro-tile__resize {
  right: 6px;
}

.metro-tile__crop {
  right: 38px;
}

.metro-tile__remove {
  right: 70px;
}

@media (hover: none) {
  .metro-tile__action {
    background: rgba(0, 0, 0, 0.55);
  }
}
</style>
