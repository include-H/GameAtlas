<template>
  <div class="media-management-panel">
    <game-media-section
      :covers="covers"
      :banners="banners"
      :preview-videos="previewVideos"
      :is-uploading-video="isUploadingVideo"
      :video-upload-progress="videoUploadProgress"
      :video-upload-file-name="videoUploadFileName"
      :screenshots="screenshots"
      :dragged-screenshot-key="draggedScreenshotKey"
      :drag-over-screenshot-key="dragOverScreenshotKey"
      :dragged-cover-key="draggedCoverKey"
      :drag-over-cover-key="dragOverCoverKey"
      :dragged-banner-key="draggedBannerKey"
      :drag-over-banner-key="dragOverBannerKey"
      :logos="logos"
      :dragged-logo-key="draggedLogoKey"
      :drag-over-logo-key="dragOverLogoKey"
      :dragged-video-key="draggedVideoKey"
      :drag-over-video-key="dragOverVideoKey"
      :logo-visible="logoVisible"
      @open-cover-selector="emit('open-cover-selector')"
      @remove-cover="(index) => emit('remove-cover', index)"
      @cover-drag-start="(key) => emit('cover-drag-start', key)"
      @cover-drag-enter="(key) => emit('cover-drag-enter', key)"
      @cover-drop="(key) => emit('cover-drop', key)"
      @cover-drag-end="emit('cover-drag-end')"
      @open-banner-selector="emit('open-banner-selector')"
      @remove-banner="(index) => emit('remove-banner', index)"
      @banner-drag-start="(key) => emit('banner-drag-start', key)"
      @banner-drag-enter="(key) => emit('banner-drag-enter', key)"
      @banner-drop="(key) => emit('banner-drop', key)"
      @banner-drag-end="emit('banner-drag-end')"
      @logo-drag-start="(key) => emit('logo-drag-start', key)"
      @logo-drag-enter="(key) => emit('logo-drag-enter', key)"
      @logo-drop="(key) => emit('logo-drop', key)"
      @logo-drag-end="emit('logo-drag-end')"
      @video-file-change="emit('video-file-change', $event)"
      @select-poster="emit('select-poster', $event)"
      @video-drag-start="emit('video-drag-start', $event)"
      @video-drag-enter="emit('video-drag-enter', $event)"
      @video-drop="emit('video-drop', $event)"
      @video-drag-end="emit('video-drag-end')"
      @remove-video="emit('remove-video', $event)"
      @open-screenshot-selector="emit('open-screenshot-selector')"
      @remove-screenshot="emit('remove-screenshot', $event)"
      @screenshot-drag-start="emit('screenshot-drag-start', $event)"
      @screenshot-drag-enter="emit('screenshot-drag-enter', $event)"
      @screenshot-drop="emit('screenshot-drop', $event)"
      @screenshot-drag-end="emit('screenshot-drag-end')"
      @open-logo-selector="emit('open-logo-selector')"
      @logo-position-change="emit('logo-position-change', $event)"
      @remove-logo="emit('remove-logo', $event)"
    />

    <edit-game-asset-import-modals
      :show-summary-selector="showSummarySelector"
      :steam-summary-search-query="steamSummarySearchQuery"
      :is-searching-steam-summary="isSearchingSteamSummary"
      :steam-summary-search-results="steamSummarySearchResults"
      :selected-steam-summary-game="selectedSteamSummaryGame"
      :steam-summary-preview="steamSummaryPreview"
      :show-cover-selector="showCoverSelector"
      :cover-source="coverSource"
      :sgdb-available="sgdbAvailable"
      :steam-cover-search-query="steamCoverSearchQuery"
      :is-searching-cover="isSearchingCover"
      :cover-search-results="coverSearchResults"
      :selected-steam-game="selectedSteamGame"
      :steam-cover-images="steamCoverImages"
      :selected-cover-image="selectedCoverImage"
      :selected-covers="selectedCovers"
      :is-downloading-steam-covers="isDownloadingSteamCovers"
      :upload-action="uploadAction"
      :upload-data="uploadData"
      :upload-headers="uploadHeaders"
      :cover-search-url="coverSearchUrl"
      :cover-preview-url="coverPreviewUrl"
      :is-downloading-cover="isDownloadingCover"
      :show-banner-selector="showBannerSelector"
      :banner-source="bannerSource"
      :steam-banner-search-query="steamBannerSearchQuery"
      :is-searching-banner="isSearchingBanner"
      :banner-search-results="bannerSearchResults"
      :selected-steam-banner-game="selectedSteamBannerGame"
      :steam-banner-images="steamBannerImages"
      :selected-banners="selectedBanners"
      :is-downloading-steam-banners="isDownloadingSteamBanners"
      :banner-upload-action="bannerUploadAction"
      :banner-upload-data="bannerUploadData"
      :banner-search-url="bannerSearchUrl"
      :banner-preview-url="bannerPreviewUrl"
      :is-downloading-banner="isDownloadingBanner"
      :show-screenshot-selector="showScreenshotSelector"
      :screenshot-source="screenshotSource"
      :screenshot-search-query="screenshotSearchQuery"
      :is-searching-screenshots="isSearchingScreenshots"
      :screenshot-search-results="screenshotSearchResults"
      :selected-screenshot-game="selectedScreenshotGame"
      :screenshot-candidates-data="screenshotCandidatesData"
      :selected-remote-screenshots="selectedRemoteScreenshots"
      :is-downloading-remote-screenshots="isDownloadingRemoteScreenshots"
      :screenshot-upload-action="screenshotUploadAction"
      :screenshot-upload-data="screenshotUploadData"
      :screenshot-search-url="screenshotSearchUrl"
      :screenshot-preview-url="screenshotPreviewUrl"
      :is-downloading-screenshot="isDownloadingScreenshot"
      :show-logo-selector="showLogoSelector"
      :logo-search-query="logoSearchQuery"
      :is-searching-logo="isSearchingLogo"
      :logo-search-results="logoSearchResults"
      :selected-logo-game="selectedLogoGame"
      :logo-images="logoImages"
      :selected-logos="selectedLogos"
      :is-downloading-logos="isDownloadingLogos"
      :logo-upload-action="logoUploadAction"
      :logo-upload-data="logoUploadData"
      :logo-search-url="logoSearchUrl"
      :logo-preview-url="logoPreviewUrl"
      :is-downloading-logo="isDownloadingLogo"
      @update:show-summary-selector="emit('update:show-summary-selector', $event)"
      @update:steam-summary-search-query="emit('update:steam-summary-search-query', $event)"
      @search-summary="emit('search-summary')"
      @clear-summary="emit('clear-summary')"
      @select-summary="emit('select-summary', $event)"
      @back-summary="emit('back-summary')"
      @confirm-summary-import="emit('confirm-summary-import')"
      @update:show-cover-selector="emit('update:show-cover-selector', $event)"
      @update:steam-cover-search-query="emit('update:steam-cover-search-query', $event)"
      @source-change-cover="emit('source-change-cover', $event)"
      @search-cover="emit('search-cover')"
      @clear-cover="emit('clear-cover')"
      @select-cover-game="emit('select-cover-game', $event)"
      @back-cover-game-search="emit('back-cover-game-search')"
      @update:selected-cover-image="emit('update:selected-cover-image', $event)"
      @download-selected-steam-cover="emit('download-selected-steam-cover')"
      @toggle-cover-selection="emit('toggle-cover-selection', $event)"
      @select-all-covers="emit('select-all-covers')"
      @invert-selection-covers="emit('invert-selection-covers')"
      @download-selected-steam-covers="emit('download-selected-steam-covers')"
      @cover-upload-success="emit('cover-upload-success', $event)"
      @cover-upload-error="emit('cover-upload-error')"
      @update:cover-search-url="emit('update:cover-search-url', $event)"
      @load-cover-from-url="emit('load-cover-from-url')"
      @confirm-cover-selection="emit('confirm-cover-selection')"
      @cover-image-error="emit('cover-image-error', $event)"
      @update:show-banner-selector="emit('update:show-banner-selector', $event)"
      @update:steam-banner-search-query="emit('update:steam-banner-search-query', $event)"
      @source-change-banner="emit('source-change-banner', $event)"
      @search-banner="emit('search-banner')"
      @clear-banner="emit('clear-banner')"
      @select-banner-game="emit('select-banner-game', $event)"
      @back-banner-game-search="emit('back-banner-game-search')"
      @toggle-banner-selection="emit('toggle-banner-selection', $event)"
      @select-all-banners="emit('select-all-banners')"
      @invert-selection-banners="emit('invert-selection-banners')"
      @download-selected-steam-banner="emit('download-selected-steam-banner')"
      @banner-upload-success="emit('banner-upload-success', $event)"
      @banner-upload-error="emit('banner-upload-error')"
      @update:banner-search-url="emit('update:banner-search-url', $event)"
      @load-banner-from-url="emit('load-banner-from-url')"
      @confirm-banner-selection="emit('confirm-banner-selection')"
      @update:show-screenshot-selector="emit('update:show-screenshot-selector', $event)"
      @source-change-screenshot="emit('source-change-screenshot', $event)"
      @update:screenshot-search-query="emit('update:screenshot-search-query', $event)"
      @search-screenshots="emit('search-screenshots')"
      @clear-screenshots="emit('clear-screenshots')"
      @select-screenshot-game="emit('select-screenshot-game', $event)"
      @back-screenshot-game-search="emit('back-screenshot-game-search')"
      @toggle-screenshot="emit('toggle-screenshot', $event)"
      @select-all-screenshots="emit('select-all-screenshots')"
      @invert-screenshots="emit('invert-screenshots')"
      @download-selected-screenshots="emit('download-selected-screenshots')"
      @screenshot-upload-success="emit('screenshot-upload-success', $event)"
      @screenshot-upload-error="emit('screenshot-upload-error')"
      @update:screenshot-search-url="emit('update:screenshot-search-url', $event)"
      @load-screenshot-preview="emit('load-screenshot-preview')"
      @confirm-screenshot-selection="emit('confirm-screenshot-selection')"
      @update:show-logo-selector="emit('update:show-logo-selector', $event)"
      @update:logo-search-query="emit('update:logo-search-query', $event)"
      @search-logo="emit('search-logo')"
      @clear-logo="emit('clear-logo')"
      @select-logo-game="emit('select-logo-game', $event)"
      @back-logo-game-search="emit('back-logo-game-search')"
      @toggle-logo-selection="emit('toggle-logo-selection', $event)"
      @select-all-logos="emit('select-all-logos')"
      @invert-selection-logos="emit('invert-selection-logos')"
      @download-selected-logos="emit('download-selected-logos')"
      @logo-upload-success="emit('logo-upload-success', $event)"
      @logo-upload-error="emit('logo-upload-error')"
      @update:logo-search-url="emit('update:logo-search-url', $event)"
      @load-logo-from-url="emit('load-logo-from-url')"
      @confirm-logo-selection="emit('confirm-logo-selection')"
    />

  </div>
