import { computed, onMounted, onUnmounted, ref, watch, type Ref } from 'vue'
import {
  createEmptyEditGameForm,
  formatEditGameReleaseDate,
  parseEditGameReleaseDate,
  type EditGameEditableBanner,
  type EditGameEditableCover,
  type EditGameEditableLogo,
  type EditGameEditableScreenshot,
  type EditGameEditableVideo,
  type EditGameForm,
} from '@/utils/edit-game-form'
import { uploadAsset, type UploadedAssetResult } from '@/services/assets'
import { buildAssetUploadUrl } from '@/services/api-url'
import { directoryService } from '@/services/directory.service'
import { proxySteamAssetUrl } from '@/services/steam.service'
import { seriesService } from '@/services/series.service'
import { developersService } from '@/services/developers.service'
import { publishersService } from '@/services/publishers.service'
import { getAssetFileExtension } from '@/utils/asset-file-extension'
import { useGameFilePaths } from '@/composables/useGameFilePaths'
import { useSteamImport } from '@/composables/useSteamImport'
import { useEditGameWorkflow } from '@/composables/useEditGameWorkflow'
import { useEditGameAssets } from '@/composables/useEditGameAssets'
import { useEditGameFormBootstrap } from '@/composables/useEditGameFormBootstrap'
import { useEditGameMediaState } from '@/composables/useEditGameMediaState'
import {
  normalizeMetadataPickerID,
  normalizeMetadataPickerIDs,
  useRemoteMetadataPicker,
} from '@/composables/useRemoteMetadataPicker'
import { sortCreatableOptionsByName } from '@/utils/creatable-select'
import type {
  AdminGameDetail,
  BannerItem,
  CoverItem,
  Developer,
  LogoItem,
  Publisher,
  ScreenshotItem,
  Series,
  VideoAssetItem,
} from '@/services/types'
import { useUiStore } from '@/stores/ui'

interface UseEditGameModalOptions {
  props: {
    visible: boolean
    game: AdminGameDetail | null
  }
  emit: {
    (event: 'update:visible', value: boolean): void
    (event: 'success'): void
    (event: 'sync'): void
  }
  uiStore: ReturnType<typeof useUiStore>
  formRef: Ref<{ validate?: () => Promise<unknown> } | undefined>
  isSubmitting: Ref<boolean>
  activeTab: Ref<string>
}

const CREATE_SERIES_OPTION_VALUE = '__create_series__'
const CREATE_DEVELOPER_OPTION_VALUE = '__create_developer__'
const CREATE_PUBLISHER_OPTION_VALUE = '__create_publisher__'

