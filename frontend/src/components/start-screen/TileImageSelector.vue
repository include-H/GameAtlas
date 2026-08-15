<template>
  <a-modal
    class="start-screen-modal"
    :visible="visible"
    title="选择磁贴图片"
    :width="760"
    :mask-closable="false"
    :footer="false"
    @cancel="emit('cancel')"
  >
    <div class="tile-image-selector">
      <div class="tile-image-selector__main">
        <div class="tile-image-selector__stage">
          <div
            ref="stageRef"
            class="tile-image-selector__canvas"
            @pointerdown="onPointerDown"
          >
            <img
              v-if="activeImage"
              :src="activeImage"
              :style="imageStyle"
              class="tile-image-selector__image"
              draggable="false"
            >
            <span v-else class="tile-image-selector__empty">选择一张图片</span>
            <span v-if="activeImage" class="tile-image-selector__reticle" :style="reticleStyle">
              <span class="tile-image-selector__reticle-h" />
              <span class="tile-image-selector__reticle-v" />
            </span>
          </div>

          <div class="tile-image-selector__previews">
            <div class="tile-image-selector__preview">
              <span class="tile-image-selector__preview-label">大磁贴 4x4</span>
              <div class="tile-image-selector__preview-box tile-image-selector__preview-box--square">
                <img v-if="activeImage" :src="activeImage" :style="imageStyle" alt="大磁贴预览" />
                <span v-else class="tile-image-selector__preview-empty">—</span>
              </div>
            </div>
            <div class="tile-image-selector__preview">
              <span class="tile-image-selector__preview-label">宽磁贴 2x4</span>
              <div class="tile-image-selector__preview-box tile-image-selector__preview-box--wide">
                <img v-if="activeImage" :src="activeImage" :style="imageStyle" alt="宽磁贴预览" />
                <span v-else class="tile-image-selector__preview-empty">—</span>
              </div>
            </div>
          </div>
          <p class="tile-image-selector__hint">大/小磁贴共用方形裁切，宽磁贴单独裁切；拖动 / 点击图片设置焦点</p>

          <div v-if="maxFlipImages > 0" class="tile-image-selector__flip">
            <span class="tile-image-selector__group-label">
              轮播帧 {{ flipSelection.length }}/{{ maxFlipImages }}
            </span>
            <div class="tile-image-selector__flip-list">
              <div v-for="(path, index) in flipSelection" :key="path" class="tile-image-selector__flip-item">
                <span class="tile-image-selector__flip-index">{{ index + 2 }}</span>
                <img :src="path" alt="" />
                <button
                  type="button"
                  class="tile-image-selector__flip-remove"
                  :title="`移除轮播帧 ${index + 2}`"
                  @click="removeFlip(path)"
                >
                  <icon-close />
                </button>
              </div>
              <span v-if="flipSelection.length === 0" class="tile-image-selector__flip-empty">未选轮播帧，磁贴保持静态</span>
            </div>
            <p class="tile-image-selector__flip-hint">
              第 1 帧是下方主图，点缩略图右上角的 + 添加轮播帧（可再点取消），最多 {{ maxFlipImages }} 张。
              仅 2x4 宽磁贴会轮播翻动，方形磁贴先选好、切到宽磁贴后生效
            </p>
          </div>
        </div>

        <div class="tile-image-selector__candidates">
          <template v-for="group in candidateGroups" :key="group.type">
            <div v-if="group.items.length > 0" class="tile-image-selector__group">
              <span class="tile-image-selector__group-label">{{ group.label }}</span>
              <div class="tile-image-selector__thumbs">
                <button
                  v-for="item in group.items"
                  :key="item"
                  type="button"
                  :class="[
                    'tile-image-selector__thumb',
                    { 'tile-image-selector__thumb--active': activeImage === item },
                    { 'tile-image-selector__thumb--flip': isFlipSelected(item) },
                  ]"
                  :title="item"
                  @click="selectImage(item)"
                >
                  <img :src="item" alt="" />
                  <span
                    v-if="maxFlipImages > 0 && item !== activeImage"
                    role="button"
                    tabindex="-1"
                    class="tile-image-selector__thumb-flip-toggle"
                    :title="isFlipSelected(item) ? '从轮播帧移除' : '添加到轮播帧'"
                    @click.stop="toggleFlip(item)"
                  >
                    <icon-plus v-if="!isFlipSelected(item)" />
                    <icon-check v-else />
                  </span>
                </button>
              </div>
            </div>
          </template>
          <p v-if="candidateCount === 0" class="tile-image-selector__none">该游戏还没有横幅 / 截图素材</p>
        </div>
      </div>

      <div class="tile-image-selector__actions">
        <a-button class="app-text-action-btn" type="text" html-type="button" @click="emit('cancel')">取消</a-button>
        <a-button type="primary" html-type="button" :disabled="!activeImage" @click="handleConfirm">确定</a-button>
      </div>
    </div>
  </a-modal>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { IconCheck, IconClose, IconPlus } from '@arco-design/web-vue/es/icon'

