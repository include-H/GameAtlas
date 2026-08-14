<script lang="ts">
export interface PickupSource {
  left: number
  top: number
  width: number
  height: number
  rot: number
}
</script>

<template>
  <!-- 拿出来的游戏盒 -->
  <Transition name="inspect">
    <div v-if="game" class="store-inspect" @click.self="putBack()">
      <div
        class="store-inspect__box"
        :class="{ 'store-inspect__box--settled': inspectSettled }"
      >
        <div
          ref="caseRef"
          class="store-inspect__case"
          :class="{ 'store-inspect__case--opening': isOpening }"
          title="开始游戏"
          @click="emit('open-case')"
        >
          <div class="store-inspect__disc">
            <img
              class="store-inspect__disc-art"
              :src="game.coverUrl"
              alt=""
              draggable="false"
            >
            <span class="store-inspect__disc-hole" />
            <span class="store-inspect__disc-shine" />
          </div>
          <div class="store-inspect__cover">
            <img :src="game.coverUrl" :alt="game.title" draggable="false">
            <span class="store-inspect__sheen" />
          </div>
        </div>
        <p
          class="store-inspect__hint"
          :class="{ 'store-inspect__hint--success': hintSuccess }"
        >
          {{ hint }}
        </p>
        <div class="store-inspect__meta">
          <h2>{{ game.title }}</h2>
          <p>
            {{ game.year }}
            <template v-if="game.titleAlt"> · {{ game.titleAlt }}</template>
          </p>
        </div>
        <div class="store-inspect__actions">
          <button type="button" class="store-btn store-btn--ghost" @click="putBack()">放回去</button>
        </div>
      </div>
    </div>
  </Transition>
</template>

<script setup lang="ts">
import { nextTick, onUnmounted, ref, watch } from 'vue'
import type { StoreShelfGame } from '@/composables/useStoreSession'

const props = defineProps<{
  game: StoreShelfGame | null
  pickupSource: PickupSource | null
  isOpening: boolean
  hint: string
  hintSuccess: boolean
}>()

const emit = defineEmits<{
  (e: 'put-back'): void
  (e: 'open-case'): void
  (e: 'dim', dim: boolean): void
}>()

interface PickupOrigin {
  x: number
  y: number
  scale: number
  rot: number
}

const caseRef = ref<HTMLElement | null>(null)
const inspectSettled = ref(false)

let pickupOrigin: PickupOrigin | null = null
let pickupAnimation: Animation | null = null
let pickupSettleTimer: number | null = null
let putBackTimer: number | null = null
let dimTimer: number | null = null