</template>

<script setup lang="ts">
import GameMediaSection from './GameMediaSection.vue'
import EditGameAssetImportModals from './EditGameAssetImportModals.vue'
import type { ImportSource } from '@/composables/useSteamImport'
import type { SteamGameSearchResult } from '@/services/types'
import type { LogoPositionChange } from '@/utils/edit-game-form'
import type { FileItem } from '@arco-design/web-vue/es/upload/interfaces'

interface EditableCover {
  asset_uid?: string
  path: string
}

interface EditableBanner {
  asset_uid?: string
  path: string
}

interface EditableLogo {
  asset_uid?: string
  path: string
  position_x: number | null
  position_y: number | null
  width_pct: number | null
}

interface EditableScreenshot {
  asset_uid?: string
  path: string
  client_key: string
}

interface EditableVideo {
  asset_uid?: string
  path: string
}

interface ScreenshotCandidatesData {
  name: string
  screenshots: string[]
  usedFallbackAssets: boolean
}

defineProps<{
  covers: EditableCover[]
  banners: EditableBanner[]
  logos: EditableLogo[]
  previewVideos: EditableVideo[]
  isUploadingVideo: boolean
  videoUploadProgress: number
  videoUploadFileName: string
  screenshots: EditableScreenshot[]
  draggedScreenshotKey: string | null
  dragOverScreenshotKey: string | null
  draggedCoverKey: string | null
  dragOverCoverKey: string | null
  draggedBannerKey: string | null
  dragOverBannerKey: string | null
  draggedLogoKey: string | null
  dragOverLogoKey: string | null
  draggedVideoKey: string | null
  dragOverVideoKey: string | null

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
  screenshotSource: ImportSource
  screenshotSearchQuery: string
  isSearchingScreenshots: boolean
  screenshotSearchResults: SteamGameSearchResult[]
  selectedScreenshotGame: SteamGameSearchResult | null
  screenshotCandidatesData: ScreenshotCandidatesData | null
  selectedRemoteScreenshots: Set<number>
  isDownloadingRemoteScreenshots: boolean
  screenshotUploadAction: string
  screenshotUploadData: Record<string, string>
  screenshotSearchUrl: string
  screenshotPreviewUrl: string
  isDownloadingScreenshot: boolean

  showLogoSelector: boolean
  logoSearchQuery: string
  isSearchingLogo: boolean
  logoSearchResults: SteamGameSearchResult[]
  selectedLogoGame: SteamGameSearchResult | null
  logoImages: string[]
  selectedLogos: Set<number>
  isDownloadingLogos: boolean
  logoUploadAction: string
  logoUploadData: Record<string, string>
  logoSearchUrl: string
  logoPreviewUrl: string
  isDownloadingLogo: boolean
  logoVisible: boolean

}>()

