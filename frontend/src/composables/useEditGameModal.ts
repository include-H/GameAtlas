import { computed, onMounted, onUnmounted, ref, watch, type Ref } from 'vue'
import { uploadAsset, type UploadedAssetResult } from '@/services/assets'
import { directoryService } from '@/services/directory.service'
import { proxySteamAssetUrl } from '@/services/steam.service'
import { seriesService } from '@/services/series.service'
import { resolveAssetCandidates } from '@/utils/asset-url'
import { getAssetFileExtension } from '@/utils/asset-file-extension'
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
  GameDetail,
  Platform,
  Publisher,
  ScreenshotItem,
  Series,
  Tag,
  TagGroup,
  VideoAssetItem,
} from '@/services/types'
import type { GameTagSelectionValue } from '@/components/edit-game/GameTagSection.vue'
import { useUiStore } from '@/stores/ui'

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
  developer_ids: Array<string | number>
  publisher_ids: Array<string | number>
  release_date: string | undefined
  engine: string
  platform_ids: (string | number)[]
  series_id: string | number | null
  tag_ids: Array<string | number>
  summary: string
  cover_image: string
  banner_image: string
  preview_videos: EditableVideo[]
  primary_preview_video_uid: string
  screenshots: EditableScreenshot[]
  file_paths: FilePathItem[]
}

interface UseEditGameModalOptions {
  props: {
    visible: boolean
    game: GameDetail | null
  }
  emit: {
    (event: 'update:visible', value: boolean): void
    (event: 'success'): void
  }
  uiStore: ReturnType<typeof useUiStore>
  formRef: Ref<{ validate?: () => Promise<unknown> } | undefined>
  isSubmitting: Ref<boolean>
}