// 抓取动画：从货架上那盒的原始位置/角度/大小飞向眼前
watch(
  () => props.game,
  (game, prevGame) => {
    if (!game || prevGame) return
    if (pickupSettleTimer !== null) {
      window.clearTimeout(pickupSettleTimer)
      pickupSettleTimer = null
    }
    if (dimTimer !== null) {
      window.clearTimeout(dimTimer)
      dimTimer = null
    }
    inspectSettled.value = false
    emit('dim', false)
    const source = props.pickupSource
    if (!source) return

    nextTick(() => {
      const caseElement = caseRef.value
      if (!caseElement || props.game?.publicId !== game.publicId) return
      const caseRect = caseElement.getBoundingClientRect()
      const origin: PickupOrigin = {
        x: source.left + source.width / 2 - (caseRect.left + caseRect.width / 2),
        y: source.top + source.height / 2 - (caseRect.top + caseRect.height / 2),
        scale: source.width / caseRect.width,
        rot: source.rot,
      }
      pickupOrigin = origin
      pickupAnimation = caseElement.animate(
        [
          // 0%：还插在货架上（平面姿态，与货架盒完全对齐，避免衔接突兀）
          {
            transform:
              `translate(${origin.x}px, ${origin.y}px) scale(${origin.scale}) ` +
              `rotate(${origin.rot}deg) rotateY(0deg) rotateX(0deg)`,
            boxShadow: '0 6px 12px rgba(0, 0, 0, 0.32), 0 2px 5px rgba(0, 0, 0, 0.28)',
            easing: 'cubic-bezier(0.25, 0.7, 0.3, 1)',
            offset: 0,
          },
          // 10%：手指捏住，轻微上抬并开始转面
          {
            transform:
              `translate(${origin.x * 0.9}px, ${origin.y * 0.9 - 14}px) ` +
              `scale(${origin.scale * 1.06}) rotate(${origin.rot * 0.85}deg) rotateY(-8deg) rotateX(5deg)`,
            boxShadow: '0 8px 16px rgba(0, 0, 0, 0.36), 0 3px 7px rgba(0, 0, 0, 0.3)',
            easing: 'cubic-bezier(0.2, 0.8, 0.3, 1)',
            offset: 0.1,
          },
          // 42%：弧线最高点，往面前带（角度转得最开）
          {
            transform:
              `translate(${origin.x * 0.3}px, ${origin.y * 0.42 - 46}px) ` +
              `scale(${origin.scale + (1 - origin.scale) * 0.55}) rotate(0deg) rotateY(-14deg) rotateX(9deg)`,
            boxShadow: '0 20px 40px rgba(0, 0, 0, 0.5), 0 8px 16px rgba(0, 0, 0, 0.38)',
            easing: 'cubic-bezier(0.35, 0.05, 0.4, 1)',
            offset: 0.42,
          },
          // 60%：减速滑到终点附近，轻微过冲（角度已固定，只动位移/缩放）
          {
            transform:
              'translate(0px, -12px) scale(1.04) rotate(0deg) rotateY(-2deg) rotateX(1deg)',
            boxShadow: '0 46px 92px rgba(0, 0, 0, 0.76), 0 17px 32px rgba(0, 0, 0, 0.54)',
            easing: 'cubic-bezier(0.25, 0.1, 0.45, 1)',
            offset: 0.6,
          },
          // 74%：回落到正位
          {
            transform:
              'translate(0px, 0px) scale(1) rotate(0deg) rotateY(-2deg) rotateX(1deg)',
            boxShadow: '0 40px 80px rgba(0, 0, 0, 0.72), 0 14px 26px rgba(0, 0, 0, 0.5)',
            easing: 'cubic-bezier(0.35, 0.2, 0.55, 1)',
            offset: 0.74,
          },
          // 87%：极轻的一次回弹，像手停稳（同样保持角度不变）
          {
            transform:
              'translate(0px, -3px) scale(1.008) rotate(0deg) rotateY(-2deg) rotateX(1deg)',
            boxShadow: '0 42px 84px rgba(0, 0, 0, 0.74), 0 15px 28px rgba(0, 0, 0, 0.52)',
            easing: 'cubic-bezier(0.45, 0.05, 0.5, 1)',
            offset: 0.87,
          },
          // 100%：落定
          {
            transform:
              'translate(0px, 0px) scale(1) rotate(0deg) rotateY(-2deg) rotateX(1deg)',
            boxShadow: '0 40px 80px rgba(0, 0, 0, 0.72), 0 14px 26px rgba(0, 0, 0, 0.5)',
            offset: 1,
          },
        ],
        {
          duration: 820,
          fill: 'forwards',
        },
      )
      // 盒子落定后再浮现文字与按钮，避免“信息跟着盒子一起飞”
      dimTimer = window.setTimeout(() => {
        emit('dim', true)
      }, 180)
      pickupSettleTimer = window.setTimeout(() => {
        inspectSettled.value = true
      }, 740)
    })
  },
)

const finishPutBack = () => {
  emit('put-back')
  pickupOrigin = null
  pickupAnimation = null
  inspectSettled.value = false
  emit('dim', false)
}

