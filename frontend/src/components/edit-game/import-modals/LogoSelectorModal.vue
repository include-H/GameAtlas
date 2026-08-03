<template>
  <a-modal
    :visible="visible"
    title="Logo"
    :width="700"
    :footer="false"
    @update:visible="emit('update:visible', $event)"
  >
    <a-tabs v-model:active-key="logoTabKey" type="rounded" size="small">
      <a-tab-pane key="import" title="更换">
        <div class="cover-selector-content">
      <div class="source-selector">
        <span class="source-selector__label">数据源</span>
        <a-button
          class="app-text-action-btn"
          :type="source === 'steam' ? 'outline' : 'text'"
          size="small"
          html-type="button"
          @click="emit('source-change', 'steam')"
        >Steam</a-button>
        <a-button
          class="app-text-action-btn"
          :type="source === 'steamgriddb' ? 'outline' : 'text'"
          size="small"
          html-type="button"
          :disabled="!sgdbAvailable"
          @click="emit('source-change', 'steamgriddb')"
        >SteamGridDB</a-button>
      </div>
      <steam-search-panel
        :query="logoSearchQuery"
        :placeholder="searchPlaceholder"
        :loading="isSearchingLogo"
        :results="logoSearchResults"
        :selected-game="selectedLogoGame"
        @update:query="emit('update:logo-search-query', $event)"
        @search="emit('search-logo')"
        @clear="emit('clear-logo')"
        @select="emit('select-logo-game', $event)"
      >
        <div v-if="selectedLogoGame && logoImages.length > 0" class="steam-images-section">
          <div class="steam-search-title">
            {{ selectedLogoGame.name }} 的 Logo
            <a-button class="app-text-action-btn" type="text" size="mini" html-type="button" @click="emit('back-logo-game-search')">返回</a-button>
          </div>
          <div class="steam-images-grid">
            <div
              v-for="(image, index) in logoImages"
              :key="index"
              class="steam-image-item"
              :class="{ 'steam-image-selected': selectedLogoImage === image }"
              @click="emit('update:selected-logo-image', image)"
            >
              <img :src="image" />
            </div>
          </div>

          <a-button
            v-if="selectedLogoImage"
            type="primary"
            long
            :loading="isDownloadingLogos"
            html-type="button"
            @click="emit('download-selected-steam-logo')"
          >
            使用此图片
          </a-button>
        </div>

        <a-empty
          v-else-if="selectedLogoGame && logoImages.length === 0 && !isSearchingLogo"
          description="未找到可用 Logo"
          class="steam-screenshots-empty"
        />
      </steam-search-panel>

      <a-divider>本地上传</a-divider>
      <a-upload
        :action="logoUploadAction"
        :data="logoUploadData"
        :headers="uploadHeaders"
        :show-file-list="false"
        accept="image/*"
        @success="emit('logo-upload-success', $event)"
        @error="emit('logo-upload-error')"
      >
        <a-button class="app-text-action-btn" type="text" long html-type="button">
          <template #icon>
            <icon-upload />
          </template>
          本地上传
        </a-button>
      </a-upload>

      <a-divider>或从 URL 加载</a-divider>
      <div class="url-input-section">
        <div class="url-input-row">
          <a-input
            :model-value="logoSearchUrl"
            class="url-input-row__field"
            placeholder="输入图片 URL..."
            @update:model-value="emit('update:logo-search-url', String($event ?? ''))"
            @press-enter="emit('load-logo-from-url')"
          />
          <a-button class="app-text-action-btn url-input-row__action" type="text" html-type="button" @click="emit('load-logo-from-url')">
            加载
          </a-button>
        </div>

        <div v-if="logoPreviewUrl" class="cover-preview-section">
          <img :src="logoPreviewUrl" class="cover-preview-img" />
        </div>
      </div>

      <div class="cover-selector-actions">
        <a-button class="app-text-action-btn" type="text" html-type="button" @click="emit('update:visible', false)">取消</a-button>
        <a-button
          type="primary"
          html-type="button"
          :disabled="!logoPreviewUrl"
          :loading="isDownloadingLogo"
          @click="emit('confirm-logo-selection')"
        >
          确定
        </a-button>
      </div>
        </div>
      </a-tab-pane>
      <a-tab-pane key="position" title="位置">
        <div class="logo-pos-editor">
          <div
            ref="logoPosEditorRef"
            class="logo-pos-editor__canvas"
            @mousedown="handleLogoPosMouseDown"
          >
            <img
              v-if="logoBannerSrc"
              :src="logoBannerSrc"
              class="logo-pos-editor__banner"
              draggable="false"
            />
            <div v-else class="logo-pos-editor__banner-empty">
              <icon-image />
              <span>无横幅图</span>
            </div>
            <img
              v-if="logoPath"
              :src="logoPath"
              class="logo-pos-editor__logo"
              :style="logoPosLogoStyle"
              draggable="false"
            />
            <div class="logo-pos-editor__hint">拖拽移动 · 滑块缩放</div>
          </div>
          <div class="logo-pos-editor__controls">
            <span class="logo-pos-editor__label">大小</span>
            <a-slider v-model="logoPosWidth" :min="10" :max="80" :step="1" :style="{ flex: 1 }" />
            <span class="logo-pos-editor__value">{{ logoPosWidth }}%</span>
          </div>
          <div class="logo-pos-editor__controls">
            <span class="logo-pos-editor__label">显示 Logo</span>
            <a-switch v-model="logoVisible" size="small" />
          </div>
          <div class="cover-selector-actions">
            <a-button class="app-text-action-btn" type="text" html-type="button" @click="emit('update:visible', false)">取消</a-button>
            <a-button type="primary" html-type="button" @click="handleLogoPosConfirm">确定</a-button>
          </div>
        </div>
      </a-tab-pane>
    </a-tabs>
  </a-modal>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { IconImage, IconUpload } from '@arco-design/web-vue/es/icon'
