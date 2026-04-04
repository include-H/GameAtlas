<template>
  <!--
    重构说明（2026-03）：
    1) 已将提交流程下沉到 useEditGameWorkflow（元数据解析、文件路径持久化、资产删除队列、排序提交）。
    2) 已将资产上传/删除与视频上传状态下沉到 useEditGameAssets。
    3) 已将表单初始化与回填下沉到 useEditGameFormBootstrap。
    4) 已将截图拖拽与视频排序交互下沉到 useEditGameMediaState。
    5) 已将“资产导入弹窗 / 预告片弹窗 / Wiki 标签弹窗”拆为独立子组件。
    当前 EditGameModal 仅承担外观层职责：UI 编排、状态注入、事件绑定。
  -->
  <a-modal
    v-model:visible="visible"
    class="edit-game-modal"
    title="编辑游戏信息"
    :width="modalWidth"
    :footer="false"
    :align-center="false"
    @cancel="handleCancel"
  >
    <a-form ref="formRef" :model="form" :rules="rules" layout="vertical" @submit="handleSubmit">
      <a-row :gutter="16">
        <a-col :xs="24" :sm="12">
          <a-form-item field="title">
            <template #label>
              <div class="field-label-action">
                <span>游戏名称</span>
                <a-button
                  class="app-text-action-btn"
                  type="text"
                  size="mini"
                  html-type="button"
                  :disabled="!props.game?.wiki_content"
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
        <a-col :xs="24" :sm="12">
          <a-form-item label="别名/英文名">
            <a-input v-model="form.title_alt" placeholder="请输入别名" />
          </a-form-item>
        </a-col>
      </a-row>

      <a-row :gutter="16">
        <a-col :xs="24" :sm="12">
          <a-form-item label="开发商">
            <a-select
              v-model="form.developer_ids"
              placeholder="选择开发商（可多选）"
              multiple
              allow-clear
              allow-search
              :loading="isSearchingDevelopers"
              :remote-search="true"
              :on-search="handleDeveloperSearch"
            >
              <a-option
                v-for="d in filteredDeveloperOptions"
                :key="d.id"
                :value="d.id"
                :label="d.name"
              >
                {{ d.name }}
              </a-option>
            </a-select>
          </a-form-item>
        </a-col>
        <a-col :xs="24" :sm="12">
          <a-form-item label="发行商">
            <a-select
              v-model="form.publisher_ids"
              placeholder="选择发行商（可多选）"
              multiple
              allow-clear
              allow-search
              :loading="isSearchingPublishers"
              :remote-search="true"
              :on-search="handlePublisherSearch"
            >
              <a-option
                v-for="p in filteredPublisherOptions"
                :key="p.id"
                :value="p.id"
                :label="p.name"
              >
                {{ p.name }}
              </a-option>
            </a-select>
          </a-form-item>
        </a-col>
      </a-row>

      <a-row :gutter="16">
        <a-col :xs="24" :sm="8">
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
        <a-col :xs="24" :sm="8">
          <a-form-item label="游戏引擎">
            <a-input v-model="form.engine" placeholder="如：Unity, Unreal" />
          </a-form-item>
        </a-col>
        <a-col :xs="24" :sm="8">
          <a-form-item label="可见性">
            <a-radio-group v-model="form.visibility" type="button">
              <a-radio value="public">公开</a-radio>
              <a-radio value="private">私有</a-radio>
            </a-radio-group>
          </a-form-item>
        </a-col>
      </a-row>

      <a-row :gutter="16">
        <a-col :xs="24" :sm="12">
          <a-form-item label="平台">
            <a-select
              v-model="form.platform_ids"
              placeholder="选择或输入平台（可多选）"
              multiple
              allow-clear
              allow-search
            >
              <a-option
                v-for="p in platformOptions"
                :key="p.id"
                :value="p.id"
                :label="p.name"
              >
                {{ p.name }}
              </a-option>
            </a-select>
          </a-form-item>
        </a-col>
        <a-col :xs="24" :sm="12">
          <a-form-item label="系列">
            <a-select
              v-model="form.series_id"
              placeholder="选择系列"
              allow-clear
              allow-search
              :loading="isSearchingSeries"
              :remote-search="true"
              :on-search="handleSeriesSearch"
            >
              <a-option
                v-for="s in filteredSeriesOptions"
                :key="s.id"
                :value="s.id"
                :label="s.name"
              >
                {{ s.name }}
              </a-option>
            </a-select>
          </a-form-item>
        </a-col>
      </a-row>

      <game-tag-section
        :tag-groups="tagGroups"
        :tag-field-values-by-group="tagFieldValuesByGroup"
        :pending-tag-options-by-group="pendingTagOptionsByGroup"
        :tag-options-by-group="tagOptionsByGroup"
        :wiki-content-exists="!!props.game?.wiki_content"
        :is-preparing-wiki-tag-candidates="isPreparingWikiTagCandidates"
        @parse-wiki-tags="handleParseWikiTags"
        @tag-selection-change="handleTagSectionSelectionChange"
      />

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
          :auto-size="{ minRows: 2, maxRows: 4 }"
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


      <game-media-section
        :title="form.title"
        :cover-image="form.cover_image"
        :banner-image="form.banner_image"
        :primary-preview-video="primaryPreviewVideo"
        :preview-video-sources="previewVideoSources"
        :screenshots="form.screenshots"
        :dragged-screenshot-key="draggedScreenshotKey"
        :drag-over-screenshot-key="dragOverScreenshotKey"
        @open-cover-selector="showCoverSelector = true"
        @remove-cover="removeCover"
        @open-banner-selector="showBannerSelector = true"
        @remove-banner="removeBanner"
        @open-video-selector="openVideoSelector"
        @open-screenshot-selector="showScreenshotSelector = true"
        @remove-screenshot="removeScreenshot"
        @screenshot-drag-start="handleScreenshotDragStart"
        @screenshot-drag-enter="handleScreenshotDragEnter"
        @screenshot-drop="handleScreenshotDrop"
        @screenshot-drag-end="handleScreenshotDragEnd"
      />

		      <a-form-item>
	        <a-space style="justify-content: flex-end; width: 100%">
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

    <edit-game-asset-import-modals
      :show-summary-selector="showSummarySelector"
      :steam-summary-search-query="steamSummarySearchQuery"
      :is-searching-steam-summary="isSearchingSteamSummary"
      :steam-summary-search-results="steamSummarySearchResults"
      :selected-steam-summary-game="selectedSteamSummaryGame"
      :steam-summary-preview="steamSummaryPreview"
      :show-cover-selector="showCoverSelector"
      :steam-cover-search-query="steamCoverSearchQuery"
      :is-searching-steam-cover="isSearchingSteamCover"
      :steam-cover-search-results="steamCoverSearchResults"
      :selected-steam-game="selectedSteamGame"
      :steam-cover-images="steamCoverImages"
      :selected-cover-image="selectedCoverImage"
      :upload-action="uploadAction"
      :upload-data="uploadData"
      :upload-headers="uploadHeaders"
      :cover-search-url="coverSearchUrl"
      :cover-preview-url="coverPreviewUrl"
      :is-downloading-cover="isDownloadingCover"
      :show-banner-selector="showBannerSelector"
      :steam-banner-search-query="steamBannerSearchQuery"
      :is-searching-steam-banner="isSearchingSteamBanner"
      :steam-banner-search-results="steamBannerSearchResults"
      :selected-steam-banner-game="selectedSteamBannerGame"
      :steam-banner-images="steamBannerImages"
      :selected-banner-image="selectedBannerImage"
      :banner-upload-action="bannerUploadAction"
      :banner-upload-data="bannerUploadData"
      :banner-search-url="bannerSearchUrl"
      :banner-preview-url="bannerPreviewUrl"
      :is-downloading-banner="isDownloadingBanner"
      :show-screenshot-selector="showScreenshotSelector"
      :steam-screenshot-search-query="steamScreenshotSearchQuery"
      :is-searching-steam-screenshots="isSearchingSteamScreenshots"
      :steam-screenshot-search-results="steamScreenshotSearchResults"
      :selected-steam-screenshot-game="selectedSteamScreenshotGame"
      :steam-screenshots-data="steamScreenshotsData"
      :selected-steam-screenshots="selectedSteamScreenshots"
      :is-downloading-steam-screenshots="isDownloadingSteamScreenshots"
      :screenshot-upload-action="screenshotUploadAction"
      :screenshot-upload-data="screenshotUploadData"
      :screenshot-search-url="screenshotSearchUrl"
      :screenshot-preview-url="screenshotPreviewUrl"
      :is-downloading-screenshot="isDownloadingScreenshot"
      @update:show-summary-selector="showSummarySelector = $event"
      @update:steam-summary-search-query="steamSummarySearchQuery = $event"
      @search-summary="searchSteamForSummary"
      @clear-summary="handleSummarySearchClear"
      @select-summary="selectSteamSummaryGame"
      @back-summary="backToSummarySearch"
      @confirm-summary-import="confirmSummaryImport"
      @update:show-cover-selector="showCoverSelector = $event"
      @update:steam-cover-search-query="steamCoverSearchQuery = $event"
      @search-cover="searchSteamForCover"
      @clear-cover="handleCoverSearchClear"
      @select-cover-game="selectSteamCoverGame"
      @back-cover-game-search="backToCoverGameSearch"
      @update:selected-cover-image="selectedCoverImage = $event"
      @download-selected-steam-cover="downloadSelectedSteamCover"
      @cover-upload-success="handleCoverUploadSuccess"
      @cover-upload-error="handleCoverUploadError"
      @update:cover-search-url="coverSearchUrl = $event"
      @load-cover-from-url="loadCoverFromUrl"
      @confirm-cover-selection="confirmCoverSelection"
      @cover-image-error="handleCoverError"
      @update:show-banner-selector="showBannerSelector = $event"
      @update:steam-banner-search-query="steamBannerSearchQuery = $event"
      @search-banner="searchSteamForBanner"
      @clear-banner="handleBannerSearchClear"
      @select-banner-game="selectSteamBannerGame"
      @back-banner-game-search="backToBannerGameSearch"
      @update:selected-banner-image="selectedBannerImage = $event"
      @download-selected-steam-banner="downloadSelectedSteamBanner"
      @banner-upload-success="handleBannerUploadSuccess"
      @banner-upload-error="handleBannerUploadError"
      @update:banner-search-url="bannerSearchUrl = $event"
      @load-banner-from-url="loadBannerFromUrl"
      @confirm-banner-selection="confirmBannerSelection"
      @update:show-screenshot-selector="showScreenshotSelector = $event"
      @update:steam-screenshot-search-query="steamScreenshotSearchQuery = $event"
      @search-screenshot="searchSteamForScreenshots"
      @clear-screenshot="handleScreenshotSearchClear"
      @select-screenshot-game="selectSteamScreenshotGame"
      @back-screenshot-game-search="backToScreenshotGameSearch"
      @toggle-steam-screenshot="toggleSteamScreenshot"
      @download-selected-steam-screenshots="downloadSelectedSteamScreenshots"
      @screenshot-upload-success="handleScreenshotUploadSuccess"
      @screenshot-upload-error="handleScreenshotUploadError"
      @update:screenshot-search-url="screenshotSearchUrl = $event"
      @load-screenshot-preview="loadScreenshotPreview"
      @confirm-screenshot-selection="confirmScreenshotSelection"
    />

    <edit-game-video-modal
      :visible="showVideoSelector"
      :is-uploading-video="isUploadingVideo"
      :video-upload-progress="videoUploadProgress"
      :video-upload-file-name="videoUploadFileName"
      :preview-videos="form.preview_videos"
      :banner-image="form.banner_image"
      :cover-image="form.cover_image"
      @update:visible="showVideoSelector = $event"
      @video-file-change="handleVideoFileChange"
      @reorder-video="reorderEditableVideos($event.key, $event.direction)"
      @remove-video="removePreviewVideo"
    />

    <edit-game-wiki-tag-picker-modal
      :visible="wikiTagPickerVisible"
      :candidates="wikiTagCandidates"
      :is-applying-wiki-tags="isApplyingWikiTags"
      @update:visible="wikiTagPickerVisible = $event"
      @group-change="handleWikiTagCandidateGroupChange($event.key, $event.value)"
      @apply="applySelectedWikiTags"
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
import { ref } from 'vue'
import { useUiStore } from '@/stores/ui'
import type { GameDetail } from '@/services/types'
import FileBrowserModal from '@/components/FileBrowserModal.vue'
import GameTagSection from '@/components/edit-game/GameTagSection.vue'
import GameFilePathsSection from '@/components/edit-game/GameFilePathsSection.vue'
import GameMediaSection from '@/components/edit-game/GameMediaSection.vue'
import EditGameAssetImportModals from '@/components/edit-game/EditGameAssetImportModals.vue'
import EditGameVideoModal from '@/components/edit-game/EditGameVideoModal.vue'
import EditGameWikiTagPickerModal from '@/components/edit-game/EditGameWikiTagPickerModal.vue'
import EditGameWikiMetadataPickerModal from '@/components/edit-game/EditGameWikiMetadataPickerModal.vue'
import { useEditGameModal } from '@/composables/useEditGameModal'

