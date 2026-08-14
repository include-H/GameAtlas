<template>
  <Teleport to="body">
    <Transition name="start-screen-overlay" @leave="onOverlayLeave">
      <div
        v-if="visible"
        ref="wrapperRef"
        class="start-screen-wrapper"
        :class="{ 'is-expanding-tiles-hidden': expandHideTiles }"
        tabindex="-1"
        @keydown.esc="handleClose"
      >
        <shared-ambient-background force-enabled />
        <div class="start-screen-scrim" @click="handleClose" />

        <div class="start-screen" @wheel.passive="handleWheel" @click.self="handleClose">
          <header class="start-screen__header">
            <div>
              <h1 class="start-screen__heading">开始</h1>
            </div>

            <div class="start-screen__header-actions">
              <span v-if="canEdit" class="start-screen__admin-name">
                {{ adminDisplayName || '管理员' }}
              </span>
              <template v-if="!isEditing && canEdit">
                <a-tooltip content="编辑磁贴">
                  <a-button
                    class="app-text-action-btn start-screen__edit-button"
                    type="text"
                    shape="circle"
                    aria-label="编辑磁贴"
                    @click="emit('startEdit')"
                  >
                    <template #icon><icon-edit /></template>
                  </a-button>
                </a-tooltip>
              </template>
              <template v-else-if="isEditing">
                <a-button class="app-text-action-btn" type="text" :disabled="isSaving" @click="emit('cancelEdit')">
                  取消
                </a-button>
                <a-button type="primary" :loading="isSaving" @click="emit('saveEdit')">
                  保存
                </a-button>
              </template>
            </div>
          </header>

          <p v-if="isEditing && saveError" class="start-screen__save-error">{{ saveError }}</p>

          <div v-if="isLoading && tiles.length === 0" class="start-screen__state">
            <a-spin :size="28" />
            <p>正在铺开你的磁贴...</p>
          </div>

          <div v-else-if="hasLoadFailure && tiles.length === 0" class="start-screen__state">
            <icon-exclamation-circle />
            <p>开始屏幕加载失败</p>
            <a-button type="primary" @click="emit('retry')">重试</a-button>
          </div>

          <div v-else-if="tiles.length === 0 && !isEditing" class="start-screen__state">
            <icon-star />
            <p>还没有磁贴</p>
            <a-space v-if="!isEditing">
              <a-button @click="handleBrowseGames">去游戏库逛逛</a-button>
            </a-space>
          </div>

          <div
            v-else
            ref="metroAreaRef"
            class="start-screen__metro"
            :class="{ 'is-editing': isEditing }"
            @click.self="handleClose"
          >
            <div class="start-screen__groups">
              <section
                v-for="(group, groupIndex) in visibleGroups"
                :key="group.columnIndex"
                class="start-screen__group"
                :style="{
                  '--start-group-cols': groupCols(group),
                  '--start-group-rows': Math.max(groupMaxRow(group), 1),
                }"
                :data-start-screen-cell="true"
                :data-column-index="group.columnIndex"
                data-row="0"
                data-col="0"
              >
                <div
                  class="start-screen__group-header"
                  :style="{ '--start-tile-enter-delay': groupEnterDelay(groupIndex) }"
                >
                  <input
                    v-if="isEditing"
                    class="start-screen__group-name-input"
                    :value="columnNameOf(group.columnIndex)"
                    :placeholder="`组 ${group.columnIndex + 1}`"                    @change="handleRenameColumn(group.columnIndex, $event)"
                  >
                  <span v-else class="start-screen__group-name">{{ columnNameOf(group.columnIndex) }}</span>
                  <button
                    v-if="isEditing && !columnHasTiles(group.columnIndex)"
                    class="app-text-action-btn start-screen__group-remove"
                    type="button"
                    :aria-label="`删除空组 ${group.columnIndex + 1}`"
                    @click="emit('removeColumn', group.columnIndex)"
                  >
                    <icon-close />
                  </button>
                </div>

                <div class="start-screen__group-grid-row">
                  <div class="start-screen__group-grid">
                    <template v-if="isEditing">
                      <div
                        v-for="cell in groupCells(group)"
                        :key="`cell-${cell.row}-${cell.col}`"
                        class="start-screen__drop-cell"
                        :class="{ 'start-screen__drop-cell--target': isDropTarget(group.columnIndex, cell.row, cell.col) }"
                        :data-start-screen-cell="true"
                        :data-column-index="group.columnIndex"
                        :data-row="cell.row"
                        :data-col="cell.col"
                        :style="{ gridColumnStart: cell.col + 1, gridRowStart: cell.row + 1 }"
                      />
                    </template>
                    <TransitionGroup
                      name="metro-tile"
                      tag="div"
                      class="start-screen__group-tiles"
                      @enter="onTileEnter"
                      @leave="onTileLeave"
                    >
                      <div
                        v-for="(slot, slotIndex) in group.slots"
                        :key="slot.tile.game_id"
                        :class="[
                          'start-screen__tile-slot',
                          `start-screen__tile-slot--${slot.tile.tile_size}`,
                          { 'start-screen__tile-slot--dragging': isDraggedTile(slot.tile) },
                          { 'start-screen__tile-slot--target': isDropTarget(group.columnIndex, slot.row, slot.col) },
                        ]"
                        :style="{
                          gridColumnStart: slot.col + 1,
                          gridColumnEnd: slot.col + 1 + tileSpan(slot.tile.tile_size).cols,
                          gridRowStart: slot.row + 1,
                          gridRowEnd: slot.row + 1 + tileSpan(slot.tile.tile_size).rows,
                          '--start-tile-enter-delay': tileEnterDelay(groupIndex, slotIndex),
                          '--start-tile-leave-delay': tileLeaveDelay(groupIndex, slotIndex),
                        }"
                        :data-tile-index="slot.globalIndex"
                        :data-start-screen-cell="true"
                        :data-column-index="group.columnIndex"
                        :data-row="slot.row"
                        :data-col="slot.col"
                        @pointerdown="onTilePointerDown(slot.globalIndex, $event)"
                      >
                        <MetroTile
                          :tile="slot.tile"
                          :color-index="slot.globalIndex"
                          :editing="isEditing"
                          @select="handleTileSelect"
                          @crop="handleCrop"
                          @resize="emit('resize', $event)"
                          @remove="emit('remove', $event)"
                        />
                      </div>
                    </TransitionGroup>
                  </div>

                  <div
                    v-if="isEditing"
                    class="start-screen__group-append-col"
                    :class="{ 'start-screen__group-append--target': isDropTarget(group.columnIndex, 0, groupCols(group)) }"
                    :data-start-screen-cell="true"
                    :data-column-index="group.columnIndex"
                    data-row="0"
                    :data-col="groupCols(group)"
                    :title="`追加列 ${groupCols(group) + 1}`"
                  >
                    <icon-plus />
                  </div>
                </div>

                <div
                  v-if="isEditing"
                  class="start-screen__group-append"
                  :class="{ 'start-screen__group-append--target': isDropTarget(group.columnIndex, groupMaxRow(group), 0) }"
                  :data-start-screen-cell="true"
                  :data-column-index="group.columnIndex"
                  :data-row="groupMaxRow(group)"
                  data-col="0"
                >
                  <icon-plus />
                  <span>拖到此处追加</span>
                </div>
              </section>

              <button
                v-if="isEditing"
                type="button"
                class="start-screen__new-column"
                :class="{ 'start-screen__new-column--target': isNewColumnTarget }"
                :data-start-screen-cell="true"
                :data-column-index="newColumnIndex"
                data-row="0"
                data-col="0"
                @click="handleAddColumn"
              >
                <icon-plus />
                <span>新列</span>
              </button>
            </div>

            <div
              v-if="dragState && dragPointer && draggedTile"
              class="start-screen__drag-ghost"
              :class="`start-screen__drag-ghost--${draggedTile.tile_size}`"
              :style="{ left: `${dragPointer.x}px`, top: `${dragPointer.y}px` }"
            >
              <MetroTile :tile="draggedTile" :color-index="dragState.fromIndex" :editing="false" />
            </div>
          </div>

          <div class="start-screen__desktop-hint" @click="handleClose">
            <icon-desktop />
            <span>回到桌面</span>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>

  <tile-crop-modal
    :visible="cropVisible"
    :image-src="cropSource"
    :banners="cropBanners"
    @confirm="handleCropConfirm"
    @cancel="cropVisible = false"
  />

  <div v-if="expand" class="start-screen-expand-host">
    <div
      ref="expandEl"
      class="start-screen-expand"
    >
      <div class="start-screen-expand__face start-screen-expand__face--front">
        <img
          v-if="expand.frontImage"
          :src="expand.frontImage"
          :alt="expand.title"
          class="start-screen-expand__cover"
        >
      </div>
      <div class="start-screen-expand__face start-screen-expand__face--back">
        <div
          v-if="expand.backImage"
          class="start-screen-expand__bg"
          :style="{ backgroundImage: `url(${expand.backImage})` }"
        />
        <div class="start-screen-expand__shade" />
      </div>
      <div ref="expandMetaEl" class="start-screen-expand__meta">
        <h2>{{ expand.title }}</h2>
        <div v-if="expandLoading" class="start-screen-expand__loading">
          <a-spin :size="28" />
          <span>正在加载...</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onUnmounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import {
  IconClose,
  IconDesktop,
  IconEdit,
  IconExclamationCircle,
  IconPlus,
  IconStar,
} from '@arco-design/web-vue/es/icon'
import MetroTile from './MetroTile.vue'
import TileCropModal from './TileCropModal.vue'
import SharedAmbientBackground from '@/components/SharedAmbientBackground.vue'
import type { StartScreenColumn, StartScreenTile, StartScreenTileSize } from '@/services/types'
import {
  findStartScreenDropTarget,
  layoutStartScreenTiles,
  START_SCREEN_FREE_COLS,
} from '@/utils/start-screen-layout'
import type { PackedStartScreenGroup } from '@/utils/start-screen-layout'
import gamesService from '@/services/games.service'
import { GAME_DETAIL_RETURN_QUERY } from '@/utils/navigation'
import { createRequestGeneration } from '@/utils/request-generation'

