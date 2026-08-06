<template>
  <div ref="storeRootRef" class="game-store">
    <div
      class="store-stage"
      :class="{ 'store-stage--dim': stageDim }"
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
        </div>
        <div class="store-poster store-poster--right">
          <img
            v-if="storePosters[1]"
            class="store-poster__img"
            :src="storePosters[1]"
            alt="畅销榜"
            decoding="async"
          >
        </div>
        <div class="store-sign">
          ATLAS&nbsp;GAMES
          <span class="store-sign__sub">阿特拉斯电玩</span>
        </div>
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
                :class="{
                  'game-box--hovered': hoveredId === cell.game.publicId,
                  'game-box--picked': pickedGame?.publicId === cell.game.publicId,
                }"
                :style="boxStyle(cell)"
                :title="cell.game.title"
                @mouseenter="hoveredId = cell.game.publicId"
                @mouseleave="hoveredId = null"
                @click="pickGame(cell.game, $event)"
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
              :src="crtVideoUrl || undefined"
              autoplay
              muted
              playsinline
              preload="auto"
              @ended="handleCrtVideoEnded"
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
      <div v-if="pickedGame" class="store-inspect" @click.self="putBack()">
        <div
          class="store-inspect__box"
          :class="{ 'store-inspect__box--settled': inspectSettled }"
        >
          <div
            ref="caseRef"
            class="store-inspect__case"
            :class="{ 'store-inspect__case--opening': isOpening }"
            title="开始游戏"
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
          <p
            class="store-inspect__hint"
            :class="{ 'store-inspect__hint--success': launchHintSuccess }"
          >
            {{ launchHint }}
          </p>
          <div class="store-inspect__meta">
            <h2>{{ pickedGame.title }}</h2>
            <p>
              {{ pickedGame.year }}
              <template v-if="pickedGame.titleAlt"> · {{ pickedGame.titleAlt }}</template>
            </p>
          </div>
          <div class="store-inspect__actions">
            <button type="button" class="store-btn store-btn--ghost" @click="putBack()">放回去</button>
          </div>
        </div>
      </div>
    </Transition>

    <!-- 多版本选择：开盒后存在多个可启动版本时弹出 -->
    <a-modal
      :visible="launchModalVisible"
      :footer="false"
      :width="480"
      :title="`开始游戏：${launchTitle}`"
      @cancel="launchModalVisible = false"
    >
      <div class="store-launch-list">
        <button
          v-for="option in launchOptions"
          :key="option.id"
          type="button"
          class="store-launch-item"
          @click="handleLaunchVersion(option)"
        >
          <icon-play-arrow />
          <span class="store-launch-item__name">{{ option.version }}</span>
          <span class="store-launch-item__action">开始游戏</span>
        </button>
      </div>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { IconPlayArrow } from '@arco-design/web-vue/es/icon'
import gamesService, { mapGameVersions } from '@/services/games.service'
import type { GameVersion } from '@/services/types'
import { useUiStore } from '@/stores/ui'
import {
  getAmbientBackgroundPoolFromGames,
  mergeAmbientBackgroundPools,
  type AmbientBackgroundPool,
} from '@/utils/ambient-background'

const uiStore = useUiStore()

interface StoreShelfGame {
  publicId: string
  title: string
  titleAlt: string
  year: string
  coverUrl: string
}

interface ShelfCell {
  game: StoreShelfGame
  dx: number
  dy: number
  rot: number
  z: number
}

const router = useRouter()
const hoveredId = ref<string | null>(null)
const pickedGame = ref<StoreShelfGame | null>(null)
const isOpening = ref(false)
const caseRef = ref<HTMLElement | null>(null)
const stageDim = ref(false)
const inspectSettled = ref(false)
const gameStoreSessionGames = ref<StoreShelfGame[]>([])
const crtVideoUrl = ref('')
const crtPlaylist = ref<string[]>([])
let crtPlaylistIndex = 0
const crtPowered = ref(true)
const crtPaused = ref(false)
const crtVideoRef = ref<HTMLVideoElement | null>(null)
const stageScale = ref(1)
const storePosters = ref<string[]>([])
const storeRootRef = ref<HTMLElement | null>(null)
const launchModalVisible = ref(false)
const launchTitle = ref('')
const launchOptions = ref<Array<{ id: string; version: string; url: string }>>([])
const launchHint = ref('点击游戏盒直接开始游戏')
const launchHintSuccess = ref(false)

