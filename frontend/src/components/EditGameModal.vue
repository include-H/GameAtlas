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
    title="编辑游戏信息"
    :width="800"
    :footer="false"
    :align-center="false"
    @cancel="handleCancel"
  >
    <a-form ref="formRef" :model="form" :rules="rules" layout="vertical" @submit="handleSubmit">
      <a-row :gutter="16">
        <a-col :span="12">
          <a-form-item field="title" label="游戏名称">
            <a-input v-model="form.title" placeholder="请输入游戏名称" />
          </a-form-item>
        </a-col>
        <a-col :span="12">
          <a-form-item label="别名/英文名">
            <a-input v-model="form.title_alt" placeholder="请输入别名" />
          </a-form-item>
        </a-col>
      </a-row>

      <a-row :gutter="16">
        <a-col :span="12">
          <a-form-item label="开发商">
            <a-select
              v-model="form.developers"
              placeholder="选择开发商（可多选）"
              multiple
              allow-clear
              allow-search
              allow-create
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
        <a-col :span="12">
          <a-form-item label="发行商">
            <a-select
              v-model="form.publishers"
              placeholder="选择发行商（可多选）"
              multiple
              allow-clear
              allow-search
              allow-create
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
        <a-col :span="8">
          <a-form-item label="发行日期">
            <a-date-picker
              v-model="releaseDate"
              :min-year="1950"
              :max-year="2100"
              placeholder="选择发行日期"
              class="w-full"
              @change="handleDateChange"
            />
          </a-form-item>
        </a-col>
        <a-col :span="8">
          <a-form-item label="游戏引擎">
            <a-input v-model="form.engine" placeholder="如：Unity, Unreal" />
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

      <a-row :gutter="16">
        <a-col :span="12">
          <a-form-item label="平台">
            <a-select
              v-model="form.platform"
              placeholder="选择或输入平台（可多选）"
              multiple
              allow-clear
              allow-search
              allow-create
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
        <a-col :span="12">
          <a-form-item label="系列">
            <a-select
              v-model="form.series"
              placeholder="选择系列"
              allow-clear
              allow-search
              allow-create
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
        :tag-selections-by-group="tagSelectionsByGroup"
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
          <a-button type="text" html-type="button" @click="handleCancel">取消</a-button>
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
      :primary-preview-video-uid="form.primary_preview_video_uid"
      :banner-image="form.banner_image"
      :cover-image="form.cover_image"
      @update:visible="showVideoSelector = $event"
      @video-file-change="handleVideoFileChange"
      @set-primary-video="setPrimaryPreviewVideo"
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
	  </a-modal>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useUiStore } from '@/stores/ui'
import { uploadAsset, type UploadedAssetResult } from '@/services/assets'
import { directoryService } from '@/services/directory.service'
import type { Game } from '@/services/types'
import FileBrowserModal from '@/components/FileBrowserModal.vue'
import GameTagSection, { type GameTagSelectionValue } from '@/components/edit-game/GameTagSection.vue'
import GameFilePathsSection from '@/components/edit-game/GameFilePathsSection.vue'
import GameMediaSection from '@/components/edit-game/GameMediaSection.vue'
import EditGameAssetImportModals from '@/components/edit-game/EditGameAssetImportModals.vue'
import EditGameVideoModal from '@/components/edit-game/EditGameVideoModal.vue'
import EditGameWikiTagPickerModal from '@/components/edit-game/EditGameWikiTagPickerModal.vue'
import { proxySteamAssetUrl } from '@/services/steam.service'
import { seriesService } from '@/services/series.service'
import { resolveAssetCandidates } from '@/utils/asset-url'
import { useGameFilePaths, type FilePathItem } from '@/composables/useGameFilePaths'
import { useTagSelection } from '@/composables/useTagSelection'
import { useSteamImport } from '@/composables/useSteamImport'
import { useEditGameWorkflow } from '@/composables/useEditGameWorkflow'
import { useEditGameAssets } from '@/composables/useEditGameAssets'
import { useEditGameFormBootstrap } from '@/composables/useEditGameFormBootstrap'
import { useEditGameMediaState } from '@/composables/useEditGameMediaState'
import {
  searchCreatableOptions,
  sortCreatableOptionsByName,
} from '@/utils/creatable-select'
import type {
  Developer,
  Platform,
  Publisher,
  ScreenshotItem,
  Series,
  Tag,
  TagGroup,
  VideoAssetItem,
} from '@/services/types'