const TILE_SPANS: Record<StartScreenTileSize, { rows: number; cols: number }> = {
  small: { rows: 2, cols: 2 },
  wide: { rows: 2, cols: 4 },
  large: { rows: 4, cols: 4 },
}

const props = defineProps<{
  visible: boolean
  tiles: StartScreenTile[]
  columns: StartScreenColumn[]
  canEdit: boolean
  adminDisplayName: string
  isLoading: boolean
  hasLoadFailure: boolean
  isEditing: boolean
  isSaving: boolean
  saveError: string | null
}>()

const emit = defineEmits<{
  close: []
  retry: []
  startEdit: []
  cancelEdit: []
  saveEdit: []
  select: [publicId: string]
  resize: [gameId: number]
  remove: [gameId: number]
  applyPlacement: [gameId: number, columnIndex: number, row: number, col: number]
  addColumn: []
  removeColumn: [index: number]
  applyCrop: [gameId: number, blobs: Record<StartScreenTileSize, Blob>]
  renameColumn: [index: number, name: string]
}>()

const router = useRouter()
const wrapperRef = ref<HTMLElement | null>(null)
const metroAreaRef = ref<HTMLElement | null>(null)
const cropVisible = ref(false)
const cropGameId = ref<number | null>(null)
const cropBanners = ref<string[]>([])
const tileDetailRequests = createRequestGeneration()
const cropDetailRequests = createRequestGeneration()

interface TileDragState {
  gameId: number
  fromIndex: number
  targetColumnIndex: number
  targetRow: number
  targetCol: number
}

const dragState = ref<TileDragState | null>(null)
const dragPointer = ref<{ x: number; y: number } | null>(null)
let pendingDrag: { gameId: number; fromIndex: number } | null = null
let dragStart: { x: number; y: number } | null = null
let edgeScrollFrame: number | null = null

const cropSource = computed(() => {
  const tile = props.tiles.find((item) => item.game_id === cropGameId.value)
  return tile?.banner_image || tile?.cover_image || ''
})

const draggedTile = computed(() => props.tiles.find((tile) => tile.game_id === dragState.value?.gameId) ?? null)

// 拖拽预览：被拖磁贴从布局中抽出，由 ghost 跟随光标；目标格高亮，落点时再写入坐标。
const displayTiles = computed(() => {
  if (!dragState.value) return props.tiles
  return props.tiles.filter((tile) => tile.game_id !== dragState.value?.gameId)
})