let isStoreDisposed = false
let storeAbortController: AbortController | null = null

/** 开盒动画（盒盖 0.6s 翻转）播完即出手，不再额外等待 */
const OPEN_CASE_DELAY_MS = 600

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

const loadStorePosters = async (signal?: AbortSignal) => {
  try {
    // 与全局抽卡池同一数据源：遍历游戏列表收集 banner 与截图
    const pools: AmbientBackgroundPool[] = []
    let page = 1
    while (true) {
      const result = await gamesService.getGames({
        query: { page, limit: 100 },
        sort: { field: 'created_at', order: 'desc' },
        signal,
      })
      if (isStoreDisposed) return
      pools.push(getAmbientBackgroundPoolFromGames(result.data))
      const totalPages = Math.max(1, result.pagination.totalPages || 1)
      if (page >= totalPages) break
      page += 1
    }
    if (isStoreDisposed) return
    storePosters.value = pickPosterImages(mergeAmbientBackgroundPools(pools))
  } catch {
    if (isStoreDisposed) return
    // 拉取失败时保留纯色海报兜底
    uiStore.addAlert('游戏店海报加载失败，已显示兜底样式', 'warning')
  }
}

const loadStoreSession = async (signal?: AbortSignal) => {
  try {
    // 每次进入生成一个随机种子，后端直接按 random 排序返回 20 个游戏
    const seed = Math.floor(Math.random() * 2_147_483_647) + 1
    const result = await gamesService.getGames({
      query: { page: 1, limit: 20 },
      sort: { field: 'random', order: 'desc', seed },
      signal,
    })
    if (isStoreDisposed) return
    // 货架固定 4 行 × 5 盒
    const picked = result.data.filter((game) => game.cover_image)
    gameStoreSessionGames.value = picked.map((game) => ({
      publicId: game.public_id,
      title: game.title,
      titleAlt: game.title_alt ?? '',
      year: game.release_date ? game.release_date.slice(0, 4) : '',
      coverUrl: game.cover_image || '',
    }))

    // CRT 播放真实预告：把本次 Session 所有预告片拍平成轮播列表，播完自动换下一个
    const videoBundles = await gamesService.getPreviewVideos(
      picked.map((game) => game.public_id),
      signal,
    )
    if (isStoreDisposed) return
    crtPlaylist.value = videoBundles.flatMap((bundle) => bundle.preview_videos.map((video) => video.path))
    crtPlaylistIndex = 0
    crtVideoUrl.value = crtPlaylist.value[0] ?? ''
  } catch {
    if (isStoreDisposed) return
    // 拉取失败时保留空货架，避免展示 mock 数据
    uiStore.addAlert('游戏店数据加载失败，货架暂时为空', 'warning')
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
  }
}

let waifuStyleTag: HTMLLinkElement | null = null
let waifuScriptTag: HTMLScriptElement | null = null
let waifuZoomTimer: number | null = null
let storeResizeObserver: ResizeObserver | null = null
let openCaseTimer: number | null = null
let putBackTimer: number | null = null
let pickupSettleTimer: number | null = null
let dimTimer: number | null = null
let pickupAnimation: Animation | null = null
let pendingLaunchPublicId: string | null = null

interface PickupOrigin {
  x: number
  y: number
  scale: number
  rot: number
}

