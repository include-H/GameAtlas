<template>
  <div class="game-store">
    <button type="button" class="store-exit" title="离开游戏店" @click="leaveStore">
      ← 离开
    </button>

    <div
      class="store-stage"
      :class="{ 'store-stage--dim': pickedGame }"
      :style="{ '--stage-scale': String(stageScale) }"
    >
      <!-- 后墙 -->
      <div class="store-backwall">
        <div class="store-wall__paper" />
        <div class="store-wall__trim" />
        <div class="store-poster store-poster--left">
          <img
            v-if="storePosters[0]"
            class="store-poster__img"
            :src="storePosters[0]"
            alt="新到货"
            decoding="async"
          >
          <span class="store-poster__kicker">NEW</span>
          <span class="store-poster__line">新到货</span>
        </div>
        <div class="store-poster store-poster--right">
          <img
            v-if="storePosters[1]"
            class="store-poster__img"
            :src="storePosters[1]"
            alt="畅销榜"
            decoding="async"
          >
          <span class="store-poster__kicker">BEST</span>
          <span class="store-poster__line">畅销榜</span>
        </div>
        <div class="store-sign">GAME&nbsp;STORE</div>
        <div class="store-sign__cord" />
      </div>

      <!-- 地面 -->
      <div class="store-floor">
        <div class="store-floor__rug" />
      </div>

      <!-- 后方主货架 -->
      <div class="store-shelf">
        <div class="store-shelf__crown" />
        <div class="store-shelf__side store-shelf__side--left" />
        <div class="store-shelf__side store-shelf__side--right" />

        <div class="store-shelf__rows">
          <div v-for="(row, rowIndex) in shelfRows" :key="rowIndex" class="store-shelf__row">
            <div class="store-shelf__row-boxes">
              <button
                v-for="cell in row"
                :key="cell.game.publicId"
                type="button"
                class="game-box"
                :class="{ 'game-box--hovered': hoveredId === cell.game.publicId }"
                :style="boxStyle(cell)"
                :title="cell.game.title"
                @mouseenter="hoveredId = cell.game.publicId"
                @mouseleave="hoveredId = null"
                @click="pickGame(cell.game)"
              >
                <img
                  class="game-box__cover"
                  :src="cell.game.coverUrl"
                  :alt="cell.game.title"
                  decoding="async"
                  draggable="false"
                >
                <span class="game-box__sheen" />
              </button>
            </div>
            <div class="store-shelf__plank">
              <div class="store-shelf__plank-shadow" />
            </div>
          </div>
        </div>

        <div class="store-shelf__base" />
      </div>

      <!-- CRT 电视 -->
      <div class="store-crt" :class="{ 'store-crt--off': !crtPowered }">
        <div class="store-crt__cabinet">
          <div class="store-crt__screen">
            <video
              ref="crtVideoRef"
              class="store-crt__video"
              :src="crtVideoUrl"
              autoplay
              muted
              loop
              playsinline
              preload="auto"
            />
            <div class="store-crt__glass" />
          </div>
          <div class="store-crt__brand">GAMEATRON</div>
          <div class="store-crt__controls">
            <button
              type="button"
              class="store-crt__knob"
              :class="{ 'is-off': !crtPowered }"
              :title="crtPowered ? '关闭电视' : '打开电视'"
              @click="toggleCrtPower"
            />
            <button
              type="button"
              class="store-crt__knob store-crt__knob--small"
              :class="{ 'is-paused': crtPaused }"
              :title="crtPaused ? '播放' : '暂停'"
              @click="toggleCrtPause"
            />
            <span class="store-crt__led" :class="{ 'is-off': !crtPowered }" />
          </div>
          <div class="store-crt__vents" />
        </div>
        <div class="store-crt__stand" />
        <div class="store-crt__cable" />
      </div>

      <!-- 前台 / 游戏吧 -->
      <div class="store-counter">
        <div class="store-counter__top">
          <div class="store-counter__top-glow" />
        </div>
        <div class="store-counter__front">
          <div class="store-counter__trim" />
        </div>
      </div>

      <!-- 灯光 / 氛围 -->
      <div class="store-light" />
      <div class="store-vignette" />
    </div>

    <!-- 拿出来的游戏盒 -->
    <Transition name="inspect">
      <div v-if="pickedGame" class="store-inspect" @click.self="putBack">
        <div class="store-inspect__box">
          <div
            class="store-inspect__case"
            :class="{ 'store-inspect__case--opening': isOpening }"
            title="打开游戏盒"
            @click="handleOpenCase"
          >
            <div class="store-inspect__disc">
              <img
                class="store-inspect__disc-art"
                :src="pickedGame.coverUrl"
                alt=""
                draggable="false"
              >
              <span class="store-inspect__disc-hole" />
              <span class="store-inspect__disc-shine" />
            </div>
            <div class="store-inspect__cover">
              <img :src="pickedGame.coverUrl" :alt="pickedGame.title" draggable="false">
              <span class="store-inspect__sheen" />
            </div>
          </div>
          <div class="store-inspect__meta">
            <h2>{{ pickedGame.title }}</h2>
            <p>{{ pickedGame.year }} · {{ pickedGame.platform }}</p>
            <p class="store-inspect__hint">点击游戏盒打开并查看详情</p>
            <p v-if="missingDetailNotice" class="store-inspect__hint store-inspect__hint--error">
              本地库暂无这个游戏（当前为演示封面），暂不能查看详情
            </p>
          </div>
          <div class="store-inspect__actions">
            <button type="button" class="store-btn store-btn--ghost" @click="putBack">放回去</button>
          </div>
        </div>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  gameStoreCrtMock,
  gameStoreSessionGames,
  type GameStoreMockGame,
} from './game-store/mock-session'
import gamesService from '@/services/games.service'
import {
  getAmbientBackgroundPoolFromGames,
  mergeAmbientBackgroundPools,
  type AmbientBackgroundPool,
} from '@/utils/ambient-background'

