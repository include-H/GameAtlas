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

        <div class="start-screen" @wheel.passive="handleWheel" @click.self="handleClose">
          <header class="start-screen__header">
            <div>
              <h1 class="start-screen__heading">开始</h1>
              <p class="start-screen__subtitle">
                {{ tiles.length > 0 ? `${tiles.length} 个磁贴` : '你的专属游戏磁贴' }}
              </p>
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
              <template v-else>
                <a-button class="app-text-action-btn" type="text" :disabled="isSaving" @click="emit('cancelEdit')">
                  取消
                </a-button>
                <a-button type="primary" :loading="isSaving" @click="emit('saveEdit')">
                  保存
                </a-button>
              </template>
            </div>
          </header>

          <p v-if="isEditing" class="start-screen__edit-hint">
            拖动排序 · 尺寸切换 · banner 裁剪 · × 移除 · + 添加
          </p>
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

          <div v-else-if="tiles.length === 0" class="start-screen__state">
            <icon-star />
            <p>{{ isEditing ? '还没有磁贴，点击 + 添加' : '还没有磁贴' }}</p>
            <a-space v-if="!isEditing">
              <a-button @click="handleBrowseGames">去游戏库逛逛</a-button>
              <a-button v-if="canEdit" type="primary" @click="emit('startEdit')">开始编辑</a-button>
            </a-space>
            <a-button v-else-if="canEdit" type="primary" @click="addVisible = true">添加磁贴</a-button>
          </div>

          <div
            v-else
            ref="metroAreaRef"
            class="start-screen__metro"
            @click.self="handleClose"
          >
            <div class="start-screen__columns">
              <div
                v-for="(column, columnIndex) in packedColumns"
                :key="columnIndex"
                class="start-screen__column"
              >
                <TransitionGroup
                  name="metro-tile"
                  tag="div"
                  class="start-screen__column-grid"
                  @enter="onTileEnter"
                  @leave="onTileLeave"
                >
                  <div
                    v-for="slot in column.slots"
                    :key="slot.tile.game_id"
                    :class="['start-screen__tile-slot', `start-screen__tile-slot--${slot.tile.tile_size}`]"
                    :style="{ gridColumnStart: slot.col + 1, gridRowStart: slot.row + 1 }"
                    :data-tile-index="slot.globalIndex"
                    :draggable="isEditing"
                    @dragstart="onDragStart(slot.globalIndex, $event)"
                    @dragover.prevent
                    @drop="onDrop(slot.globalIndex)"
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
                v-if="isEditing && tiles.length > 0"
                class="start-screen__add-tile"
                title="添加磁贴"
                @click="addVisible = true"
              >
                <icon-plus />
                <span>添加</span>
              </div>
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

  <a-modal
    :visible="addVisible"
    :footer="false"
    :width="520"
    title="添加磁贴（来自收藏）"
    @cancel="addVisible = false"
  >
    <div class="start-screen-add-list">
      <p v-if="addCandidates.length === 0" class="start-screen-add-list__empty">
        没有可添加的收藏了，先去游戏库收藏一些游戏吧
      </p>
      <button
        v-for="game in addCandidates"
        :key="game.public_id"
        type="button"
        class="start-screen-add-item"
        @click="handleAdd(game)"
      >
        <img
          v-if="game.cover_image"
          :src="game.cover_image"
          :alt="game.title"
          class="start-screen-add-item__cover"
          loading="lazy"
        >
        <span v-else class="start-screen-add-item__placeholder">{{ game.title.charAt(0) }}</span>
        <span class="start-screen-add-item__title">{{ game.title }}</span>
      </button>
    </div>
  </a-modal>

  <tile-crop-modal
    :visible="cropVisible"
    :image-src="cropSource"
    @confirm="handleCropConfirm"
    @cancel="cropVisible = false"
  />
</template>

<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import {
  IconDesktop,
  IconEdit,
  IconExclamationCircle,
  IconPlus,
  IconStar,
} from '@arco-design/web-vue/es/icon'
import MetroTile from './MetroTile.vue'
import TileCropModal from './TileCropModal.vue'
import type { GameListItem, StartScreenTile, StartScreenTileSize } from '@/services/types'
import { packStartScreenTiles } from '@/utils/start-screen-layout'

const props = defineProps<{
  visible: boolean
  tiles: StartScreenTile[]
  favoritePool: GameListItem[]
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
  move: [fromIndex: number, toIndex: number]
  add: [game: GameListItem]
  applyCrop: [gameId: number, blobs: Record<StartScreenTileSize, Blob>]
}>()

