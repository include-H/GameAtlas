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
        <shared-ambient-background force-enabled />
        <div class="start-screen-scrim" @click="handleClose" />

        <div class="start-screen" @wheel.passive="handleWheel" @click.self="handleClose">
          <header class="start-screen__header">
            <div>
              <h1 class="start-screen__heading">开始</h1>
            </div>

            <div class="start-screen__header-actions">
              <template v-if="!isEditing && canEdit">
                <a-button
                  class="app-text-action-btn"
                  type="text"
                  @click="emit('startEdit')"
                >
                  <template #icon><icon-edit /></template>
                  编辑磁贴
                </a-button>
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
            <div class="start-screen__columns">
              <div
                v-for="(column, columnIndex) in visibleColumns"
                :key="columnIndex"
                class="start-screen__column"
              >
                <div class="start-screen__column-header">
                  <input
                    v-if="isEditing"
                    class="start-screen__column-name-input"
                    :value="columnNameOf(columnIndex)"
                    :placeholder="`列 ${columnIndex + 1}`"
                    @change="handleRenameColumn(columnIndex, $event)"
                  >
                  <span v-else class="start-screen__column-name">{{ columnNameOf(columnIndex) }}</span>
                  <button
                    v-if="isEditing && !columnHasTiles(columnIndex)"
                    class="app-text-action-btn start-screen__column-remove"
                    type="button"
                    :aria-label="`删除空列 ${columnIndex + 1}`"
                    @click="emit('removeColumn', columnIndex)"
                  >
                    <icon-close />
                  </button>
                </div>

                <div class="start-screen__column-grid">
                  <template v-if="isEditing">
                    <div
                      v-for="cell in gridCells"
                      :key="`cell-${cell.row}-${cell.col}`"
                      class="start-screen__drop-cell"
                      :class="{ 'start-screen__drop-cell--target': isDropTarget(columnIndex, cell.row, cell.col) }"
                      :data-start-screen-cell="true"
                      :data-column-index="columnIndex"
                      :data-row="cell.row"
                      :data-col="cell.col"
                      :style="{ gridColumnStart: cell.col + 1, gridRowStart: cell.row + 1 }"
                    />
                  </template>
                  <TransitionGroup
                    name="metro-tile"
                    tag="div"
                    class="start-screen__column-tiles"
                    @enter="onTileEnter"
                    @leave="onTileLeave"
                  >
                    <div
                      v-for="slot in column.slots"
                      :key="slot.tile.game_id"
                      :class="[
                        'start-screen__tile-slot',
                        `start-screen__tile-slot--${slot.tile.tile_size}`,
                        { 'start-screen__tile-slot--dragging': isDraggedTile(slot.tile) },
                        { 'start-screen__tile-slot--target': isDropTarget(columnIndex, slot.row, slot.col) },
                      ]"
                      :style="{ gridColumnStart: slot.col + 1, gridRowStart: slot.row + 1 }"
                      :data-tile-index="slot.globalIndex"
                      :data-start-screen-cell="true"
                      :data-column-index="columnIndex"
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
              </div>

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

  <a-modal
    class="start-screen-modal"
    :visible="launchModalVisible"
    :footer="false"
    :width="480"
    :title="`开始游戏：${launchTitle}`"
    @cancel="launchModalVisible = false"
  >
    <div class="start-screen-launch-list">
      <button
        v-for="option in launchOptions"
        :key="option.id"
        type="button"
        class="start-screen-launch-item"
        @click="handleLaunchVersion(option)"
      >
        <icon-play-arrow />
        <span class="start-screen-launch-item__name">{{ option.version }}</span>
        <span class="start-screen-launch-item__action">开始游戏</span>
      </button>
    </div>
  </a-modal>
</template>

<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import {
  IconClose,
  IconDesktop,
  IconEdit,
  IconExclamationCircle,
  IconPlayArrow,
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
  START_SCREEN_COLUMN_COLS,
  START_SCREEN_COLUMN_ROWS,
} from '@/utils/start-screen-layout'
import gamesService, { mapGameVersions } from '@/services/games.service'