const emit = defineEmits<{
  'open-cover-selector': []
  'remove-cover': [index: number]
  'cover-drag-start': [key: string]
  'cover-drag-enter': [key: string]
  'cover-drop': [key: string]
  'cover-drag-end': []
  'open-banner-selector': []
  'remove-banner': [index: number]
  'banner-drag-start': [key: string]
  'banner-drag-enter': [key: string]
  'banner-drop': [key: string]
  'banner-drag-end': []
  'logo-drag-start': [key: string]
  'logo-drag-enter': [key: string]
  'logo-drop': [key: string]
  'logo-drag-end': []
  'video-file-change': [event: Event]
  'select-poster': [video: unknown]
  'video-drag-start': [key: string]
  'video-drag-enter': [key: string]
  'video-drop': [key: string]
  'video-drag-end': []
  'remove-video': [assetUid?: string]
  'open-screenshot-selector': []
  'remove-screenshot': [clientKey: string]
  'screenshot-drag-start': [clientKey: string]
  'screenshot-drag-enter': [clientKey: string]
  'screenshot-drop': [clientKey: string]
  'screenshot-drag-end': []
  'open-logo-selector': []
  'logo-position-change': [payload: LogoPositionChange]
  'remove-logo': [index: number]

  'update:show-summary-selector': [value: boolean]
  'update:steam-summary-search-query': [value: string]
  'search-summary': []
  'clear-summary': []
  'select-summary': [game: SteamGameSearchResult]
  'back-summary': []
  'confirm-summary-import': []
  'update:show-cover-selector': [value: boolean]
  'update:steam-cover-search-query': [value: string]
  'source-change-cover': [source: ImportSource]
  'search-cover': []
  'clear-cover': []
  'select-cover-game': [game: SteamGameSearchResult]
  'back-cover-game-search': []
  'update:selected-cover-image': [value: string]
  'download-selected-steam-cover': []
  'toggle-cover-selection': [index: number]
  'select-all-covers': []
  'invert-selection-covers': []
  'download-selected-steam-covers': []
  'cover-upload-success': [fileItem: FileItem]
  'cover-upload-error': []
  'update:cover-search-url': [value: string]
  'load-cover-from-url': []
  'confirm-cover-selection': []
  'cover-image-error': [event: Event]
  'update:show-banner-selector': [value: boolean]
  'update:steam-banner-search-query': [value: string]
  'source-change-banner': [source: ImportSource]
  'search-banner': []
  'clear-banner': []
  'select-banner-game': [game: SteamGameSearchResult]
  'back-banner-game-search': []
  'toggle-banner-selection': [index: number]
  'select-all-banners': []
  'invert-selection-banners': []
  'download-selected-steam-banner': []
  'banner-upload-success': [fileItem: FileItem]
  'banner-upload-error': []
  'update:banner-search-url': [value: string]
  'load-banner-from-url': []
  'confirm-banner-selection': []
  'update:show-screenshot-selector': [value: boolean]
  'source-change-screenshot': [source: ImportSource]
  'update:screenshot-search-query': [value: string]
  'search-screenshots': []
  'clear-screenshots': []
  'select-screenshot-game': [game: SteamGameSearchResult]
  'back-screenshot-game-search': []
  'toggle-screenshot': [index: number]
  'select-all-screenshots': []
  'invert-screenshots': []
  'download-selected-screenshots': []
  'screenshot-upload-success': [fileItem: FileItem]
  'screenshot-upload-error': []
  'update:screenshot-search-url': [value: string]
  'load-screenshot-preview': []
  'confirm-screenshot-selection': []
  'update:show-logo-selector': [value: boolean]
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
.media-management-panel {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
</style>