export const useEditGameModal = ({
  props,
  emit,
  uiStore,
  formRef,
  isSubmitting,
  activeTab,
}: UseEditGameModalOptions) => {
  const viewportWidth = ref(typeof window !== 'undefined' ? window.innerWidth : 1280)
  const isUploadingVideo = ref(false)
  const videoUploadProgress = ref(0)
  const videoUploadFileName = ref('')

  const rules = {
    title: [{ required: true, message: '请输入游戏名称' }],
  }

  const form = ref<EditGameForm>(createEmptyEditGameForm())
  const seriesPicker = useRemoteMetadataPicker<Series>({
    selectedIds: () => form.value.series_id === null ? [] : [form.value.series_id],
    search: (query) => seriesService.searchSeries(query),
    create: (name) => seriesService.createSeries({ name }),
  })
  const developerPicker = useRemoteMetadataPicker<Developer>({
    selectedIds: () => form.value.developer_ids,
    search: (query) => developersService.listDevelopers({ query }),
    create: (name) => developersService.createDeveloper({ name }),
  })
  const publisherPicker = useRemoteMetadataPicker<Publisher>({
    selectedIds: () => form.value.publisher_ids,
    search: (query) => publishersService.listPublishers({ query }),
    create: (name) => publishersService.createPublisher({ name }),
  })
  const seriesOptions = seriesPicker.options
  const developerOptions = developerPicker.options
  const publisherOptions = publisherPicker.options
  const isSearchingSeries = seriesPicker.isSearching
  const isSearchingDevelopers = developerPicker.isSearching
  const isSearchingPublishers = publisherPicker.isSearching
  const isCreatingSeries = seriesPicker.isCreating
  const isCreatingDevelopers = developerPicker.isCreating
  const isCreatingPublishers = publisherPicker.isCreating
  const canCreateSeriesOption = seriesPicker.canCreate
  const canCreateDeveloperOption = developerPicker.canCreate
  const canCreatePublisherOption = publisherPicker.canCreate

  const openLogoSelector = () => {
    showLogoSelector.value = true
  }

  const modalWidth = computed(() => {
    const isMediaTab = activeTab.value === 'media'
    if (viewportWidth.value <= 576) return 'calc(100vw - 24px)'
    if (isMediaTab) return 'min(70vw, 1400px)'
    if (viewportWidth.value <= 912) return 'min(800px, calc(100vw - 48px))'
    return 800
  })

  const filteredSeriesOptions = computed(() => {
    // 2026-04-04: keep authoring pickers alphabetized for scan speed.
    // Impact: this is UI-only option ordering; do not treat it as backend metadata sort semantics.
    return sortCreatableOptionsByName(seriesOptions.value)
  })

  const filteredDeveloperOptions = computed(() => {
    // 2026-04-04: keep authoring pickers alphabetized for scan speed.
    // Impact: this is UI-only option ordering; do not treat it as backend metadata sort semantics.
    return sortCreatableOptionsByName(developerOptions.value)
  })

  const filteredPublisherOptions = computed(() => {
    // 2026-04-04: keep authoring pickers alphabetized for scan speed.
    // Impact: this is UI-only option ordering; do not treat it as backend metadata sort semantics.
    return sortCreatableOptionsByName(publisherOptions.value)
  })

  const currentGame = computed(() => props.game)
  const currentGameId = computed(() => props.game?.id)
  // 2026-04-06: Wiki-derived authoring actions depend on parsable content,
  // not merely on the transport field being present.
  const hasParsableWikiContent = computed(() => Boolean(props.game?.wiki_content?.trim()))
  const releaseDate = computed<Date | null>({
    get: () => parseEditGameReleaseDate(form.value.release_date),
    set: (value) => {
      form.value.release_date = formatEditGameReleaseDate(value)
    },
  })
  const addAlert = (message: string, type: 'success' | 'warning' | 'error') => {
    uiStore.addAlert(message, type)
  }

  let assetSyncTimer: ReturnType<typeof setTimeout> | undefined
  const emitAssetSync = () => {
    if (typeof window === 'undefined') {
      emit('sync')
      return
    }
    if (assetSyncTimer) clearTimeout(assetSyncTimer)
    assetSyncTimer = setTimeout(() => {
      assetSyncTimer = undefined
      emit('sync')
    }, 600)
  }

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
    if (assetSyncTimer) clearTimeout(assetSyncTimer)
  })

  const handleSeriesSearch = async (query: string) => {
    try {
      await seriesPicker.search(query)
    } catch {
      addAlert('系列搜索失败', 'error')
    }
  }

  const handleDeveloperSearch = async (query: string) => {
    try {
      await developerPicker.search(query)
    } catch {
      addAlert('开发商搜索失败', 'error')
    }
  }

  const handlePublisherSearch = async (query: string) => {
    try {
      await publisherPicker.search(query)
    } catch {
      addAlert('发行商搜索失败', 'error')
    }
  }

  const createSeriesFromSearch = async () => {
    try {
      const item = await seriesPicker.createFromQuery()
      if (!item) return
      form.value.series_id = item.id
      seriesPicker.clearSearch()
    } catch {
      addAlert('创建系列失败', 'error')
    }
  }

  const handleSeriesEnter = async () => {
    try {
      const item = await seriesPicker.resolveQuery()
      if (!item) return
      form.value.series_id = item.id
      seriesPicker.clearSearch()
    } catch {
      addAlert('选择或创建系列失败', 'error')
    }
  }

  const createDeveloperFromSearch = async () => {
    try {
      const item = await developerPicker.createFromQuery()
      if (!item) return
      form.value.developer_ids = Array.from(new Set([...form.value.developer_ids, item.id]))
      developerPicker.clearSearch()
    } catch {
      addAlert('创建开发商失败', 'error')
    }
  }

  const createPublisherFromSearch = async () => {
    try {
      const item = await publisherPicker.createFromQuery()
      if (!item) return
      form.value.publisher_ids = Array.from(new Set([...form.value.publisher_ids, item.id]))
      publisherPicker.clearSearch()
    } catch {
      addAlert('创建发行商失败', 'error')
    }
  }

  const handleSeriesSelection = (value: unknown) => {
    if (value === CREATE_SERIES_OPTION_VALUE) {
      void createSeriesFromSearch()
      return
    }
    form.value.series_id = normalizeMetadataPickerID(value)
    seriesPicker.clearSearch()
  }

  const handleDeveloperSelection = (value: unknown) => {
    const values = Array.isArray(value) ? value : []
    form.value.developer_ids = normalizeMetadataPickerIDs(values)
    if (values.includes(CREATE_DEVELOPER_OPTION_VALUE)) {
      void createDeveloperFromSearch()
    }
  }

  const handlePublisherSelection = (value: unknown) => {
    const values = Array.isArray(value) ? value : []
    form.value.publisher_ids = normalizeMetadataPickerIDs(values)
    if (values.includes(CREATE_PUBLISHER_OPTION_VALUE)) {
      void createPublisherFromSearch()
    }
  }


  const uploadAction = computed(() => {
    return buildAssetUploadUrl('cover')
  })

  const uploadData = computed(() => ({
    game_id: String(props.game?.id || ''),
    sort_order: '0',
  }))

  const bannerUploadAction = computed(() => {
    return buildAssetUploadUrl('banner')
  })

  const bannerUploadData = computed(() => ({
    game_id: String(props.game?.id || ''),
    sort_order: '0',
  }))

  const screenshotUploadAction = computed(() => {
    return buildAssetUploadUrl('screenshot')
  })

  const screenshotUploadData = computed(() => ({
    game_id: String(props.game?.id || ''),
    sort_order: String(form.value.screenshots.length),
  }))

  const logoUploadAction = computed(() => {
    return buildAssetUploadUrl('logo')
  })

  const logoUploadData = computed(() => ({
    game_id: String(props.game?.id || ''),
    sort_order: '0',
  }))

  const uploadHeaders = computed(() => ({}))

  const createScreenshotKey = (
    asset: Pick<EditGameEditableScreenshot, 'id' | 'asset_uid' | 'path'>,
    index = 0,
  ) => {
    if (asset.asset_uid) return `uid:${asset.asset_uid}`
    if (typeof asset.id === 'number') return `db:${asset.id}`
    return `path:${asset.path}:${index}:${Date.now()}`
  }

  const createEditableScreenshot = (
    asset: ScreenshotItem | UploadedAssetResult | string,
    index: number,
  ): EditGameEditableScreenshot => {
    if (typeof asset === 'string') {
      return {
        path: asset,
        client_key: createScreenshotKey({ path: asset }, index),
      }
    }

    const screenshotId = 'id' in asset ? asset.id : undefined

    return {
      id: screenshotId,
      asset_uid: asset.asset_uid,
      path: asset.path,
      client_key: createScreenshotKey({
        id: screenshotId,
        asset_uid: asset.asset_uid,
        path: asset.path,
      }, index),
    }
  }

  const createEditableVideo = (asset: VideoAssetItem | UploadedAssetResult | string): EditGameEditableVideo => {
    if (typeof asset === 'string') {
      return { path: asset }
    }
    return {
      id: 'id' in asset ? asset.id : undefined,
      asset_uid: asset.asset_uid,
      path: asset.path,
      poster_path: 'poster_path' in asset ? (asset.poster_path ?? null) : null,
    }
  }

  const createEditableCover = (asset: CoverItem | UploadedAssetResult | string): EditGameEditableCover => {
    if (typeof asset === 'string') {
      return { path: asset }
    }
    return {
      id: 'id' in asset ? asset.id : undefined,
      asset_uid: asset.asset_uid,
      path: asset.path,
    }
  }

  const createEditableBanner = (asset: BannerItem | UploadedAssetResult | string): EditGameEditableBanner => {
    if (typeof asset === 'string') {
      return { path: asset }
    }
    return {
      id: 'id' in asset ? asset.id : undefined,
      asset_uid: asset.asset_uid,
      path: asset.path,
    }
  }

  const createEditableLogo = (asset: LogoItem | UploadedAssetResult | string): EditGameEditableLogo => {
    if (typeof asset === 'string') {
      return { path: asset, position_x: null, position_y: null, width_pct: null }
    }
    const isLogoItem = 'sort_order' in asset
    return {
      id: 'id' in asset ? asset.id : undefined,
      asset_uid: asset.asset_uid,
      path: asset.path,
      position_x: isLogoItem ? (asset as LogoItem).position_x ?? null : null,
      position_y: isLogoItem ? (asset as LogoItem).position_y ?? null : null,
      width_pct: isLogoItem ? (asset as LogoItem).width_pct ?? null : null,
    }
  }

  const {
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
    handleScreenshotDragStart,
    handleScreenshotDragEnter,
    handleScreenshotDrop,
    handleScreenshotDragEnd,
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
  } = useEditGameMediaState({
    form,
  })

  const {
    showFileBrowser,
    initialPath,
    addFilePath,
    removeFilePath,
    openFileBrowser,
    handleFileSelect,
    handleFilePathItemUpdate,
    resetFileBrowserState,
  } = useGameFilePaths({
    form,
    getDefaultDirectory: () => directoryService.getDefaultDirectory(),
    onResolveInitialPathError: (message) => {
      addAlert(message, 'error')
    },
  })

  const visible = computed({
    get: () => props.visible,
    set: (value) => emit('update:visible', value),
  })

  const { hydrateFormFromGame, initializeOptions } = useEditGameFormBootstrap({
    form,
    seriesOptions,
    developerOptions,
    publisherOptions,
    addAlert,
    createEditableCover,
    createEditableBanner,
    createEditableLogo,
    createEditableScreenshot,
    createEditableVideo,
  })

  const {
    handleSubmit,
  } = useEditGameWorkflow({
    game: currentGame,
    form,
    isSubmitting,
    validateForm: async () => {
      try {
        await formRef.value?.validate?.()
        return true
      } catch {
        return false
      }
    },
    addAlert,
    emitSuccess: () => {
      emit('success')
    },
    closeModal: () => {
      visible.value = false
    },
  })

  const uploadAssetFromUrl = async (
    url: string,
    assetType: 'cover' | 'banner' | 'screenshot' | 'video' | 'logo',
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
    selectedCovers,
    isDownloadingSteamCovers,
    showBannerSelector,
    bannerSearchUrl,
    bannerPreviewUrl,
    isDownloadingBanner,
    steamBannerImages,
    selectedBanners,
    isDownloadingSteamBanners,
    showScreenshotSelector,
    screenshotSearchUrl,
    screenshotPreviewUrl,
    isDownloadingScreenshot,
    screenshotCandidatesData,
    selectedRemoteScreenshots,
    isDownloadingRemoteScreenshots,
    steamSummarySearchQuery,
    steamSummarySearchResults,
    selectedSteamSummaryGame,
    isSearchingSteamSummary,
    steamCoverSearchQuery,
    coverSearchResults,
    selectedSteamGame,
    isSearchingCover,
    steamBannerSearchQuery,
    bannerSearchResults,
    selectedSteamBannerGame,
    isSearchingBanner,
    screenshotSearchQuery,
    screenshotSearchResults,
    selectedScreenshotGame,
    isSearchingScreenshots,
    showLogoSelector,
    logoSearchUrl,
    logoPreviewUrl,
    isDownloadingLogo,
    logoImages,
    selectedLogos,
    isDownloadingLogos,
    logoSearchQuery,
    logoSearchResults,
    selectedLogoGame,
    isSearchingLogo,
    coverSource,
    bannerSource,
    screenshotSource,
    sgdbAvailable,
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
    downloadSelectedSteamCovers,
    toggleCoverSelection,
    selectAllCovers,
    invertSelectionCovers,
    handleBannerSearchClear,
    searchSteamForBanner,
    selectSteamBannerGame,
    backToBannerGameSearch,
    loadBannerFromUrl,
    confirmBannerSelection,
    downloadSelectedSteamBanner,
    toggleBannerSelection,
    selectAllBanners,
    invertSelectionBanners,
    handleScreenshotSearchClear,
    searchScreenshots,
    selectScreenshotGame,
    backToScreenshotGameSearch,
    toggleScreenshot,
    selectAllScreenshots,
    invertSelectionScreenshots,
    loadScreenshotPreview,
    confirmScreenshotSelection,
    downloadSelectedScreenshots,
    handleLogoSearchClear,
    searchLogos,
    selectLogoGame,
    backToLogoGameSearch,
    toggleLogoSelection,
    selectAllLogos,
    invertSelectionLogos,
    downloadSelectedLogos,
    loadLogoFromUrl,
    confirmLogoSelection,
    resetSteamImportState,
  } = useSteamImport({
    form,
    gameId: currentGameId,
    getWikiContent: () => props.game?.wiki_content || '',
    uploadAssetFromUrl,
    createEditableCover,
    createEditableBanner,
    createEditableLogo,
    createEditableScreenshot,
    ensureDeveloperNames: developerPicker.ensureNames,
    ensurePublisherNames: publisherPicker.ensureNames,
    addAlert,
    onAssetPersisted: emitAssetSync,
  })

  const handleCoverError = (event: Event) => {
    const img = event.target as HTMLImageElement
    img.src = 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"%3E%3Crect fill="%23333" width="100" height="100"/%3E%3Ctext x="50%25" y="50%25" text-anchor="middle" dy=".3em" fill="%23666" font-size="12"%3E加载失败%3C/text%3E%3C/svg%3E'
  }

  const {
    handleCoverUploadSuccess,
    handleCoverUploadError,
    handleLogoUploadSuccess,
    handleLogoUploadError,
    handleBannerUploadSuccess,
    handleBannerUploadError,
    handleScreenshotUploadSuccess,
    handleScreenshotUploadError,
    handleVideoFileChange,
    removeCover,
    removeLogo,
    removeBanner,
    removeScreenshot,
    removePreviewVideo,
    handleLogoPositionChange,
    resetVideoUploadState,
  } = useEditGameAssets({
    form,
    gameId: currentGameId,
    showCoverSelector,
    showBannerSelector,
    showScreenshotSelector,
    isUploadingVideo,
    videoUploadProgress,
    videoUploadFileName,
    createEditableCover,
    createEditableBanner,
    createEditableLogo,
    createEditableScreenshot,
    createEditableVideo,
    addAlert,
    onAssetPersisted: emitAssetSync,
  })

  const resetTransientState = () => {
    seriesPicker.reset()
    developerPicker.reset()
    publisherPicker.reset()
    resetFileBrowserState()
    resetSteamImportState()
    resetVideoUploadState()
  }

  watch(() => props.game, async (game, previousGame) => {
    await initializeOptions(game)
    const isSameGame = game?.id && previousGame?.id && game.id === previousGame.id
    if (props.visible && isSameGame) {
      // Keep in-progress form edits intact while background asset sync refreshes the source game payload.
      return
    }
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
  }

  return {
    CREATE_DEVELOPER_OPTION_VALUE,
    CREATE_PUBLISHER_OPTION_VALUE,
    CREATE_SERIES_OPTION_VALUE,
    bannerUploadAction,
    bannerUploadData,
    bannerPreviewUrl,
    bannerSearchUrl,
    backToBannerGameSearch,
    backToCoverGameSearch,
    backToLogoGameSearch,
    backToScreenshotGameSearch,
    backToSummarySearch,
    canCreateDeveloperOption,
    canCreatePublisherOption,
    canCreateSeriesOption,
    confirmBannerSelection,
    confirmCoverSelection,
    confirmScreenshotSelection,
    confirmSummaryImport,
    coverPreviewUrl,
    coverSearchUrl,
    downloadSelectedSteamBanner,
    toggleBannerSelection,
    selectAllBanners,
    invertSelectionBanners,
    downloadSelectedSteamCover,
    downloadSelectedSteamCovers,
    downloadSelectedLogos,
    toggleLogoSelection,
    selectAllLogos,
    invertSelectionLogos,
    downloadSelectedScreenshots,
    confirmLogoSelection,
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
    filteredDeveloperOptions,
    filteredPublisherOptions,
    filteredSeriesOptions,
    form,
    handleBannerSearchClear,
    handleBannerUploadError,
    handleBannerUploadSuccess,
    handleCancel,
    hasParsableWikiContent,
    handleCoverError,
    handleCoverSearchClear,
    handleCoverUploadError,
    handleCoverUploadSuccess,
    handleLogoUploadSuccess,
    handleLogoUploadError,
    handleLogoSearchClear,
    handleDeveloperSearch,
    handleDeveloperSelection,
    handleFilePathItemUpdate,
    handleFileSelect,
    handlePublisherSearch,
    handlePublisherSelection,
    handleCoverDragEnd,
    handleCoverDragEnter,
    handleCoverDragStart,
    handleCoverDrop,
    handleBannerDragEnd,
    handleBannerDragEnter,
    handleBannerDragStart,
    handleBannerDrop,
    handleLogoDragEnd,
    handleLogoDragEnter,
    handleLogoDragStart,
    handleLogoDrop,
    handleVideoDragEnd,
    handleVideoDragEnter,
    handleVideoDragStart,
    handleVideoDrop,
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
    isApplyingWikiMetadata,
    isDownloadingBanner,
    isDownloadingCover,
    isDownloadingScreenshot,
    isDownloadingSteamCovers,
    isDownloadingRemoteScreenshots,
    isDownloadingLogos,
    isDownloadingLogo,
    isCreatingDevelopers,
    isCreatingPublishers,
    isCreatingSeries,
    isSearchingLogo,
    isPreparingWikiMetadataCandidates,
    isSearchingSeries,
    isSearchingDevelopers,
    isSearchingPublishers,
    isSearchingBanner,
    isSearchingCover,
    isSearchingScreenshots,
    isSearchingSteamSummary,
    isUploadingVideo,
    loadBannerFromUrl,
    loadCoverFromUrl,
    loadLogoFromUrl,
    loadScreenshotPreview,
    logoUploadAction,
    logoUploadData,
    logoSearchUrl,
    logoPreviewUrl,
    logoSearchResults,
    developerSearchQuery: developerPicker.query,
    publisherSearchQuery: publisherPicker.query,
    seriesSearchQuery: seriesPicker.query,
    logoImages,
    logoSearchQuery,
    selectedLogos,
    selectedLogoGame,
    modalWidth,
    openFileBrowser,
    openLogoSelector,
    releaseDate,
    removeBanner,
    removeCover,
    removeLogo,
    removeFilePath,
    removePreviewVideo,
    removeScreenshot,
    handleLogoPositionChange,
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
    selectSteamBannerGame,
    selectSteamCoverGame,
    selectLogoGame,
    selectScreenshotGame,
    selectSteamSummaryGame,
    selectedBanners,
    isDownloadingSteamBanners,
    selectedCoverImage,
    selectedCovers,
    selectedSteamBannerGame,
    selectedSteamGame,
    selectedScreenshotGame,
    selectedRemoteScreenshots,
    selectedSteamSummaryGame,
    showBannerSelector,
    showCoverSelector,
    showFileBrowser,
    showScreenshotSelector,
    showSummarySelector,
    showLogoSelector,
    steamBannerImages,
    steamBannerSearchQuery,
    bannerSearchResults,
    steamCoverImages,
    steamCoverSearchQuery,
    coverSearchResults,
    screenshotSearchQuery,
    screenshotSearchResults,
    screenshotCandidatesData,
    steamSummaryPreview,
    steamSummarySearchQuery,
    steamSummarySearchResults,
    toggleCoverSelection,
    selectAllCovers,
    invertSelectionCovers,
    toggleScreenshot,
    selectAllScreenshots,
    invertSelectionScreenshots,
    uploadAction,
    uploadData,
    uploadHeaders,
    videoUploadFileName,
    videoUploadProgress,
    visible,
    wikiMetadataCandidates,
    wikiMetadataPickerVisible,
    applySelectedWikiMetadata,
    addFilePath,
    coverSource,
    bannerSource,
    screenshotSource,
    sgdbAvailable,
  }
}