import SteamSearchPanel from '@/components/SteamSearchPanel.vue'
import type { ImportSource } from '@/composables/useSteamImport'
import type { SteamGameSearchResult } from '@/services/types'
import type { FileItem } from '@arco-design/web-vue/es/upload/interfaces'

interface Props {
  visible: boolean
  source: ImportSource
  sgdbAvailable: boolean
  logoSearchQuery: string
  isSearchingLogo: boolean
  logoSearchResults: SteamGameSearchResult[]
  selectedLogoGame: SteamGameSearchResult | null
  logoImages: string[]
  selectedLogoImage: string
  isDownloadingLogos: boolean
  logoUploadAction: string
  logoUploadData: Record<string, string>
  uploadHeaders: Record<string, string>
  logoSearchUrl: string
  logoPreviewUrl: string
  isDownloadingLogo: boolean
  logoBannerSrc: string
  logoPath: string
  logoPositionX: number | null
  logoPositionY: number | null
  logoWidthPct: number | null
  logoVisible: boolean
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'source-change': [source: ImportSource]
  'update:logo-search-query': [value: string]
  'search-logo': []
  'clear-logo': []
  'select-logo-game': [game: SteamGameSearchResult]
  'back-logo-game-search': []
  'update:selected-logo-image': [value: string]
  'download-selected-steam-logo': []
  'logo-upload-success': [fileItem: FileItem]
  'logo-upload-error': []
  'update:logo-search-url': [value: string]
  'load-logo-from-url': []
  'confirm-logo-selection': []
  'confirm-logo-position': [payload: { position_x: number; position_y: number; width_pct: number; logo_visible: boolean }]
}>()

const searchPlaceholder = computed(() =>
  props.source === 'steamgriddb' ? '搜索 SteamGridDB...' : '搜索 Steam 游戏...'
)

// Logo position editor state (local to this component)
const logoTabKey = ref('import')
const logoPosEditorRef = ref<HTMLElement | null>(null)
const logoPosWidth = ref(30)
const logoPosX = ref(50)
const logoPosY = ref(50)
const logoVisible = ref(true)

watch(() => props.visible, (v) => {
  if (v) {
    logoPosX.value = props.logoPositionX ?? 50
    logoPosY.value = props.logoPositionY ?? 50
    logoPosWidth.value = props.logoWidthPct ?? 30
    logoVisible.value = props.logoVisible ?? true
    logoTabKey.value = 'import'
  }
})

const logoPosLogoStyle = computed(() => ({
  left: `${logoPosX.value}%`,
  top: `${logoPosY.value}%`,
  width: `${logoPosWidth.value}%`,
  transform: 'translate(-50%, -50%)',
}))