interface ShelfCell {
  game: GameStoreMockGame
  dx: number
  dy: number
  rot: number
  z: number
}

const router = useRouter()
const hoveredId = ref<string | null>(null)
const pickedGame = ref<GameStoreMockGame | null>(null)
const isOpening = ref(false)
const crtVideoUrl = gameStoreCrtMock.videoUrl
const crtPowered = ref(true)
const crtPaused = ref(false)
const crtVideoRef = ref<HTMLVideoElement | null>(null)
const stageScale = ref(1)
const storePosters = ref<string[]>([])
const knownGameIds = ref<Set<string>>(new Set())
const missingDetailNotice = ref(false)

/**
 * 固定 1280×720 设计稿，按窗口尺寸等比缩放整个场景，
 * 保证任何分辨率下货架/封面/电视的相对位置都不错位。
 */
const DESIGN_WIDTH = 1280
const DESIGN_HEIGHT = 720

const updateStageScale = () => {
  stageScale.value = Math.min(
    window.innerWidth / DESIGN_WIDTH,
    window.innerHeight / DESIGN_HEIGHT,
  )
}

const pickPosterImages = (pool: AmbientBackgroundPool): string[] => {
  const picks: string[] = []
  const add = (url: string | undefined | null) => {
    if (url && !picks.includes(url)) picks.push(url)
  }
  add(pool.screenshots[Math.floor(Math.random() * pool.screenshots.length)])
  add(pool.banners[Math.floor(Math.random() * pool.banners.length)])
  if (picks.length < 2) {
    const rest = [...pool.screenshots, ...pool.banners].filter((url) => !picks.includes(url))
    if (rest.length > 0) {
      add(rest[Math.floor(Math.random() * rest.length)])
    }
  }
  return picks
}

const loadStorePosters = async () => {
  try {
    // 与全局抽卡池同一数据源：遍历游戏列表收集 banner 与截图
    const pools: AmbientBackgroundPool[] = []
    let page = 1
    while (true) {
      const result = await gamesService.getGames({
        query: { page, limit: 100 },
        sort: { field: 'created_at', order: 'desc' },
      })
      for (const game of result.data) {
        knownGameIds.value.add(game.public_id)
      }
      pools.push(getAmbientBackgroundPoolFromGames(result.data))
      const totalPages = Math.max(1, result.pagination.totalPages || 1)
      if (page >= totalPages) break
      page += 1
    }
    storePosters.value = pickPosterImages(mergeAmbientBackgroundPools(pools))
  } catch {
    // 拉取失败时保留纯色海报兜底
  }
}

interface Live2DWidgetConfig {
  waifuPath: string
  cdnPath: string
  cubism2Path: string
  cubism5Path: string
  tools: string[]
  drag: boolean
  logLevel: string
  modelId: number
  showToggleAfterQuit: boolean
}

declare global {
  interface Window {
    initWidget?: (config: Live2DWidgetConfig) => void
    __waifuManager?: {
      cubism2model?: {
        gl?: unknown
        modelScaling: (factor: number) => void
        viewMatrix?: {
          getScaleX?: () => number
        }
      }
    }
    __waifuTools?: {
      el: () => HTMLElement | null
      move: (left: number, bottom: number) => void
      zoom: (factor: number) => void
      report: () => Record<string, number | null>
    }
  }
}

let waifuStyleTag: HTMLLinkElement | null = null
let waifuScriptTag: HTMLScriptElement | null = null
let waifuZoomTimer: number | null = null
let openCaseTimer: number | null = null

const WAIFU_TARGET_ZOOM = 1.35

