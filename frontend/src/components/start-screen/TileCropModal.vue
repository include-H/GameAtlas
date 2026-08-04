<template>
  <a-modal
    :visible="visible"
    title="裁剪磁贴图片（banner）"
    :width="640"
    :mask-closable="false"
    :footer="false"
    @cancel="emit('cancel')"
  >
    <div class="tile-crop">
      <div class="tile-crop__modes">
        <a-radio-group v-model="mode" type="button" size="small">
          <a-radio value="square">方形（大 / 小）</a-radio>
          <a-radio value="wide">宽幅</a-radio>
        </a-radio-group>
        <span class="tile-crop__mode-tip">{{ modeTip }}</span>
      </div>

      <div ref="canvasRef" class="tile-crop__canvas" @mousedown="onMouseDown">
        <img
          ref="imgRef"
          :src="imageSrc"
          class="tile-crop__image"
          draggable="false"
          @load="onImageLoad"
        />
        <div class="tile-crop__overlay" />
        <div class="tile-crop__window" :style="windowStyle" />
      </div>

      <div class="tile-crop__previews">
        <div class="tile-crop__preview">
          <img v-if="previewLarge" :src="previewLarge" alt="大磁贴预览" class="tile-crop__preview-img tile-crop__preview-img--large" />
          <span v-else class="tile-crop__preview-empty">大 2×2</span>
        </div>
        <div class="tile-crop__preview">
          <img v-if="previewWide" :src="previewWide" alt="宽磁贴预览" class="tile-crop__preview-img tile-crop__preview-img--wide" />
          <span v-else class="tile-crop__preview-empty">宽 2×1</span>
        </div>
        <div class="tile-crop__preview">
          <img v-if="previewSmall" :src="previewSmall" alt="小磁贴预览" class="tile-crop__preview-img tile-crop__preview-img--small" />
          <span v-else class="tile-crop__preview-empty">小 1×1</span>
        </div>
      </div>

      <div class="tile-crop__actions">
        <a-button class="app-text-action-btn" type="text" html-type="button" @click="emit('cancel')">取消</a-button>
        <a-button type="primary" html-type="button" :disabled="!imageLoaded" @click="handleConfirm">确定裁剪</a-button>
      </div>
    </div>
  </a-modal>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { StartScreenTileSize } from '@/services/types'

const props = defineProps<{
  visible: boolean
  imageSrc: string
}>()

const emit = defineEmits<{
  confirm: [blobs: Record<StartScreenTileSize, Blob>]
  cancel: []
}>()

type CropMode = 'square' | 'wide'

const mode = ref<CropMode>('square')
const canvasRef = ref<HTMLElement | null>(null)
const imgRef = ref<HTMLImageElement | null>(null)
const imageLoaded = ref(false)
const displayWidth = ref(0)
const displayHeight = ref(0)
const naturalWidth = ref(0)
const naturalHeight = ref(0)
const squareLeft = ref(0)
const wideLeft = ref(0)
const previewLarge = ref('')
const previewWide = ref('')
const previewSmall = ref('')

const ASPECT: Record<CropMode, number> = {
  square: 1,
  wide: 2,
}

const modeTip = computed(() => (mode.value === 'square' ? '方形裁切会同时生成大、小两种磁贴图' : '宽幅裁切会生成宽磁贴图'))

const currentLeft = computed(() => (mode.value === 'square' ? squareLeft.value : wideLeft.value))

const cropWindowWidth = computed(() => {
  if (!displayHeight.value) return 0
  return displayHeight.value * ASPECT[mode.value]
})

const maxLeft = computed(() => Math.max(0, displayWidth.value - cropWindowWidth.value))

const windowStyle = computed(() => ({
  left: `${currentLeft.value}px`,
  width: `${cropWindowWidth.value}px`,
  height: `${displayHeight.value}px`,
}))

watch(() => props.visible, (value) => {
  if (value) {
    imageLoaded.value = false
    squareLeft.value = 0
    wideLeft.value = 0
    previewLarge.value = ''
    previewWide.value = ''
    previewSmall.value = ''
  }
})

const onImageLoad = () => {
  const img = imgRef.value
  if (!img) return
  naturalWidth.value = img.naturalWidth
  naturalHeight.value = img.naturalHeight
  displayWidth.value = img.clientWidth
  displayHeight.value = img.clientHeight
  imageLoaded.value = true
  squareLeft.value = Math.max(0, (displayWidth.value - displayHeight.value) / 2)
  wideLeft.value = Math.max(0, (displayWidth.value - displayHeight.value * 2) / 2)
  renderPreviews()
}

