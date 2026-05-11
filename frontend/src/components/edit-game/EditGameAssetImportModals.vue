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
      <div class="source-selector">
        <span class="source-selector__label">数据源</span>
        <a-button
          class="app-text-action-btn"
          :type="coverSource === 'steam' ? 'outline' : 'text'"
          size="small"
          html-type="button"
          @click="emit('source-change-cover', 'steam')"
        >Steam</a-button>
        <a-button
          class="app-text-action-btn"
          :type="coverSource === 'steamgriddb' ? 'outline' : 'text'"
          size="small"
          html-type="button"
          :disabled="!sgdbAvailable"
          @click="emit('source-change-cover', 'steamgriddb')"
        >SteamGridDB</a-button>
      </div>
      <steam-search-panel
        :query="steamCoverSearchQuery"
        :placeholder="coverSource === 'steamgriddb' ? '搜索 SteamGridDB...' : '搜索 Steam 游戏...'"
        :loading="isSearchingCover"
        :results="coverSearchResults"
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
              :class="{ 'steam-image-selected': selectedCovers.has(index) }"
              @click="emit('toggle-cover-selection', index)"
            >
              <img :src="image" />
              <div v-if="selectedCovers.has(index)" class="steam-screenshot-check">
                <icon-check />
              </div>
            </div>
          </div>
          <a-button
            v-if="selectedCovers.size > 0"
            type="primary"
            long
            :loading="isDownloadingSteamCovers"
            html-type="button"
            @click="emit('download-selected-steam-covers')"
          >
            下载选中的 {{ selectedCovers.size }} 张封面
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
      <div class="source-selector">
        <span class="source-selector__label">数据源</span>
        <a-button
          class="app-text-action-btn"
          :type="bannerSource === 'steam' ? 'outline' : 'text'"
          size="small"
          html-type="button"
          @click="emit('source-change-banner', 'steam')"
        >Steam</a-button>
        <a-button
          class="app-text-action-btn"
          :type="bannerSource === 'steamgriddb' ? 'outline' : 'text'"
          size="small"
          html-type="button"
          :disabled="!sgdbAvailable"
          @click="emit('source-change-banner', 'steamgriddb')"
        >SteamGridDB</a-button>
      </div>
      <steam-search-panel
        :query="steamBannerSearchQuery"
        :placeholder="bannerSource === 'steamgriddb' ? '搜索 SteamGridDB...' : '搜索 Steam 游戏...'"
        :loading="isSearchingBanner"
        :results="bannerSearchResults"
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
              :class="{ 'steam-image-selected': selectedBanners.has(index) }"
              @click="emit('toggle-banner-selection', index)"
            >
              <img :src="image" />
              <div v-if="selectedBanners.has(index)" class="steam-screenshot-check">
                <icon-check />
              </div>
            </div>
          </div>
          <a-button
            v-if="selectedBanners.size > 0"
            type="primary"
            long
            :loading="isDownloadingSteamBanners"
            html-type="button"
            @click="emit('download-selected-steam-banner')"
          >
            下载选中的 {{ selectedBanners.size }} 张横幅
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
      <steam-search-panel
        :query="steamScreenshotSearchQuery"
        placeholder="搜索 Steam 游戏..."
        :loading="isSearchingScreenshots"
        :results="screenshotSearchResults"
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

  <a-modal
    :visible="showLogoSelector"
    title="Logo"
    :width="700"
    :footer="false"
    @update:visible="emit('update:show-logo-selector', $event)"
  >
    <a-tabs v-model:active-key="logoTabKey" type="rounded" size="small">
      <a-tab-pane key="import" title="更换">
        <div class="cover-selector-content">
      <steam-search-panel
        :query="steamLogoSearchQuery"
        placeholder="搜索 Steam 游戏..."
        :loading="isSearchingLogo"
        :results="logoSearchResults"
        :selected-game="selectedSteamLogoGame"
        @update:query="emit('update:steam-logo-search-query', $event)"
        @search="emit('search-logo')"
        @clear="emit('clear-logo')"
        @select="emit('select-logo-game', $event)"
      >
        <div v-if="selectedSteamLogoGame && steamLogoImages.length > 0" class="steam-images-section">
          <div class="steam-search-title">
            {{ selectedSteamLogoGame.name }} 的 Logo
            <a-button class="app-text-action-btn" type="text" size="mini" html-type="button" @click="emit('back-logo-game-search')">返回</a-button>
          </div>
          <div class="steam-images-grid">
            <div
              v-for="(image, index) in steamLogoImages"
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
            :loading="isDownloadingSteamLogos"
            html-type="button"
            @click="emit('download-selected-steam-logo')"
          >
            使用此图片
          </a-button>
        </div>

        <a-empty
          v-else-if="selectedSteamLogoGame && steamLogoImages.length === 0 && !isSearchingLogo"
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
        <a-button class="app-text-action-btn" type="text" html-type="button" @click="emit('update:show-logo-selector', false)">取消</a-button>
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
          <div class="cover-selector-actions">
            <a-button class="app-text-action-btn" type="text" html-type="button" @click="emit('update:show-logo-selector', false)">取消</a-button>
            <a-button type="primary" html-type="button" @click="handleLogoPosConfirm">确定</a-button>
          </div>
        </div>
      </a-tab-pane>
    </a-tabs>
  </a-modal>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { IconCheck, IconImage, IconUpload } from '@arco-design/web-vue/es/icon'