const router = useRouter()
const wrapperRef = ref<HTMLElement | null>(null)
const metroAreaRef = ref<HTMLElement | null>(null)
const addVisible = ref(false)
const cropVisible = ref(false)
const cropGameId = ref<number | null>(null)
const draggedIndex = ref<number | null>(null)

const cropSource = computed(() => {
  const tile = props.tiles.find((item) => item.game_id === cropGameId.value)
  return tile?.banner_image || tile?.cover_image || ''
})

const addCandidates = computed(() => {
  const pinned = new Set(props.tiles.map((tile) => tile.game_id))
  return props.favoritePool.filter((game) => !pinned.has(game.id))
})

const packedColumns = computed(() => packStartScreenTiles(props.tiles))

const handleClose = () => {
  emit('close')
}

const handleTileSelect = (publicId: string) => {
  emit('close')
  router.push({ name: 'game-detail', params: { publicId } })
}

const handleBrowseGames = () => {
  emit('close')
  router.push({ name: 'games' })
}

const handleAdd = (game: GameListItem) => {
  emit('add', game)
  addVisible.value = false
}

const handleCrop = (gameId: number) => {
  cropGameId.value = gameId
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

const onDragStart = (index: number, event: DragEvent) => {
  draggedIndex.value = index
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = 'move'
  }
}

const onDrop = (index: number) => {
  if (draggedIndex.value === null) return
  emit('move', draggedIndex.value, index)
  draggedIndex.value = null
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
  if (typeof document !== 'undefined') {
    document.body.style.overflow = ''
  }
})
</script>

<style>
/* 开始屏幕是沉浸式品牌页特例：全屏场景色、Win8 磁贴网格与动效留在组件内，不外溢到全局 token。
   列模板：一列 = 1 个大正方形（2x2）+ 2 个宽长方形（2x1）+ 4 个小正方形（1x1），
   列高 5 行、列宽 4 格，塞满就往右另开一列，列间距 50px。 */
.start-screen-wrapper {
  position: fixed;
  inset: 0;
  z-index: 1500;
  outline: none;
}

/* 开始屏幕（1500）内部会打开裁剪/添加弹窗：Arco modal 默认 1000/1001，
   这里统一提到 1600，保证弹窗盖在开始屏幕之上、全局告警（2000）之下。 */
.arco-modal-container {
  z-index: 1600 !important;
}

.start-screen-scrim {
  position: absolute;
  inset: 0;
  background: rgba(8, 10, 16, 0.76);
  backdrop-filter: blur(6px);
}

.start-screen {
  position: relative;
  height: 100%;
  display: flex;
  flex-direction: column;
  padding: 40px 56px 28px;
  color: #fff;
  /* 列模板：一列 5 行高（大 2 行 + 宽 2 行 + 小 1 行），行高随视口高度自适应，720p 可整列放下 */
  --start-cell: clamp(72px, 11vh, 110px);
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

.start-screen__subtitle {
  margin: 8px 0 0;
  font-size: 14px;
  opacity: 0.65;
}

.start-screen__header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.start-screen__edit-hint {
  margin: 0 0 12px;
  font-size: 13px;
  opacity: 0.7;
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
  grid-template-columns: repeat(4, var(--start-cell));
  grid-template-rows: repeat(5, var(--start-cell));
  gap: var(--start-gap);
  width: max-content;
}

.start-screen__tile-slot {
  min-width: 0;
  min-height: 0;
  /* Win8.1 磁贴层：上浮 10px 弹入（back-out 轻微过冲）+ 淡入 */
  transition:
    opacity 280ms ease,
    transform 340ms cubic-bezier(0.34, 1.56, 0.64, 1);
}

.start-screen__tile-slot--small {
  grid-column: span 1;
  grid-row: span 1;
}

.start-screen__tile-slot--wide {
  grid-column: span 2;
  grid-row: span 1;
}

.start-screen__tile-slot--large {
  grid-column: span 2;
  grid-row: span 2;
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

.start-screen-add-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 60vh;
  overflow-y: auto;
}

.start-screen-add-list__empty {
  margin: 0;
  padding: 24px 0;
  text-align: center;
  color: var(--color-text-3);
}

.start-screen-add-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--color-text-1);
  cursor: pointer;
  text-align: left;
  transition: background 120ms ease;
}

.start-screen-add-item:hover {
  background: var(--color-fill-2);
}

.start-screen-add-item__cover,
.start-screen-add-item__placeholder {
  width: 44px;
  height: 44px;
  border-radius: 6px;
  object-fit: cover;
  flex-shrink: 0;
}

.start-screen-add-item__placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-fill-2);
  color: var(--color-text-2);
  font-weight: 600;
}

.start-screen-add-item__title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
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