const putBack = (animate = true) => {
  if (putBackTimer !== null) {
    window.clearTimeout(putBackTimer)
    putBackTimer = null
  }
  if (pickupSettleTimer !== null) {
    window.clearTimeout(pickupSettleTimer)
    pickupSettleTimer = null
  }
  if (dimTimer !== null) {
    window.clearTimeout(dimTimer)
    dimTimer = null
  }
  inspectSettled.value = false
  emit('dim', false)

  // 已开盒或没有抓取起点时直接收起，不播反向动画
  const caseElement = caseRef.value
  if (!animate || props.isOpening || !pickupOrigin || !caseElement) {
    finishPutBack()
    return
  }

  const origin = pickupOrigin
  // 若还在飞行途中，就从盒子当前所处位置开始收回去
  let startTransform =
    'translate(0px, 0px) scale(1) rotate(0deg) rotateY(-2deg) rotateX(1deg)'
  let startShadow = '0 40px 80px rgba(0, 0, 0, 0.72), 0 14px 26px rgba(0, 0, 0, 0.5)'
  if (pickupAnimation && pickupAnimation.playState === 'running') {
    pickupAnimation.cancel()
    const computedStyle = window.getComputedStyle(caseElement)
    startTransform = computedStyle.transform
    startShadow = computedStyle.boxShadow
  }
  const reverse = caseElement.animate(
    [
      {
        // 0%：正位
        transform: startTransform,
        boxShadow: startShadow,
        easing: 'cubic-bezier(0.3, 0.1, 0.45, 1)',
        offset: 0,
      },
      // 28%：临走前轻轻抬一下
      {
        transform:
          'translate(0px, -8px) scale(1.02) rotate(0deg) rotateY(-6deg) rotateX(4deg)',
        boxShadow: '0 36px 72px rgba(0, 0, 0, 0.66), 0 12px 24px rgba(0, 0, 0, 0.46)',
        easing: 'cubic-bezier(0.35, 0.05, 0.4, 1)',
        offset: 0.28,
      },
      // 58%：弧线回去
      {
        transform:
          `translate(${origin.x * 0.3}px, ${origin.y * 0.42 - 40}px) ` +
          `scale(${origin.scale + (1 - origin.scale) * 0.5}) rotate(0deg) rotateY(-12deg) rotateX(8deg)`,
        boxShadow: '0 18px 34px rgba(0, 0, 0, 0.45), 0 6px 12px rgba(0, 0, 0, 0.35)',
        easing: 'cubic-bezier(0.3, 0.1, 0.45, 1)',
        offset: 0.58,
      },
      // 84%：贴近货架，减速并转回平面姿态
      {
        transform:
          `translate(${origin.x * 0.9}px, ${origin.y * 0.9 - 4}px) ` +
          `scale(${origin.scale * 1.04}) rotate(${origin.rot * 0.9}deg) rotateY(-7deg) rotateX(4deg)`,
        boxShadow: '0 9px 18px rgba(0, 0, 0, 0.38), 0 3px 7px rgba(0, 0, 0, 0.3)',
        easing: 'cubic-bezier(0.35, 0.05, 0.45, 1)',
        offset: 0.84,
      },
      // 100%：插回货架（与货架盒完全同姿态，无痕衔接）
      {
        transform:
          `translate(${origin.x}px, ${origin.y}px) scale(${origin.scale}) ` +
          `rotate(${origin.rot}deg) rotateY(0deg) rotateX(0deg)`,
        boxShadow: '0 6px 12px rgba(0, 0, 0, 0.32), 0 2px 5px rgba(0, 0, 0, 0.28)',
        offset: 1,
      },
    ],
    {
      duration: 560,
      fill: 'forwards',
    },
  )
  reverse.onfinish = () => finishPutBack()
  // 兜底：动画被中断时也能收场
  putBackTimer = window.setTimeout(() => {
    if (props.game) finishPutBack()
  }, 900)
}

defineExpose({
  putBack,
})