const layoutGroups = computed(() => layoutStartScreenTiles(displayTiles.value, props.columns.length))
const visibleGroups = computed(() => {
  const groups = layoutGroups.value
  if (!props.isEditing) return groups.filter((group) => group.slots.length > 0)
  if (props.tiles.length === 0 && props.columns.length === 0) return []
  return groups
})
const tileSpan = (size: StartScreenTileSize) => TILE_SPANS[size]
// EntranceThemeTransition 复刻：磁贴从其最终位置右下方约 10px 处上浮 + 淡入到原位，
// 位移克制（10px 量级，无大滑入无缩放）。错峰：组间重叠启动（下一组在第一组播到
// 约 1/3 时开始），组内磁贴轻微级联（40ms），整体呈现"一组一块逐组收拢"的层次。
const groupEnterDelay = (groupIndex: number) => `${groupIndex * 110}ms`
const tileEnterDelay = (groupIndex: number, slotIndex: number) => {
  const delay = groupIndex * 110 + slotIndex * 40
  return `${delay}ms`
}
const totalTiles = computed(() =>
  visibleGroups.value.reduce((count, group) => count + group.slots.length, 0),
)
// 退场波浪反向收拢：后进先出的磁贴先飞出。
const tileLeaveDelay = (groupIndex: number, slotIndex: number) => {
  const maxGroup = visibleGroups.value.length - 1
  const maxSlot = (visibleGroups.value[groupIndex]?.slots.length ?? 1) - 1
  const delay = (maxGroup - groupIndex) * 110 + Math.max(maxSlot - slotIndex, 0) * 40
  return `${delay}ms`
}
const groupMaxRow = (group: PackedStartScreenGroup) =>
  group.slots.reduce(
    (max, slot) => Math.max(max, slot.row + tileSpan(slot.tile.tile_size).rows),
    0,
  )
// 组宽按磁贴实际占用收缩（上限 12 列）：磁贴少时网格紧凑，不撑满一屏空白；
// 至少留 2 列兜底（编辑模式空组也有可拖入的区域）。
const groupCols = (group: PackedStartScreenGroup) => {
  const needed = group.slots.reduce(
    (max, slot) => Math.max(max, slot.col + tileSpan(slot.tile.tile_size).cols),
    0,
  )
  return Math.min(START_SCREEN_FREE_COLS, Math.max(needed, 2))
}
const groupCells = (group: PackedStartScreenGroup) => {
  // 网格只精确覆盖磁贴占用的行，不向下延伸虚空格；组尾由追加条承接拖放。
  const rows = groupMaxRow(group)
  const cols = groupCols(group)
  const cells: Array<{ row: number; col: number }> = []
  for (let row = 0; row < rows; row += 1) {
    for (let col = 0; col < cols; col += 1) {
      cells.push({ row, col })
    }
  }
  return cells
}
const newColumnIndex = computed(() => {
  const maxTileColumn = props.tiles.reduce(
    (max, tile) => Math.max(max, Number.isFinite(tile.column_index) ? tile.column_index : 0),
    -1,
  )
  return Math.max(props.columns.length, maxTileColumn + 1)
})

const isNewColumnTarget = computed(() =>
  Boolean(dragState.value && dragState.value.targetColumnIndex === newColumnIndex.value),
)

const isDraggedTile = (tile: StartScreenTile) => dragState.value?.gameId === tile.game_id

const isDropTarget = (columnIndex: number, row: number, col: number) =>
  Boolean(
    dragState.value &&
    dragState.value.targetColumnIndex === columnIndex &&
    dragState.value.targetRow === row &&
    dragState.value.targetCol === col,
  )

const columnHasTiles = (columnIndex: number) =>
  props.tiles.some((tile) => (Number.isFinite(tile.column_index) ? tile.column_index : 0) === columnIndex)

const columnNameOf = (index: number) => {
  const name = props.columns[index]?.name?.trim()
  return name || `组 ${index + 1}`
}

const handleRenameColumn = (index: number, event: Event) => {
  const target = event.target as HTMLInputElement
  emit('renameColumn', index, target.value)
}

const handleAddColumn = () => {
  emit('addColumn')
}

const handleClose = () => {
  if (dragState.value) {
    endDrag()
    return
  }
  // 编辑中禁用空白处/ESC 退出，避免误操作丢掉编辑现场；只能通过"取消/保存"离开。
  if (props.isEditing) return
  emit('close')
}

// Xbox 式展开层：点击磁贴后，从磁贴位置翻转放大成 16:9 大图（当前显示的 banner），
// 期间并行拉取详情；展开动画结束后开始屏退场，随后启动游戏。
interface ExpandState {
  title: string
  /** 起始面：必须和点击时的磁贴图片一致，保证首帧没有换图闪烁。 */
  frontImage: string
  /** 展开面：翻转后显示的游戏 banner/截图。 */
  backImage: string
  size: StartScreenTileSize
  rect: { x: number; y: number; width: number; height: number }
}

const expand = ref<ExpandState | null>(null)
const expandLoading = ref(false)
const expandHideTiles = ref(false)
const expandEl = ref<HTMLElement | null>(null)
const expandMetaEl = ref<HTMLElement | null>(null)
let expandAnim: Animation | null = null
let expandHideTilesTimer: number | null = null

const EXPAND_DELAY_MS = 120
const EXPAND_DURATION_MS = 1500

const prefersReducedMotion = () =>
  typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches

const resetExpandTileVisibility = () => {
  if (expandHideTilesTimer !== null) {
    window.clearTimeout(expandHideTilesTimer)
    expandHideTilesTimer = null
  }
  expandHideTiles.value = false
}