const loadWaifuResource = (url: string, type: 'css' | 'js'): Promise<void> => {
  return new Promise((resolve, reject) => {
    if (type === 'css') {
      const link = document.createElement('link')
      link.rel = 'stylesheet'
      link.href = url
      link.onload = () => resolve()
      link.onerror = () => reject(new Error(`看板娘样式加载失败：${url}`))
      document.head.appendChild(link)
      waifuStyleTag = link
      return
    }
    const script = document.createElement('script')
    script.type = 'module'
    script.src = url
    script.onload = () => resolve()
    script.onerror = () => reject(new Error(`看板娘脚本加载失败：${url}`))
    document.head.appendChild(script)
    waifuScriptTag = script
  })
}

const waitForElement = (selector: string, timeoutMs = 8000): Promise<HTMLElement | null> => {
  const existing = document.querySelector<HTMLElement>(selector)
  if (existing) return Promise.resolve(existing)
  return new Promise((resolve) => {
    const startedAt = Date.now()
    const timer = window.setInterval(() => {
      const element = document.querySelector<HTMLElement>(selector)
      if (element) {
        window.clearInterval(timer)
        resolve(element)
        return
      }
      if (Date.now() - startedAt > timeoutMs) {
        window.clearInterval(timer)
        resolve(null)
      }
    }, 100)
  })
}

const initWaifu = async () => {
  // 热更新或重复挂载时先清掉旧的看板娘节点
  document.getElementById('waifu')?.remove()
  document.getElementById('waifu-toggle')?.remove()

  await loadWaifuResource('/live2d-widget/waifu.css', 'css')
  if (!window.initWidget) {
    await loadWaifuResource('/live2d-widget/waifu-tips.js', 'js')
  }

  // 清掉旧 manager，避免上一次会话的缩放状态被误判为“已生效”
  window.__waifuManager = undefined

  window.initWidget?.({
    waifuPath: '/live2d-config/waifu-tips.json',
    cdnPath: '/live2d-models/',
    cubism2Path: '/live2d-widget/live2d.min.js',
    cubism5Path: '',
    tools: [],
    drag: false,
    logLevel: 'warn',
    modelId: 0,
    showToggleAfterQuit: false,
  })

  // 把看板娘挂到场景内部，跟随 1280×720 设计稿一起缩放定位
  const stageElement = document.querySelector<HTMLElement>('.store-stage')
  const waifuElement = await waitForElement('#waifu')
  if (stageElement && waifuElement) {
    stageElement.appendChild(waifuElement)
  }

  // 模型加载完成后把人物缩放到目标值（鼠标滚轮的缩放不会自动保存）
  const applyTargetZoom = () => {
    const model = window.__waifuManager?.cubism2model
    // 必须等模型初始化完成（gl 就绪、viewMatrix 存在）再设置缩放
    if (!model || !model.gl || !model.viewMatrix) return false
    const current = model.viewMatrix?.getScaleX?.() ?? 1
    if (Math.abs(current - WAIFU_TARGET_ZOOM) > 0.01) {
      model.modelScaling(WAIFU_TARGET_ZOOM / current)
    }
    return true
  }
  if (!applyTargetZoom()) {
    const startedAt = Date.now()
    waifuZoomTimer = window.setInterval(() => {
      if (applyTargetZoom() || Date.now() - startedAt > 15000) {
        if (waifuZoomTimer !== null) {
          window.clearInterval(waifuZoomTimer)
          waifuZoomTimer = null
        }
      }
    }, 200)
  }
}

const cleanupWaifu = () => {
  if (waifuZoomTimer !== null) {
    window.clearInterval(waifuZoomTimer)
    waifuZoomTimer = null
  }
  document.getElementById('waifu')?.remove()
  document.getElementById('waifu-toggle')?.remove()
  waifuScriptTag?.remove()
  waifuStyleTag?.remove()
  waifuScriptTag = null
  waifuStyleTag = null
}

/**
 * 确定性伪随机摆放：每次进入页面结果一致，
 * 但每盒都有少量位置偏移与角度变化，避免“像素级整齐”。
 */
const shelfRows = computed<ShelfCell[][]>(() => {
  const rows: ShelfCell[][] = []
  gameStoreSessionGames.forEach((game, index) => {
    const rowIndex = Math.floor(index / 5)
    if (!rows[rowIndex]) rows[rowIndex] = []
    // 上层货架在前：row0 最前，row3 最后，避免下层盒子顶部穿到上层时遮挡关系错误
    const rowZ = [40, 30, 20, 10][rowIndex] ?? 10
    rows[rowIndex].push({
      game,
      dx: ((index * 37 + 11) % 5) - 2,
      dy: 0,
      rot: (((index * 29 + 7) % 7) - 3) / 12,
      z: rowZ + ((index * 7) % 5),
    })
  })
  return rows
})

const boxStyle = (cell: ShelfCell) => ({
  '--dx': `${cell.dx}px`,
  '--dy': `${cell.dy}px`,
  '--rot': `${cell.rot}deg`,
  '--box-z': String(cell.z),
})

const pickGame = (game: GameStoreMockGame) => {
  hoveredId.value = null
  missingDetailNotice.value = false
  pickedGame.value = game
}

