<template>
  <a-modal
    v-model:visible="visible"
    class="edit-game-modal"
    title="编辑游戏"
    :width="modalWidth"
    :footer="false"
    :align-center="false"
    :body-style="modalBodyStyle"
    @cancel="handleCancel"
  >
    <a-form ref="formRef" :model="form" :rules="rules" layout="vertical" @submit="handleSubmit">
      <a-tabs v-model:active-key="activeTab" class="edit-modal-tabs">
        <a-tab-pane key="info" title="游戏信息">
          <a-row :gutter="12">
            <a-col :span="12">
              <a-form-item field="title">
                <template #label>
                  <div class="field-label-action">
                    <span>游戏名称</span>
                    <a-button
                      class="app-text-action-btn"
                      type="text"
                      size="mini"
                      html-type="button"
                      :disabled="!hasParsableWikiContent"
                      :loading="isPreparingWikiMetadataCandidates"
                      @click="importMetadataFromWiki"
                    >
                      从 Wiki 提取
                    </a-button>
                  </div>
                </template>
                <a-input v-model="form.title" placeholder="请输入游戏名称" />
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item label="别名/英文名">
                <a-input v-model="form.title_alt" placeholder="请输入别名" />
              </a-form-item>
            </a-col>
          </a-row>

          <a-row :gutter="12">
            <a-col :span="12">
              <a-form-item label="开发商">
                <a-select
                  :model-value="form.developer_ids"
                  placeholder="选择开发商（可多选）"
                  multiple
                  allow-clear
                  allow-search
                  :loading="isSearchingDevelopers || isCreatingDevelopers"
                  :filter-option="false"
                  @search="handleDeveloperSearch"
                  @update:model-value="handleDeveloperSelection"
                >
                  <a-option
                    v-for="d in filteredDeveloperOptions"
                    :key="d.id"
                    :value="d.id"
                    :label="d.name"
                  >
                    {{ d.name }}
                  </a-option>
                  <a-option
                    v-if="canCreateDeveloperOption"
                    :value="CREATE_DEVELOPER_OPTION_VALUE"
                    :label="`创建“${developerSearchQuery.trim()}”`"
                  >
                    创建“{{ developerSearchQuery.trim() }}”
                  </a-option>
                </a-select>
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item label="发行商">
                <a-select
                  :model-value="form.publisher_ids"
                  placeholder="选择发行商（可多选）"
                  multiple
                  allow-clear
                  allow-search
                  :loading="isSearchingPublishers || isCreatingPublishers"
                  :filter-option="false"
                  @search="handlePublisherSearch"
                  @update:model-value="handlePublisherSelection"
                >
                  <a-option
                    v-for="p in filteredPublisherOptions"
                    :key="p.id"
                    :value="p.id"
                    :label="p.name"
                  >
                    {{ p.name }}
                  </a-option>
                  <a-option
                    v-if="canCreatePublisherOption"
                    :value="CREATE_PUBLISHER_OPTION_VALUE"
                    :label="`创建“${publisherSearchQuery.trim()}”`"
                  >
                    创建“{{ publisherSearchQuery.trim() }}”
                  </a-option>
                </a-select>
              </a-form-item>
            </a-col>
          </a-row>

          <a-row :gutter="12">
            <a-col :span="8">
              <a-form-item label="系列">
                <a-select
                  :model-value="form.series_id"
                  v-model:input-value="seriesSearchQuery"
                  placeholder="选择系列"
                  allow-clear
                  allow-search
                  :loading="isSearchingSeries || isCreatingSeries"
                  :filter-option="false"
                  @keydown.enter.capture.stop.prevent="handleSeriesEnter"
                  @search="handleSeriesSearch"
                  @update:model-value="handleSeriesSelection"
                >
                  <a-option
                    v-for="s in filteredSeriesOptions"
                    :key="s.id"
                    :value="s.id"
                    :label="s.name"
                  >
                    {{ s.name }}
                  </a-option>
                  <a-option
                    v-if="canCreateSeriesOption"
                    :value="CREATE_SERIES_OPTION_VALUE"
                    :label="`创建“${seriesSearchQuery.trim()}”`"
                  >
                    创建“{{ seriesSearchQuery.trim() }}”
                  </a-option>
                </a-select>
              </a-form-item>
            </a-col>
            <a-col :span="8">
              <a-form-item label="发行日期">
                <a-date-picker
                  v-model="releaseDate"
                  :min-year="1950"
                  :max-year="2100"
                  placeholder="选择发行日期"
                  class="w-full"
                />
              </a-form-item>
            </a-col>
            <a-col :span="8">
              <a-form-item label="可见性">
                <a-radio-group v-model="form.visibility" type="button">
                  <a-radio value="public">公开</a-radio>
                  <a-radio value="private">私有</a-radio>
                </a-radio-group>
              </a-form-item>
            </a-col>
          </a-row>

          <a-form-item>
            <template #label>
              <div class="summary-label">
                <span>简介</span>
                <a-button
                  class="app-text-action-btn"
                  type="text"
                  size="mini"
                  html-type="button"
                  @click="showSummarySelector = true"
                >
                  从 Steam 导入
                </a-button>
              </div>
            </template>
            <a-textarea
              v-model="form.summary"
              placeholder="简短描述..."
              :auto-size="{ minRows: 3, maxRows: 6 }"
              show-word-limit
            />
          </a-form-item>

          <game-file-paths-section
            :file-paths="form.file_paths"
            @update-item="handleFilePathItemUpdate"
            @add="addFilePath"
            @remove="removeFilePath"
            @browse="openFileBrowser"
          />
        </a-tab-pane>

        <a-tab-pane key="media" title="游戏素材">
          <div class="media-tab-toolbar">
            <a-button
              class="app-text-action-btn"
              type="text"
              size="small"
              html-type="button"
              @click="openStandaloneMediaPage"
            >
              <template #icon><icon-launch /></template>
              在新页面管理素材
            </a-button>
          </div>
          <media-management-panel
            :title="form.title"
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
            :logo-source="logoSource"
            :logo-search-query="logoSearchQuery"
            :is-searching-logo="isSearchingLogo"
            :logo-search-results="logoSearchResults"
            :selected-logo-game="selectedLogoGame"
            :logo-images="logoImages"
            :selected-logo-image="selectedLogoImage"
            :is-downloading-logos="isDownloadingLogos"
            :logo-upload-action="logoUploadAction"
            :logo-upload-data="logoUploadData"
            :logo-search-url="logoSearchUrl"
            :logo-preview-url="logoPreviewUrl"
            :is-downloading-logo="isDownloadingLogo"
            :logo-banner-src="logoBannerSrc"
            :logo-path="editingLogo?.path ?? ''"
            :logo-position-x="editingLogo?.position_x ?? null"
            :logo-position-y="editingLogo?.position_y ?? null"
            :logo-width-pct="editingLogo?.width_pct ?? null"
            :logo-visible="form.logo_visible ?? true"
            :logo-initial-tab="logoInitialTab"
            :show-banner-crop-modal="showBannerCropModal"
            :banner-crop-src="bannerCropSrc"
            @open-cover-selector="showCoverSelector = true"
            @remove-cover="removeCover"
            @set-primary-cover="setPrimaryCover"
            @cover-drag-start="handleCoverDragStart"
            @cover-drag-enter="handleCoverDragEnter"
            @cover-drop="handleCoverDrop"
            @cover-drag-end="handleCoverDragEnd"
            @open-banner-selector="showBannerSelector = true"
            @remove-banner="removeBanner"
            @set-primary-banner="handleSetPrimaryBanner"
            @banner-drag-start="handleBannerDragStart"
            @banner-drag-enter="handleBannerDragEnter"
            @banner-drop="handleBannerDrop"
            @banner-drag-end="handleBannerDragEnd"
            @logo-drag-start="handleLogoDragStart"
            @logo-drag-enter="handleLogoDragEnter"
            @logo-drop="handleLogoDrop"
            @logo-drag-end="handleLogoDragEnd"
            @video-file-change="handleVideoFileChange"
            @reorder-video="reorderEditableVideos($event.key, $event.direction)"
            @remove-video="removePreviewVideo"
            @open-screenshot-selector="showScreenshotSelector = true"
            @remove-screenshot="removeScreenshot"
            @screenshot-drag-start="handleScreenshotDragStart"
            @screenshot-drag-enter="handleScreenshotDragEnter"
            @screenshot-drop="handleScreenshotDrop"
            @screenshot-drag-end="handleScreenshotDragEnd"
            @open-logo-selector="openLogoSelector"
            @open-logo-position-editor="openLogoPositionEditor"
            @remove-logo="removeLogo"
            @set-primary-logo="setPrimaryLogo"
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
            @source-change-logo="logoSource = $event"
            @update:logo-search-query="logoSearchQuery = $event"
            @search-logo="searchLogos"
            @clear-logo="handleLogoSearchClear"
            @select-logo-game="selectLogoGame"
            @back-logo-game-search="backToLogoGameSearch"
            @update:selected-logo-image="selectedLogoImage = $event"
            @download-selected-logo="downloadSelectedLogo"
            @logo-upload-success="handleLogoUploadSuccess"
            @logo-upload-error="handleLogoUploadError"
            @update:logo-search-url="logoSearchUrl = $event"
            @load-logo-from-url="loadLogoFromUrl"
            @confirm-logo-selection="confirmLogoSelection"
            @confirm-logo-position="handleLogoPositionConfirm"
            @banner-crop-confirm="handleBannerCropConfirm"
            @banner-crop-cancel="showBannerCropModal = false"
          />
        </a-tab-pane>
      </a-tabs>

      <a-form-item>
        <a-space class="edit-modal-footer">
          <a-button class="app-text-action-btn" type="text" html-type="button" @click="handleCancel">取消</a-button>
          <a-button type="primary" html-type="submit" :loading="isSubmitting">
            保存
          </a-button>
        </a-space>
      </a-form-item>
    </a-form>

    <!-- File Browser Modal -->
    <file-browser-modal
      v-model:visible="showFileBrowser"
      :initial-path="initialPath"
      @select="handleFileSelect"
    />

    <edit-game-wiki-metadata-picker-modal
      :visible="wikiMetadataPickerVisible"
      :candidates="wikiMetadataCandidates"
      :is-applying-wiki-metadata="isApplyingWikiMetadata"
      @update:visible="wikiMetadataPickerVisible = $event"
      @selection-change="handleWikiMetadataCandidateSelectionChange($event.key, $event.selected)"
      @apply="applySelectedWikiMetadata"
    />

  </a-modal>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { IconLaunch } from '@arco-design/web-vue/es/icon'