// Win8 式展开分成三个清晰阶段：
// 1. 磁贴保持原尺寸，先翻成左高右低的窄梯形；
// 2. 以这条“磁铁边”为视觉锚点横移到屏幕中线；
// 3. 到中线后才放大、摊平并转正。
// 参考帧里的节奏依赖这个停顿，不能让放大和横移从第一帧同时开始。
const startExpandAnimation = () => {
  const el = expandEl.value
  if (!el || !expand.value) return
  const rect = expand.value.rect
  const viewportWidth = window.innerWidth
  const viewportHeight = window.innerHeight

  type ExpandFrame = { left: number; top: number; width: number; height: number }
  const centeredFrame = (
    widthRatio: number,
    heightRatio: number,
    centerX = 0.5,
    centerY = 0.5,
  ): ExpandFrame => {
    const width = viewportWidth * widthRatio
    const height = viewportHeight * heightRatio
    return {
      left: viewportWidth * centerX - width / 2,
      top: viewportHeight * centerY - height / 2,
      width,
      height,
    }
  }
  const mixFrame = (from: ExpandFrame, to: ExpandFrame, amount: number): ExpandFrame => ({
    left: from.left + (to.left - from.left) * amount,
    top: from.top + (to.top - from.top) * amount,
    width: from.width + (to.width - from.width) * amount,
    height: from.height + (to.height - from.height) * amount,
  })
  const frameStyle = (frame: ExpandFrame) => ({
    left: `${frame.left}px`,
    top: `${frame.top}px`,
    width: `${frame.width}px`,
    height: `${frame.height}px`,
  })

  const originFrame: ExpandFrame = {
    left: rect.x,
    top: rect.y,
    width: rect.width,
    height: rect.height,
  }
  // 先只移动卡片，不改变它的尺寸。旋转到约 80 度后，投影自然成为参考中的窄梯形。
  // 这里把窄面中心放在屏幕中线略左处，横移过程会比直接缩放到中间更明显。
  const edgeFrame = centeredFrame(0.24, 0.62, 0.48, 0.5)
  const unfoldFrame = centeredFrame(0.7, 0.82, 0.49, 0.5)
  const revealFrame = centeredFrame(0.94, 0.94, 0.5, 0.5)
  const fullFrame = centeredFrame(1, 1)
  const settleFrame = mixFrame(originFrame, edgeFrame, 0.16)

  expandAnim?.cancel()
  expandAnim = el.animate(
    [
      {
        ...frameStyle(originFrame),
        // 父面预置 180deg，因此 -180deg 时起始面正对用户。
        transform: 'rotateY(-180deg) rotateZ(0deg)',
        opacity: 1,
      },
      {
        ...frameStyle(originFrame),
        transform: 'rotateY(-152deg) rotateZ(0deg)',
        offset: 0.08,
      },
      {
        ...frameStyle(settleFrame),
        transform: 'rotateY(-112deg) rotateZ(0deg)',
        offset: 0.16,
      },
      {
        // 窄梯形作为连续路径中的转折点，让观众看见磁铁面已经变成侧面。
        ...frameStyle(edgeFrame),
        transform: 'rotateY(-84deg) rotateZ(0deg)',
        offset: 0.31,
      },
      {
        // 中线定位完成后，才开始把窄面摊成大屏。
        ...frameStyle(unfoldFrame),
        transform: 'rotateY(-50deg) rotateZ(-2deg)',
        offset: 0.5,
      },
      {
        ...frameStyle(revealFrame),
        transform: 'rotateY(-18deg) rotateZ(-2.2deg)',
        offset: 0.7,
      },
      {
        ...frameStyle(fullFrame),
        transform: 'rotateY(-5deg) rotateZ(-0.8deg)',
        offset: 0.88,
      },
      {
        ...frameStyle(fullFrame),
        transform: 'rotateY(0deg) rotateZ(0deg)',
      },
    ],
    {
      duration: prefersReducedMotion() ? 1 : EXPAND_DURATION_MS,
      delay: prefersReducedMotion() ? 0 : EXPAND_DELAY_MS,
      // 所有属性共用一条速度曲线，避免每个关键帧都重新减速造成“分段感”。
      easing: 'cubic-bezier(0.22, 1, 0.36, 1)',
      fill: 'both',
    },
  )

  // 标题在背面已经展开到足够大之后出现，不抢侧面翻书的视觉焦点。
  // 注意：WAAPI 的 transform 会覆盖 CSS 的 translateX(-50%) 居中，keyframes 必须带上。
  const meta = expandMetaEl.value
  if (meta) {
    meta.animate(
      [
        { opacity: 0, transform: 'translateX(-50%) translateY(14px)', filter: 'blur(4px)' },
        { opacity: 1, transform: 'translateX(-50%) translateY(0)', filter: 'blur(0)' },
      ],
      {
        duration: prefersReducedMotion() ? 1 : 240,
        delay: prefersReducedMotion() ? 0 : EXPAND_DELAY_MS + 1040,
        easing: 'cubic-bezier(0.22, 1, 0.36, 1)',
        fill: 'both',
      },
    )
  }
}

const openExpand = (publicId: string, rect: DOMRect) => {
  const tile = props.tiles.find((item) => item.public_id === publicId)
  if (!tile) return
  if (expandHideTilesTimer !== null) {
    window.clearTimeout(expandHideTilesTimer)
    expandHideTilesTimer = null
  }
  expandHideTiles.value = false
  const frontImage = tile[`image_${tile.tile_size}_path`] || tile.cover_image || tile.banner_image || ''
  const backImage = tile.image_wide_path || tile.banner_image || tile.cover_image || frontImage
  // 预加载两面：起始面要和磁贴无缝衔接，展开面要在翻正前准备好。
  for (const image of new Set([frontImage, backImage])) {
    if (image) {
      const preload = new Image()
      preload.src = image
    }
  }
  expandLoading.value = true
  expand.value = {
    title: tile.title,
    frontImage,
    backImage,
    size: tile.tile_size,
    rect: { x: rect.x, y: rect.y, width: rect.width, height: rect.height },
  }
  // 保留 200ms 的磁贴原位画面，让展开层和被点击磁贴先完成无缝接管。
  expandHideTilesTimer = window.setTimeout(() => {
    expandHideTiles.value = true
    expandHideTilesTimer = null
  }, 200)
  void nextTick(() => {
    startExpandAnimation()
  })
}

// 展开结束的优雅过渡：详情页已 push 到位，展开层轻微放大 + 淡出（zoom-out 风格，
// 320ms WAAPI），onfinish 后卸载——避免硬切。
const closeExpand = () => {
  resetExpandTileVisibility()
  expandAnim?.cancel()
  expandAnim = null
  const el = expandEl.value
  if (!el || prefersReducedMotion()) {
    expand.value = null
    return
  }
  const leave = el.animate(
    [
      { opacity: 1, transform: 'scale(1)' },
      { opacity: 0, transform: 'scale(1.04)' },
    ],
    {
      duration: 320,
      easing: 'cubic-bezier(0.22, 1, 0.36, 1)',
      fill: 'forwards',
    },
  )
  leave.onfinish = () => {
    leave.cancel()
    expand.value = null
  }
}

const waitForExpand = () =>
  new Promise<void>((resolve) => {
    if (typeof window === 'undefined') {
      resolve()
      return
    }
    if (prefersReducedMotion()) {
      resolve()
      return
    }
    // 等待展开层完成后再切详情页，避免页面切换截断翻书动作。
    window.setTimeout(resolve, EXPAND_DELAY_MS + EXPAND_DURATION_MS + 40)
  })