const toggleCrtPower = () => {
  crtPowered.value = !crtPowered.value
  const video = crtVideoRef.value
  if (!video) return
  if (crtPowered.value) {
    void video.play().catch(() => {})
  } else {
    video.pause()
  }
}

const toggleCrtPause = () => {
  if (!crtPowered.value) return
  crtPaused.value = !crtPaused.value
  const video = crtVideoRef.value
  if (!video) return
  if (crtPaused.value) {
    video.pause()
  } else {
    void video.play().catch(() => {})
  }
}

const putBack = () => {
  if (openCaseTimer !== null) {
    window.clearTimeout(openCaseTimer)
    openCaseTimer = null
  }
  isOpening.value = false
  pickedGame.value = null
}

const handleOpenCase = () => {
  if (isOpening.value || !pickedGame.value) return
  if (!knownGameIds.value.has(pickedGame.value.publicId)) {
    missingDetailNotice.value = true
    return
  }
  isOpening.value = true
  const publicId = pickedGame.value.publicId
  openCaseTimer = window.setTimeout(() => {
    putBack()
    router.push(`/games/${publicId}`)
  }, 750)
}

const leaveStore = () => {
  router.push({ name: 'games' })
}

const handleKeydown = (event: KeyboardEvent) => {
  if (event.key === 'Escape') putBack()
}

onMounted(() => {
  window.addEventListener('keydown', handleKeydown)
  window.addEventListener('resize', updateStageScale)
  updateStageScale()
  void initWaifu()
  void loadStorePosters()

  // 调试工具：移动/缩放/报告看板娘位置（1280×720 设计稿坐标）
  window.__waifuTools = {
    el: () => document.getElementById('waifu'),
    move: (left, bottom) => {
      const element = document.getElementById('waifu')
      if (!element) return
      element.style.setProperty('left', `${left}px`, 'important')
      element.style.setProperty('bottom', `${bottom}px`, 'important')
    },
    zoom: (factor) => {
      window.__waifuManager?.cubism2model?.modelScaling(factor)
    },
    report: () => {
      const element = document.getElementById('waifu')
      const rect = element?.getBoundingClientRect()
      const stage = document.querySelector<HTMLElement>('.store-stage')
      const scale = stage ? new DOMMatrixReadOnly(getComputedStyle(stage).transform).a : 1
      const model = window.__waifuManager?.cubism2model
      const info = {
        designLeft: rect ? (rect.left - (window.innerWidth - 1280 * scale) / 2) / scale : null,
        designBottom: rect
          ? (window.innerHeight - rect.bottom - (window.innerHeight - 720 * scale) / 2) / scale
          : null,
        width: rect?.width ?? null,
        height: rect?.height ?? null,
        viewScale: model?.viewMatrix?.getScaleX?.() ?? null,
      }
      console.log(JSON.stringify(info, null, 2))
      return info
    },
  }
})
onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
  window.removeEventListener('resize', updateStageScale)
  if (openCaseTimer !== null) {
    window.clearTimeout(openCaseTimer)
    openCaseTimer = null
  }
  cleanupWaifu()
})
</script>

<style scoped>
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

