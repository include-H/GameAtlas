<template>
  <a-modal
    :visible="visible"
    title="添加截图"
    :width="800"
    :footer="false"
    @update:visible="emit('update:visible', $event)"
  >
    <div class="screenshot-selector-content">
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
        :query="screenshotSearchQuery"
        :placeholder="searchPlaceholder"
        :loading="isSearchingScreenshots"
        :results="screenshotSearchResults"
        :selected-game="selectedScreenshotGame"
        @update:query="emit('update:screenshot-search-query', $event)"
        @search="emit('search-screenshots')"
        @clear="emit('clear-screenshots')"
        @select="emit('select-screenshot-game', $event)"
      >
        <div v-if="screenshotCandidatesData" class="steam-screenshots-section">
          <div class="steam-game-info">
            <span>{{ screenshotCandidatesData.name }}</span>
            <a-button class="app-text-action-btn" type="text" size="mini" html-type="button" @click="emit('back-screenshot-game-search')">返回</a-button>
            <a-button class="app-text-action-btn" type="text" size="mini" html-type="button" @click="emit('select-all-screenshots')">全选</a-button>
            <a-button class="app-text-action-btn" type="text" size="mini" html-type="button" @click="emit('invert-screenshots')">反选</a-button>
          </div>

          <div v-if="screenshotCandidatesData.usedFallbackAssets" class="steam-screenshot-hint">
            Steam 未返回截图，以下为可用商店素材
          </div>

          <div v-if="screenshotCandidatesData.screenshots.length > 0" class="steam-screenshots-grid">
            <div
              v-for="(screenshot, index) in screenshotCandidatesData.screenshots"
              :key="index"
              class="steam-screenshot-item"
              :class="{ 'steam-screenshot-selected': selectedRemoteScreenshots.has(index) }"
              @click="emit('toggle-screenshot', index)"
            >
              <img :src="screenshot" />
              <div v-if="selectedRemoteScreenshots.has(index)" class="steam-screenshot-check">
                <icon-check />
              </div>
            </div>
          </div>

          <a-empty
            v-else
            description="未找到可用截图"
            class="steam-screenshots-empty"
          />

          <a-button
            v-if="selectedRemoteScreenshots.size > 0"
            type="primary"
            long
            :loading="isDownloadingRemoteScreenshots"
            html-type="button"
            @click="emit('download-selected-screenshots')"
          >
            下载选中的 {{ selectedRemoteScreenshots.size }} 张截图
          </a-button>
        </div>
      </steam-search-panel>

      <a-divider>本地上传</a-divider>
      <a-upload
        multiple
        :action="screenshotUploadAction"
        :data="screenshotUploadData"
        :headers="uploadHeaders"
        :show-file-list="false"
        accept="image/*"
        @success="emit('screenshot-upload-success', $event)"
        @error="emit('screenshot-upload-error')"
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
            :model-value="screenshotSearchUrl"
            class="url-input-row__field"
            placeholder="输入图片 URL..."
            @update:model-value="emit('update:screenshot-search-url', String($event ?? ''))"
            @press-enter="emit('load-screenshot-preview')"
          />
          <a-button class="app-text-action-btn url-input-row__action" type="text" html-type="button" @click="emit('load-screenshot-preview')">
            加载
          </a-button>
        </div>

        <div v-if="screenshotPreviewUrl" class="cover-preview-section">
          <img :src="screenshotPreviewUrl" class="cover-preview-img" />
        </div>
      </div>

      <div class="cover-selector-actions">
        <a-button class="app-text-action-btn" type="text" html-type="button" @click="emit('update:visible', false)">取消</a-button>
        <a-button
          type="primary"
          html-type="button"
          :disabled="!screenshotPreviewUrl"
          :loading="isDownloadingScreenshot"
          @click="emit('confirm-screenshot-selection')"
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

interface ScreenshotCandidatesData {
  name: string
  screenshots: string[]
  usedFallbackAssets: boolean
}

interface Props {
  visible: boolean
  source: ImportSource
  sgdbAvailable: boolean
  screenshotSearchQuery: string
  isSearchingScreenshots: boolean
  screenshotSearchResults: SteamGameSearchResult[]
  selectedScreenshotGame: SteamGameSearchResult | null
  screenshotCandidatesData: ScreenshotCandidatesData | null
  selectedRemoteScreenshots: Set<number>
  isDownloadingRemoteScreenshots: boolean
  screenshotUploadAction: string
  screenshotUploadData: Record<string, string>
  uploadHeaders: Record<string, string>
  screenshotSearchUrl: string
  screenshotPreviewUrl: string
  isDownloadingScreenshot: boolean
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'source-change': [source: ImportSource]
  'update:screenshot-search-query': [value: string]
  'search-screenshots': []
  'clear-screenshots': []
  'select-screenshot-game': [game: SteamGameSearchResult]
  'back-screenshot-game-search': []
  'toggle-screenshot': [index: number]
  'download-selected-screenshots': []
  'select-all-screenshots': []
  'invert-screenshots': []
  'screenshot-upload-success': [fileItem: FileItem]
  'screenshot-upload-error': []
  'update:screenshot-search-url': [value: string]
  'load-screenshot-preview': []
  'confirm-screenshot-selection': []
}>()

const searchPlaceholder = computed(() =>
  props.source === 'steamgriddb' ? '搜索 SteamGridDB...' : '搜索 Steam 游戏...'
)
</script>

<style scoped>
.screenshot-selector-content {
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

.steam-screenshots-section {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.steam-game-info {
  display: flex;
  align-items: center;
  gap: 10px;
}

.steam-game-info .app-text-action-btn:first-of-type {
  margin-left: auto;
}

.steam-screenshot-hint {
  font-size: 12px;
  color: var(--color-text-3);
}



.steam-screenshots-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
  gap: 10px;
}

.steam-screenshot-item {
  position: relative;
  border: 1px solid var(--color-border-2);
  border-radius: 8px;
  overflow: hidden;
  cursor: pointer;
}

.steam-screenshot-item img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.steam-screenshot-selected {
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
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid var(--color-border-2);
}

.cover-preview-img {
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