// 点击磁贴 = 进入应用：展开动画（翻转放大当前图）期间后台预取详情页数据，
// 动画播完直接切到游戏详情页（详情页自带启动入口，无弹窗打断）。
const handleTileSelect = async (publicId: string, rect?: DOMRect) => {
  const request = tileDetailRequests.begin()
  if (rect) {
    openExpand(publicId, rect)
  }
  const detailPromise = gamesService.getGameDetail(publicId).catch(() => null)
  await waitForExpand()
  if (!request.isCurrent()) return
  const detail = await detailPromise
  if (!request.isCurrent()) return
  // 详情返回后把正面图升级为游戏 banner/截图（翻转完成前替换，转正即显示内容图）
  if (detail && expand.value) {
    const frontImage =
      detail.banner_image ||
      (detail.screenshots && detail.screenshots.length > 0 ? detail.screenshots[0].path : '') ||
      expand.value.frontImage
    if (frontImage && frontImage !== expand.value.backImage) {
      const preload = new Image()
      preload.src = frontImage
      expand.value.backImage = frontImage
    }
  }
  // 点击磁贴的关闭跳过退场动画：展开动画已演出，直接切详情页
  skipOverlayLeave = true
  emit('close')
  // 开始屏幕是全局覆盖层：覆盖在游戏库上时保留筛选上下文，覆盖在其他页面上时统一回游戏库。
  const currentRoute = router.currentRoute.value
  const returnTo = currentRoute.name === 'games' ? currentRoute.fullPath : '/games'
  router.push({
    name: 'game-detail',
    params: { publicId },
    query: { [GAME_DETAIL_RETURN_QUERY]: returnTo },
  })
  closeExpand()
}

const handleBrowseGames = () => {
  emit('close')
  router.push({ name: 'games' })
}

const handleCrop = async (gameId: number) => {
  const request = cropDetailRequests.begin()
  cropGameId.value = gameId
  cropBanners.value = []
  const tile = props.tiles.find((item) => item.game_id === gameId)
  if (tile?.public_id) {
    try {
      const detail = await gamesService.getGameDetail(tile.public_id)
      if (!request.isCurrent()) return
      cropBanners.value = detail.banners.map((banner) => banner.path).filter((path): path is string => Boolean(path))
    } catch {
      // 拉取 banner 列表失败时仍可用磁贴默认 banner / 封面裁剪。
    }
  }
  if (!request.isCurrent()) return
  cropVisible.value = true
}

const handleCropConfirm = (blobs: Record<StartScreenTileSize, Blob>) => {
  if (cropGameId.value === null) return
  emit('applyCrop', cropGameId.value, blobs)
  cropVisible.value = false
  cropGameId.value = null
}

/* Win8 组横排：滚轮横向翻组 */
const handleWheel = (event: WheelEvent) => {
  const area = metroAreaRef.value
  if (!area) return
  area.scrollLeft += event.deltaY
}

const onTilePointerDown = (index: number, event: PointerEvent) => {
  if (!props.isEditing) return
  const tile = props.tiles[index]
  if (!tile) return
  if ((event.target as HTMLElement).closest('.metro-tile__action')) return
  pendingDrag = { gameId: tile.game_id, fromIndex: index }
  dragStart = { x: event.clientX, y: event.clientY }
  window.addEventListener('pointermove', onWindowPointerMove)
  window.addEventListener('pointerup', onWindowPointerUp)
  window.addEventListener('pointercancel', onWindowPointerCancel)
}

const onWindowPointerMove = (event: PointerEvent) => {
  if (!pendingDrag || !dragStart) return
  if (!dragState.value && Math.hypot(event.clientX - dragStart.x, event.clientY - dragStart.y) < 6) {
    return
  }
  if (!dragState.value) {
    const fromTile = props.tiles[pendingDrag.fromIndex]
    dragState.value = {
      gameId: pendingDrag.gameId,
      fromIndex: pendingDrag.fromIndex,
      targetColumnIndex: fromTile?.column_index ?? 0,
      targetRow: fromTile?.grid_row ?? 0,
      targetCol: fromTile?.grid_col ?? 0,
    }
  }
  dragPointer.value = { x: event.clientX, y: event.clientY }
  updateDragTarget(event.clientX, event.clientY)
  updateEdgeScroll(event.clientX)
}

const updateDragTarget = (x: number, y: number) => {
  if (!dragState.value) return
  const hit = document.elementFromPoint(x, y)
  const cell = hit?.closest?.('[data-start-screen-cell]') as HTMLElement | null
  if (!cell) return
  const rawColumn = cell.dataset.columnIndex
  const rawRow = cell.dataset.row
  const rawCol = cell.dataset.col
  const columnIndex = Number(rawColumn)
  const row = Number(rawRow)
  const col = Number(rawCol)
  const dragged = draggedTile.value
  if (
    !dragged ||
    !Number.isInteger(columnIndex) ||
    !Number.isInteger(row) ||
    !Number.isInteger(col) ||
    columnIndex < 0 ||
    row < 0 ||
    col < 0
  ) {
    return
  }
  const target = findStartScreenDropTarget(
    props.tiles,
    dragState.value.gameId,
    columnIndex,
    row,
    col,
    dragged.tile_size,
  )
  dragState.value.targetColumnIndex = target.columnIndex
  dragState.value.targetRow = target.row
  dragState.value.targetCol = target.col
}

const updateEdgeScroll = (x: number) => {
  const area = metroAreaRef.value
  if (!area) return
  const rect = area.getBoundingClientRect()
  const nearLeft = x < rect.left + 70
  const nearRight = x > rect.right - 70
  if (edgeScrollFrame !== null) {
    cancelAnimationFrame(edgeScrollFrame)
    edgeScrollFrame = null
  }
  if (!nearLeft && !nearRight) return
  const step = () => {
    if (!dragState.value) {
      edgeScrollFrame = null
      return
    }
    if (nearLeft) area.scrollLeft -= 14
    if (nearRight) area.scrollLeft += 14
    edgeScrollFrame = requestAnimationFrame(step)
  }
  edgeScrollFrame = requestAnimationFrame(step)
}

const onWindowPointerUp = () => {
  if (dragState.value) {
    emit(
      'applyPlacement',
      dragState.value.gameId,
      dragState.value.targetColumnIndex,
      dragState.value.targetRow,
      dragState.value.targetCol,
    )
  }
  endDrag()
}

const onWindowPointerCancel = () => {
  endDrag()
}

const endDrag = () => {
  dragState.value = null
  dragPointer.value = null
  pendingDrag = null
  dragStart = null
  if (edgeScrollFrame !== null) {
    cancelAnimationFrame(edgeScrollFrame)
    edgeScrollFrame = null
  }
  window.removeEventListener('pointermove', onWindowPointerMove)
  window.removeEventListener('pointerup', onWindowPointerUp)
  window.removeEventListener('pointercancel', onWindowPointerCancel)
}

// 入场动画与磁贴渲染解耦：磁贴是异步加载的，overlay 的 enter-active 类在加载完成前就可能
// 被 Vue 移除，动画永远赶不上。改为由组件显式控制：tiles 渲染完成后加 --entering 类触发
// 右滑入场，最晚磁贴动画结束后移除。
const entranceActive = ref(false)
let entranceTriggered = false
let entranceTimer: number | null = null

