<template>
  <a-modal
    :visible="visible"
    title="Logo"
    :width="700"
    :footer="false"
    @update:visible="emit('update:visible', $event)"
  >
    <div class="cover-selector-content">
      <steam-search-panel
        :query="logoSearchQuery"
        placeholder="搜索 SteamGridDB..."
        :loading="isSearchingLogo"
        :results="logoSearchResults"
        :selected-game="selectedLogoGame"
        @update:query="emit('update:logo-search-query', $event)"
        @search="emit('search-logo')"
        @clear="emit('clear-logo')"
        @select="emit('select-logo-game', $event)"
      >
        <div v-if="selectedLogoGame && logoImages.length > 0" class="steam-images-section">
          <div class="steam-game-info">
            <span>{{ selectedLogoGame.name }} 的 Logo</span>
            <a-button class="app-text-action-btn" type="text" size="mini" html-type="button" @click="emit('back-logo-game-search')">返回</a-button>
            <a-button class="app-text-action-btn" type="text" size="mini" html-type="button" @click="emit('select-all-logos')">全选</a-button>
            <a-button class="app-text-action-btn" type="text" size="mini" html-type="button" @click="emit('invert-selection-logos')">反选</a-button>
          </div>
          <div class="steam-images-grid">
            <div
              v-for="(image, index) in logoImages"
              :key="index"
              class="steam-image-item"
              :class="{ 'steam-image-selected': selectedLogos.has(index) }"
              @click="emit('toggle-logo-selection', index)"
            >
              <img :src="image" />
              <div v-if="selectedLogos.has(index)" class="steam-screenshot-check">
                <icon-check />
              </div>
            </div>
          </div>

          <a-button
            v-if="selectedLogos.size > 0"
            type="primary"
            long
            :loading="isDownloadingLogos"
            html-type="button"
            @click="emit('download-selected-logos')"
          >
            下载选中的 {{ selectedLogos.size }} 张 Logo
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
        multiple
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
  </a-modal>
</template>

<script setup lang="ts">
import { IconCheck, IconUpload } from '@arco-design/web-vue/es/icon'
import SteamSearchPanel from '@/components/SteamSearchPanel.vue'
import type { SteamGameSearchResult } from '@/services/types'
import type { FileItem } from '@arco-design/web-vue/es/upload/interfaces'

interface Props {
  visible: boolean
  logoSearchQuery: string
  isSearchingLogo: boolean
  logoSearchResults: SteamGameSearchResult[]
  selectedLogoGame: SteamGameSearchResult | null
  logoImages: string[]
  selectedLogos: Set<number>
  isDownloadingLogos: boolean
  logoUploadAction: string
  logoUploadData: Record<string, string>
  uploadHeaders: Record<string, string>
  logoSearchUrl: string
  logoPreviewUrl: string
  isDownloadingLogo: boolean
}

defineProps<Props>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'update:logo-search-query': [value: string]
  'search-logo': []
  'clear-logo': []
  'select-logo-game': [game: SteamGameSearchResult]
  'back-logo-game-search': []
  'toggle-logo-selection': [index: number]
  'select-all-logos': []
  'invert-selection-logos': []
  'download-selected-logos': []
  'logo-upload-success': [fileItem: FileItem]
  'logo-upload-error': []
  'update:logo-search-url': [value: string]
  'load-logo-from-url': []
  'confirm-logo-selection': []
}>()
</script>

<style scoped>
.cover-selector-content {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.steam-game-info {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 14px;
  font-weight: 700;
  color: var(--color-text-1);
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

</style>