onUnmounted(() => {
  pickupAnimation?.cancel()
  pickupAnimation = null
  if (putBackTimer !== null) {
    window.clearTimeout(putBackTimer)
    putBackTimer = null
  }
  if (pickupSettleTimer !== null) {
    window.clearTimeout(pickupSettleTimer)
    pickupSettleTimer = null
  }
  if (dimTimer !== null) {
    window.clearTimeout(dimTimer)
    dimTimer = null
  }
})
</script>

<style scoped>
/* ---------- 拿出游戏盒 ---------- */
.store-inspect {
  position: absolute;
  inset: 0;
  z-index: 30;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(8, 5, 3, 0.72);
}

.store-inspect__box {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 20px;
  padding: 18px 26px 26px;
  /* 透视从动画里移到父级，飞行插值不再因透视矩阵产生形变 */
  perspective: 900px;
}

.store-inspect__case {
  position: relative;
  width: 320px;
  /* 与货架游戏盒统一 0.72（宽:高），起飞/落回时与货架盒无痕衔接 */
  aspect-ratio: 0.72;
  border-radius: 4px;
  cursor: pointer;
  perspective: 1100px;
  transform: rotateY(-2deg) rotateX(1deg);
  box-shadow:
    0 40px 80px rgba(0, 0, 0, 0.72),
    0 14px 26px rgba(0, 0, 0, 0.5);
  will-change: transform, box-shadow;
}

.store-inspect__disc {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  background:
    linear-gradient(180deg, #2b211a, #1b1410 78%);
  box-shadow:
    inset 0 0 0 1px rgba(232, 213, 173, 0.32),
    inset 0 0 0 10px rgba(0, 0, 0, 0.3);
}

.store-inspect__disc-art {
  position: absolute;
  left: 50%;
  top: 50%;
  width: 68%;
  aspect-ratio: 1;
  transform: translate(-50%, -50%);
  border-radius: 50%;
  object-fit: cover;
  object-position: center;
  display: block;
  border: 2px solid rgba(255, 255, 255, 0.5);
  box-shadow:
    0 6px 16px rgba(0, 0, 0, 0.55);
}

.store-inspect__disc-hole {
  position: absolute;
  width: 15%;
  aspect-ratio: 1;
  border-radius: 50%;
  background: radial-gradient(circle at 45% 40%, #3a3a40, #141418 70%);
  box-shadow: inset 0 2px 5px rgba(0, 0, 0, 0.9);
}

.store-inspect__disc-shine {
  position: absolute;
  inset: 0;
  border-radius: 4px;
  background:
    linear-gradient(118deg, rgba(255, 255, 255, 0.14) 0%, transparent 32%),
    radial-gradient(circle at 30% 22%, rgba(255, 255, 255, 0.18), transparent 42%);
  pointer-events: none;
}

.store-inspect__cover {
  position: absolute;
  inset: 0;
  background:
    linear-gradient(135deg, rgba(255, 255, 255, 0.1), rgba(255, 255, 255, 0.02) 30%, transparent 55%),
    linear-gradient(180deg, #26201a, #17120e 82%);
  border-radius: 4px;
  box-shadow:
    inset 0 0 0 1px rgba(232, 213, 173, 0.35),
    inset 0 0 0 9px rgba(0, 0, 0, 0.22);
  transform-origin: left center;
  transition: transform 0.6s cubic-bezier(0.22, 0.61, 0.36, 1);
  will-change: transform;
}

.store-inspect__case--opening .store-inspect__cover {
  transform: rotateY(-112deg);
}

.store-inspect__cover img {
  position: absolute;
  inset: 10px;
  width: calc(100% - 20px);
  height: calc(100% - 20px);
  object-fit: cover;
  border-radius: 2px;
  box-shadow:
    inset 0 0 0 1px rgba(255, 255, 255, 0.12),
    0 1px 4px rgba(0, 0, 0, 0.55);
}

/* 盒顶厚度高光 */
.store-inspect__cover::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 3px;
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.24), rgba(255, 255, 255, 0.04));
  border-radius: 4px 4px 0 0;
  pointer-events: none;
}