const scheduleEntrance = () => {
  if (entranceTriggered || !props.visible) return
  if (totalTiles.value === 0) return
  entranceTriggered = true
  void nextTick(() => {
    entranceActive.value = true
    playEntrance()
    const longestDelay =
      Math.max(visibleGroups.value.length - 1, 0) * 110 +
      Math.max(totalTiles.value - 1, 0) * 40
    entranceTimer = window.setTimeout(() => {
      entranceActive.value = false
      entranceTimer = null
    }, longestDelay + 400)
  })
}

// 入场动画（WAAPI）：Win8 克制上浮 + 淡入，磁贴/标题按各自 --start-tile-enter-delay 错峰
const playEntrance = () => {
  const wrapper = wrapperRef.value
  if (!wrapper || prefersReducedMotion()) return
  const elements = [
    ...Array.from(wrapper.querySelectorAll<HTMLElement>('.start-screen__header')),
    ...Array.from(wrapper.querySelectorAll<HTMLElement>('.start-screen__group-header')),
    ...Array.from(wrapper.querySelectorAll<HTMLElement>('.start-screen__tile-slot')),
  ]
  elements.forEach((el) => {
    const delay = parseFloat(el.style.getPropertyValue('--start-tile-enter-delay')) || 0
    el.animate(
      [
        { opacity: 0, transform: 'translate3d(10px, 10px, 0)' },
        { opacity: 1, transform: 'translate3d(0, 0, 0)' },
      ],
      { duration: 320, delay, easing: 'cubic-bezier(0.2, 0, 0.2, 1)', fill: 'both' },
    )
  })
}

watch(
  () => props.visible,
  (value) => {
    if (value) {
      if (!expand.value) resetExpandTileVisibility()
      scheduleEntrance()
    } else {
      resetExpandTileVisibility()
      entranceActive.value = false
      entranceTriggered = false
      if (entranceTimer !== null) {
        clearTimeout(entranceTimer)
        entranceTimer = null
      }
    }
  },
)

watch(
  () => props.tiles.length,
  () => {
    scheduleEntrance()
  },
)

const onTileEnter = (el: Element) => {
  const element = el as HTMLElement
  // 初次打开由 playEntrance 统一驱动；编辑中新增的磁贴单独播 WAAPI 入场
  if (entranceActive.value) return
  const delay = parseFloat(element.style.getPropertyValue('--start-tile-enter-delay')) || 0
  if (prefersReducedMotion()) return
  element.animate(
    [
      { opacity: 0, transform: 'translate3d(48px, 0, 0)' },
      { opacity: 1, transform: 'translate3d(0, 0, 0)' },
    ],
    { duration: 250, delay, easing: 'cubic-bezier(0, 0, 0.2, 1)', fill: 'both' },
  )
}

const onTileLeave = (el: Element) => {
  const element = el as HTMLElement
  if (prefersReducedMotion()) return
  element.animate(
    [{ opacity: 1 }, { opacity: 0 }],
    { duration: 167, easing: 'cubic-bezier(0.4, 0, 1, 1)', fill: 'both' },
  )
}

// 退场动画（WAAPI）：磁贴向回落 + 淡出（后进先出收拢），标题快速退场；
// 播完后调用 done 让 Vue Transition 移除 DOM。
// 点击磁贴进入游戏的关闭跳过退场动画（展开动画已演出，直接切详情页），
// 仅空白处/ESC/回到桌面退出时播放。
let skipOverlayLeave = false

const onOverlayLeave = (el: Element, done: () => void) => {
  if (skipOverlayLeave) {
    skipOverlayLeave = false
    done()
    return
  }
  const wrapper = el as HTMLElement
  let longest = 0
  if (!prefersReducedMotion()) {
    const slots = wrapper.querySelectorAll<HTMLElement>('.start-screen__tile-slot')
    slots.forEach((slot) => {
      const delay = parseFloat(slot.style.getPropertyValue('--start-tile-leave-delay')) || 0
      longest = Math.max(longest, delay + 280)
      slot.animate(
        [
          { opacity: 1, transform: 'translate3d(0, 0, 0)' },
          { opacity: 0, transform: 'translate3d(10px, 10px, 0)' },
        ],
        { duration: 280, delay, easing: 'cubic-bezier(0.4, 0, 1, 1)', fill: 'both' },
      )
    })
    const headers = wrapper.querySelectorAll<HTMLElement>('.start-screen__header, .start-screen__group-header')
    headers.forEach((header) => {
      header.animate(
        [
          { opacity: 1, transform: 'translate3d(0, 0, 0)' },
          { opacity: 0, transform: 'translate3d(10px, 10px, 0)' },
        ],
        { duration: 167, easing: 'cubic-bezier(0.4, 0, 1, 1)', fill: 'both' },
      )
    })
  }
  window.setTimeout(done, longest + 30)
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
  tileDetailRequests.invalidate()
  cropDetailRequests.invalidate()
  endDrag()
  if (entranceTimer !== null) {
    clearTimeout(entranceTimer)
    entranceTimer = null
  }
  resetExpandTileVisibility()
  if (typeof document !== 'undefined') {
    document.body.style.overflow = ''
  }
})
</script>

<style>
/* 开始屏幕是沉浸式品牌页特例：全屏场景色、Win8 磁贴网格与动效留在组件内，不外溢到全局 token。
   全屏自定义网格（Win8.1/Win10 形态）：组（列）只是顶部标签，磁贴在 12 列无限行的
   自由网格内摆放，组与组垂直堆叠；间距对齐 Win8 的 5px。 */