interface Props {
  visible: boolean
  game: GameDetail | null
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'success': []
}>()

const uiStore = useUiStore()
const formRef = ref()
const isSubmitting = ref(false)

const {
  addFilePath,
  applySelectedWikiMetadata,
  applySelectedWikiTags,
  backToBannerGameSearch,
  backToCoverGameSearch,
  backToScreenshotGameSearch,
  backToSummarySearch,
  bannerPreviewUrl,
  bannerSearchUrl,
  bannerUploadAction,
  bannerUploadData,
  confirmBannerSelection,
  confirmCoverSelection,
  confirmScreenshotSelection,
  confirmSummaryImport,
  coverPreviewUrl,
  coverSearchUrl,
  downloadSelectedSteamBanner,
  downloadSelectedSteamCover,
  downloadSelectedSteamScreenshots,
  draggedScreenshotKey,
  dragOverScreenshotKey,
  filteredDeveloperOptions,
  filteredPublisherOptions,
  filteredSeriesOptions,
  form,
  handleBannerSearchClear,
  handleBannerUploadError,
  handleBannerUploadSuccess,
  handleCancel,
  handleCoverError,
  handleCoverSearchClear,
  handleCoverUploadError,
  handleCoverUploadSuccess,
  handleDeveloperSearch,
  handleFilePathItemUpdate,
  handleFileSelect,
  handleParseWikiTags,
  handlePublisherSearch,
  handleScreenshotDragEnd,
  handleScreenshotDragEnter,
  handleScreenshotDragStart,
  handleScreenshotDrop,
  handleScreenshotSearchClear,
  handleScreenshotUploadError,
  handleScreenshotUploadSuccess,
  handleSeriesSearch,
  handleSubmit,
  handleSummarySearchClear,
  handleTagSectionSelectionChange,
  handleVideoFileChange,
  handleWikiMetadataCandidateSelectionChange,
  handleWikiTagCandidateGroupChange,
  importMetadataFromWiki,
  initialPath,
  isApplyingWikiMetadata,
  isApplyingWikiTags,
  isDownloadingBanner,
  isDownloadingCover,
  isDownloadingScreenshot,
  isDownloadingSteamScreenshots,
  isPreparingWikiMetadataCandidates,
  isPreparingWikiTagCandidates,
  isSearchingDevelopers,
  isSearchingPublishers,
  isSearchingSeries,
  isSearchingSteamBanner,
  isSearchingSteamCover,
  isSearchingSteamScreenshots,
  isSearchingSteamSummary,
  isUploadingVideo,
  loadBannerFromUrl,
  loadCoverFromUrl,
  loadScreenshotPreview,
  modalWidth,
  openFileBrowser,
  openVideoSelector,
  pendingTagOptionsByGroup,
  platformOptions,
  previewVideoSources,
  primaryPreviewVideo,
  releaseDate,
  removeBanner,
  removeCover,
  removeFilePath,
  removePreviewVideo,
  removeScreenshot,
  reorderEditableVideos,
  rules,
  screenshotPreviewUrl,
  screenshotSearchUrl,
  screenshotUploadAction,
  screenshotUploadData,
  searchSteamForBanner,
  searchSteamForCover,
  searchSteamForScreenshots,
  searchSteamForSummary,
  selectSteamBannerGame,
  selectSteamCoverGame,
  selectSteamScreenshotGame,
  selectSteamSummaryGame,
  selectedBannerImage,
  selectedCoverImage,
  selectedSteamBannerGame,
  selectedSteamGame,
  selectedSteamScreenshotGame,
  selectedSteamScreenshots,
  selectedSteamSummaryGame,
  showBannerSelector,
  showCoverSelector,
  showFileBrowser,
  showScreenshotSelector,
  showSummarySelector,
  showVideoSelector,
  steamBannerImages,
  steamBannerSearchQuery,
  steamBannerSearchResults,
  steamCoverImages,
  steamCoverSearchQuery,
  steamCoverSearchResults,
  steamScreenshotSearchQuery,
  steamScreenshotSearchResults,
  steamScreenshotsData,
  steamSummaryPreview,
  steamSummarySearchQuery,
  steamSummarySearchResults,
  tagGroups,
  tagOptionsByGroup,
  tagFieldValuesByGroup,
  toggleSteamScreenshot,
  uploadAction,
  uploadData,
  uploadHeaders,
  videoUploadFileName,
  videoUploadProgress,
  visible,
  wikiMetadataCandidates,
  wikiMetadataPickerVisible,
  wikiTagCandidates,
  wikiTagPickerVisible,
} = useEditGameModal({
  props,
  emit,
  uiStore,
  formRef,
  isSubmitting,
})
</script>

<style scoped src="./edit-game/EditGameModal.css"></style>