import { useUiStore } from '@/stores/ui'
import type { AdminGameDetail } from '@/services/types'
import FileBrowserModal from '@/components/FileBrowserModal.vue'
import GameFilePathsSection from '@/components/edit-game/GameFilePathsSection.vue'
import MediaManagementPanel from '@/components/edit-game/MediaManagementPanel.vue'
import EditGameWikiMetadataPickerModal from '@/components/edit-game/EditGameWikiMetadataPickerModal.vue'
import { useEditGameModal } from '@/composables/useEditGameModal'
import { uploadAsset } from '@/services/assets'

interface Props {
  visible: boolean
  game: AdminGameDetail | null
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'success': []
  'sync': []
}>()

const uiStore = useUiStore()
const router = useRouter()
const formRef = ref()
const isSubmitting = ref(false)
const activeTab = ref('info')
const modalBodyStyle = computed(() => (
  activeTab.value === 'media'
    ? { maxHeight: 'calc(85vh - 120px)', overflowY: 'auto' as const }
    : undefined
))

const openStandaloneMediaPage = () => {
  if (!props.game?.public_id) return
  visible.value = false
  void router.push({ name: 'game-media', params: { publicId: props.game.public_id } })
}

const {
  CREATE_DEVELOPER_OPTION_VALUE,
  CREATE_PUBLISHER_OPTION_VALUE,
  CREATE_SERIES_OPTION_VALUE,
  addFilePath,
  applySelectedWikiMetadata,
  backToBannerGameSearch,
  backToCoverGameSearch,
  backToScreenshotGameSearch,
  backToSummarySearch,
  backToLogoGameSearch,
  bannerPreviewUrl,
  bannerSearchUrl,
  bannerUploadAction,
  bannerUploadData,
  canCreateDeveloperOption,
  canCreatePublisherOption,
  canCreateSeriesOption,
  confirmBannerSelection,
  confirmCoverSelection,
  confirmLogoSelection,
  confirmScreenshotSelection,
  confirmSummaryImport,
  coverPreviewUrl,
  coverSearchUrl,
  developerSearchQuery,
  downloadSelectedSteamBanner,
  downloadSelectedSteamCover,
  downloadSelectedSteamCovers,
  downloadSelectedScreenshots,
  downloadSelectedLogo,
  draggedScreenshotKey,
  dragOverScreenshotKey,
  draggedCoverKey,
  dragOverCoverKey,
  draggedBannerKey,
  dragOverBannerKey,
  draggedLogoKey,
  dragOverLogoKey,
  filteredDeveloperOptions,
  filteredPublisherOptions,
  filteredSeriesOptions,
  form,
  handleBannerSearchClear,
  handleBannerUploadError,
  handleBannerUploadSuccess,
  handleLogoSearchClear,
  handleLogoPositionConfirm,
  handleLogoUploadError,
  handleLogoUploadSuccess,
  handleCancel,
  hasParsableWikiContent,
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
  handleDeveloperSearch,
  handleDeveloperSelection,
  handleFilePathItemUpdate,
  handleFileSelect,
  handlePublisherSearch,
  handlePublisherSelection,
  handleScreenshotDragEnd,
  handleScreenshotDragEnter,
  handleScreenshotDragStart,
  handleScreenshotDrop,
  handleScreenshotSearchClear,
  handleScreenshotUploadError,
  handleScreenshotUploadSuccess,
  handleSeriesEnter,
  handleSeriesSearch,
  handleSeriesSelection,
  handleSubmit,
  handleSummarySearchClear,
  handleVideoFileChange,
  handleWikiMetadataCandidateSelectionChange,
  importMetadataFromWiki,
  initialPath,
  invertSelectionBanners,
  invertSelectionCovers,
  invertSelectionScreenshots,
  isApplyingWikiMetadata,
  isDownloadingBanner,
  isDownloadingCover,
  isDownloadingLogo,
  isDownloadingScreenshot,
  isDownloadingSteamCovers,
  isDownloadingLogos,
  isDownloadingRemoteScreenshots,
  isCreatingDevelopers,
  isCreatingPublishers,
  isCreatingSeries,
  isPreparingWikiMetadataCandidates,
  isSearchingSeries,
  isSearchingDevelopers,
  isSearchingPublishers,
  isSearchingBanner,
  isSearchingCover,
  isSearchingLogo,
  isSearchingScreenshots,
  isSearchingSteamSummary,
  isUploadingVideo,
  loadBannerFromUrl,
  loadCoverFromUrl,
  loadLogoFromUrl,
  loadScreenshotPreview,
  logoPreviewUrl,
  logoSearchResults,
  logoSearchUrl,
  logoUploadAction,
  logoUploadData,
  logoBannerSrc,
  editingLogo,
  logoInitialTab,
  openLogoSelector,
  openLogoPositionEditor,
  modalWidth,
  openFileBrowser,
  setPrimaryLogo,
  releaseDate,
  removeBanner,
  removeCover,
  removeFilePath,
  removeLogo,
  removePreviewVideo,
  removeScreenshot,
  reorderEditableVideos,
  setPrimaryCover,
  rules,
  screenshotPreviewUrl,
  screenshotSearchUrl,
  screenshotUploadAction,
  screenshotUploadData,
  searchSteamForBanner,
  searchSteamForCover,
  searchLogos,
  searchScreenshots,
  searchSteamForSummary,
  selectAllBanners,
  selectAllCovers,
  selectAllScreenshots,
  selectSteamBannerGame,
  selectSteamCoverGame,
  selectLogoGame,
  selectScreenshotGame,
  selectSteamSummaryGame,
  selectedBanners,
  isDownloadingSteamBanners,
  toggleBannerSelection,
  setPrimaryBanner,
  selectedCoverImage,
  selectedCovers,
  selectedLogoImage,
  selectedSteamBannerGame,
  selectedSteamGame,
  selectedLogoGame,
  selectedScreenshotGame,
  selectedRemoteScreenshots,
  selectedSteamSummaryGame,
  publisherSearchQuery,
  showBannerSelector,
  showCoverSelector,
  showFileBrowser,
  showLogoSelector,
  showScreenshotSelector,
  showSummarySelector,
  seriesSearchQuery,
  steamBannerImages,
  steamBannerSearchQuery,
  bannerSearchResults,
  steamCoverImages,
  steamCoverSearchQuery,
  coverSearchResults,
  logoImages,
  logoSearchQuery,
  screenshotSearchQuery,
  screenshotSearchResults,
  screenshotCandidatesData,
  steamSummaryPreview,
  steamSummarySearchQuery,
  steamSummarySearchResults,
  toggleCoverSelection,
  toggleScreenshot,
  uploadAction,
  uploadData,
  uploadHeaders,
  videoUploadFileName,
  videoUploadProgress,
  visible,
  wikiMetadataCandidates,
  wikiMetadataPickerVisible,
  coverSource,
  bannerSource,
  screenshotSource,
  logoSource,
  sgdbAvailable,
} = useEditGameModal({
  props,
  emit,
  uiStore,
  formRef,
  isSubmitting,
  activeTab,
})

const showBannerCropModal = ref(false)
const bannerCropSrc = ref('')

const handleSetPrimaryBanner = (index: number) => {
  const banner = form.value.banners[index]
  if (!banner) return

  const img = new Image()
  img.src = banner.path
  img.onload = () => {
    if (img.naturalWidth / img.naturalHeight > 1.6) {
      bannerCropSrc.value = banner.path
      showBannerCropModal.value = true
    } else {
      setPrimaryBanner(index)
    }
  }
  img.onerror = () => {
    setPrimaryBanner(index)
  }
}

const handleBannerCropConfirm = async (blob: Blob) => {
  const gameId = props.game?.id
  if (!gameId) return

  showBannerCropModal.value = false
  try {
    const file = new File([blob], 'banner-crop.png', { type: 'image/png' })
    const result = await uploadAsset('banner', gameId, file)
    if (result) {
      form.value.banners.unshift({ asset_uid: result.asset_uid, path: result.path })
    }
  } catch {
    uiStore.addAlert('横幅裁剪上传失败', 'error')
  }
}
</script>

<style scoped src="./edit-game/EditGameModal.css"></style>

<style>
.edit-game-modal .arco-modal-body {
  padding: 2px 8px;
}
</style>
