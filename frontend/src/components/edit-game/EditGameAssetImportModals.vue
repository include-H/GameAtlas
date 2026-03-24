<template>
  <a-modal
    :visible="showSummarySelector"
    title="导入 Steam 简介"
    :width="800"
    :footer="false"
    @update:visible="emit('update:show-summary-selector', $event)"
  >
    <div class="cover-selector-content">
      <steam-search-panel
        :query="steamSummarySearchQuery"
        placeholder="搜索 Steam 游戏..."
        :loading="isSearchingSteamSummary"
        :results="steamSummarySearchResults"
        :selected-game="selectedSteamSummaryGame"
        @update:query="emit('update:steam-summary-search-query', $event)"
        @search="emit('search-summary')"
        @clear="emit('clear-summary')"
        @select="emit('select-summary', $event)"
      >
        <div v-if="selectedSteamSummaryGame" class="steam-summary-section">
          <div class="steam-search-title">
            {{ selectedSteamSummaryGame.name }} 的简介
            <a-button class="app-text-action-btn" type="text" size="mini" html-type="button" @click="emit('back-summary')">返回</a-button>
          </div>

          <div v-if="steamSummaryPreview" class="steam-summary-preview">
            {{ steamSummaryPreview }}
          </div>

          <a-empty
            v-else-if="!isSearchingSteamSummary"
            description="Steam 未返回可用简介"
            class="steam-summary-empty"
          />

          <a-button
            v-if="steamSummaryPreview"
            type="primary"
            long
            html-type="button"
            @click="emit('confirm-summary-import')"
          >
            导入这段简介
          </a-button>
        </div>
      </steam-search-panel>
    </div>
  </a-modal>

  <a-modal
    :visible="showCoverSelector"
    title="选择封面图"
    :width="700"
    :footer="false"
    @update:visible="emit('update:show-cover-selector', $event)"
  >
    <div class="cover-selector-content">
      <a-divider>从 Steam 获取</a-divider>
      <steam-search-panel
        :query="steamCoverSearchQuery"
        placeholder="搜索 Steam 游戏..."
        :loading="isSearchingSteamCover"
        :results="steamCoverSearchResults"
        :selected-game="selectedSteamGame"
        @update:query="emit('update:steam-cover-search-query', $event)"
        @search="emit('search-cover')"
        @clear="emit('clear-cover')"
        @select="emit('select-cover-game', $event)"
      >
        <div v-if="selectedSteamGame && steamCoverImages.length > 0" class="steam-images-section">
          <div class="steam-search-title">
            {{ selectedSteamGame.name }} 的封面
            <a-button class="app-text-action-btn" type="text" size="mini" html-type="button" @click="emit('back-cover-game-search')">返回</a-button>
          </div>
          <div class="steam-images-grid">
            <div
              v-for="(image, index) in steamCoverImages"
              :key="index"
              class="steam-image-item"
              :class="{ 'steam-image-selected': selectedCoverImage === image }"
              @click="emit('update:selected-cover-image', image)"
            >
              <img :src="image" />
            </div>
          </div>
          <a-button
            v-if="selectedCoverImage"
            type="primary"
            long
            :loading="isSearchingSteamCover"
            html-type="button"
            @click="emit('download-selected-steam-cover')"
          >
            下载选中的封面
          </a-button>
        </div>
      </steam-search-panel>

      <a-divider>本地上传</a-divider>
      <a-upload
        :action="uploadAction"
        :data="uploadData"
        :headers="uploadHeaders"
        :show-file-list="false"
        accept="image/*"
        @success="emit('cover-upload-success', $event)"
        @error="emit('cover-upload-error')"
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
          :model-value="coverSearchUrl"
          class="url-input-row__field"
          placeholder="输入图片 URL..."
          @update:model-value="emit('update:cover-search-url', String($event ?? ''))"
          @press-enter="emit('load-cover-from-url')"
        />
        <a-button class="app-text-action-btn url-input-row__action" type="text" html-type="button" @click="emit('load-cover-from-url')">
          加载
        </a-button>
      </div>
      <div v-if="coverPreviewUrl" class="cover-preview-large">
        <img :src="coverPreviewUrl" @error="emit('cover-image-error', $event)" />
      </div>
      <div class="cover-selector-actions">
        <a-button class="app-text-action-btn" type="text" html-type="button" @click="emit('update:show-cover-selector', false)">取消</a-button>
        <a-button
          type="primary"
          html-type="button"
          :disabled="!coverPreviewUrl"
          :loading="isDownloadingCover"
          @click="emit('confirm-cover-selection')"
        >
          确定
        </a-button>
      </div>
    </div>
  </a-modal>

  <a-modal
    :visible="showBannerSelector"
    title="选择横幅图"
    :width="800"
    :footer="false"
    @update:visible="emit('update:show-banner-selector', $event)"
  >
    <div class="cover-selector-content">
      <a-divider>从 Steam 获取</a-divider>
      <steam-search-panel
        :query="steamBannerSearchQuery"
        placeholder="搜索 Steam 游戏..."
        :loading="isSearchingSteamBanner"
        :results="steamBannerSearchResults"
        :selected-game="selectedSteamBannerGame"
        @update:query="emit('update:steam-banner-search-query', $event)"
        @search="emit('search-banner')"
        @clear="emit('clear-banner')"
        @select="emit('select-banner-game', $event)"
      >
        <div v-if="selectedSteamBannerGame && steamBannerImages.length > 0" class="steam-images-section">
          <div class="steam-search-title">
            {{ selectedSteamBannerGame.name }} 的横幅
            <a-button class="app-text-action-btn" type="text" size="mini" html-type="button" @click="emit('back-banner-game-search')">返回</a-button>
          </div>
          <div class="steam-images-grid">
            <div
              v-for="(image, index) in steamBannerImages"
              :key="index"
              class="steam-image-item banner-thumb"
              :class="{ 'steam-image-selected': selectedBannerImage === image }"
              @click="emit('update:selected-banner-image', image)"
            >
              <img :src="image" />
            </div>
          </div>
          <a-button
            v-if="selectedBannerImage"
            type="primary"
            long
            :loading="isSearchingSteamBanner"
            html-type="button"
            @click="emit('download-selected-steam-banner')"
          >
            下载选中的横幅
          </a-button>
        </div>
      </steam-search-panel>

      <a-divider>本地上传</a-divider>
      <a-upload
        :action="bannerUploadAction"
        :data="bannerUploadData"
        :headers="uploadHeaders"
        :show-file-list="false"
        accept="image/*"
        @success="emit('banner-upload-success', $event)"
        @error="emit('banner-upload-error')"
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
          :model-value="bannerSearchUrl"
          class="url-input-row__field"
          placeholder="输入图片 URL..."
          @update:model-value="emit('update:banner-search-url', String($event ?? ''))"
          @press-enter="emit('load-banner-from-url')"
        />
        <a-button class="app-text-action-btn url-input-row__action" type="text" html-type="button" @click="emit('load-banner-from-url')">
          加载
        </a-button>
      </div>
      <div v-if="bannerPreviewUrl" class="cover-preview-large">
        <img :src="bannerPreviewUrl" @error="emit('cover-image-error', $event)" />
      </div>
      <div class="cover-selector-actions">
        <a-button class="app-text-action-btn" type="text" html-type="button" @click="emit('update:show-banner-selector', false)">取消</a-button>
        <a-button
          type="primary"
          html-type="button"
          :disabled="!bannerPreviewUrl"
          :loading="isDownloadingBanner"
          @click="emit('confirm-banner-selection')"
        >
          确定
        </a-button>
      </div>
    </div>
  </a-modal>

  <a-modal
    :visible="showScreenshotSelector"
    title="添加截图"
    :width="800"
    :footer="false"
    @update:visible="emit('update:show-screenshot-selector', $event)"
  >
    <div class="screenshot-selector-content">
      <a-divider>从 Steam 获取</a-divider>
      <steam-search-panel
        :query="steamScreenshotSearchQuery"
        placeholder="搜索 Steam 游戏..."
        :loading="isSearchingSteamScreenshots"
        :results="steamScreenshotSearchResults"
        :selected-game="selectedSteamScreenshotGame"
        @update:query="emit('update:steam-screenshot-search-query', $event)"
        @search="emit('search-screenshot')"
        @clear="emit('clear-screenshot')"
        @select="emit('select-screenshot-game', $event)"
      >
        <div v-if="steamScreenshotsData" class="steam-screenshots-section">
          <div class="steam-game-info">
            <img :src="steamScreenshotsData.cover" :alt="steamScreenshotsData.name" />
            <span>{{ steamScreenshotsData.name }}</span>
            <a-button class="app-text-action-btn" type="text" size="mini" html-type="button" @click="emit('back-screenshot-game-search')">返回</a-button>
          </div>

          <div v-if="steamScreenshotsData.usedFallbackAssets" class="steam-screenshot-hint">
            Steam 未返回截图，以下为可用商店素材
          </div>

          <div v-if="steamScreenshotsData.screenshots.length > 0" class="steam-screenshots-grid">
            <div
              v-for="(screenshot, index) in steamScreenshotsData.screenshots"
              :key="index"
              class="steam-screenshot-item"
              :class="{ 'steam-screenshot-selected': selectedSteamScreenshots.has(index) }"
              @click="emit('toggle-steam-screenshot', index)"
            >
              <img :src="screenshot" />
              <div v-if="selectedSteamScreenshots.has(index)" class="steam-screenshot-check">
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
            v-if="selectedSteamScreenshots.size > 0"
            type="primary"
            long
            :loading="isDownloadingSteamScreenshots"
            html-type="button"
            @click="emit('download-selected-steam-screenshots')"
          >
            下载选中的 {{ selectedSteamScreenshots.size }} 张截图
          </a-button>
        </div>
      </steam-search-panel>

      <a-divider>本地上传</a-divider>
      <a-upload
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
        <a-button class="app-text-action-btn" type="text" html-type="button" @click="emit('update:show-screenshot-selector', false)">取消</a-button>
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
import { IconCheck, IconUpload } from '@arco-design/web-vue/es/icon'
import SteamSearchPanel from '@/components/SteamSearchPanel.vue'
import type { SteamGameSearchResult } from '@/services/types'