const onMouseDown = (event: MouseEvent) => {
  const startX = event.clientX
  const startLeft = currentLeft.value

  const onMove = (ev: MouseEvent) => {
    const dx = ev.clientX - startX
    const next = Math.round(Math.min(maxLeft.value, Math.max(0, startLeft + dx)))
    if (mode.value === 'square') {
      squareLeft.value = next
    } else {
      wideLeft.value = next
    }
    renderPreviews()
  }

  const onUp = () => {
    document.removeEventListener('mousemove', onMove)
    document.removeEventListener('mouseup', onUp)
  }

  document.addEventListener('mousemove', onMove)
  document.addEventListener('mouseup', onUp)
}

const cropRegion = (aspect: number) => {
  const img = imgRef.value
  if (!img || !naturalWidth.value || !naturalHeight.value) return null
  const scale = displayWidth.value / naturalWidth.value
  const left = aspect === 1 ? squareLeft.value : wideLeft.value
  const width = naturalHeight.value * aspect
  const x = left / scale
  return { img, x, y: 0, width, height: naturalHeight.value }
}

const renderPreview = (aspect: number, targetWidth: number, targetHeight: number): string => {
  const region = cropRegion(aspect)
  if (!region) return ''
  const canvas = document.createElement('canvas')
  canvas.width = targetWidth
  canvas.height = targetHeight
  const ctx = canvas.getContext('2d')
  if (!ctx) return ''
  ctx.drawImage(region.img, region.x, region.y, region.width, region.height, 0, 0, targetWidth, targetHeight)
  return canvas.toDataURL('image/jpeg', 0.82)
}

const renderPreviews = () => {
  if (!imageLoaded.value) return
  previewLarge.value = renderPreview(1, 234, 234)
  previewWide.value = renderPreview(2, 234, 110)
  previewSmall.value = renderPreview(1, 110, 110)
}

const regionToBlob = (aspect: number, targetWidth: number, targetHeight: number): Promise<Blob> => {
  return new Promise((resolve, reject) => {
    const region = cropRegion(aspect)
    if (!region) {
      reject(new Error('图片尚未就绪'))
      return
    }
    const canvas = document.createElement('canvas')
    canvas.width = targetWidth
    canvas.height = targetHeight
    const ctx = canvas.getContext('2d')
    if (!ctx) {
      reject(new Error('canvas 不可用'))
      return
    }
    ctx.drawImage(region.img, region.x, region.y, region.width, region.height, 0, 0, targetWidth, targetHeight)
    canvas.toBlob((blob) => {
      if (blob) {
        resolve(blob)
      } else {
        reject(new Error('生成图片失败'))
      }
    }, 'image/png')
  })
}

const handleConfirm = async () => {
  try {
    const [large, wide, small] = await Promise.all([
      regionToBlob(1, 234, 234),
      regionToBlob(2, 234, 110),
      regionToBlob(1, 110, 110),
    ])
    emit('confirm', { large, wide, small })
  } catch {
    // 图片未就绪时确定按钮是禁用的，这里只是兜底。
  }
}
</script>

<style scoped>
.tile-crop {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.tile-crop__modes {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.tile-crop__mode-tip {
  font-size: 12px;
  color: var(--color-text-3);
}

.tile-crop__canvas {
  position: relative;
  overflow: hidden;
  border-radius: 8px;
  cursor: ew-resize;
  user-select: none;
}

.tile-crop__image {
  display: block;
  width: 100%;
  height: auto;
}

.tile-crop__overlay {
  position: absolute;
  inset: 0;
  background: var(--app-scrim);
  pointer-events: none;
}

.tile-crop__window {
  position: absolute;
  top: 0;
  box-shadow: 0 0 0 9999px var(--app-scrim);
  border: 2px solid var(--color-text-on-dark);
  pointer-events: none;
}

.tile-crop__previews {
  display: flex;
  align-items: flex-end;
  gap: 10px;
}

.tile-crop__preview {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 72px;
  border-radius: 6px;
  overflow: hidden;
  background: var(--color-fill-2);
}

.tile-crop__preview-img {
  object-fit: cover;
  width: 100%;
  height: 100%;
}

.tile-crop__preview-img--large {
  aspect-ratio: 1 / 1;
  max-height: 96px;
}

.tile-crop__preview-img--wide {
  aspect-ratio: 2 / 1;
  max-height: 96px;
}

.tile-crop__preview-img--small {
  aspect-ratio: 1 / 1;
  max-height: 72px;
}

.tile-crop__preview-empty {
  font-size: 12px;
  color: var(--color-text-3);
}

.tile-crop__actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