const props = defineProps<{
  visible: boolean
  tiles: StartScreenTile[]
  columns: StartScreenColumn[]
  canEdit: boolean
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
const launchModalVisible = ref(false)
const launchTitle = ref('')
const launchOptions = ref<Array<{ id: string; version: string; url: string }>>([])

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

const layoutColumns = computed(() => layoutStartScreenTiles(displayTiles.value, props.columns.length))
const visibleColumns = computed(() => {
  const columns = layoutColumns.value
  if (!props.isEditing) return columns.filter((column) => column.slots.length > 0)
  if (props.tiles.length === 0 && props.columns.length === 0) return []
  return columns
})
const gridCells = Array.from({ length: START_SCREEN_COLUMN_ROWS * START_SCREEN_COLUMN_COLS }, (_, index) => ({
  row: Math.floor(index / START_SCREEN_COLUMN_COLS),
  col: index % START_SCREEN_COLUMN_COLS,
}))
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
  return name || `列 ${index + 1}`
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

// 点击磁贴 = 开始游戏：单个可启动版本直接下载启动脚本，多个弹窗选择，无则回退详情页。
const handleTileSelect = async (publicId: string) => {
  emit('close')
  try {
    const detail = await gamesService.getGameDetail(publicId)
    const launchable = mapGameVersions(detail).filter((version) => version.canLaunch && version.launchScriptUrl)
    if (launchable.length === 0) {
      router.push({ name: 'game-detail', params: { publicId } })
      return
    }
    if (launchable.length === 1) {
      window.location.assign(launchable[0].launchScriptUrl!)
      return
    }
    launchTitle.value = detail.title
    launchOptions.value = launchable.map((version) => ({
      id: version.id,
      version: version.version,
      url: version.launchScriptUrl!,
    }))
    launchModalVisible.value = true
  } catch {
    // 拉取失败时回退到详情页，不阻塞用户。
    router.push({ name: 'game-detail', params: { publicId } })
  }
}

const handleLaunchVersion = (option: { id: string; version: string; url: string }) => {
  launchModalVisible.value = false
  window.location.assign(option.url)
}

const handleBrowseGames = () => {
  emit('close')
  router.push({ name: 'games' })
}

const handleCrop = async (gameId: number) => {
  cropGameId.value = gameId
  cropBanners.value = []
  const tile = props.tiles.find((item) => item.game_id === gameId)
  if (tile?.public_id) {
    try {
      const detail = await gamesService.getGameDetail(tile.public_id)
      cropBanners.value = detail.banners.map((banner) => banner.path).filter((path): path is string => Boolean(path))
    } catch {
      // 拉取 banner 列表失败时仍可用磁贴默认 banner / 封面裁剪。
    }
  }
  cropVisible.value = true
}

const handleCropConfirm = (blobs: Record<StartScreenTileSize, Blob>) => {
  if (cropGameId.value === null) return
  emit('applyCrop', cropGameId.value, blobs)
  cropVisible.value = false
  cropGameId.value = null
}

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

const onTileEnter = (el: Element) => {
  const element = el as HTMLElement
  const index = Number(element.dataset.tileIndex)
  if (Number.isNaN(index)) return
  // Win8.1 磁贴层：延迟 50ms 起按序淡入，轻微上浮弹入由 CSS 过渡完成。
  element.style.transitionDelay = `${50 + Math.min(index, 24) * 30}ms`
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
  endDrag()
  if (typeof document !== 'undefined') {
    document.body.style.overflow = ''
  }
})
</script>

<style>
/* 开始屏幕是沉浸式品牌页特例：全屏场景色、Win8 磁贴网格与动效留在组件内，不外溢到全局 token。
   磁贴自由排列：列只是 2 格宽 × 6 行高的容器（约三个 2x2 大正方形占地），
   按顺序行优先塞入，当前列放不下就向右另开一列，列间距 50px。 */
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

.start-screen-scrim {
  position: absolute;
  inset: 0;
  /* 半透明遮罩：让开始屏幕透出当前全局背景（自定义 bg / 环境背景池），同时保证文字可读 */
  background: rgba(8, 10, 16, 0.46);
  backdrop-filter: blur(4px);
}