interface SteamScreenshotsData {
  name: string
  cover: string
  screenshots: string[]
  appId: string
  usedFallbackAssets: boolean
}

defineProps<{
  showSummarySelector: boolean
  steamSummarySearchQuery: string
  isSearchingSteamSummary: boolean
  steamSummarySearchResults: SteamGameSearchResult[]
  selectedSteamSummaryGame: SteamGameSearchResult | null
  steamSummaryPreview: string

  showCoverSelector: boolean
  steamCoverSearchQuery: string
  isSearchingSteamCover: boolean
  steamCoverSearchResults: SteamGameSearchResult[]
  selectedSteamGame: SteamGameSearchResult | null
  steamCoverImages: string[]
  selectedCoverImage: string
  uploadAction: string
  uploadData: Record<string, string>
  uploadHeaders: Record<string, string>
  coverSearchUrl: string
  coverPreviewUrl: string
  isDownloadingCover: boolean

  showBannerSelector: boolean
  steamBannerSearchQuery: string
  isSearchingSteamBanner: boolean
  steamBannerSearchResults: SteamGameSearchResult[]
  selectedSteamBannerGame: SteamGameSearchResult | null
  steamBannerImages: string[]
  selectedBannerImage: string
  bannerUploadAction: string
  bannerUploadData: Record<string, string>
  bannerSearchUrl: string
  bannerPreviewUrl: string
  isDownloadingBanner: boolean

  showScreenshotSelector: boolean
  steamScreenshotSearchQuery: string
  isSearchingSteamScreenshots: boolean
  steamScreenshotSearchResults: SteamGameSearchResult[]
  selectedSteamScreenshotGame: SteamGameSearchResult | null
  steamScreenshotsData: SteamScreenshotsData | null
  selectedSteamScreenshots: Set<number>
  isDownloadingSteamScreenshots: boolean
  screenshotUploadAction: string
  screenshotUploadData: Record<string, string>
  screenshotSearchUrl: string
  screenshotPreviewUrl: string
  isDownloadingScreenshot: boolean
}>()