export interface TileImageCandidateGroup {
  type: 'cover' | 'banner' | 'screenshot'
  label: string
  items: string[]
}

const props = withDefaults(defineProps<{
  visible: boolean
  image: string
  focusX: number
  focusY: number
  /** 已选的轮播追加帧（宽磁贴活磁贴；方形磁贴传空）。 */
  flipImages?: string[]
  /** 轮播追加帧上限（0 = 不启用活磁贴）。 */
  maxFlipImages?: number
  candidates: TileImageCandidateGroup[]
}>(), {
  image: '',
  focusX: 50,
  focusY: 50,
  flipImages: () => [],
  maxFlipImages: 0,
  candidates: () => [],
})

const emit = defineEmits<{
  confirm: [imagePath: string, focusX: number, focusY: number, flipImages: string[]]
  cancel: []
}>()

const activeImage = ref('')
const focusX = ref(50)
const focusY = ref(50)
const flipSelection = ref<string[]>([])
const stageRef = ref<HTMLElement | null>(null)

watch(() => props.visible, (value) => {
  if (!value) return
  activeImage.value = props.image
  focusX.value = Math.max(0, Math.min(100, props.focusX))
  focusY.value = Math.max(0, Math.min(100, props.focusY))
  flipSelection.value = props.flipImages.filter((path) => path !== props.image)
})

const candidateGroups = computed(() =>
  props.candidates.filter((group) => group.items.length > 0),
)
const candidateCount = computed(() =>
  props.candidates.reduce((count, group) => count + group.items.length, 0),
)

const imageStyle = computed(() => ({
  objectPosition: `${focusX.value}% ${focusY.value}%`,
}))
const reticleStyle = computed(() => ({
  left: `${focusX.value}%`,
  top: `${focusY.value}%`,
}))

const selectImage = (path: string) => {
  activeImage.value = path
}

const onPointerDown = (event: PointerEvent) => {
  const stage = stageRef.value
  if (!stage) return
  const rect = stage.getBoundingClientRect()
  const apply = (clientX: number, clientY: number) => {
    const x = ((clientX - rect.left) / rect.width) * 100
    const y = ((clientY - rect.top) / rect.height) * 100
    focusX.value = Math.max(0, Math.min(100, x))
    focusY.value = Math.max(0, Math.min(100, y))
  }
  apply(event.clientX, event.clientY)

  const onMove = (ev: PointerEvent) => apply(ev.clientX, ev.clientY)
  const onUp = () => {
    window.removeEventListener('pointermove', onMove)
    window.removeEventListener('pointerup', onUp)
  }
  window.addEventListener('pointermove', onMove)
  window.addEventListener('pointerup', onUp)
}

const handleConfirm = () => {
  if (!activeImage.value) return
  emit('confirm', activeImage.value, Math.round(focusX.value), Math.round(focusY.value), [...flipSelection.value])
}

const isFlipSelected = (path: string) => flipSelection.value.includes(path)

const toggleFlip = (path: string) => {
  if (path === activeImage.value) return
  if (isFlipSelected(path)) {
    flipSelection.value = flipSelection.value.filter((item) => item !== path)
    return
  }
  if (flipSelection.value.length >= props.maxFlipImages) return
  flipSelection.value = [...flipSelection.value, path]
}

const removeFlip = (path: string) => {
  flipSelection.value = flipSelection.value.filter((item) => item !== path)
}
</script>