/* ---------- 后墙 ---------- */
.store-backwall {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 480px;
  z-index: 1;
  background:
    linear-gradient(180deg, rgba(58, 44, 34, 0.55), rgba(46, 35, 27, 0) 30%),
    repeating-linear-gradient(90deg, rgba(24, 18, 14, 0.12) 0 0.67px, transparent 0.67px 80px),
    linear-gradient(180deg, #b3986f 0%, #9a7f5b 55%, #846b4b 100%);
  box-shadow: inset 0 -12px 20px rgba(0, 0, 0, 0.35);
}

.store-wall__paper {
  position: absolute;
  inset: 0;
  opacity: 0.18;
  background:
    radial-gradient(ellipse at 50% 42%, rgba(255, 226, 170, 0.28), transparent 58%),
    repeating-linear-gradient(0deg, rgba(255, 240, 210, 0.05) 0 1.33px, transparent 1.33px 6px);
}

.store-wall__trim {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  height: 9.33px;
  background: linear-gradient(180deg, #5d4a36, #3f3024);
  box-shadow: 0 2.67px 6.67px rgba(0, 0, 0, 0.45);
}

/* ---------- 海报 ---------- */
.store-poster {
  position: absolute;
  top: 64px;
  width: 200px;
  aspect-ratio: 16 / 9;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 3.33px;
  border-radius: 2.67px;
  overflow: hidden;
  border: 2px solid #e8d5ad;
  box-shadow: 0 6.67px 16px rgba(0, 0, 0, 0.42);
  background:
    linear-gradient(180deg, rgba(30, 22, 16, 0.72), rgba(18, 13, 9, 0.82)),
    #3a2a1c;
}

.store-poster--left {
  left: 60px;
  transform: rotate(-0.8deg);
}

.store-poster--right {
  right: 60px;
  transform: rotate(0.8deg);
}

.store-poster__img {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.store-poster__kicker,
.store-poster__line {
  position: relative;
  z-index: 1;
  text-shadow: 0 1.33px 5.33px rgba(0, 0, 0, 0.9);
}

.store-poster__kicker {
  font-size: 10px;
  letter-spacing: 2px;
  color: #f7e9c8;
  font-weight: 700;
}

.store-poster__line {
  font-size: 13.33px;
  color: #ffe9b8;
  letter-spacing: 2.67px;
}

/* ---------- 霓虹招牌 ---------- */
.store-sign {
  position: absolute;
  top: 9.33px;
  left: 50%;
  transform: translateX(-50%);
  font-size: 24px;
  font-weight: 700;
  letter-spacing: 5.33px;
  color: #ffd9a0;
  text-shadow:
    0 0 4px rgba(255, 190, 110, 0.95),
    0 0 12px rgba(255, 160, 70, 0.75),
    0 0 28px rgba(255, 130, 40, 0.55);
  background: rgba(30, 20, 14, 0.35);
  padding: 3.33px 14.67px 4.67px;
  border-radius: 2.67px;
}

.store-sign__cord {
  position: absolute;
  top: 100%;
  left: 50%;
  width: 1.33px;
  height: 17.33px;
  background: rgba(20, 14, 10, 0.8);
}

/* ---------- 地面 ---------- */
.store-floor {
  position: absolute;
  left: -4%;
  right: -4%;
  top: 413.33px;
  bottom: 200px;
  z-index: 1;
  transform: perspective(466.67px) rotateX(32deg);
  transform-origin: top center;
  background:
    repeating-linear-gradient(0deg, rgba(0, 0, 0, 0.22) 0 1.33px, transparent 1.33px 28px),
    linear-gradient(180deg, #4a3525, #2b1d13 90%);
  box-shadow: inset 0 12px 22.67px rgba(0, 0, 0, 0.5);
}

.store-floor__rug {
  position: absolute;
  left: 50%;
  bottom: 6%;
  width: 46%;
  height: 46%;
  transform: translateX(-50%);
  background: radial-gradient(ellipse at 50% 30%, rgba(130, 52, 42, 0.55), rgba(70, 28, 24, 0.18) 70%, transparent 72%);
}

/* ---------- 主货架 ---------- */
.store-shelf {
  position: absolute;
  top: 50.67px;
  left: 50%;
  transform: translateX(-50%);
  width: 600px;
  height: 480px;
  z-index: 2;
  background: linear-gradient(180deg, #6d5138, #543c28 18%, #462f1f 100%);
  border: 6.67px solid #3d2b1d;
  border-radius: 4px;
  box-shadow:
    0 16px 29.33px rgba(0, 0, 0, 0.6),
    inset 0 1.33px 0 rgba(255, 230, 190, 0.18),
    inset 0 -1.33px 0 rgba(0, 0, 0, 0.5);
}

.store-shelf__crown {
  position: absolute;
  top: -6.67px;
  left: -6.67px;
  right: -6.67px;
  height: 17.33px;
  background: linear-gradient(180deg, #7d5d3e, #543c28);
  border-radius: 4px 4px 0 0;
  box-shadow: 0 2px 5.33px rgba(0, 0, 0, 0.35);
}

.store-shelf__side {
  position: absolute;
  top: 10.67px;
  bottom: 10.67px;
  width: 12px;
  background: linear-gradient(90deg, #4a3423, #2c1f14);
  box-shadow: inset 0 0 5.33px rgba(0, 0, 0, 0.55);
}

.store-shelf__side--left {
  left: -6.67px;
  border-radius: 2.67px 0 0 2.67px;
}

.store-shelf__side--right {
  right: -6.67px;
  border-radius: 0 2.67px 2.67px 0;
}

.store-shelf__rows {
  position: absolute;
  inset: 10.67px 12px 6.67px;
  display: flex;
  flex-direction: column;
}

.store-shelf__row {
  position: relative;
  flex: 1;
  min-height: 0;
}

.store-shelf__row-boxes {
  position: absolute;
  inset: 0 0 9.33px;
  display: flex;
  justify-content: center;
  align-items: flex-end;
}

.store-shelf__plank {
  position: absolute;
  left: -1.33px;
  right: -1.33px;
  bottom: 0;
  height: 9.33px;
  background:
    linear-gradient(180deg, #8a6a47 0%, #6b4f33 55%, #4e3824 100%);
  border-radius: 1.33px;
  box-shadow:
    0 1.33px 3.33px rgba(0, 0, 0, 0.45),
    inset 0 0.67px 0 rgba(255, 230, 190, 0.22);
  z-index: 5;
}

.store-shelf__plank-shadow {
  position: absolute;
  top: 100%;
  left: 0;
  right: 0;
  height: 12px;
  background: linear-gradient(180deg, rgba(0, 0, 0, 0.5), transparent);
}

.store-shelf__base {
  position: absolute;
  left: -6.67px;
  right: -6.67px;
  bottom: -12px;
  height: 14.67px;
  background: linear-gradient(180deg, #543c28, #352516 70%);
  border-radius: 0 0 3.33px 3.33px;
  box-shadow: 0 6.67px 12px rgba(0, 0, 0, 0.55);
}

/* ---------- 游戏盒 ---------- */
.game-box {
  appearance: none;
  border: 0;
  padding: 0;
  margin: 0 14.67px;
  position: relative;
  /* 封面源图统一为 600×900（2:3），盒子按 2:3 显示，避免 cover 裁切 */
  width: 68px;
  height: 102px;
  aspect-ratio: 2 / 3;
  background: transparent;
  cursor: pointer;
  outline: none;
  overflow: hidden;
  transform:
    translate(var(--dx), var(--dy))
    rotate(var(--rot));
  transform-style: preserve-3d;
  z-index: var(--box-z, 10);
  transition:
    transform 0.3s cubic-bezier(0.22, 0.61, 0.36, 1),
    box-shadow 0.3s ease,
    filter 0.3s ease;
  filter: brightness(0.94);
}

.game-box__cover {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
  background: #241a12;
  box-shadow:
    inset 0 0 0 0.67px rgba(255, 255, 255, 0.08),
    0 3.33px 6px rgba(0, 0, 0, 0.42);
}

/* 盒脊：右侧厚度 */
.game-box::after {
  content: '';
  position: absolute;
  top: 0;
  right: 0;
  width: 2.67px;
  height: 100%;
  background: linear-gradient(90deg, rgba(255, 255, 255, 0.24), rgba(0, 0, 0, 0.5));
  border-radius: 0 1.33px 1.33px 0;
  z-index: 2;
}

/* 盒顶厚度 */
.game-box::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 2px;
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.3), rgba(0, 0, 0, 0.2));
  border-radius: 1.33px 1.33px 0 0;
  z-index: 1;
}

.game-box__sheen {
  position: absolute;
  inset: 0;
  z-index: 2;
  pointer-events: none;
  background:
    linear-gradient(115deg, rgba(255, 255, 255, 0.16) 0%, transparent 26%),
    linear-gradient(0deg, rgba(0, 0, 0, 0.28), transparent 34%);
}

.game-box--hovered,
.game-box:hover {
  z-index: 60 !important;
  transform:
    translate(calc(var(--dx) - 5.33px), calc(var(--dy) - 4.67px))
    rotate(var(--rot))
    scale(1.08);
  box-shadow: 0 14.67px 22.67px rgba(0, 0, 0, 0.58);
  filter: brightness(1.08);
}

.game-box--hovered .game-box__cover,
.game-box:hover .game-box__cover {
  box-shadow:
    inset 0 0 0 0.67px rgba(255, 255, 255, 0.14),
    0 9.33px 16px rgba(0, 0, 0, 0.55);
}

/* ---------- CRT 电视 ---------- */
.store-crt {
  position: absolute;
  right: 120px;
  bottom: 200px;
  width: 280px;
  z-index: 3;
  transform: none;
}

.store-crt__cabinet {
  position: relative;
  padding: 10.67px 10.67px 8px;
  border-radius: 12px 12px 8px 8px;
  background:
    linear-gradient(180deg, #d8c9a8 0%, #b8a37e 34%, #8f7a5b 100%);
  border: 1.33px solid #5d4a32;
  box-shadow:
    0 12px 20px rgba(0, 0, 0, 0.55),
    inset 0 1.33px 0 rgba(255, 245, 220, 0.5),
    inset 0 -4px 9.33px rgba(0, 0, 0, 0.28);
}

.store-crt__screen {
  position: relative;
  aspect-ratio: 16 / 9;
  border-radius: 6.67px;
  overflow: hidden;
  background:
    radial-gradient(ellipse at 42% 36%, rgba(90, 120, 96, 0.5), rgba(12, 22, 16, 0.95) 72%),
    #08140c;
  border: 2.67px solid #2c2116;
  box-shadow:
    inset 0 0 17.33px rgba(0, 0, 0, 0.9),
    0 0 14.67px rgba(190, 235, 190, 0.16),
    0 0 40px rgba(160, 220, 160, 0.08);
  transition: background 0.4s ease, box-shadow 0.4s ease;
}

.store-crt__video {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
  background: #08140c;
  transition: opacity 0.4s ease;
}

.store-crt__glass {
  position: absolute;
  inset: 0;
  background:
    linear-gradient(118deg, rgba(255, 255, 255, 0.13) 0%, transparent 24%),
    linear-gradient(0deg, rgba(255, 255, 255, 0.05), transparent 40%);
  box-shadow: inset 0 0 26.67px rgba(255, 255, 255, 0.05);
  animation: crt-flicker 7s ease-in-out infinite;
}

.store-crt--off .store-crt__screen {
  background: radial-gradient(ellipse at 42% 36%, rgba(86, 86, 86, 0.26), #050505 78%), #050505;
  box-shadow: inset 0 0 20px rgba(0, 0, 0, 0.95);
}

.store-crt--off .store-crt__video {
  opacity: 0;
}

.store-crt--off .store-crt__glass {
  opacity: 0.25;
  animation: none;
}

.store-crt__brand {
  margin-top: 5.33px;
  text-align: center;
  font-size: 6px;
  letter-spacing: 2.67px;
  color: #4a3824;
  font-weight: 700;
}

.store-crt__controls {
  position: absolute;
  right: 12px;
  bottom: 13.33px;
  display: flex;
  align-items: center;
  gap: 4px;
}

.store-crt__knob {
  appearance: none;
  padding: 0;
  cursor: pointer;
  width: 9.33px;
  height: 9.33px;
  border-radius: 50%;
  background: radial-gradient(circle at 35% 30%, #efe3c8, #8a7556 70%);
  border: 0.67px solid #4a3824;
  box-shadow: 0 1.33px 2px rgba(0, 0, 0, 0.4);
}

.store-crt__knob:hover {
  filter: brightness(1.12);
}

.store-crt__knob.is-off {
  background: radial-gradient(circle at 35% 30%, #6f6a5e, #4a4238 70%);
}

.store-crt__knob--small {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 6.67px;
  height: 6.67px;
}

.store-crt__knob--small::after {
  content: '❚❚';
  font-size: 4.67px;
  line-height: 1;
  color: rgba(30, 20, 10, 0.85);
}

.store-crt__knob--small.is-paused::after {
  content: '▶';
  font-size: 4px;
}

.store-crt__led {
  width: 3.33px;
  height: 3.33px;
  border-radius: 50%;
  background: #7be37b;
  box-shadow: 0 0 4px #7be37b;
  animation: crt-breath 4.8s ease-in-out infinite;
}

.store-crt__led.is-off {
  background: #5d2222;
  box-shadow: 0 0 2.67px rgba(140, 40, 40, 0.5);
  animation: none;
}

.store-crt__vents {
  position: absolute;
  left: 12px;
  bottom: 12px;
  width: 56px;
  height: 6.67px;
  background: repeating-linear-gradient(90deg, #6d5a3e 0 1.33px, transparent 1.33px 3.33px);
  border-radius: 1.33px;
  opacity: 0.75;
}

.store-crt__stand {
  width: 72%;
  height: 14.67px;
  margin: 0 auto;
  background: linear-gradient(180deg, #7d6240, #4e3824);
  border-radius: 0 0 5.33px 5.33px;
  box-shadow: 0 8px 12px rgba(0, 0, 0, 0.5);
}

.store-crt__cable {
  position: absolute;
  right: -10%;
  bottom: -25.33px;
  width: 60%;
  height: 34.67px;
  border: 2px solid #241a12;
  border-top: 0;
  border-left: 0;
  border-radius: 0 0 40px 0;
  opacity: 0.85;
}

/* ---------- 前台 / 游戏吧 ---------- */
.store-counter {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  height: 200px;
  z-index: 4;
}

.store-counter__top {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 10.67px;
  background:
    linear-gradient(180deg, #b08a5c 0%, #8d6a42 72%, #6f4f2e 100%);
  box-shadow:
    0 4px 8px rgba(0, 0, 0, 0.55),
    inset 0 0.67px 0 rgba(255, 235, 200, 0.4);
  transform: perspective(266.67px) rotateX(24deg);
  transform-origin: bottom center;
  border-radius: 2px 2px 0 0;
}

.store-counter__top-glow {
  position: absolute;
  inset: 6.67px 0 0 0;
  background: linear-gradient(90deg, transparent 20%, rgba(255, 230, 170, 0.25) 50%, transparent 80%);
}

.store-counter__front {
  position: absolute;
  top: 10.67px;
  left: 0;
  right: 0;
  bottom: 0;
  background:
    repeating-linear-gradient(0deg, rgba(0, 0, 0, 0.12) 0 0.67px, transparent 0.67px 6px),
    linear-gradient(180deg, #6f4f2e, #4a3320 78%);
  box-shadow:
    inset 0 0.67px 0 rgba(255, 235, 200, 0.16),
    inset 0 -16px 26.67px rgba(0, 0, 0, 0.5);
}

.store-counter__trim {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 2px;
  background: rgba(255, 235, 200, 0.14);
}

/* ---------- 灯光 / 氛围 ---------- */
.store-light {
  position: absolute;
  inset: 0;
  z-index: 5;
  pointer-events: none;
  background:
    radial-gradient(ellipse at 50% -8%, rgba(255, 196, 120, 0.28), transparent 46%),
    radial-gradient(ellipse at 50% 38%, rgba(255, 180, 100, 0.07), transparent 62%);
  mix-blend-mode: screen;
}

.store-vignette {
  position: absolute;
  inset: 0;
  z-index: 6;
  pointer-events: none;
  background:
    linear-gradient(180deg, rgba(10, 7, 5, 0.42), transparent 26%),
    radial-gradient(ellipse at 50% 46%, transparent 52%, rgba(8, 5, 3, 0.62) 100%);
}

/* ---------- 离开按钮 ---------- */
.store-exit {
  position: absolute;
  top: 16px;
  left: 16px;
  z-index: 20;
  border: 1px solid rgba(255, 225, 180, 0.24);
  background: rgba(24, 16, 11, 0.48);
  color: rgba(255, 230, 190, 0.82);
  font-size: 15px;
  padding: 10px 20px;
  border-radius: 999px;
  cursor: pointer;
  backdrop-filter: blur(6px);
  -webkit-backdrop-filter: blur(6px);
  transition: background 0.25s ease, color 0.25s ease, border-color 0.25s ease;
}

.store-exit:hover {
  background: rgba(58, 38, 24, 0.72);
  color: #ffe9c8;
  border-color: rgba(255, 225, 180, 0.5);
}

/* ---------- 拿出游戏盒 ---------- */
.store-inspect {
  position: absolute;
  inset: 0;
  z-index: 30;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(8, 5, 3, 0.58);
  backdrop-filter: blur(2px);
  -webkit-backdrop-filter: blur(2px);
}

.store-inspect__box {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 20px;
  padding: 18px 26px 26px;
}

.store-inspect__case {
  position: relative;
  width: 320px;
  aspect-ratio: 0.72;
  border-radius: 4px;
  cursor: pointer;
  perspective: 1100px;
  transform: perspective(900px) rotateY(-2deg) rotateX(1deg);
  box-shadow:
    0 40px 80px rgba(0, 0, 0, 0.72),
    0 14px 26px rgba(0, 0, 0, 0.5);
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
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.06);
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
  background: #1b1410;
  border-radius: 4px;
  transform-origin: left center;
  transition: transform 0.6s cubic-bezier(0.22, 0.61, 0.36, 1);
  will-change: transform;
}

.store-inspect__case--opening .store-inspect__cover {
  transform: rotateY(-112deg);
}

.store-inspect__cover img {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
  border-radius: 4px;
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.12);
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
  margin-top: 10px !important;
  font-size: 12px !important;
  color: rgba(255, 230, 190, 0.52) !important;
  letter-spacing: 1px !important;
}

.store-inspect__hint--error {
  color: rgba(255, 170, 130, 0.92) !important;
}

.store-inspect__actions {
  display: flex;
  gap: 14px;
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

/* ---------- 动画 ---------- */
@keyframes crt-noise {
  0% { background-position: 0 0; }
  25% { background-position: -20px 8px; }
  50% { background-position: 12px -14.67px; }
  75% { background-position: -8px -20px; }
  100% { background-position: 16px 12px; }
}

@keyframes crt-flicker {
  0%, 100% { opacity: 1; }
  92% { opacity: 1; }
  93% { opacity: 0.78; }
  94% { opacity: 1; }
  97% { opacity: 0.9; }
  98% { opacity: 1; }
}

@keyframes crt-breath {
  0%, 100% { opacity: 0.55; }
  50% { opacity: 1; }
}

.inspect-enter-active,
.inspect-leave-active {
  transition: opacity 0.28s ease;
}

.inspect-enter-from,
.inspect-leave-to {
  opacity: 0;
}

.inspect-enter-active .store-inspect__case,
.inspect-leave-active .store-inspect__case {
  transition: transform 0.3s cubic-bezier(0.22, 0.61, 0.36, 1), opacity 0.28s ease;
}

.inspect-enter-from .store-inspect__case,
.inspect-leave-to .store-inspect__case {
  transform: perspective(900px) rotateY(-2deg) rotateX(1deg) scale(0.72);
  opacity: 0;
}


</style>

<!-- 看板娘：live2d-widget 创建的是 body 级节点，需用全局样式把它定位进店内场景 -->
<style>
#waifu {
  position: absolute !important;
  left: 0px !important;
  bottom: 6.67px !important;
  width: 533.33px !important;
  height: 533.33px !important;
  transform: none !important;
  z-index: 3 !important;
  pointer-events: none !important;
  transition: bottom 1s ease-in-out;
}

#waifu.waifu-active {
  bottom: 6.67px !important;
}

#waifu:hover {
  transform: none !important;
}

#waifu-toggle {
  display: none !important;
}

#live2d {
  width: 533.33px !important;
  height: 533.33px !important;
  pointer-events: none !important;
}

#waifu-tips {
  left: 340px !important;
  top: 20px !important;
  margin: 0 !important;
  width: 146.67px !important;
  min-height: 36px;
  font-size: 8.67px;
  line-height: 14px;
}

@media (max-width: 768px) {
  #waifu {
    display: none;
  }
}
</style>