interface Props {
  visible: boolean
  game: Game | null
}

interface EditableScreenshot {
  id?: number
  asset_uid?: string
  path: string
  sort_order?: number
  client_key: string
}

interface EditableVideo {
  id?: number
  asset_uid?: string
  path: string
  sort_order?: number
}

interface GameForm {
  title: string
  title_alt: string
  visibility: 'public' | 'private'
  developers: Array<string | number>
  publishers: Array<string | number>
  release_date: string | undefined
  engine: string
  platform: (string | number)[]
  series: string | number | null
  tag_ids: Array<string | number>
  summary: string
  cover_image: string
  banner_image: string
  preview_videos: EditableVideo[]
  primary_preview_video_uid: string
  screenshots: EditableScreenshot[]
  file_paths: FilePathItem[]
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'success': []
}>()

const uiStore = useUiStore()
const formRef = ref()
const isSubmitting = ref(false)

// Form validation rules
const rules = {
  title: [{ required: true, message: '请输入游戏名称' }]
}

const seriesOptions = ref<Series[]>([])
const platformOptions = ref<Platform[]>([])
const tagGroups = ref<TagGroup[]>([])
const tagOptions = ref<Tag[]>([])
const developerOptions = ref<Developer[]>([])
const publisherOptions = ref<Publisher[]>([])
const isSearchingSeries = ref(false)
const isSearchingDevelopers = ref(false)
const isSearchingPublishers = ref(false)

const primaryPreviewVideo = computed(() => {
  if (form.value.preview_videos.length === 0) return null
  const selected = form.value.preview_videos.find((item) => item.asset_uid === form.value.primary_preview_video_uid)
  return selected || form.value.preview_videos[0]
})
const previewVideoSources = computed(() => resolveAssetCandidates(primaryPreviewVideo.value?.path || ''))

const filteredSeriesOptions = computed(() => {
  return [...seriesOptions.value].sort((a, b) => a.name.localeCompare(b.name, 'zh-Hans-CN'))
})

const handleSeriesSearch = async (query: string) => {
  if (!query) return
  isSearchingSeries.value = true
  try {
    const results = await seriesService.searchSeries(query)
    // Add results but keep current selection if it exists
    const current = seriesOptions.value.find(s => s.id === form.value.series)
    seriesOptions.value = results
    if (current && !results.find(s => s.id === current.id)) {
      seriesOptions.value.push(current)
    }
  } finally {
    isSearchingSeries.value = false
  }
}

const filteredDeveloperOptions = computed(() => {
  return sortCreatableOptionsByName(developerOptions.value)
})

const handleDeveloperSearch = async (query: string) => {
  if (!query) return
  isSearchingDevelopers.value = true
  try {
    const { developersService } = await import('@/services/developers.service')
    developerOptions.value = await searchCreatableOptions({
      query,
      selectedValues: form.value.developers,
      currentOptions: developerOptions.value,
      search: (keyword) => developersService.searchDevelopers(keyword),
    })
  } finally {
    isSearchingDevelopers.value = false
  }
}

const filteredPublisherOptions = computed(() => {
  return sortCreatableOptionsByName(publisherOptions.value)
})

const handlePublisherSearch = async (query: string) => {
  if (!query) return
  isSearchingPublishers.value = true
  try {
    const { publishersService } = await import('@/services/publishers.service')
    publisherOptions.value = await searchCreatableOptions({
      query,
      selectedValues: form.value.publishers,
      currentOptions: publisherOptions.value,
      search: (keyword) => publishersService.searchPublishers(keyword),
    })
  } finally {
    isSearchingPublishers.value = false
  }
}

const form = ref<GameForm>({
  title: '',
  title_alt: '',
  visibility: 'public',
  developers: [],
  publishers: [],
  release_date: undefined,
  engine: '',
  platform: [],
  series: null,
  tag_ids: [],
  summary: '',
  cover_image: '',
  banner_image: '',
  preview_videos: [],
  primary_preview_video_uid: '',
  screenshots: [],
  file_paths: [{ path: '', label: '' }]
})

