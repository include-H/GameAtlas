<template>
  <div class="game-media-page">
    <header class="game-media-page__header glass-header">
      <div class="game-media-page__heading">
        <a-button
          class="app-text-action-btn game-media-page__back"
          type="text"
          html-type="button"
          @click="goBack"
        >
          <template #icon><icon-left /></template>
          返回
        </a-button>
        <h1 class="game-media-page__title">
          {{ game?.title || form.title || '素材管理' }}
        </h1>
      </div>
      <div class="game-media-page__actions">
        <a-button
          class="app-text-action-btn"
          type="text"
          html-type="button"
          :disabled="isSubmitting"
          @click="handleSave"
        >
          <template #icon><icon-save /></template>
          保存
        </a-button>
        <a-button
          type="primary"
          html-type="button"
          :loading="isSubmitting"
          @click="handleSaveAndExit"
        >
          <template #icon><icon-save /></template>
          保存并退出
        </a-button>
      </div>
    </header>

    <main class="game-media-page__body">
      <a-spin v-if="loading" class="game-media-page__loading" :size="28" />
      <a-empty v-else-if="loadError" class="game-media-page__error">
        <template #description>
          <div class="game-media-page__error-text">
            <h3>加载素材失败</h3>
            <p>没有取到这条游戏的素材数据，请稍后重试。</p>
          </div>
        </template>
        <template #extra>
          <a-button type="primary" html-type="button" @click="loadGame">重试</a-button>
        </template>
      </a-empty>
      <template v-else-if="game">
        <media-management-panel
          :covers="form.covers"
          :banners="form.banners"
          :preview-videos="form.preview_videos"
          :is-uploading-video="isUploadingVideo"
          :video-upload-progress="videoUploadProgress"
          :video-upload-file-name="videoUploadFileName"
          :screenshots="form.screenshots"
          :dragged-screenshot-key="draggedScreenshotKey"
          :drag-over-screenshot-key="dragOverScreenshotKey"
          :dragged-cover-key="draggedCoverKey"
          :drag-over-cover-key="dragOverCoverKey"
          :dragged-banner-key="draggedBannerKey"
          :drag-over-banner-key="dragOverBannerKey"
          :logos="form.logos"
          :dragged-logo-key="draggedLogoKey"
          :drag-over-logo-key="dragOverLogoKey"
          :dragged-video-key="draggedVideoKey"
          :drag-over-video-key="dragOverVideoKey"
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
          :logo-visible="form.logo_visible ?? true"
          @open-cover-selector="showCoverSelector = true"
          @remove-cover="removeCover"
          @cover-drag-start="handleCoverDragStart"
          @cover-drag-enter="handleCoverDragEnter"
          @cover-drop="handleCoverDrop"
          @cover-drag-end="handleCoverDragEnd"
          @open-banner-selector="showBannerSelector = true"
          @remove-banner="removeBanner"
          @banner-drag-start="handleBannerDragStart"
          @banner-drag-enter="handleBannerDragEnter"
          @banner-drop="handleBannerDrop"
          @banner-drag-end="handleBannerDragEnd"
          @logo-drag-start="handleLogoDragStart"
          @logo-drag-enter="handleLogoDragEnter"
          @logo-drop="handleLogoDrop"
          @logo-drag-end="handleLogoDragEnd"
          @video-file-change="handleVideoFileChange"
          @video-drag-start="handleVideoDragStart"
          @video-drag-enter="handleVideoDragEnter"
          @video-drop="handleVideoDrop"
          @video-drag-end="handleVideoDragEnd"
          @remove-video="removePreviewVideo"
          @open-screenshot-selector="showScreenshotSelector = true"
          @remove-screenshot="removeScreenshot"
          @screenshot-drag-start="handleScreenshotDragStart"
          @screenshot-drag-enter="handleScreenshotDragEnter"
          @screenshot-drop="handleScreenshotDrop"
          @screenshot-drag-end="handleScreenshotDragEnd"
          @open-logo-selector="openLogoSelector"
          @logo-position-change="handleLogoPositionChange"
          @remove-logo="removeLogo"
          @update:show-summary-selector="showSummarySelector = $event"
          @update:steam-summary-search-query="steamSummarySearchQuery = $event"
          @search-summary="searchSteamForSummary"
          @clear-summary="handleSummarySearchClear"
          @select-summary="selectSteamSummaryGame"
          @back-summary="backToSummarySearch"
          @confirm-summary-import="confirmSummaryImport"
          @update:show-cover-selector="showCoverSelector = $event"
          @update:steam-cover-search-query="steamCoverSearchQuery = $event"
          @source-change-cover="coverSource = $event"
          @search-cover="searchSteamForCover"
          @clear-cover="handleCoverSearchClear"
          @select-cover-game="selectSteamCoverGame"
          @back-cover-game-search="backToCoverGameSearch"
          @update:selected-cover-image="selectedCoverImage = $event"
          @download-selected-steam-cover="downloadSelectedSteamCover"
          @toggle-cover-selection="toggleCoverSelection"
          @select-all-covers="selectAllCovers"
          @invert-selection-covers="invertSelectionCovers"
          @download-selected-steam-covers="downloadSelectedSteamCovers"
          @cover-upload-success="handleCoverUploadSuccess"
          @cover-upload-error="handleCoverUploadError"
          @update:cover-search-url="coverSearchUrl = $event"
          @load-cover-from-url="loadCoverFromUrl"
          @confirm-cover-selection="confirmCoverSelection"
          @cover-image-error="handleCoverError"
          @update:show-banner-selector="showBannerSelector = $event"
          @update:steam-banner-search-query="steamBannerSearchQuery = $event"
          @source-change-banner="bannerSource = $event"
          @search-banner="searchSteamForBanner"
          @clear-banner="handleBannerSearchClear"
          @select-banner-game="selectSteamBannerGame"
          @back-banner-game-search="backToBannerGameSearch"
          @toggle-banner-selection="toggleBannerSelection"
          @select-all-banners="selectAllBanners"
          @invert-selection-banners="invertSelectionBanners"
          @download-selected-steam-banner="downloadSelectedSteamBanner"
          @banner-upload-success="handleBannerUploadSuccess"
          @banner-upload-error="handleBannerUploadError"
          @update:banner-search-url="bannerSearchUrl = $event"
          @load-banner-from-url="loadBannerFromUrl"
          @confirm-banner-selection="confirmBannerSelection"
          @update:show-screenshot-selector="showScreenshotSelector = $event"
          @source-change-screenshot="screenshotSource = $event"
          @update:screenshot-search-query="screenshotSearchQuery = $event"
          @search-screenshots="searchScreenshots"
          @clear-screenshots="handleScreenshotSearchClear"
          @select-screenshot-game="selectScreenshotGame"
          @back-screenshot-game-search="backToScreenshotGameSearch"
          @toggle-screenshot="toggleScreenshot"
          @select-all-screenshots="selectAllScreenshots"
          @invert-screenshots="invertSelectionScreenshots"
          @download-selected-screenshots="downloadSelectedScreenshots"
          @screenshot-upload-success="handleScreenshotUploadSuccess"
          @screenshot-upload-error="handleScreenshotUploadError"
          @update:screenshot-search-url="screenshotSearchUrl = $event"
          @load-screenshot-preview="loadScreenshotPreview"
          @confirm-screenshot-selection="confirmScreenshotSelection"
          @update:show-logo-selector="showLogoSelector = $event"
          @update:logo-search-query="logoSearchQuery = $event"
          @search-logo="searchLogos"
          @clear-logo="handleLogoSearchClear"
          @select-logo-game="selectLogoGame"
          @back-logo-game-search="backToLogoGameSearch"
          @toggle-logo-selection="toggleLogoSelection"
          @select-all-logos="selectAllLogos"
          @invert-selection-logos="invertSelectionLogos"
          @download-selected-logos="downloadSelectedLogos"
          @logo-upload-success="handleLogoUploadSuccess"
          @logo-upload-error="handleLogoUploadError"
          @update:logo-search-url="logoSearchUrl = $event"
          @load-logo-from-url="loadLogoFromUrl"
          @confirm-logo-selection="confirmLogoSelection"
        />
      </template>
    </main>

  </div>