import SteamSearchPanel from '@/components/SteamSearchPanel.vue'
import type { SteamGameSearchResult } from '@/services/types'
import type { ImportSource } from '@/composables/useSteamImport'
import type { FileItem } from '@arco-design/web-vue/es/upload/interfaces'

interface SteamScreenshotsData {
  name: string
  cover: string
  screenshots: string[]
  appId: string
  usedFallbackAssets: boolean
}

const props = defineProps<{
  showSummarySelector: boolean
  steamSummarySearchQuery: string
  isSearchingSteamSummary: boolean
  steamSummarySearchResults: SteamGameSearchResult[]
  selectedSteamSummaryGame: SteamGameSearchResult | null
  steamSummaryPreview: string

  showCoverSelector: boolean
  coverSource: ImportSource
  sgdbAvailable: boolean
  steamCoverSearchQuery: string
  isSearchingCover: boolean
  coverSearchResults: SteamGameSearchResult[]
  selectedSteamGame: SteamGameSearchResult | null
  steamCoverImages: string[]
  selectedCoverImage: string
  selectedCovers: Set<number>
  isDownloadingSteamCovers: boolean
  uploadAction: string
  uploadData: Record<string, string>
  uploadHeaders: Record<string, string>
  coverSearchUrl: string
  coverPreviewUrl: string
  isDownloadingCover: boolean

  showBannerSelector: boolean
  bannerSource: ImportSource
  steamBannerSearchQuery: string
  isSearchingBanner: boolean
  bannerSearchResults: SteamGameSearchResult[]
  selectedSteamBannerGame: SteamGameSearchResult | null
  steamBannerImages: string[]
  selectedBanners: Set<number>
  isDownloadingSteamBanners: boolean
  bannerUploadAction: string
  bannerUploadData: Record<string, string>
  bannerSearchUrl: string
  bannerPreviewUrl: string
  isDownloadingBanner: boolean

  showScreenshotSelector: boolean
  steamScreenshotSearchQuery: string
  isSearchingScreenshots: boolean
  screenshotSearchResults: SteamGameSearchResult[]
  selectedSteamScreenshotGame: SteamGameSearchResult | null
  steamScreenshotsData: SteamScreenshotsData | null
  selectedSteamScreenshots: Set<number>
  isDownloadingSteamScreenshots: boolean
  screenshotUploadAction: string
  screenshotUploadData: Record<string, string>
  screenshotSearchUrl: string
  screenshotPreviewUrl: string
  isDownloadingScreenshot: boolean

  showLogoSelector: boolean
  steamLogoSearchQuery: string
  isSearchingLogo: boolean
  logoSearchResults: SteamGameSearchResult[]
  selectedSteamLogoGame: SteamGameSearchResult | null
  steamLogoImages: string[]
  selectedLogoImage: string
  isDownloadingSteamLogos: boolean
  logoUploadAction: string
  logoUploadData: Record<string, string>
  logoSearchUrl: string
  logoPreviewUrl: string
  isDownloadingLogo: boolean
  logoBannerSrc: string
  logoPath: string
  logoPositionX: number | null
  logoPositionY: number | null
  logoWidthPct: number | null
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
  'source-change-cover': [source: ImportSource]
  'update:steam-cover-search-query': [value: string]
  'search-cover': []
  'clear-cover': []
  'select-cover-game': [game: SteamGameSearchResult]
  'back-cover-game-search': []
  'update:selected-cover-image': [value: string]
  'download-selected-steam-cover': []
  'toggle-cover-selection': [index: number]
  'download-selected-steam-covers': []
  'cover-upload-success': [fileItem: FileItem]
  'cover-upload-error': []
  'update:cover-search-url': [value: string]
  'load-cover-from-url': []
  'confirm-cover-selection': []
  'cover-image-error': [event: Event]

  'update:show-banner-selector': [value: boolean]
  'source-change-banner': [source: ImportSource]
  'update:steam-banner-search-query': [value: string]
  'search-banner': []
  'clear-banner': []
  'select-banner-game': [game: SteamGameSearchResult]
  'back-banner-game-search': []
  'toggle-banner-selection': [index: number]
  'download-selected-steam-banner': []
  'banner-upload-success': [fileItem: FileItem]
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
  'screenshot-upload-success': [fileItem: FileItem]
  'screenshot-upload-error': []
  'update:screenshot-search-url': [value: string]
  'load-screenshot-preview': []
  'confirm-screenshot-selection': []

  'update:show-logo-selector': [value: boolean]
  'update:steam-logo-search-query': [value: string]
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
  'confirm-logo-position': [payload: { position_x: number; position_y: number; width_pct: number }]
}>()

// Logo position editor state
const logoTabKey = ref('import')
const logoPosEditorRef = ref<HTMLElement | null>(null)
const logoPosWidth = ref(30)
const logoPosX = ref(50)
const logoPosY = ref(50)

watch(() => props.showLogoSelector, (v) => {
  if (v) {
    logoPosX.value = props.logoPositionX ?? 50
    logoPosY.value = props.logoPositionY ?? 50
    logoPosWidth.value = props.logoWidthPct ?? 30
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
  })
  emit('update:show-logo-selector', false)
}
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
  max-height: 400px;
  overflow-y: auto;
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
  color: rgba(255, 255, 255, 0.5);
  background: rgba(0, 0, 0, 0.4);
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