.start-screen {
  position: relative;
  height: 100%;
  display: flex;
  flex-direction: column;
  padding: 40px 56px 28px;
  color: #fff;
  /* 一列 = 三个大正方形占地纵向堆叠（大 2x2 + 两个长 2x1 + 四个 1x1 的 2x2），
     列高 6 行、列宽 2 格；行高随视口自适应，720p 可整列放下 */
  --start-cell: clamp(56px, 8.5vh, 96px);
  --start-gap: clamp(8px, 1.2vh, 14px);
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

.start-screen__columns {
  display: flex;
  align-items: flex-start;
  gap: 50px;
  width: max-content;
}

.start-screen__column-grid {
  display: grid;
  grid-template-columns: repeat(2, var(--start-cell));
  grid-template-rows: repeat(6, var(--start-cell));
  gap: var(--start-gap);
  position: relative;
  width: max-content;
}

.start-screen__column {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.start-screen__column-header {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.start-screen__column-name {
  flex: 1;
  min-width: 0;
  font-size: 14px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.78);
  text-shadow: 0 1px 3px rgba(0, 0, 0, 0.55);
  white-space: nowrap;
}

.start-screen__column-name-input {
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

.start-screen__column-remove {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  padding: 0;
  border-radius: 6px;
}

.start-screen__column-name-input:focus {
  border-color: rgba(255, 255, 255, 0.75);
  background: rgba(255, 255, 255, 0.14);
}

.start-screen__column-tiles {
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

.start-screen__new-column {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  flex-shrink: 0;
  width: calc(var(--start-cell) * 2 + var(--start-gap));
  min-height: calc(var(--start-cell) * 6 + var(--start-gap) * 5);
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
  /* Win8.1 磁贴层：上浮 10px 弹入（back-out 轻微过冲）+ 淡入 */
  transition:
    opacity 280ms ease,
    transform 340ms cubic-bezier(0.34, 1.56, 0.64, 1);
}

.start-screen__tile-slot--small {
  grid-column-end: span 1;
  grid-row-end: span 1;
}

.start-screen__tile-slot--wide {
  grid-column-end: span 2;
  grid-row-end: span 1;
}

.start-screen__tile-slot--large {
  grid-column-end: span 2;
  grid-row-end: span 2;
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
  width: var(--start-cell);
  height: var(--start-cell);
}

.start-screen__drag-ghost--wide {
  width: calc(var(--start-cell) * 2 + var(--start-gap));
  height: var(--start-cell);
}

.start-screen__drag-ghost--large {
  width: calc(var(--start-cell) * 2 + var(--start-gap));
  height: calc(var(--start-cell) * 2 + var(--start-gap));
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

.metro-tile-enter-from,
.metro-tile-leave-to {
  opacity: 0;
  transform: translateY(10px) scale(0.96);
}

.metro-tile-enter-to {
  opacity: 1;
  transform: translateY(0) scale(1);
}

.metro-tile-leave-active {
  /* 退出时磁贴先快速淡出并下压 */
  transition:
    opacity 140ms ease,
    transform 160ms cubic-bezier(0.4, 0, 1, 1);
}

.metro-tile-leave-active {
  position: absolute;
}

.metro-tile-leave-to {
  opacity: 0;
  transform: translateY(6px) scale(0.97);
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
  /* Win8.1：从"中心偏下约 1/3"的锚点由 88% 放大到 100%，ease-out 先快后慢 */
  transition:
    opacity 350ms ease,
    transform 350ms cubic-bezier(0.22, 1, 0.36, 1);
  transform-origin: 50% 66%;
}

.start-screen-overlay-enter-from {
  opacity: 0;
  transform: scale(0.88);
}

.start-screen-overlay-leave-active {
  /* 退出：整屏向锚点"吸回" */
  transition:
    opacity 260ms ease,
    transform 260ms cubic-bezier(0.4, 0, 0.6, 1);
  transform-origin: 50% 66%;
}

.start-screen-overlay-leave-to {
  opacity: 0;
  transform: scale(0.94);
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

.start-screen-overlay-enter-active .start-screen__header {
  animation: start-screen-header-in 420ms cubic-bezier(0.22, 1, 0.36, 1) 60ms both;
}

@keyframes start-screen-header-in {
  from {
    opacity: 0;
    transform: translateY(-14px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* 性能/偏好容错：系统要求减少动效时直接瞬显，保留缩放节奏兜底 */
@media (prefers-reduced-motion: reduce) {
  .start-screen-overlay-enter-active,
  .start-screen-overlay-leave-active,
  .start-screen__tile-slot,
  .start-screen-scrim,
  .start-screen__header {
    transition: none !important;
    animation: none !important;
  }

  .start-screen-overlay-enter-from,
  .start-screen-overlay-leave-to,
  .metro-tile-enter-from,
  .metro-tile-leave-to {
    opacity: 1;
    transform: none;
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
