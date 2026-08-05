<template>
  <summary-import-modal
    :visible="showSummarySelector"
    :steam-summary-search-query="steamSummarySearchQuery"
    :is-searching-steam-summary="isSearchingSteamSummary"
    :steam-summary-search-results="steamSummarySearchResults"
    :selected-steam-summary-game="selectedSteamSummaryGame"
    :steam-summary-preview="steamSummaryPreview"
    @update:visible="emit('update:show-summary-selector', $event)"
    @update:steam-summary-search-query="emit('update:steam-summary-search-query', $event)"
    @search-summary="emit('search-summary')"
    @clear-summary="emit('clear-summary')"
    @select-summary="emit('select-summary', $event)"
    @back-summary="emit('back-summary')"
    @confirm-summary-import="emit('confirm-summary-import')"
  />

  <cover-banner-selector-modal
    mode="cover"
    :visible="showCoverSelector"
    :source="coverSource"
    :sgdb-available="sgdbAvailable"
    :search-query="steamCoverSearchQuery"
    :is-searching="isSearchingCover"
    :search-results="coverSearchResults"
    :selected-game="selectedSteamGame"
    :images="steamCoverImages"
    :selected-images="selectedCovers"
    :is-downloading-steam="isDownloadingSteamCovers"
    :upload-action="uploadAction"
    :upload-data="uploadData"
    :upload-headers="uploadHeaders"
    :search-url="coverSearchUrl"
    :preview-url="coverPreviewUrl"
    :is-downloading="isDownloadingCover"
    @update:visible="emit('update:show-cover-selector', $event)"
    @source-change="emit('source-change-cover', $event)"
    @update:search-query="emit('update:steam-cover-search-query', $event)"
    @search="emit('search-cover')"
    @clear="emit('clear-cover')"
    @select-game="emit('select-cover-game', $event)"
    @back-game-search="emit('back-cover-game-search')"
    @toggle-selection="emit('toggle-cover-selection', $event)"
    @select-all="emit('select-all-covers')"
    @invert-selection="emit('invert-selection-covers')"
    @download-selected-steam="emit('download-selected-steam-covers')"
    @upload-success="emit('cover-upload-success', $event)"
    @upload-error="emit('cover-upload-error')"
    @update:search-url="emit('update:cover-search-url', $event)"
    @load-from-url="emit('load-cover-from-url')"
    @confirm-selection="emit('confirm-cover-selection')"
    @image-error="emit('cover-image-error', $event)"
  />

  <cover-banner-selector-modal
    mode="banner"
    :visible="showBannerSelector"
    :source="bannerSource"
    :sgdb-available="sgdbAvailable"
    :search-query="steamBannerSearchQuery"
    :is-searching="isSearchingBanner"
    :search-results="bannerSearchResults"
    :selected-game="selectedSteamBannerGame"
    :images="steamBannerImages"
    :selected-images="selectedBanners"
    :is-downloading-steam="isDownloadingSteamBanners"
    :upload-action="bannerUploadAction"
    :upload-data="bannerUploadData"
    :upload-headers="uploadHeaders"
    :search-url="bannerSearchUrl"
    :preview-url="bannerPreviewUrl"
    :is-downloading="isDownloadingBanner"
    @update:visible="emit('update:show-banner-selector', $event)"
    @source-change="emit('source-change-banner', $event)"
    @update:search-query="emit('update:steam-banner-search-query', $event)"
    @search="emit('search-banner')"
    @clear="emit('clear-banner')"
    @select-game="emit('select-banner-game', $event)"
    @back-game-search="emit('back-banner-game-search')"
    @toggle-selection="emit('toggle-banner-selection', $event)"
    @select-all="emit('select-all-banners')"
    @invert-selection="emit('invert-selection-banners')"
    @download-selected-steam="emit('download-selected-steam-banner')"
    @upload-success="emit('banner-upload-success', $event)"
    @upload-error="emit('banner-upload-error')"
    @update:search-url="emit('update:banner-search-url', $event)"
    @load-from-url="emit('load-banner-from-url')"
    @confirm-selection="emit('confirm-banner-selection')"
    @image-error="emit('cover-image-error', $event)"
  />

  <screenshot-selector-modal
    :visible="showScreenshotSelector"
    :source="screenshotSource"
    :sgdb-available="sgdbAvailable"
    :screenshot-search-query="screenshotSearchQuery"
    :is-searching-screenshots="isSearchingScreenshots"
    :screenshot-search-results="screenshotSearchResults"
    :selected-screenshot-game="selectedScreenshotGame"
    :screenshot-candidates-data="screenshotCandidatesData"
    :selected-remote-screenshots="selectedRemoteScreenshots"
    :is-downloading-remote-screenshots="isDownloadingRemoteScreenshots"
    :screenshot-upload-action="screenshotUploadAction"
    :screenshot-upload-data="screenshotUploadData"
    :upload-headers="uploadHeaders"
    :screenshot-search-url="screenshotSearchUrl"
    :screenshot-preview-url="screenshotPreviewUrl"
    :is-downloading-screenshot="isDownloadingScreenshot"
    @update:visible="emit('update:show-screenshot-selector', $event)"
    @source-change="emit('source-change-screenshot', $event)"
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
  />

  <logo-selector-modal
    :visible="showLogoSelector"
    :source="logoSource"
    :sgdb-available="sgdbAvailable"
    :logo-search-query="logoSearchQuery"
    :is-searching-logo="isSearchingLogo"
    :logo-search-results="logoSearchResults"
    :selected-logo-game="selectedLogoGame"
    :logo-images="logoImages"
    :selected-logo-image="selectedLogoImage"
    :is-downloading-logos="isDownloadingLogos"
    :logo-upload-action="logoUploadAction"
    :logo-upload-data="logoUploadData"
    :upload-headers="uploadHeaders"
    :logo-search-url="logoSearchUrl"
    :logo-preview-url="logoPreviewUrl"
    :is-downloading-logo="isDownloadingLogo"
    :logo-banner-src="logoBannerSrc"
    :logo-path="logoPath"
    :logo-position-x="logoPositionX"
    :logo-position-y="logoPositionY"
    :logo-width-pct="logoWidthPct"
    :logo-visible="logoVisible"
    :initial-tab="logoInitialTab"
    @update:visible="emit('update:show-logo-selector', $event)"
    @source-change="emit('source-change-logo', $event)"
    @update:logo-search-query="emit('update:logo-search-query', $event)"
    @search-logo="emit('search-logo')"
    @clear-logo="emit('clear-logo')"
    @select-logo-game="emit('select-logo-game', $event)"
    @back-logo-game-search="emit('back-logo-game-search')"
    @update:selected-logo-image="emit('update:selected-logo-image', $event)"
    @download-selected-steam-logo="emit('download-selected-logo')"
    @logo-upload-success="emit('logo-upload-success', $event)"
    @logo-upload-error="emit('logo-upload-error')"
    @update:logo-search-url="emit('update:logo-search-url', $event)"
    @load-logo-from-url="emit('load-logo-from-url')"
    @confirm-logo-selection="emit('confirm-logo-selection')"
    @confirm-logo-position="emit('confirm-logo-position', $event)"
  />