.start-screen-wrapper {
  position: fixed;
  inset: 0;
  z-index: 1500;
  outline: none;
  /* 不透明基底：与 app 全局底色一致，盖住下层页面；背景图层在其上叠加处理后的 bg */
  background: var(--color-bg-1, #0d1117);
}

/* 开始屏幕（1500）内部会打开裁剪/添加弹窗：Arco modal 默认 1000/1001，
   这里统一提到 1600，保证弹窗盖在开始屏幕之上、全局告警（2000）之下。
   注意只能限定 start-screen-modal（开始屏自己的弹窗），不能全局抬高：
   Arco select 下拉 popup 由内部管理器分配 z-index（默认 1001+），
   全局抬 .arco-modal-container 会把全站弹窗内的下拉压到弹窗底下。 */
.start-screen-modal.arco-modal-container {
  z-index: 1600 !important;
}

/* 展开动画由单一 Web Animations API 逻辑驱动（startExpandAnimation）。
   宿主提供透视，卡片本身只负责尺寸、旋转和内容；这样卡片旋转时会真实投影成侧面。 */
.start-screen-expand-host {
  position: fixed;
  inset: 0;
  z-index: 1700;
  pointer-events: none;
  perspective: 1400px;
}

.start-screen-expand {
  position: fixed;
  top: 0;
  left: 0;
  width: 100vw;
  height: 100vh;
  color: #fff;
  background: #0d1117;
  transform-origin: 50% 50%;
  will-change: transform, opacity, left, top, width, height;
  /* 不能有 overflow:hidden——它会压平 preserve-3d，使翻到侧面的面被裁成黑块。 */
  transform-style: preserve-3d;
}

/* 双面翻转：容器 rotateY -180°→0°，正面（预置 180°）先正对显示磁贴原图，
   翻转过程中换面，终点背面（不预置）正对显示展开大图，全程无镜像。
   换面完全由 backface-visibility 按真实 3D 朝向处理，无需额外黑屏/淡出逻辑。 */
.start-screen-expand__face {
  position: absolute;
  inset: 0;
  backface-visibility: hidden;
}

.start-screen-expand__face--front {
  transform: rotateY(180deg);
}

.start-screen-expand__cover {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.start-screen-expand__bg {
  position: absolute;
  inset: 0;
  background-size: cover;
  background-position: center;
}

.start-screen-expand__shade {
  position: absolute;
  inset: 0;
  background: linear-gradient(180deg, rgba(0, 0, 0, 0.12) 0%, rgba(0, 0, 0, 0.5) 100%);
}

.start-screen-expand__meta {
  position: absolute;
  left: 50%;
  bottom: 18%;
  transform: translateX(-50%);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  text-align: center;
  max-width: 72vw;
}

.start-screen-expand__meta h2 {
  margin: 0;
  font-size: 44px;
  font-weight: 700;
  letter-spacing: 0.08em;
  line-height: 1.1;
  text-shadow: 0 2px 14px rgba(0, 0, 0, 0.65);
}

.start-screen-expand__loading {
  display: flex;
  align-items: center;
  gap: 10px;
  color: rgba(255, 255, 255, 0.88);
  font-size: 14px;
}

.start-screen-scrim {
  position: absolute;
  inset: 0;
  /* 半透明遮罩：让开始屏幕透出当前全局背景（自定义 bg / 环境背景池），同时保证文字可读。
     背景本身已模糊（SharedAmbientBackground 静态 blur），无需 backdrop-filter */
  background: rgba(8, 10, 16, 0.46);
}

.start-screen {
  position: relative;
  height: 100%;
  display: flex;
  flex-direction: column;
  padding: 40px 56px 28px;
  color: #fff;
  /* 12 列网格 + Win10 磁贴比例（2x2/2x4/4x4）：单元约 4vw，
     2x2 小磁贴 ≈ Win8 的 125px 量级；组间横向滚动翻屏 */
  --start-cell: clamp(32px, 4vw, 60px);
  --start-gap: 5px;
}

.start-screen__header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 24px;
  margin-bottom: 16px;
}

.start-screen__heading {
  margin: 0;
  font-size: clamp(42px, 5vw, 58px);
  font-weight: 700;
  line-height: 1;
  letter-spacing: 0.1em;
}

.start-screen__header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.start-screen__admin-name {
  margin-right: 4px;
  font-size: 24px;
  font-weight: 600;
  line-height: 1;
}

.start-screen__edit-button {
  width: 36px;
  height: 36px;
  padding: 0;
  font-size: 24px;
}

.start-screen__save-error {
  margin: 0 0 12px;
  padding: 8px 12px;
  border-radius: 6px;
  background: rgba(255, 77, 79, 0.16);
  color: #ff7875;
  font-size: 13px;
}

.start-screen__metro {
  flex: 1;
  min-height: 0;
  overflow-x: auto;
  overflow-y: hidden;
  padding: 8px 4px 16px;
  scrollbar-width: thin;
}

.start-screen__groups {
  display: flex;
  flex-direction: row;
  align-items: flex-start;
  gap: 48px;
  width: max-content;
  transition: opacity 120ms ease;
}

/* 展开层接管磁贴后，背景网格淡出，避免其他磁贴穿过 3D 卡片露出来。 */
.start-screen-wrapper.is-expanding-tiles-hidden .start-screen__groups {
  opacity: 0;
  pointer-events: none;
}

.start-screen__group {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.start-screen__group-grid-row {
  display: flex;
  align-items: flex-start;
  gap: var(--start-gap);
}

.start-screen__group-grid {
  display: grid;
  /* 组宽按磁贴实际占用收缩（--start-group-cols，上限 12），磁贴少时紧凑排列 */
  grid-template-columns: repeat(var(--start-group-cols, 12), var(--start-cell));
  grid-auto-rows: var(--start-cell);
  gap: var(--start-gap);
  position: relative;
  width: max-content;
}

.start-screen__group-header {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.start-screen__group-name {
  flex: 1;
  min-width: 0;
  font-size: 14px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.78);
  text-shadow: 0 1px 3px rgba(0, 0, 0, 0.55);
  white-space: nowrap;
}

.start-screen__group-name-input {
  flex: 1;
  min-width: 0;
  box-sizing: border-box;
  padding: 6px 10px;
  border: 1px solid rgba(255, 255, 255, 0.35);
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.08);
  color: #fff;
  font-size: 14px;
  font-weight: 600;
  outline: none;
}

.start-screen__group-remove {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  padding: 0;
  border-radius: 6px;
}

.start-screen__group-name-input:focus {
  border-color: rgba(255, 255, 255, 0.75);
  background: rgba(255, 255, 255, 0.14);
}

.start-screen__group-tiles {
  display: contents;
}

.start-screen__drop-cell {
  position: relative;
  z-index: 0;
  min-width: 0;
  min-height: 0;
  border: 1px dashed rgba(255, 255, 255, 0.16);
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.04);
  transition: background 120ms ease, border-color 120ms ease, box-shadow 120ms ease;
}

.start-screen__drop-cell--target {
  border-color: rgba(255, 255, 255, 0.8);
  background: rgba(255, 255, 255, 0.16);
  box-shadow: inset 0 0 0 2px rgba(255, 255, 255, 0.45);
}