const handleLogoPosMouseDown = (e: MouseEvent) => {
  const target = e.target as HTMLElement
  if (!target.classList.contains('logo-pos-editor__logo')) return
  e.preventDefault()
  const editor = logoPosEditorRef.value
  if (!editor) return

  const startMouseX = e.clientX
  const startMouseY = e.clientY
  const startPosX = logoPosX.value
  const startPosY = logoPosY.value
  const rect = editor.getBoundingClientRect()

  const onMove = (ev: MouseEvent) => {
    const dx = ev.clientX - startMouseX
    const dy = ev.clientY - startMouseY
    logoPosX.value = Math.round(Math.min(95, Math.max(5, startPosX + (dx / rect.width) * 100)) * 10) / 10
    logoPosY.value = Math.round(Math.min(95, Math.max(5, startPosY + (dy / rect.height) * 100)) * 10) / 10
  }

  const onUp = () => {
    document.removeEventListener('mousemove', onMove)
    document.removeEventListener('mouseup', onUp)
  }
  document.addEventListener('mousemove', onMove)
  document.addEventListener('mouseup', onUp)
}

const handleLogoPosConfirm = () => {
  emit('confirm-logo-position', {
    position_x: logoPosX.value,
    position_y: logoPosY.value,
    width_pct: logoPosWidth.value,
    logo_visible: logoVisible.value,
  })
  emit('update:visible', false)
}
</script>

<style scoped>
.cover-selector-content {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.source-selector {
  display: flex;
  align-items: center;
  gap: 8px;
}

.source-selector__label {
  font-size: 14px;
  color: var(--color-text-2);
  flex-shrink: 0;
}

.steam-search-title {
  font-size: 14px;
  font-weight: 700;
  color: var(--color-text-1);
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.steam-images-section {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.steam-images-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 10px;
  max-height: 400px;
  overflow-y: auto;
}

.steam-image-item {
  position: relative;
  aspect-ratio: 16 / 9;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 10px;
  border: 1px solid var(--color-border-2);
  border-radius: 8px;
  overflow: hidden;
  cursor: pointer;
  background: color-mix(in srgb, var(--app-card-surface) 88%, transparent);
  transition: border-color 0.2s ease, transform 0.2s ease;
}

.steam-image-item:hover {
  border-color: rgba(var(--primary-6), 0.6);
  transform: translateY(-1px);
}

.steam-image-item img {
  width: 100%;
  height: 100%;
  object-fit: contain;
  display: block;
}

.steam-image-selected {
  border-color: rgb(var(--primary-6));
  box-shadow: 0 0 0 1px rgba(var(--primary-6), 0.45);
}

.steam-screenshots-empty {
  margin: 4px 0 8px;
}

.url-input-section {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.url-input-row {
  display: flex;
  gap: 8px;
  align-items: center;
}

.url-input-row__field {
  flex: 1;
  min-width: 0;
}

.url-input-row__action {
  flex-shrink: 0;
  min-width: 72px;
}

.cover-preview-section {
  min-height: 140px;
  max-height: 300px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid var(--color-border-2);
  background: color-mix(in srgb, var(--app-card-surface) 88%, transparent);
}

.cover-preview-img {
  max-width: 100%;
  max-height: 268px;
  width: auto;
  height: auto;
  object-fit: contain;
  display: block;
}

.cover-selector-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 4px;
}

/* Logo position editor */
.logo-pos-editor {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.logo-pos-editor__canvas {
  position: relative;
  width: 100%;
  aspect-ratio: 460 / 215;
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid var(--app-card-border);
  background: color-mix(in srgb, var(--app-card-surface) 86%, transparent);
  user-select: none;
}

.logo-pos-editor__banner {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.logo-pos-editor__banner-empty {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  font-size: 2rem;
  color: var(--color-text-4);
}

.logo-pos-editor__banner-empty span {
  font-size: 12px;
}

.logo-pos-editor__logo {
  position: absolute;
  max-width: 90%;
  max-height: 90%;
  height: auto;
  object-fit: contain;
  pointer-events: auto;
  cursor: grab;
  z-index: 2;
}

.logo-pos-editor__logo:active {
  cursor: grabbing;
}

.logo-pos-editor__hint {
  position: absolute;
  bottom: 6px;
  left: 50%;
  transform: translateX(-50%);
  font-size: 10px;
  color: var(--color-text-3);
  background: var(--app-scrim-light);
  padding: 2px 8px;
  border-radius: 4px;
  pointer-events: none;
  white-space: nowrap;
  z-index: 3;
}

.logo-pos-editor__controls {
  display: flex;
  align-items: center;
  gap: 8px;
}

.logo-pos-editor__label {
  font-size: 13px;
  color: var(--color-text-3);
  white-space: nowrap;
}

.logo-pos-editor__value {
  font-size: 13px;
  color: var(--color-text-2);
  min-width: 36px;
  text-align: right;
}
</style>