</template>

<script setup lang="ts">
import SummaryImportModal from '@/components/edit-game/import-modals/SummaryImportModal.vue'
import CoverBannerSelectorModal from '@/components/edit-game/import-modals/CoverBannerSelectorModal.vue'
import ScreenshotSelectorModal from '@/components/edit-game/import-modals/ScreenshotSelectorModal.vue'
import LogoSelectorModal from '@/components/edit-game/import-modals/LogoSelectorModal.vue'
import type { SteamGameSearchResult } from '@/services/types'
import type { ImportSource } from '@/composables/useSteamImport'
import type { FileItem } from '@arco-design/web-vue/es/upload/interfaces'

interface ScreenshotCandidatesData {
  name: string
  screenshots: string[]
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
  logoSource: ImportSource
  logoSearchQuery: string
  isSearchingLogo: boolean
  logoSearchResults: SteamGameSearchResult[]
  selectedLogoGame: SteamGameSearchResult | null
  logoImages: string[]
  selectedLogoImage: string
  isDownloadingLogos: boolean
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
  logoVisible: boolean
  logoInitialTab: 'import' | 'position'
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
  'source-change-banner': [source: ImportSource]
  'update:steam-banner-search-query': [value: string]
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
  'source-change-logo': [source: ImportSource]
  'update:logo-search-query': [value: string]
  'search-logo': []
  'clear-logo': []
  'select-logo-game': [game: SteamGameSearchResult]
  'back-logo-game-search': []
  'update:selected-logo-image': [value: string]
  'download-selected-logo': []
  'logo-upload-success': [fileItem: FileItem]
  'logo-upload-error': []
  'update:logo-search-url': [value: string]
  'load-logo-from-url': []
  'confirm-logo-selection': []
  'confirm-logo-position': [payload: { position_x: number; position_y: number; width_pct: number; logo_visible: boolean }]
}>()
</script>