/* 组尾追加条：承接网格之外（磁贴下方新行）的拖放，替代向下延伸的虚空格 */
.start-screen__group-append {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  width: calc(var(--start-cell) * var(--start-group-cols, 2) + var(--start-gap) * (var(--start-group-cols, 2) - 1));
  height: var(--start-cell);
  box-sizing: border-box;
  border: 2px dashed rgba(255, 255, 255, 0.22);
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.03);
  color: rgba(255, 255, 255, 0.42);
  font-size: 12px;
  transition: background 120ms ease, border-color 120ms ease, color 120ms ease;
}

.start-screen__group-append--target {
  border-color: rgba(255, 255, 255, 0.85);
  background: rgba(255, 255, 255, 0.16);
  color: #fff;
}

/* 组尾横向追加条：承接网格右侧（新列）的拖放，拖入后组宽自动扩展 */
.start-screen__group-append-col {
  display: flex;
  align-items: center;
  justify-content: center;
  width: var(--start-cell);
  height: calc(var(--start-cell) * var(--start-group-rows, 1) + var(--start-gap) * (var(--start-group-rows, 1) - 1));
  box-sizing: border-box;
  border: 2px dashed rgba(255, 255, 255, 0.22);
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.03);
  color: rgba(255, 255, 255, 0.42);
  cursor: pointer;
  transition: background 120ms ease, border-color 120ms ease, color 120ms ease;
}

.start-screen__group-append-col:hover {
  border-color: rgba(255, 255, 255, 0.6);
  color: rgba(255, 255, 255, 0.85);
}

.start-screen__new-column {
  display: flex;
  flex-direction: row;
  align-items: center;
  justify-content: center;
  gap: 10px;
  width: calc(var(--start-cell) * 12 + var(--start-gap) * 11);
  min-height: 72px;
  border: 2px dashed rgba(255, 255, 255, 0.42);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.05);
  color: rgba(255, 255, 255, 0.72);
  cursor: pointer;
  font-size: 14px;
  transition: background 120ms ease, border-color 120ms ease, color 120ms ease, box-shadow 120ms ease;
}

.start-screen__new-column:hover,
.start-screen__new-column--target {
  border-color: rgba(255, 255, 255, 0.9);
  background: rgba(255, 255, 255, 0.14);
  color: #fff;
  box-shadow: inset 0 0 0 2px rgba(255, 255, 255, 0.45);
}

.start-screen__new-column .arco-icon {
  font-size: 28px;
}

.start-screen__tile-slot {
  position: relative;
  z-index: 1;
  min-width: 0;
  min-height: 0;
  /* EntranceThemeTransition：统一水平偏移 + 淡入，布局位置本身不参与动画。 */
  transition:
    opacity 250ms cubic-bezier(0, 0, 0.2, 1),
    transform 250ms cubic-bezier(0, 0, 0.2, 1);
}

.start-screen__tile-slot--dragging {
  opacity: 0;
  pointer-events: none;
}

.start-screen__tile-slot--target {
  outline: 3px solid rgba(255, 255, 255, 0.7);
  outline-offset: 2px;
  border-radius: 6px;
}

.start-screen__metro.is-editing .start-screen__tile-slot {
  /* 编辑拖拽时禁止触摸滚动，pointer 事件接管 */
  touch-action: none;
}

.start-screen__drag-ghost {
  position: fixed;
  z-index: 2000;
  pointer-events: none;
  transform: translate(-50%, -50%) scale(1.05);
  filter: drop-shadow(0 14px 28px rgba(0, 0, 0, 0.55));
  opacity: 0.92;
}

.start-screen__drag-ghost--small {
  width: calc(var(--start-cell) * 2 + var(--start-gap));
  height: calc(var(--start-cell) * 2 + var(--start-gap));
}

.start-screen__drag-ghost--wide {
  width: calc(var(--start-cell) * 4 + var(--start-gap) * 3);
  height: calc(var(--start-cell) * 2 + var(--start-gap));
}

.start-screen__drag-ghost--large {
  width: calc(var(--start-cell) * 4 + var(--start-gap) * 3);
  height: calc(var(--start-cell) * 4 + var(--start-gap) * 3);
}

.start-screen-launch-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.start-screen-launch-item {
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

.start-screen-launch-item:hover {
  background: var(--color-fill-3);
  border-color: var(--app-glass-border-hover);
}

.start-screen-launch-item__name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.start-screen-launch-item__action {
  font-size: 13px;
  color: var(--color-primary-6);
}

.start-screen__add-tile {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  width: var(--start-cell);
  height: var(--start-cell);
  flex-shrink: 0;
  border: 2px dashed rgba(255, 255, 255, 0.45);
  border-radius: 6px;
  color: rgba(255, 255, 255, 0.75);
  cursor: pointer;
  font-size: 13px;
  transition: border-color 120ms ease, color 120ms ease, background 120ms ease;
}

.start-screen__add-tile:hover {
  border-color: rgba(255, 255, 255, 0.85);
  color: #fff;
  background: rgba(255, 255, 255, 0.08);
}

/* 编辑增删磁贴：入场/退场已由 WAAPI（onTileEnter/onTileLeave）驱动，
   只保留 leave 时的绝对定位（防止磁贴离场时周围布局跳动） */
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

.start-screen-overlay-enter-active {
  /* 微软文档明确没有 enterPage；外层只负责桌面遮罩的淡入。 */
  transition: opacity 250ms cubic-bezier(0, 0, 0.2, 1);
}

.start-screen-overlay-enter-from {
  opacity: 0;
}

.start-screen-overlay-leave-active {
  /* 退场由内容层动画演出（磁贴右飞、背景渐暗、标题移出），
     外层不整体淡出——否则 167ms 全透明会盖住磁贴的飞出动画。 */
  transition: none;
}

.start-screen-scrim {
  /* 背景层比磁贴层更快定色：前 200ms 完成淡入 */
  transition: opacity 200ms ease;
}

.start-screen-overlay-enter-from .start-screen-scrim {
  opacity: 0;
}

.start-screen-overlay-leave-active .start-screen-scrim {
  opacity: 0;
}

/* 入场/退场/编辑增删动效已全部由 WAAPI 驱动（playEntrance / onOverlayLeave /
   onTileEnter / onTileLeave），此处只保留 overlay 与 scrim 的简单淡入淡出。 */

/* 性能/偏好容错：系统要求减少动效时直接瞬显。 */
@media (prefers-reduced-motion: reduce) {
  .start-screen-overlay-enter-active,
  .start-screen-overlay-leave-active,
  .start-screen-scrim {
    transition: none !important;
  }

  .start-screen-overlay-enter-from,
  .start-screen-overlay-leave-to {
    opacity: 1;
  }
}

@media (max-width: 768px) {
  .start-screen {
    padding: 28px 20px 20px;
  }

  .start-screen__heading {
    font-size: 34px;
  }
}
</style>