/* 右侧盒脊：封面边缘的厚度与受光 */
.store-inspect__cover::after {
  content: '';
  position: absolute;
  top: 0;
  right: 0;
  width: 13px;
  height: 100%;
  background: linear-gradient(90deg, rgba(0, 0, 0, 0.18), rgba(255, 255, 255, 0.1) 42%, rgba(0, 0, 0, 0.5));
  border-left: 1px solid rgba(232, 213, 173, 0.28);
  pointer-events: none;
}

.store-inspect__sheen {
  position: absolute;
  inset: 0;
  border-radius: 4px;
  background: linear-gradient(115deg, rgba(255, 255, 255, 0.2), transparent 30%);
  pointer-events: none;
}

.store-inspect__meta {
  text-align: center;
  color: #ffe9c8;
  text-shadow: 0 2px 8px rgba(0, 0, 0, 0.8);
}

.store-inspect__meta h2 {
  margin: 0;
  font-size: 28px;
  letter-spacing: 2px;
}

.store-inspect__meta p {
  margin: 6px 0 0;
  font-size: 15px;
  color: rgba(255, 230, 190, 0.78);
  letter-spacing: 1px;
}

.store-inspect__hint {
  font-size: 12px !important;
  color: rgba(255, 230, 190, 0.52) !important;
  letter-spacing: 1px !important;
}

.store-inspect__hint--error {
  color: rgba(255, 170, 130, 0.92) !important;
}

.store-inspect__hint--success {
  color: rgba(255, 205, 120, 0.95) !important;
}

.store-inspect__actions {
  display: flex;
  gap: 14px;
}

/* 盒子落定后文字与按钮再浮现，避免跟着飞行动画一起“生硬弹入” */
.store-inspect__meta,
.store-inspect__hint,
.store-inspect__actions {
  opacity: 0;
  transform: translateY(14px);
  transition:
    opacity 0.45s ease,
    transform 0.5s cubic-bezier(0.22, 0.61, 0.36, 1);
}

.store-inspect__box--settled .store-inspect__meta,
.store-inspect__box--settled .store-inspect__hint,
.store-inspect__box--settled .store-inspect__actions {
  opacity: 1;
  transform: translateY(0);
}

.store-btn {
  appearance: none;
  border: 0;
  cursor: pointer;
  font-size: 14px;
  letter-spacing: 2px;
  padding: 9px 22px;
  border-radius: 999px;
  transition: transform 0.2s ease, box-shadow 0.2s ease, background 0.2s ease, color 0.2s ease;
}

.store-btn:hover {
  transform: translateY(-1px);
}

.store-btn--ghost {
  background: rgba(255, 230, 190, 0.08);
  color: rgba(255, 230, 190, 0.9);
  border: 1px solid rgba(255, 230, 190, 0.32);
}

.store-btn--ghost:hover {
  background: rgba(255, 230, 190, 0.16);
}

.store-btn--primary {
  background: linear-gradient(180deg, #d99b4e, #a9682a);
  color: #2b180a;
  font-weight: 700;
  box-shadow: 0 8px 18px rgba(0, 0, 0, 0.42);
}

.store-btn--primary:hover {
  box-shadow: 0 12px 24px rgba(0, 0, 0, 0.5);
}

.inspect-enter-active,
.inspect-leave-active {
  transition: opacity 0.28s ease;
}

.inspect-enter-from,
.inspect-leave-to {
  opacity: 0;
}

.inspect-enter-active .store-inspect__case {
  transition: opacity 0.18s ease;
}

.inspect-leave-active .store-inspect__case {
  transition: opacity 0.24s ease;
}

.inspect-enter-from .store-inspect__case,
.inspect-leave-to .store-inspect__case {
  opacity: 0;
}
</style>
