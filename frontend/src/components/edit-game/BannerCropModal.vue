<template>
  <a-modal
    :visible="visible"
    title="裁剪横幅"
    :width="560"
    :mask-closable="false"
    :footer="false"
    @cancel="emit('cancel')"
  >
    <div class="banner-crop">
      <div ref="canvasRef" class="banner-crop__canvas" @mousedown="onMouseDown">
        <img
          ref="imgRef"
          :src="imageSrc"
          class="banner-crop__image"
          draggable="false"
          @load="onImageLoad"
        />
        <div class="banner-crop__overlay" />
        <div
          class="banner-crop__window"
          :style="windowStyle"
        />
      </div>
      <div class="banner-crop__actions">
        <a-button class="app-text-action-btn" type="text" html-type="button" @click="emit('cancel')">取消</a-button>
        <a-button type="primary" html-type="button" :disabled="!imageLoaded" @click="handleConfirm">确定裁剪</a-button>
      </div>
    </div>
  </a-modal>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'

const props = defineProps<{
  visible: boolean
  imageSrc: string
}>()

const emit = defineEmits<{
  confirm: [blob: Blob]
  cancel: []
}>()

const canvasRef = ref<HTMLElement | null>(null)
const imgRef = ref<HTMLImageElement | null>(null)
const imageLoaded = ref(false)
const cropLeft = ref(0)
const displayWidth = ref(0)
const displayHeight = ref(0)
const naturalWidth = ref(0)
const naturalHeight = ref(0)

const ASPECT_RATIO = 16 / 9

const cropWindowWidth = computed(() => {
  if (!displayHeight.value) return 0
  return displayHeight.value * ASPECT_RATIO
})

const maxLeft = computed(() => {
  return Math.max(0, displayWidth.value - cropWindowWidth.value)
})

const windowStyle = computed(() => ({
  left: `${cropLeft.value}px`,
  width: `${cropWindowWidth.value}px`,
  height: `${displayHeight.value}px`,
}))

watch(() => props.visible, (v) => {
  if (v) {
    imageLoaded.value = false
    cropLeft.value = 0
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
  cropLeft.value = Math.max(0, (displayWidth.value - cropWindowWidth.value) / 2)
}

const onMouseDown = (e: MouseEvent) => {
  const startX = e.clientX
  const startLeft = cropLeft.value

  const onMove = (ev: MouseEvent) => {
    const dx = ev.clientX - startX
    cropLeft.value = Math.round(Math.min(maxLeft.value, Math.max(0, startLeft + dx)))
  }

  const onUp = () => {
    document.removeEventListener('mousemove', onMove)
    document.removeEventListener('mouseup', onUp)
  }

  document.addEventListener('mousemove', onMove)
  document.addEventListener('mouseup', onUp)
}

const handleConfirm = () => {
  const img = imgRef.value
  if (!img || !naturalWidth.value || !naturalHeight.value) return

  const scale = displayWidth.value / naturalWidth.value
  const cropX = cropLeft.value / scale
  const cropW = cropWindowWidth.value / scale
  const cropH = naturalHeight.value

  const canvas = document.createElement('canvas')
  canvas.width = Math.round(cropW)
  canvas.height = Math.round(cropH)
  const ctx = canvas.getContext('2d')
  if (!ctx) return

  ctx.drawImage(img, cropX, 0, cropW, cropH, 0, 0, canvas.width, canvas.height)

  canvas.toBlob((blob) => {
    if (blob) emit('confirm', blob)
  }, 'image/png')
}
</script>

<style scoped>
.banner-crop {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.banner-crop__canvas {
  position: relative;
  overflow: hidden;
  border-radius: 8px;
  cursor: ew-resize;
  user-select: none;
}

.banner-crop__image {
  display: block;
  width: 100%;
  height: auto;
}

.banner-crop__overlay {
  position: absolute;
  inset: 0;
  background: var(--app-scrim);
  pointer-events: none;
}

.banner-crop__window {
  position: absolute;
  top: 0;
  box-shadow: 0 0 0 9999px var(--app-scrim);
  border: 2px solid var(--color-text-on-dark);
  pointer-events: none;
}

.banner-crop__actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