</template>

<script setup lang="ts">
import { nextTick, onBeforeUnmount, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { IconLeft, IconSave } from '@arco-design/web-vue/es/icon'
import gamesService from '@/services/games.service'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import { useEditGameModal } from '@/composables/useEditGameModal'
import MediaManagementPanel from '@/components/edit-game/MediaManagementPanel.vue'
import type { AdminGameDetail } from '@/services/types'
import {
  getAmbientBackgroundPoolFromGameDetail,
  hasAmbientBackgroundPoolImages,
} from '@/utils/ambient-background'
import { navigateBackOrFallback } from '@/utils/navigation'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const uiStore = useUiStore()

const publicId = ref(String(route.params.publicId || ''))
const game = ref<AdminGameDetail | null>(null)
const loading = ref(true)
const loadError = ref(false)
const AMBIENT_BACKGROUND_OWNER = 'game-media'

const formRef = ref()
const isSubmitting = ref(false)
const activeTab = ref('media')

const modalProps = reactive({
  visible: true,
  game,
})

const pendingSaveMode = ref<'stay' | 'exit'>('exit')

const goBack = () => {
  navigateBackOrFallback(router, {
    name: 'game-detail',
    params: { publicId: publicId.value },
  })
}

const modalEmit = (event: 'update:visible' | 'success' | 'sync', _value?: boolean) => {
  if (event !== 'success') return
  if (pendingSaveMode.value === 'exit') {
    goBack()
    return
  }
  void loadGame()
}

const {
  bannerPreviewUrl,
  bannerSearchUrl,
  bannerSource,
  bannerUploadAction,
  bannerUploadData,
  backToBannerGameSearch,
  backToCoverGameSearch,
  backToLogoGameSearch,
  backToScreenshotGameSearch,
  backToSummarySearch,
  confirmBannerSelection,
  confirmCoverSelection,
  confirmLogoSelection,
  confirmScreenshotSelection,
  confirmSummaryImport,
  coverPreviewUrl,
  coverSearchUrl,
  coverSource,
  downloadSelectedSteamBanner,
  downloadSelectedSteamCover,
  downloadSelectedSteamCovers,
  downloadSelectedLogos,
  downloadSelectedScreenshots,
  draggedScreenshotKey,
  dragOverScreenshotKey,
  draggedCoverKey,
  dragOverCoverKey,
  draggedBannerKey,
  dragOverBannerKey,
  draggedLogoKey,
  dragOverLogoKey,
  draggedVideoKey,
  dragOverVideoKey,
  form,
  handleBannerSearchClear,
  handleBannerUploadError,
  handleBannerUploadSuccess,
  handleCoverError,
  handleCoverSearchClear,
  handleCoverUploadError,
  handleCoverUploadSuccess,
  handleCoverDragStart,
  handleCoverDragEnter,
  handleCoverDrop,
  handleCoverDragEnd,
  handleBannerDragStart,
  handleBannerDragEnter,
  handleBannerDrop,
  handleBannerDragEnd,
  handleLogoDragStart,
  handleLogoDragEnter,
  handleLogoDrop,
  handleLogoDragEnd,
  handleVideoDragStart,
  handleVideoDragEnter,
  handleVideoDrop,
  handleVideoDragEnd,
  handleLogoSearchClear,
  handleLogoPositionChange,
  handleLogoUploadError,
  handleLogoUploadSuccess,
  handleScreenshotDragEnd,
  handleScreenshotDragEnter,
  handleScreenshotDragStart,
  handleScreenshotDrop,
  handleScreenshotSearchClear,
  handleScreenshotUploadError,
  handleScreenshotUploadSuccess,
  handleSubmit,
  handleSummarySearchClear,
  handleVideoFileChange,
  invertSelectionBanners,
  invertSelectionCovers,
  invertSelectionLogos,
  invertSelectionScreenshots,
  isDownloadingBanner,
  isDownloadingCover,
  isDownloadingLogo,
  isDownloadingScreenshot,
  isDownloadingSteamBanners,
  isDownloadingSteamCovers,
  isDownloadingRemoteScreenshots,
  isDownloadingLogos,
  isSearchingLogo,
  isSearchingBanner,
  isSearchingCover,
  isSearchingScreenshots,
  isSearchingSteamSummary,
  isUploadingVideo,
  loadBannerFromUrl,
  loadCoverFromUrl,
  loadLogoFromUrl,
  loadScreenshotPreview,
  logoImages,
  logoPreviewUrl,
  logoSearchUrl,
  logoSearchResults,
  logoSearchQuery,
  logoUploadAction,
  logoUploadData,
  openLogoSelector,
  removeBanner,
  removeCover,
  removeLogo,
  removePreviewVideo,
  removeScreenshot,
  searchLogos,
  searchSteamForBanner,
  searchSteamForCover,
  searchScreenshots,
  searchSteamForSummary,
  selectAllBanners,
  selectAllCovers,
  selectAllLogos,
  selectAllScreenshots,
  selectSteamBannerGame,
  selectSteamCoverGame,
  selectLogoGame,
  selectScreenshotGame,
  selectSteamSummaryGame,
  selectedBanners,
  selectedCoverImage,
  selectedCovers,
  selectedLogos,
  selectedLogoGame,
  selectedSteamBannerGame,
  selectedSteamGame,
  selectedScreenshotGame,
  selectedRemoteScreenshots,
  selectedSteamSummaryGame,
  sgdbAvailable,
  showBannerSelector,
  showCoverSelector,
  showLogoSelector,
  showScreenshotSelector,
  showSummarySelector,
  steamBannerImages,
  steamBannerSearchQuery,
  bannerSearchResults,
  steamCoverImages,
  steamCoverSearchQuery,
  coverSearchResults,
  screenshotSearchQuery,
  screenshotSearchResults,
  screenshotSearchUrl,
  screenshotPreviewUrl,
  screenshotSource,
  screenshotUploadAction,
  screenshotUploadData,
  screenshotCandidatesData,
  steamSummaryPreview,
  steamSummarySearchQuery,
  steamSummarySearchResults,
  toggleBannerSelection,
  toggleCoverSelection,
  toggleLogoSelection,
  toggleScreenshot,
  uploadAction,
  uploadData,
  uploadHeaders,
  videoUploadFileName,
  videoUploadProgress,
} = useEditGameModal({
  props: modalProps,
  emit: modalEmit,
  uiStore,
  formRef,
  isSubmitting,
  activeTab,
})

const handleSave = () => {
  pendingSaveMode.value = 'stay'
  void handleSubmit()
}

const handleSaveAndExit = () => {
  pendingSaveMode.value = 'exit'
  void handleSubmit()
}

const syncAmbientBackground = () => {
  if (!game.value?.public_id) {
    uiStore.clearAmbientBackgroundSource(AMBIENT_BACKGROUND_OWNER)
    return
  }
  const pool = getAmbientBackgroundPoolFromGameDetail(game.value)
  if (!hasAmbientBackgroundPoolImages(pool)) {
    uiStore.clearAmbientBackgroundSource(AMBIENT_BACKGROUND_OWNER)
    return
  }
  uiStore.setAmbientBackgroundSource({
    owner: AMBIENT_BACKGROUND_OWNER,
    key: game.value.public_id,
    pool,
  })
}

const loadGame = async () => {
  if (!publicId.value) return
  loading.value = true
  loadError.value = false
  try {
    const detail = await gamesService.getAdminGameDetail(publicId.value)
    game.value = detail
    syncAmbientBackground()
    await nextTick()
  } catch {
    loadError.value = true
    uiStore.clearAmbientBackgroundSource(AMBIENT_BACKGROUND_OWNER)
    uiStore.addAlert('加载素材失败', 'error')
  } finally {
    loading.value = false
  }
}

onBeforeUnmount(() => {
  uiStore.clearAmbientBackgroundSource(AMBIENT_BACKGROUND_OWNER)
})

if (!authStore.isAdmin) {
  void router.replace({ name: 'game-detail', params: { publicId: publicId.value } })
} else {
  void loadGame()
}
</script>

<style scoped>
.game-media-page {
  display: flex;
  flex-direction: column;
  min-height: 100%;
  padding: 16px 24px 32px;
  gap: 16px;
  max-width: 1600px;
  margin: 0 auto;
}

.game-media-page__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
  position: sticky;
  top: 0;
  z-index: 20;
  margin: -16px -24px 0;
  padding: 12px 24px;
}

.game-media-page__heading {
  display: flex;
  align-items: center;
  flex: 1 1 auto;
  gap: 12px;
  min-width: 0;
}

.game-media-page__back {
  flex-shrink: 0;
}

.game-media-page__title {
  margin: 0;
  font-size: 18px;
  font-weight: 700;
  color: var(--color-text-1);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.game-media-page__actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  flex: 0 0 auto;
  margin-left: auto;
  flex-wrap: wrap;
  gap: 8px;
}

.game-media-page__body {
  min-height: 60vh;
}

.game-media-page__loading {
  display: flex;
  justify-content: center;
  padding: 80px 0;
}

.game-media-page__error {
  padding: 80px 0;
}

.game-media-page__error-text {
  color: var(--color-text-3);
}

.game-media-page__error-text h3 {
  margin: 0 0 8px;
  color: var(--color-text-1);
}

@media (max-width: 768px) {
  .game-media-page {
    padding: 12px 12px 24px;
  }

  .game-media-page__header {
    align-items: flex-start;
    margin: -12px -12px 0;
    padding: 12px;
  }

  .game-media-page__heading,
  .game-media-page__actions {
    width: 100%;
  }

  .game-media-page__actions {
    justify-content: flex-end;
  }
}
</style>