const emit = defineEmits<{
  'update:show-summary-selector': [value: boolean]
  'update:steam-summary-search-query': [value: string]
  'search-summary': []
  'clear-summary': []
  'select-summary': [game: SteamGameSearchResult]
  'back-summary': []
  'confirm-summary-import': []

  'update:show-cover-selector': [value: boolean]
  'update:steam-cover-search-query': [value: string]
  'search-cover': []
  'clear-cover': []
  'select-cover-game': [game: SteamGameSearchResult]
  'back-cover-game-search': []
  'update:selected-cover-image': [value: string]
  'download-selected-steam-cover': []
  'cover-upload-success': [fileItem: unknown]
  'cover-upload-error': []
  'update:cover-search-url': [value: string]
  'load-cover-from-url': []
  'confirm-cover-selection': []
  'cover-image-error': [event: Event]

  'update:show-banner-selector': [value: boolean]
  'update:steam-banner-search-query': [value: string]
  'search-banner': []
  'clear-banner': []
  'select-banner-game': [game: SteamGameSearchResult]
  'back-banner-game-search': []
  'update:selected-banner-image': [value: string]
  'download-selected-steam-banner': []
  'banner-upload-success': [fileItem: unknown]
  'banner-upload-error': []
  'update:banner-search-url': [value: string]
  'load-banner-from-url': []
  'confirm-banner-selection': []

  'update:show-screenshot-selector': [value: boolean]
  'update:steam-screenshot-search-query': [value: string]
  'search-screenshot': []
  'clear-screenshot': []
  'select-screenshot-game': [game: SteamGameSearchResult]
  'back-screenshot-game-search': []
  'toggle-steam-screenshot': [index: number]
  'download-selected-steam-screenshots': []
  'screenshot-upload-success': [fileItem: unknown]
  'screenshot-upload-error': []
  'update:screenshot-search-url': [value: string]
  'load-screenshot-preview': []
  'confirm-screenshot-selection': []
}>()
</script>

<style scoped>
.cover-selector-content {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.steam-summary-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.steam-summary-preview {
  max-height: 280px;
  overflow-y: auto;
  white-space: pre-wrap;
  line-height: 1.6;
  padding: 12px;
  border-radius: 8px;
  background: var(--color-fill-2);
  color: var(--color-text-2);
}

.steam-summary-empty {
  margin: 8px 0;
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
}

.steam-image-item {
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

.screenshot-selector-content {
  display: flex;
  flex-direction: column;
  gap: 12px;
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

.steam-game-info img {
  width: 80px;
  height: 40px;
  object-fit: cover;
  border-radius: 4px;
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
  color: #fff;
}

.steam-screenshots-empty {
  margin: 4px 0 8px;
}

.url-input-section {
  display: flex;
  flex-direction: column;
  gap: 10px;
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
</style>