const {
  isPreparingWikiTagCandidates,
  isApplyingWikiTags,
  wikiTagPickerVisible,
  wikiTagCandidates,
  tagOptionsByGroup,
  tagSelectionsByGroup,
  pendingTagOptionsByGroup,
  handleTagSelectionChange,
  handleParseWikiTags,
  handleWikiTagCandidateGroupChange,
  applySelectedWikiTags,
  resolveTagSelections,
  resetTagSelectionState,
} = useTagSelection({
  tagGroups,
  tagOptions,
  formTagIds: computed({
    get: () => form.value.tag_ids,
    set: (value) => {
      form.value.tag_ids = value
    },
  }),
  getWikiContent: () => props.game?.wiki_content || '',
  addAlert: (message, type) => {
    uiStore.addAlert(message, type)
  },
})

// Cover image state
const uploadAction = computed(() => {
  const baseUrl = import.meta.env.VITE_API_BASE_URL || '/api'
  return `${baseUrl}/assets/cover`
})
const uploadData = computed(() => ({
  game_id: String(props.game?.id || ''),
  sort_order: '0',
}))

// Banner image state
const bannerUploadAction = computed(() => {
  const baseUrl = import.meta.env.VITE_API_BASE_URL || '/api'
  return `${baseUrl}/assets/banner`
})
const bannerUploadData = computed(() => ({
  game_id: String(props.game?.id || ''),
  sort_order: '0',
}))

// Screenshot state
const showVideoSelector = ref(false)
const isUploadingVideo = ref(false)
const videoUploadProgress = ref(0)
const videoUploadFileName = ref('')
const screenshotUploadAction = computed(() => {
  const baseUrl = import.meta.env.VITE_API_BASE_URL || '/api'
  return `${baseUrl}/assets/screenshot`
})
const screenshotUploadData = computed(() => ({
  game_id: String(props.game?.id || ''),
  sort_order: String(form.value.screenshots.length),
}))
const uploadHeaders = computed(() => {
  return {}
})

const createScreenshotKey = (asset: Pick<EditableScreenshot, 'id' | 'asset_uid' | 'path'>, index = 0) => {
  if (asset.asset_uid) return `uid:${asset.asset_uid}`
  if (typeof asset.id === 'number') return `db:${asset.id}`
  return `path:${asset.path}:${index}:${Date.now()}`
}

const createEditableScreenshot = (
  asset: ScreenshotItem | UploadedAssetResult | string,
  index: number,
): EditableScreenshot => {
  if (typeof asset === 'string') {
    return {
      path: asset,
      sort_order: index,
      client_key: createScreenshotKey({ path: asset }, index),
    }
  }

  const screenshotId = 'id' in asset ? asset.id : ('asset_id' in asset ? asset.asset_id : undefined)
  const screenshotSortOrder = 'sort_order' in asset ? asset.sort_order : index

  return {
    id: screenshotId,
    asset_uid: asset.asset_uid,
    path: asset.path,
    sort_order: screenshotSortOrder ?? index,
    client_key: createScreenshotKey({
      id: screenshotId,
      asset_uid: asset.asset_uid,
      path: asset.path,
    }, index),
  }
}

const createEditableVideo = (asset: VideoAssetItem | UploadedAssetResult | string): EditableVideo => {
  if (typeof asset === 'string') {
    return { path: asset }
  }
  return {
    id: 'id' in asset ? asset.id : ('asset_id' in asset ? asset.asset_id : undefined),
    asset_uid: asset.asset_uid,
    path: asset.path,
    sort_order: 'sort_order' in asset ? asset.sort_order : undefined,
  }
}

const {
  draggedScreenshotKey,
  dragOverScreenshotKey,
  reorderEditableVideos,
  handleScreenshotDragStart,
  handleScreenshotDragEnter,
  handleScreenshotDrop,
  handleScreenshotDragEnd,
  setPrimaryPreviewVideo,
} = useEditGameMediaState({
  form: computed({
    get: () => form.value,
    set: (value) => {
      form.value = value
    },
  }),
})

const {
  showFileBrowser,
  initialPath,
  addFilePath,
  removeFilePath,
  openFileBrowser,
  handleFileSelect,
  resetFileBrowserState,
} = useGameFilePaths({
  filePaths: computed(() => form.value.file_paths),
  getDefaultDirectory: () => directoryService.getDefaultDirectory(),
  onResolveInitialPathError: (error) => {
    console.error('Failed to get default directory:', error)
  },
})