<style scoped>
.tile-image-selector {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.tile-image-selector__main {
  display: flex;
  align-items: flex-start;
  gap: 14px;
}

.tile-image-selector__stage {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.tile-image-selector__canvas {
  position: relative;
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid var(--color-border-2);
  background: var(--color-fill-2);
  aspect-ratio: 16 / 9;
  cursor: crosshair;
  user-select: none;
  touch-action: none;
}

.tile-image-selector__image {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
  pointer-events: none;
}

.tile-image-selector__empty {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text-3);
  font-size: 14px;
}

.tile-image-selector__reticle {
  position: absolute;
  width: 22px;
  height: 22px;
  transform: translate(-50%, -50%);
  pointer-events: none;
}

.tile-image-selector__reticle-h,
.tile-image-selector__reticle-v {
  position: absolute;
  background: rgba(255, 255, 255, 0.92);
  box-shadow: 0 0 4px rgba(0, 0, 0, 0.7);
}

.tile-image-selector__reticle-h {
  left: 0;
  right: 0;
  top: 50%;
  height: 2px;
}

.tile-image-selector__reticle-v {
  top: 0;
  bottom: 0;
  left: 50%;
  width: 2px;
}

.tile-image-selector__previews {
  display: grid;
  grid-template-columns: 1fr 2fr;
  gap: 10px;
  align-items: start;
}

.tile-image-selector__preview {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.tile-image-selector__preview-label {
  font-size: 12px;
  color: var(--color-text-3);
}

.tile-image-selector__preview-box {
  position: relative;
  width: 100%;
  border-radius: 6px;
  overflow: hidden;
  background: var(--color-fill-2);
  border: 1px solid var(--color-border-2);
}

.tile-image-selector__preview-box--square {
  aspect-ratio: 1;
}

.tile-image-selector__preview-box--wide {
  aspect-ratio: 2 / 1;
}

.tile-image-selector__preview-box img {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.tile-image-selector__preview-empty {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text-3);
  font-size: 12px;
}

.tile-image-selector__hint {
  margin: 0;
  font-size: 12px;
  color: var(--color-text-3);
}

.tile-image-selector__candidates {
  flex: 1;
  min-width: 220px;
  max-height: 400px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.tile-image-selector__group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.tile-image-selector__group-label {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-2);
}

.tile-image-selector__thumbs {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(96px, 1fr));
  gap: 8px;
}

.tile-image-selector__thumb {
  position: relative;
  aspect-ratio: 16 / 9;
  padding: 0;
  border: 2px solid transparent;
  border-radius: 6px;
  overflow: hidden;
  cursor: pointer;
  background: var(--color-fill-2);
  opacity: 0.72;
  transition: border-color var(--transition-fast), opacity var(--transition-fast);
}

.tile-image-selector__thumb img {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.tile-image-selector__thumb:hover {
  opacity: 0.9;
}

.tile-image-selector__thumb--active,
.tile-image-selector__thumb--active:hover {
  border-color: var(--color-primary-6);
  opacity: 1;
}

.tile-image-selector__none {
  margin: 0;
  font-size: 13px;
  color: var(--color-text-3);
}

.tile-image-selector__flip {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.tile-image-selector__flip-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.tile-image-selector__flip-item {
  position: relative;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 6px 4px 4px;
  border-radius: 6px;
  border: 1px solid var(--color-border-2);
  background: var(--color-fill-2);
}

.tile-image-selector__flip-item img {
  width: 48px;
  height: 24px;
  object-fit: cover;
  border-radius: 4px;
}

.tile-image-selector__flip-index {
  font-size: 12px;
  color: var(--color-text-2);
  min-width: 14px;
  text-align: center;
}

.tile-image-selector__flip-remove {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  padding: 0;
  border: none;
  border-radius: 50%;
  cursor: pointer;
  color: var(--color-text-3);
  background: transparent;
}

.tile-image-selector__flip-remove:hover {
  color: var(--color-danger-6);
  background: var(--color-fill-3);
}

.tile-image-selector__flip-empty {
  font-size: 12px;
  color: var(--color-text-3);
}

.tile-image-selector__flip-hint {
  margin: 0;
  font-size: 12px;
  color: var(--color-text-3);
}

.tile-image-selector__thumb-flip-toggle {
  position: absolute;
  top: 4px;
  right: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  cursor: pointer;
  color: #fff;
  background: rgba(0, 0, 0, 0.62);
  transition: background var(--transition-fast);
}

.tile-image-selector__thumb-flip-toggle:hover {
  background: rgba(0, 0, 0, 0.85);
}

.tile-image-selector__thumb--flip {
  border-color: rgb(var(--primary-6));
}

.tile-image-selector__thumb--flip:hover {
  border-color: rgb(var(--primary-6));
}

.tile-image-selector__actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

@media (max-width: 768px) {
  .tile-image-selector__main {
    flex-direction: column;
  }

  .tile-image-selector__candidates {
    max-height: 220px;
    width: 100%;
  }
}
</style>