let pickupOrigin: PickupOrigin | null = null

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
  if (isStoreDisposed) return Promise.resolve(null)
  const existing = document.querySelector<HTMLElement>(selector)
  if (existing) return Promise.resolve(existing)
  return new Promise((resolve) => {
    const startedAt = Date.now()
    const timer = window.setInterval(() => {
      if (isStoreDisposed) {
        window.clearInterval(timer)
        resolve(null)
        return
      }
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
  if (isStoreDisposed) return
  // 热更新或重复挂载时先清掉旧的看板娘节点
  document.getElementById('waifu')?.remove()
  document.getElementById('waifu-toggle')?.remove()

  await loadWaifuResource('/live2d-widget/waifu.css', 'css')
  if (isStoreDisposed) return
  if (!window.initWidget) {
    await loadWaifuResource('/live2d-widget/waifu-tips.js', 'js')
    if (isStoreDisposed) return
  }

  // 清掉旧 manager，避免上一次会话的缩放状态被误判为“已生效”
  window.__waifuManager = undefined

  if (isStoreDisposed) return
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
  if (isStoreDisposed) return
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
  if (isStoreDisposed) return
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
  gameStoreSessionGames.value.forEach((game, index) => {
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

const pickGame = (game: StoreShelfGame, event: MouseEvent) => {
  const button = event.currentTarget as HTMLElement | null
  const buttonRect = button?.getBoundingClientRect()
  hoveredId.value = null
  inspectSettled.value = false
  stageDim.value = false
  launchHint.value = '点击游戏盒直接开始游戏'
  launchHintSuccess.value = false
  pickedGame.value = game
  if (pickupSettleTimer !== null) {
    window.clearTimeout(pickupSettleTimer)
    pickupSettleTimer = null
  }
  if (dimTimer !== null) {
    window.clearTimeout(dimTimer)
    dimTimer = null
  }
  if (!buttonRect) return

  // “抓取”动画：从货架上那盒的原始位置/角度/大小飞向眼前
  nextTick(() => {
    const caseElement = caseRef.value
    if (!caseElement || pickedGame.value?.publicId !== game.publicId) return
    const caseRect = caseElement.getBoundingClientRect()
    const origin: PickupOrigin = {
      x: buttonRect.left + buttonRect.width / 2 - (caseRect.left + caseRect.width / 2),
      y: buttonRect.top + buttonRect.height / 2 - (caseRect.top + caseRect.height / 2),
      scale: buttonRect.width / caseRect.width,
      rot: Number.parseFloat(button?.style.getPropertyValue('--rot') ?? '') || 0,
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
      stageDim.value = true
    }, 180)
    pickupSettleTimer = window.setTimeout(() => {
      inspectSettled.value = true
    }, 740)
  })
}

const handleCrtVideoEnded = () => {
  if (crtPlaylist.value.length === 0) return
  crtPlaylistIndex = (crtPlaylistIndex + 1) % crtPlaylist.value.length
  crtVideoUrl.value = crtPlaylist.value[crtPlaylistIndex]
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

const finishPutBack = () => {
  isOpening.value = false
  pickedGame.value = null
  pickupOrigin = null
  pickupAnimation = null
  inspectSettled.value = false
  stageDim.value = false
}

const putBack = (animate = true) => {
  if (openCaseTimer !== null) {
    window.clearTimeout(openCaseTimer)
    openCaseTimer = null
  }
  pendingLaunchPublicId = null
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
  stageDim.value = false

  // 已开盒或没有抓取起点时直接收起，不播反向动画
  const caseElement = caseRef.value
  if (!animate || isOpening.value || !pickupOrigin || !caseElement) {
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
    if (pickedGame.value) finishPutBack()
  }, 900)
}

// 开盒即开玩（与开始屏幕一致）：详情拉取与开盒动画并行，动画结束直接启动；
// 无可启动版本或拉取失败才回退详情页，多个版本时弹窗选择。
// 触发启动后游戏盒保持打开，等待浏览器下载启动脚本，用户可点“放回去”收起。
const handleOpenCase = () => {
  if (isOpening.value || !pickedGame.value) return
  isOpening.value = true
  const publicId = pickedGame.value.publicId
  pendingLaunchPublicId = publicId
  const detailPromise = gamesService.getGameDetail(publicId).catch(() => null)

  openCaseTimer = window.setTimeout(async () => {
    openCaseTimer = null
    const detail = await detailPromise
    if (pendingLaunchPublicId !== publicId) return
    pendingLaunchPublicId = null
    if (!detail) {
      putBack()
      router.push({ name: 'game-detail', params: { publicId } })
      return
    }
    const launchable = mapGameVersions(detail).filter(
      (version) => version.canLaunch && version.launchScriptUrl,
    )
    if (launchable.length === 0) {
      putBack()
      router.push({ name: 'game-detail', params: { publicId } })
      return
    }
    if (launchable.length === 1) {
      launchVersion(launchable[0])
      return
    }
    launchTitle.value = detail.title
    launchOptions.value = launchable.map((version) => ({
      id: version.id,
      version: version.version,
      url: version.launchScriptUrl!,
    }))
    launchModalVisible.value = true
  }, OPEN_CASE_DELAY_MS)
}

const launchVersion = (version: GameVersion) => {
  if (!version.launchScriptUrl) return
  window.location.assign(version.launchScriptUrl)
  launchHint.value = `已生成启动脚本：${version.version}，请查看浏览器下载`
  launchHintSuccess.value = true
}

const handleLaunchVersion = (option: { id: string; version: string; url: string }) => {
  launchModalVisible.value = false
  window.location.assign(option.url)
  launchHint.value = `已生成启动脚本：${option.version}，请查看浏览器下载`
  launchHintSuccess.value = true
}

const handleKeydown = (event: KeyboardEvent) => {
  if (event.key === 'Escape') putBack()
}

onMounted(() => {
  isStoreDisposed = false
  storeAbortController = new AbortController()
  const storeSignal = storeAbortController.signal
  window.addEventListener('keydown', handleKeydown)
  updateStageScale()
  // 侧边栏收起/展开不会触发 window.resize，监听容器尺寸变化才能让场景跟随内容区缩放。
  if (storeRootRef.value) {
    storeResizeObserver = new ResizeObserver(() => updateStageScale())
    storeResizeObserver.observe(storeRootRef.value)
  }
  void initWaifu().catch(() => {
    cleanupWaifu()
    if (!isStoreDisposed) {
      uiStore.addAlert('看板娘加载失败，已停用', 'warning')
    }
  })
  void loadStorePosters(storeSignal)
  void loadStoreSession(storeSignal)
})
onUnmounted(() => {
  isStoreDisposed = true
  storeAbortController?.abort()
  storeAbortController = null
  pickupAnimation?.cancel()
  pickupAnimation = null
  window.removeEventListener('keydown', handleKeydown)
  storeResizeObserver?.disconnect()
  storeResizeObserver = null
  if (openCaseTimer !== null) {
    window.clearTimeout(openCaseTimer)
    openCaseTimer = null
  }
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
  pendingLaunchPublicId = null
  cleanupWaifu()
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
  transform: rotate(-0.8deg) scale(1.33);
}

.store-poster--right {
  right: 60px;
  transform: rotate(0.8deg) scale(1.33);
}

.store-poster__img {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

/* ---------- 霓虹招牌 ---------- */
.store-sign {
  position: absolute;
  top: 4px;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.67px;
  font-size: 21px;
  font-weight: 700;
  letter-spacing: 4.67px;
  color: #ffd9a0;
  text-shadow:
    0 0 4px rgba(255, 190, 110, 0.95),
    0 0 12px rgba(255, 160, 70, 0.75),
    0 0 28px rgba(255, 130, 40, 0.55);
  background: rgba(30, 20, 14, 0.35);
  padding: 2.67px 12px 4px;
  border-radius: 2.67px;
}

.store-sign__sub {
  font-size: 9px;
  font-weight: 400;
  letter-spacing: 2.67px;
  color: rgba(255, 217, 160, 0.82);
  text-shadow:
    0 0 3.33px rgba(255, 190, 110, 0.8),
    0 0 8px rgba(255, 160, 70, 0.5);
}

.store-sign__cord {
  position: absolute;
  top: 100%;
  left: 50%;
  width: 1.33px;
  height: 9.33px;
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
  /* 盒身比例统一 0.72（宽:高 = 18:25），与检查盒一致；
     封面源图 600×900 用 object-fit: cover 轻微裁左右出血 */
  width: 72px;
  height: 100px;
  aspect-ratio: 0.72;
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

/* 已拿在手里的那盒：货架上原位置留空，模拟“盒子在我手上” */
.game-box--picked {
  visibility: hidden;
  pointer-events: none;
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
  bottom: 8.33px;
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

/* ---------- 多版本启动选择 ---------- */
.store-launch-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.store-launch-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border: 1px solid var(--app-card-border);
  border-radius: 8px;
  background: var(--color-fill-2);
  color: var(--color-text-1);
  cursor: pointer;
  text-align: left;
  transition: background 120ms ease, border-color 120ms ease;
}

.store-launch-item:hover {
  background: var(--color-fill-3);
  border-color: var(--app-glass-border-hover);
}

.store-launch-item__name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.store-launch-item__action {
  font-size: 13px;
  color: var(--color-primary-6);
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