export const useEditGameModal = ({
  props,
  emit,
  uiStore,
  formRef,
  isSubmitting,
}: UseEditGameModalOptions) => {
  const viewportWidth = ref(typeof window !== 'undefined' ? window.innerWidth : 1280)
  const seriesOptions = ref<Series[]>([])
  const platformOptions = ref<Platform[]>([])
  const tagGroups = ref<TagGroup[]>([])
  const tagOptions = ref<Tag[]>([])
  const developerOptions = ref<Developer[]>([])
  const publisherOptions = ref<Publisher[]>([])
  const isSearchingSeries = ref(false)
  const isSearchingDevelopers = ref(false)
  const isSearchingPublishers = ref(false)
  const showVideoSelector = ref(false)
  const isUploadingVideo = ref(false)
  const videoUploadProgress = ref(0)
  const videoUploadFileName = ref('')
  const releaseDate = ref<Date | null>(null)

  const rules = {
    title: [{ required: true, message: '请输入游戏名称' }],
  }

  const form = ref<GameForm>({
    title: '',
    title_alt: '',
    visibility: 'public',
    developer_ids: [],
    publisher_ids: [],
    release_date: undefined,
    engine: '',
    platform_ids: [],
    series_id: null,
    tag_ids: [],
    summary: '',
    cover_image: '',
    banner_image: '',
    preview_videos: [],
    primary_preview_video_uid: '',
    screenshots: [],
    file_paths: [{ path: '', label: '' }],
  })

  const primaryPreviewVideo = computed(() => {
    if (form.value.preview_videos.length === 0) return null
    const selected = form.value.preview_videos.find((item) => item.asset_uid === form.value.primary_preview_video_uid)
    return selected || form.value.preview_videos[0]
  })

  const previewVideoSources = computed(() => resolveAssetCandidates(primaryPreviewVideo.value?.path || ''))

  const modalWidth = computed(() => {
    if (viewportWidth.value <= 576) return 'calc(100vw - 24px)'
    if (viewportWidth.value <= 912) return 'min(800px, calc(100vw - 48px))'
    return 800
  })

  const filteredSeriesOptions = computed(() => {
    return [...seriesOptions.value].sort((a, b) => a.name.localeCompare(b.name, 'zh-Hans-CN'))
  })

  const filteredDeveloperOptions = computed(() => {
    return sortCreatableOptionsByName(developerOptions.value)
  })

  const filteredPublisherOptions = computed(() => {
    return sortCreatableOptionsByName(publisherOptions.value)
  })

  const syncViewportWidth = () => {
    viewportWidth.value = window.innerWidth
  }

  onMounted(() => {
    if (typeof window === 'undefined') return
    syncViewportWidth()
    window.addEventListener('resize', syncViewportWidth)
  })

  onUnmounted(() => {
    if (typeof window === 'undefined') return
    window.removeEventListener('resize', syncViewportWidth)
  })

  const handleSeriesSearch = async (query: string) => {
    if (!query) return
    isSearchingSeries.value = true
    try {
      const results = await seriesService.searchSeries(query)
      const currentSeriesId = form.value.series_id
      const current = seriesOptions.value.find((item) => item.id === currentSeriesId)
      seriesOptions.value = results
      if (current && !results.find((item) => item.id === current.id)) {
        seriesOptions.value.push(current)
      }
    } finally {
      isSearchingSeries.value = false
    }
  }

  const handleDeveloperSearch = async (query: string) => {
    if (!query) return
    isSearchingDevelopers.value = true
    try {
      const { developersService } = await import('@/services/developers.service')
      developerOptions.value = await searchCreatableOptions({
        query,
        selectedValues: form.value.developer_ids,
        currentOptions: developerOptions.value,
        search: (keyword) => developersService.searchDevelopers(keyword),
      })
    } finally {
      isSearchingDevelopers.value = false
    }
  }

  const handlePublisherSearch = async (query: string) => {
    if (!query) return
    isSearchingPublishers.value = true
    try {
      const { publishersService } = await import('@/services/publishers.service')
      publisherOptions.value = await searchCreatableOptions({
        query,
        selectedValues: form.value.publisher_ids,
        currentOptions: publisherOptions.value,
        search: (keyword) => publishersService.searchPublishers(keyword),
      })
    } finally {
      isSearchingPublishers.value = false
    }
  }

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

  const uploadAction = computed(() => {
    const baseUrl = import.meta.env.VITE_API_BASE_URL || '/api'
    return `${baseUrl}/assets/cover`
  })

  const uploadData = computed(() => ({
    game_id: String(props.game?.id || ''),
    sort_order: '0',
  }))

  const bannerUploadAction = computed(() => {
    const baseUrl = import.meta.env.VITE_API_BASE_URL || '/api'
    return `${baseUrl}/assets/banner`
  })

  const bannerUploadData = computed(() => ({
    game_id: String(props.game?.id || ''),
    sort_order: '0',
  }))

  const screenshotUploadAction = computed(() => {
    const baseUrl = import.meta.env.VITE_API_BASE_URL || '/api'
    return `${baseUrl}/assets/screenshot`
  })

  const screenshotUploadData = computed(() => ({
    game_id: String(props.game?.id || ''),
    sort_order: String(form.value.screenshots.length),
  }))

  const uploadHeaders = computed(() => ({}))

  const createScreenshotKey = (
    asset: Pick<EditableScreenshot, 'id' | 'asset_uid' | 'path'>,
    index = 0,
  ) => {
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
    onResolveInitialPathError: (message) => {
      console.error(message)
    },
  })

  const visible = computed({
    get: () => props.visible,
    set: (value) => emit('update:visible', value),
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

  const handleDateChange = (value: Date | number | string | null) => {
    if (value) {
      const dateObj = value instanceof Date ? value : new Date(value)
      const year = dateObj.getFullYear()
      const month = String(dateObj.getMonth() + 1).padStart(2, '0')
      const day = String(dateObj.getDate()).padStart(2, '0')
      form.value.release_date = `${year}-${month}-${day}`
      return
    }
    form.value.release_date = undefined
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

  const {
    queueAssetDeletion,
    resetPendingDeleteAssets,
    handleSubmit,
  } = useEditGameWorkflow({
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
        await formRef.value?.validate?.()
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
    const ext = getAssetFileExtension(blob.type, assetType)
    const file = new File([blob], `${assetType}-${Date.now()}.${ext}`, {
      type: blob.type || 'image/jpeg',
    })

    return uploadAsset(assetType, props.game.id, file, sortOrder)
  }

  const {
    showSummarySelector,
    steamSummaryPreview,
    isPreparingWikiMetadataCandidates,
    isApplyingWikiMetadata,
    wikiMetadataPickerVisible,
    wikiMetadataCandidates,
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
    importMetadataFromWiki,
    handleWikiMetadataCandidateSelectionChange,
    applySelectedWikiMetadata,
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
    getWikiContent: () => props.game?.wiki_content || '',
    uploadAssetFromUrl,
    queueAssetDeletion,
    createEditableScreenshot,
    addAlert: (message, type) => {
      uiStore.addAlert(message, type)
    },
  })

  const handleCoverError = (event: Event) => {
    const img = event.target as HTMLImageElement
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

  const resetTransientState = () => {
    resetTagSelectionState()
    resetPendingDeleteAssets()
    resetFileBrowserState()
    resetSteamImportState()
    resetVideoUploadState()
  }

  watch(() => props.game, async (game) => {
    await initializeOptions(game)
    hydrateFormFromGame(game)
  }, { immediate: true })

  watch(visible, async (value) => {
    resetTransientState()
    if (value) {
      await initializeOptions(props.game)
      hydrateFormFromGame(props.game)
    }
  })

  const handleCancel = () => {
    visible.value = false
    resetPendingDeleteAssets()
  }

  return {
    bannerUploadAction,
    bannerUploadData,
    bannerPreviewUrl,
    bannerSearchUrl,
    backToBannerGameSearch,
    backToCoverGameSearch,
    backToScreenshotGameSearch,
    backToSummarySearch,
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
    handleDateChange,
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
    seriesOptions,
    setPrimaryPreviewVideo,
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
    tagSelectionsByGroup,
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
    applySelectedWikiMetadata,
    applySelectedWikiTags,
    addFilePath,
  }
}