// Release date for date picker (Date object)
const releaseDate = ref<Date | null>(null)

const visible = computed({
  get: () => props.visible,
  set: (value) => emit('update:visible', value)
})

const { hydrateFormFromGame, initializeOptions } = useEditGameFormBootstrap({
  form: computed({
    get: () => form.value,
    set: (value) => {
      form.value = value
    },
  }),
  releaseDate,
  seriesOptions,
  platformOptions,
  tagGroups,
  tagOptions,
  developerOptions,
  publisherOptions,
  resetTagSelectionState,
  createEditableScreenshot,
  createEditableVideo,
})

const resetTransientState = () => {
  resetTagSelectionState()
  resetPendingDeleteAssets()
  resetFileBrowserState()
  resetSteamImportState()
  resetVideoUploadState()
}

// Initialize form and options
watch(() => props.game, async (game) => {
  await initializeOptions(game)
  hydrateFormFromGame(game)
}, { immediate: true })

// Reset state when modal opens
watch(visible, async (val) => {
  resetTransientState()
  if (val) {
    await initializeOptions(props.game)
    hydrateFormFromGame(props.game)
  }
})

// Handle date change from date picker
const handleDateChange = (value: Date | number | string | null) => {
  if (value) {
    // Convert to Date object if needed
    const dateObj = value instanceof Date ? value : new Date(value)
    // Set release_date as YYYY-MM-DD format (avoid timezone issues)
    const year = dateObj.getFullYear()
    const month = String(dateObj.getMonth() + 1).padStart(2, '0')
    const day = String(dateObj.getDate()).padStart(2, '0')
    form.value.release_date = `${year}-${month}-${day}`
  } else {
    form.value.release_date = undefined
  }
}

const handleTagSectionSelectionChange = (payload: {
  groupId: number
  value: GameTagSelectionValue
}) => {
  if (payload.value === null || payload.value === undefined) {
    handleTagSelectionChange(payload.groupId, undefined)
    return
  }

  if (typeof payload.value === 'string' || typeof payload.value === 'number') {
    handleTagSelectionChange(payload.groupId, payload.value)
    return
  }

  const arrayValue = payload.value
  if (arrayValue.every((item) => typeof item === 'string')) {
    handleTagSelectionChange(payload.groupId, arrayValue as string[])
    return
  }

  if (arrayValue.every((item) => typeof item === 'number')) {
    handleTagSelectionChange(payload.groupId, arrayValue as number[])
    return
  }

  handleTagSelectionChange(payload.groupId, arrayValue.map((item) => String(item)))
}

const handleFilePathItemUpdate = (payload: {
  index: number
  field: 'path' | 'label'
  value: string
}) => {
  const target = form.value.file_paths[payload.index]
  if (!target) return
  target[payload.field] = payload.value
}

const { queueAssetDeletion, resetPendingDeleteAssets, handleSubmit } = useEditGameWorkflow({
  game: computed(() => props.game),
  form: computed({
    get: () => form.value,
    set: (value) => {
      form.value = value
    },
  }),
  isSubmitting,
  seriesOptions,
  developerOptions,
  publisherOptions,
  platformOptions,
  validateForm: async () => {
    try {
      await formRef.value?.validate()
      return true
    } catch {
      return false
    }
  },
  resolveTagSelections,
  addAlert: (message, type) => {
    uiStore.addAlert(message, type)
  },
  emitSuccess: () => {
    emit('success')
  },
  closeModal: () => {
    visible.value = false
  },
})

const uploadAssetFromUrl = async (
  url: string,
  assetType: 'cover' | 'banner' | 'screenshot' | 'video',
  sortOrder = 0,
) => {
  if (!props.game?.id) {
    throw new Error('缺少游戏 ID')
  }

  const response = await fetch(proxySteamAssetUrl(url))
  if (!response.ok) {
    throw new Error(`下载远程图片失败: ${response.status}`)
  }

  const blob = await response.blob()
  const ext = blob.type.split('/')[1] || 'jpg'
  const file = new File([blob], `${assetType}-${Date.now()}.${ext}`, {
    type: blob.type || 'image/jpeg',
  })

  return uploadAsset(assetType, props.game.id, file, sortOrder)
}

