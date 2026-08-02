<template>
  <a-modal
    :visible="visible"
    :title="modalTitle"
    :width="modalWidth"
    :footer="false"
    @update:visible="emit('update:visible', $event)"
  >
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
        :query="searchQuery"
        :placeholder="searchPlaceholder"
        :loading="isSearching"
        :results="searchResults"
        :selected-game="selectedGame"
        @update:query="emit('update:search-query', $event)"
        @search="emit('search')"
        @clear="emit('clear')"
        @select="emit('select-game', $event)"
      >
        <div v-if="selectedGame && images.length > 0" class="steam-images-section">
          <div class="steam-game-info">
            <span>{{ selectedGame.name }} 的{{ titleLabel }}</span>
            <a-button class="app-text-action-btn" type="text" size="mini" html-type="button" @click="emit('back-game-search')">返回</a-button>
            <a-button class="app-text-action-btn" type="text" size="mini" html-type="button" @click="emit('select-all')">全选</a-button>
            <a-button class="app-text-action-btn" type="text" size="mini" html-type="button" @click="emit('invert-selection')">反选</a-button>
          </div>
          <div class="steam-images-grid">
            <div
              v-for="(image, index) in images"
              :key="index"
              class="steam-image-item"
              :class="[imageItemClass, { 'steam-image-selected': selectedImages.has(index) }]"
              @click="emit('toggle-selection', index)"
            >
              <img :src="image" />
              <div v-if="selectedImages.has(index)" class="steam-screenshot-check">
                <icon-check />
              </div>
            </div>
          </div>
          <a-button
            v-if="selectedImages.size > 0"
            type="primary"
            long
            :loading="isDownloadingSteam"
            html-type="button"
            @click="emit('download-selected-steam')"
          >
            下载选中的 {{ selectedImages.size }} 张{{ titleLabel }}
          </a-button>
        </div>
      </steam-search-panel>

      <a-divider>本地上传</a-divider>
      <a-upload
        multiple
        :action="uploadAction"
        :data="uploadData"
        :headers="uploadHeaders"
        :show-file-list="false"
        accept="image/*"
        @success="emit('upload-success', $event)"
        @error="emit('upload-error')"
      >
        <a-button class="app-text-action-btn" type="text" long html-type="button">
          <template #icon>
            <icon-upload />
          </template>
          点击上传本地图片
        </a-button>
      </a-upload>

      <a-divider>或从 URL 加载</a-divider>
      <div class="url-input-row">
        <a-input
          :model-value="searchUrl"
          class="url-input-row__field"
          placeholder="输入图片 URL..."
          @update:model-value="emit('update:search-url', String($event ?? ''))"
          @press-enter="emit('load-from-url')"
        />
        <a-button class="app-text-action-btn url-input-row__action" type="text" html-type="button" @click="emit('load-from-url')">
          加载
        </a-button>
      </div>
      <div v-if="previewUrl" class="cover-preview-large">
        <img :src="previewUrl" @error="emit('image-error', $event)" />
      </div>
      <div class="cover-selector-actions">
        <a-button class="app-text-action-btn" type="text" html-type="button" @click="emit('update:visible', false)">取消</a-button>
        <a-button
          type="primary"
          html-type="button"
          :disabled="!previewUrl"
          :loading="isDownloading"
          @click="emit('confirm-selection')"
        >
          确定
        </a-button>
      </div>
    </div>
  </a-modal>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { IconCheck, IconUpload } from '@arco-design/web-vue/es/icon'
import SteamSearchPanel from '@/components/SteamSearchPanel.vue'
import type { ImportSource } from '@/composables/useSteamImport'
import type { SteamGameSearchResult } from '@/services/types'
import type { FileItem } from '@arco-design/web-vue/es/upload/interfaces'

interface Props {
  mode: 'cover' | 'banner'
  visible: boolean
  source: ImportSource
  sgdbAvailable: boolean
  searchQuery: string
  isSearching: boolean
  searchResults: SteamGameSearchResult[]
  selectedGame: SteamGameSearchResult | null
  images: string[]
  selectedImages: Set<number>
  isDownloadingSteam: boolean
  uploadAction: string
  uploadData: Record<string, string>
  uploadHeaders: Record<string, string>
  searchUrl: string
  previewUrl: string
  isDownloading: boolean
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'source-change': [source: ImportSource]
  'update:search-query': [value: string]
  'search': []
  'clear': []
  'select-game': [game: SteamGameSearchResult]
  'back-game-search': []
  'toggle-selection': [index: number]
  'download-selected-steam': []
  'select-all': []
  'invert-selection': []
  'upload-success': [fileItem: FileItem]
  'upload-error': []
  'update:search-url': [value: string]
  'load-from-url': []
  'confirm-selection': []
  'image-error': [event: Event]
}>()

const modalTitle = computed(() => props.mode === 'cover' ? '选择封面图' : '选择横幅图')
const modalWidth = computed(() => props.mode === 'cover' ? 700 : 800)
const titleLabel = computed(() => props.mode === 'cover' ? '封面' : '横幅')
const searchPlaceholder = computed(() =>
  props.source === 'steamgriddb' ? '搜索 SteamGridDB...' : '搜索 Steam 游戏...'
)
const imageItemClass = computed(() => props.mode === 'banner' ? 'banner-thumb' : '')
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


.steam-game-info {
  display: flex;
  align-items: center;
  gap: 10px;
}

.steam-game-info .app-text-action-btn:first-of-type {
  margin-left: auto;
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
  border: 1px solid var(--color-border-2);
  border-radius: 8px;
  overflow: hidden;
  cursor: pointer;
  transition: border-color 0.2s ease, transform 0.2s ease;
}

.steam-image-item:hover {
  border-color: rgba(var(--primary-6), 0.6);
  transform: translateY(-1px);
}

.steam-image-item img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.steam-image-selected {
  border-color: rgb(var(--primary-6));
  box-shadow: 0 0 0 1px rgba(var(--primary-6), 0.45);
}

.banner-thumb {
  aspect-ratio: 16 / 9;
}

.steam-screenshot-check {
  position: absolute;
  top: 6px;
  right: 6px;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  background: rgba(var(--primary-6), 0.9);
  color: var(--color-text-on-dark);
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

.cover-preview-large {
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid var(--color-border-2);
  background: var(--color-fill-2);
}

.cover-preview-large img {
  width: 100%;
  height: auto;
  display: block;
}

.cover-selector-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 4px;
}
</style>
