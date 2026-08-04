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

          <div v-else ref="metroAreaRef" class="start-screen__metro">
            <TransitionGroup
              name="metro-tile"
              tag="div"
              class="start-screen__metro-grid"
              @enter="onTileEnter"
              @leave="onTileLeave"
            >
              <div
                v-for="(tile, index) in tiles"
                :key="tile.game_id"
                :class="['start-screen__tile-slot', `start-screen__tile-slot--${tile.tile_size}`]"
                :data-tile-index="index"
                :draggable="isEditing"
                @dragstart="onDragStart(index, $event)"
                @dragover.prevent
                @drop="onDrop(index)"
              >
                <MetroTile
                  :tile="tile"
                  :color-index="index"
                  :editing="isEditing"
                  @select="handleTileSelect"
                  @crop="handleCrop"
                  @resize="emit('resize', $event)"
                  @remove="emit('remove', $event)"
                />
              </div>

              <div
                v-if="isEditing"
                class="start-screen__tile-slot start-screen__tile-slot--small start-screen__add-tile"
                title="添加磁贴"
                @click="addVisible = true"
              >
                <icon-plus />
                <span>添加</span>
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
/* 开始屏幕是沉浸式品牌页特例：全屏场景色、Win8 磁贴网格与动效留在组件内，不外溢到全局 token。
   720p（1280x720）基准：磁贴区高约 482px，单元格 110px、间距 14px → 竖着 4 行；
   大磁贴 2x2 占 2 行，因此 720p 竖着可放 2 个大磁贴或 4 个小磁贴。 */
.start-screen-wrapper {
  position: fixed;
  inset: 0;
  z-index: 1500;
  outline: none;
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

.start-screen__metro-grid {
  display: grid;
  grid-template-rows: repeat(4, 110px);
  grid-auto-flow: column;
  grid-auto-columns: 110px;
  gap: 14px;
  width: max-content;
}

.start-screen__tile-slot {
  min-width: 0;
  min-height: 0;
  /* Win8 磁贴入场：back-out 回弹曲线，配合 onTileEnter 的错峰 delay */
  transition:
    opacity 260ms ease,
    transform 480ms cubic-bezier(0.34, 1.56, 0.64, 1);
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
  transform: translateY(28px) scale(0.45);
}

.metro-tile-enter-to {
  opacity: 1;
  transform: translateY(0) scale(1);
}

.metro-tile-leave-active {
  transition:
    opacity 160ms ease,
    transform 160ms ease;
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

.start-screen-overlay-enter-active {
  transition:
    opacity 260ms ease,
    transform 340ms cubic-bezier(0.22, 1, 0.36, 1);
}

.start-screen-overlay-enter-from {
  opacity: 0;
  transform: scale(0.96);
}

.start-screen-overlay-leave-active {
  transition: opacity 180ms ease;
}

.start-screen-overlay-leave-to {
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

@media (min-height: 900px) {
  .start-screen__metro-grid {
    grid-template-rows: repeat(5, 110px);
  }
}

@media (min-height: 1080px) {
  .start-screen__metro-grid {
    grid-template-rows: repeat(6, 110px);
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