const {
  showSummarySelector,
  steamSummaryPreview,
  showCoverSelector,
  coverSearchUrl,
  coverPreviewUrl,
  isDownloadingCover,
  steamCoverImages,
  selectedCoverImage,
  showBannerSelector,
  bannerSearchUrl,
  bannerPreviewUrl,
  isDownloadingBanner,
  steamBannerImages,
  selectedBannerImage,
  showScreenshotSelector,
  screenshotSearchUrl,
  screenshotPreviewUrl,
  isDownloadingScreenshot,
  steamScreenshotsData,
  selectedSteamScreenshots,
  isDownloadingSteamScreenshots,
  steamSummarySearchQuery,
  steamSummarySearchResults,
  selectedSteamSummaryGame,
  isSearchingSteamSummary,
  steamCoverSearchQuery,
  steamCoverSearchResults,
  selectedSteamGame,
  isSearchingSteamCover,
  steamBannerSearchQuery,
  steamBannerSearchResults,
  selectedSteamBannerGame,
  isSearchingSteamBanner,
  steamScreenshotSearchQuery,
  steamScreenshotSearchResults,
  selectedSteamScreenshotGame,
  isSearchingSteamScreenshots,
  handleSummarySearchClear,
  searchSteamForSummary,
  selectSteamSummaryGame,
  backToSummarySearch,
  confirmSummaryImport,
  handleCoverSearchClear,
  searchSteamForCover,
  selectSteamCoverGame,
  backToCoverGameSearch,
  loadCoverFromUrl,
  confirmCoverSelection,
  downloadSelectedSteamCover,
  handleBannerSearchClear,
  searchSteamForBanner,
  selectSteamBannerGame,
  backToBannerGameSearch,
  loadBannerFromUrl,
  confirmBannerSelection,
  downloadSelectedSteamBanner,
  handleScreenshotSearchClear,
  searchSteamForScreenshots,
  selectSteamScreenshotGame,
  backToScreenshotGameSearch,
  toggleSteamScreenshot,
  loadScreenshotPreview,
  confirmScreenshotSelection,
  downloadSelectedSteamScreenshots,
  resetSteamImportState,
} = useSteamImport({
  form: computed({
    get: () => form.value,
    set: (value) => {
      form.value = value
    },
  }),
  releaseDate,
  gameId: computed(() => props.game?.id),
  uploadAssetFromUrl,
  queueAssetDeletion,
  createEditableScreenshot,
  addAlert: (message, type) => {
    uiStore.addAlert(message, type)
  },
})

// Cover image handlers
const handleCoverError = (e: Event) => {
  const img = e.target as HTMLImageElement
  img.src = 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"%3E%3Crect fill="%23333" width="100" height="100"/%3E%3Ctext x="50%25" y="50%25" text-anchor="middle" dy=".3em" fill="%23666" font-size="12"%3E加载失败%3C/text%3E%3C/svg%3E'
}

const {
  handleCoverUploadSuccess,
  handleCoverUploadError,
  handleBannerUploadSuccess,
  handleBannerUploadError,
  handleScreenshotUploadSuccess,
  handleScreenshotUploadError,
  openVideoSelector,
  handleVideoFileChange,
  removeCover,
  removeBanner,
  removeScreenshot,
  removePreviewVideo,
  resetVideoUploadState,
} = useEditGameAssets({
  form: computed({
    get: () => form.value,
    set: (value) => {
      form.value = value
    },
  }),
  gameId: computed(() => props.game?.id),
  showCoverSelector,
  showBannerSelector,
  showScreenshotSelector,
  showVideoSelector,
  isUploadingVideo,
  videoUploadProgress,
  videoUploadFileName,
  queueAssetDeletion,
  createEditableScreenshot: (asset, index) => createEditableScreenshot(asset, index),
  createEditableVideo: (asset) => createEditableVideo(asset),
  addAlert: (message, type) => {
    uiStore.addAlert(message, type)
  },
})

const handleCancel = () => {
  visible.value = false
  resetPendingDeleteAssets()
}
</script>

<style scoped src="./edit-game/EditGameModal.css"></style>
