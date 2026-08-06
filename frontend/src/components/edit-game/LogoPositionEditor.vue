<template>
  <div
    class="media-card__preview media-card__preview--logo"
    @pointerdown="handleLogoPositionPointerDown"
  >
    <div ref="logoPositionEditorRef" class="logo-position-editor__canvas">
      <img
        v-if="logoBannerSrc"
        :src="logoBannerSrc"
        class="logo-position-editor__banner"
        alt=""
        draggable="false"
      />
      <div v-else class="logo-position-editor__banner-empty">
        <icon-image />
        <span>无横幅图</span>
      </div>
      <img
        v-if="logoVisible"
        :src="logos[0].path"
        class="logo-position-editor__logo"
        :style="logoPositionStyle"
        alt=""
        draggable="false"
      />
      <div v-else class="logo-position-editor__logo-hidden">Logo 已隐藏</div>
    </div>
    <div class="logo-position-editor__controls">
      <span class="logo-position-editor__label">大小</span>
      <a-slider
        :model-value="logoWidthPct"
        :min="10"
        :max="80"
        :step="1"
        :style="{ flex: 1 }"
        @update:model-value="handleLogoWidthChange"
      />
      <span class="logo-position-editor__value">{{ logoWidthPct }}%</span>
      <span class="logo-position-editor__label">显示 Logo</span>
      <a-switch
        :model-value="logoVisible"
        size="small"
        @update:model-value="handleLogoVisibilityChange"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { IconImage } from '@arco-design/web-vue/es/icon'
import type { LogoPositionChange } from '@/utils/edit-game-form'

interface EditableLogo {
  asset_uid?: string
  path: string
  position_x: number | null
  position_y: number | null
  width_pct: number | null
}

interface EditableBanner {
  asset_uid?: string
  path: string
}

interface EditableCover {
  asset_uid?: string
  path: string
}

const props = defineProps<{
  logos: EditableLogo[]
  banners: EditableBanner[]
  covers: EditableCover[]
  logoVisible: boolean
}>()

const emit = defineEmits<{
  'logo-position-change': [payload: LogoPositionChange]
}>()

const logoPositionEditorRef = ref<HTMLElement | null>(null)
let logoPositionDragCleanup: (() => void) | null = null

const primaryLogo = computed(() => props.logos[0] ?? null)
const logoBannerSrc = computed(() => props.banners[0]?.path || props.covers[0]?.path || '')
const logoPositionX = computed(() => primaryLogo.value?.position_x ?? 50)
const logoPositionY = computed(() => primaryLogo.value?.position_y ?? 50)
const logoWidthPct = computed(() => primaryLogo.value?.width_pct ?? 30)
const logoPositionStyle = computed(() => ({
  left: `${logoPositionX.value}%`,
  top: `${logoPositionY.value}%`,
  width: `${logoWidthPct.value}%`,
  transform: 'translate(-50%, -50%)',
}))

const emitLogoPositionChange = (changes: Partial<Omit<LogoPositionChange, 'key'>>) => {
  const logo = primaryLogo.value
  if (!logo) return
  emit('logo-position-change', {
    key: logo.asset_uid || logo.path,
    position_x: changes.position_x ?? logoPositionX.value,
    position_y: changes.position_y ?? logoPositionY.value,
    width_pct: changes.width_pct ?? logoWidthPct.value,
    logo_visible: changes.logo_visible ?? props.logoVisible,
  })
}

const handleLogoWidthChange = (value: number | number[]) => {
  const width = Array.isArray(value) ? value[0] : value
  if (typeof width === 'number' && Number.isFinite(width)) {
    emitLogoPositionChange({ width_pct: width })
  }
}

const handleLogoVisibilityChange = (value: string | number | boolean) => {
  if (typeof value === 'boolean') {
    emitLogoPositionChange({ logo_visible: value })
  }
}

const stopLogoPositionDrag = () => {
  logoPositionDragCleanup?.()
  logoPositionDragCleanup = null
}

const handleLogoPositionPointerDown = (event: PointerEvent) => {
  const target = event.target as HTMLElement
  if (!props.logoVisible || !target.classList.contains('logo-position-editor__logo')) return
  const editor = logoPositionEditorRef.value
  if (!editor) return

  event.preventDefault()
  stopLogoPositionDrag()

  const startMouseX = event.clientX
  const startMouseY = event.clientY
  const startPosX = logoPositionX.value
  const startPosY = logoPositionY.value
  const rect = editor.getBoundingClientRect()

  const onMove = (moveEvent: PointerEvent) => {
    const dx = moveEvent.clientX - startMouseX
    const dy = moveEvent.clientY - startMouseY
    const nextX = Math.round(Math.min(95, Math.max(5, startPosX + (dx / rect.width) * 100)) * 10) / 10
    const nextY = Math.round(Math.min(95, Math.max(5, startPosY + (dy / rect.height) * 100)) * 10) / 10
    emitLogoPositionChange({ position_x: nextX, position_y: nextY })
  }

  const onUp = () => stopLogoPositionDrag()
  window.addEventListener('pointermove', onMove)
  window.addEventListener('pointerup', onUp, { once: true })
  logoPositionDragCleanup = () => {
    window.removeEventListener('pointermove', onMove)
    window.removeEventListener('pointerup', onUp)
  }
}

onBeforeUnmount(() => {
  stopLogoPositionDrag()
})
</script>

<style scoped>
.media-card__preview {
  position: relative;
  flex: 1 1 auto;
  min-height: 0;
  width: 100%;
  border: 1px solid var(--app-card-border);
  border-radius: 10px;
  overflow: hidden;
  background:
    linear-gradient(180deg, var(--color-border-1), transparent),
    color-mix(in srgb, var(--app-card-surface) 90%, transparent);
  display: flex;
  align-items: center;
  justify-content: center;
}

.media-card__preview--logo {
  flex: 0 0 auto;
  height: auto;
  flex-direction: column;
  align-items: stretch;
  justify-content: flex-start;
  gap: 8px;
  padding: 8px;
}

.logo-position-editor__canvas {
  position: relative;
  flex: 0 0 auto;
  min-height: 0;
  width: 100%;
  aspect-ratio: 460 / 215;
  overflow: hidden;
  border-radius: 7px;
  background: var(--color-media-bg);
  user-select: none;
  touch-action: none;
}

.logo-position-editor__banner {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.logo-position-editor__banner-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  gap: 4px;
  color: var(--color-text-4);
  font-size: 24px;
}

.logo-position-editor__banner-empty span,
.logo-position-editor__logo-hidden {
  font-size: 12px;
}

.logo-position-editor__logo {
  position: absolute;
  max-width: 90%;
  max-height: 90%;
  height: auto;
  object-fit: contain;
  cursor: grab;
  pointer-events: auto;
  touch-action: none;
  user-select: none;
  z-index: 2;
}

.logo-position-editor__logo:active {
  cursor: grabbing;
}

.logo-position-editor__logo-hidden {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  padding: 4px 8px;
  border-radius: 4px;
  color: var(--color-text-2);
  background: var(--app-scrim-light);
  white-space: nowrap;
}

.logo-position-editor__controls {
  display: flex;
  align-items: center;
  flex: 0 0 auto;
  width: 100%;
  min-width: 0;
  gap: 8px;
  min-height: 28px;
  color: var(--color-text-2);
}

.logo-position-editor__controls :deep(.arco-slider) {
  flex: 1 1 auto;
  min-width: 0;
}

.logo-position-editor__label {
  flex-shrink: 0;
  font-size: 12px;
  color: var(--color-text-3);
  white-space: nowrap;
}

.logo-position-editor__value {
  flex-shrink: 0;
  min-width: 34px;
  font-size: 12px;
  color: var(--color-text-2);
  text-align: right;
}
</style>
