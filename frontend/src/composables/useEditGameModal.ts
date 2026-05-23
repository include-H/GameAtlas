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
import { resolveAssetCandidates } from '@/utils/asset-url'
import { getAssetFileExtension } from '@/utils/asset-file-extension'
import { useGameFilePaths } from '@/composables/useGameFilePaths'
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
  const developerOptions = ref<Developer[]>([])
  const publisherOptions = ref<Publisher[]>([])
  const isSearchingSeries = ref(false)
  const isSearchingDevelopers = ref(false)
  const isSearchingPublishers = ref(false)
  const showVideoSelector = ref(false)
  const isUploadingVideo = ref(false)
  const videoUploadProgress = ref(0)
  const videoUploadFileName = ref('')

  const rules = {
    title: [{ required: true, message: '请输入游戏名称' }],
  }

  const form = ref<EditGameForm>(createEmptyEditGameForm())

  const primaryCover = computed(() => {
    return form.value.covers[0] || null
  })

  const primaryLogo = computed(() => {
    return form.value.logo || null
  })

  const logoBannerSrc = computed(() => form.value.banners[0]?.path || form.value.banner_image || form.value.covers[0]?.path || '')
  const logoPath = computed(() => form.value.logo?.path || '')

  const primaryPreviewVideo = computed(() => {
    return form.value.preview_videos[0] || null
  })

  const previewVideoSources = computed(() => resolveAssetCandidates(primaryPreviewVideo.value?.path || ''))

  const modalWidth = computed(() => {
    if (viewportWidth.value <= 576) return 'calc(100vw - 24px)'
    if (viewportWidth.value <= 912) return 'min(800px, calc(100vw - 48px))'
    return 800
  })

  const filteredSeriesOptions = computed(() => {
    // 2026-04-04: keep authoring pickers alphabetized for scan speed.
    // Impact: this is UI-only option ordering; do not treat it as backend metadata sort semantics.
    return [...seriesOptions.value].sort((a, b) => a.name.localeCompare(b.name, 'zh-Hans-CN'))
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

  const emitAssetSync = async () => {
    emit('sync')
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
    } catch {
      // 2026-04-08: picker search failures must stay distinguishable from a real empty result set.
      // Impact: keep the last successful options visible and surface an explicit error instead of "no options".
      addAlert('系列搜索失败', 'error')
    } finally {
      isSearchingSeries.value = false
    }
  }

  const handleDeveloperSearch = async (query: string) => {
    if (!query) return
    isSearchingDevelopers.value = true
    try {
      developerOptions.value = await searchCreatableOptions({
        query,
        selectedValues: form.value.developer_ids,
        currentOptions: developerOptions.value,
        search: (keyword) => developersService.listDevelopers({ query: keyword }),
      })
    } catch {
      addAlert('开发商搜索失败', 'error')
    } finally {
      isSearchingDevelopers.value = false
    }
  }

  const handlePublisherSearch = async (query: string) => {
    if (!query) return
    isSearchingPublishers.value = true
    try {
      publisherOptions.value = await searchCreatableOptions({
        query,
        selectedValues: form.value.publisher_ids,
        currentOptions: publisherOptions.value,
        search: (keyword) => publishersService.listPublishers({ query: keyword }),
      })
    } catch {
      addAlert('发行商搜索失败', 'error')
    } finally {
      isSearchingPublishers.value = false
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
    reorderEditableCovers,
    reorderEditableVideos,
    handleScreenshotDragStart,
    handleScreenshotDragEnter,
    handleScreenshotDrop,
    handleScreenshotDragEnd,
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
    seriesOptions,
    developerOptions,
    publisherOptions,
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
    steamScreenshotsData,
    selectedSteamScreenshots,
    isDownloadingSteamScreenshots,
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
    steamScreenshotSearchQuery,
    screenshotSearchResults,
    selectedSteamScreenshotGame,
    isSearchingScreenshots,
    showLogoSelector,
    logoSearchUrl,
    logoPreviewUrl,
    isDownloadingLogo,
    steamLogoImages,
    selectedLogoImage,
    isDownloadingSteamLogos,
    steamLogoSearchQuery,
    logoSearchResults,
    selectedSteamLogoGame,
    isSearchingLogo,
    coverSource,
    bannerSource,
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
    handleBannerSearchClear,
    searchSteamForBanner,
    selectSteamBannerGame,
    backToBannerGameSearch,
    loadBannerFromUrl,
    confirmBannerSelection,
    downloadSelectedSteamBanner,
    toggleBannerSelection,
    handleScreenshotSearchClear,
    searchSteamForScreenshots,
    selectSteamScreenshotGame,
    backToScreenshotGameSearch,
    toggleSteamScreenshot,
    loadScreenshotPreview,
    confirmScreenshotSelection,
    downloadSelectedSteamScreenshots,
    handleLogoSearchClear,
    searchSteamForLogo,
    selectSteamLogoGame,
    backToLogoGameSearch,
    downloadSelectedSteamLogo,
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
    openVideoSelector,
    handleVideoFileChange,
    removeCover,
    removeLogo,
    removeBanner,
    removeScreenshot,
    removePreviewVideo,
    setPrimaryCover,
    setPrimaryBanner,
    handleLogoPositionConfirm,
    resetVideoUploadState,
  } = useEditGameAssets({
    form,
    gameId: currentGameId,
    showCoverSelector,
    showBannerSelector,
    showScreenshotSelector,
    showVideoSelector,
    showLogoSelector,
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
    bannerUploadAction,
    bannerUploadData,
    bannerPreviewUrl,
    bannerSearchUrl,
    backToBannerGameSearch,
    backToCoverGameSearch,
    backToLogoGameSearch,
    backToScreenshotGameSearch,
    backToSummarySearch,
    confirmBannerSelection,
    confirmCoverSelection,
    confirmScreenshotSelection,
    confirmSummaryImport,
    coverPreviewUrl,
    coverSearchUrl,
    downloadSelectedSteamBanner,
    toggleBannerSelection,
    downloadSelectedSteamCover,
    downloadSelectedSteamCovers,
    downloadSelectedSteamLogo,
    downloadSelectedSteamScreenshots,
    confirmLogoSelection,
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
    hasParsableWikiContent,
    handleCoverError,
    handleCoverSearchClear,
    handleCoverUploadError,
    handleCoverUploadSuccess,
    handleLogoUploadSuccess,
    handleLogoUploadError,
    handleLogoSearchClear,
    handleDeveloperSearch,
    handleFilePathItemUpdate,
    handleFileSelect,
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
    handleVideoFileChange,
    handleWikiMetadataCandidateSelectionChange,
    importMetadataFromWiki,
    initialPath,
    isApplyingWikiMetadata,
    isDownloadingBanner,
    isDownloadingCover,
    isDownloadingScreenshot,
    isDownloadingSteamCovers,
    isDownloadingSteamScreenshots,
    isDownloadingSteamLogos,
    isDownloadingLogo,
    isSearchingLogo,
    isPreparingWikiMetadataCandidates,
    isSearchingSeries,
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
    steamLogoImages,
    steamLogoSearchQuery,
    selectedLogoImage,
    selectedSteamLogoGame,
    modalWidth,
    openFileBrowser,
    openVideoSelector,
    previewVideoSources,
    primaryCover,
    primaryLogo,
    logoBannerSrc,
    logoPath,
    primaryPreviewVideo,
    releaseDate,
    removeBanner,
    removeCover,
    removeLogo,
    removeFilePath,
    removePreviewVideo,
    removeScreenshot,
    reorderEditableCovers,
    reorderEditableVideos,
    setPrimaryCover,
    setPrimaryBanner,
    handleLogoPositionConfirm,
    rules,
    screenshotPreviewUrl,
    screenshotSearchUrl,
    screenshotUploadAction,
    screenshotUploadData,
    searchSteamForBanner,
    searchSteamForCover,
    searchSteamForLogo,
    searchSteamForScreenshots,
    searchSteamForSummary,
    selectSteamBannerGame,
    selectSteamCoverGame,
    selectSteamLogoGame,
    selectSteamScreenshotGame,
    selectSteamSummaryGame,
    selectedBanners,
    isDownloadingSteamBanners,
    selectedCoverImage,
    selectedCovers,
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
    showLogoSelector,
    steamBannerImages,
    steamBannerSearchQuery,
    bannerSearchResults,
    steamCoverImages,
    steamCoverSearchQuery,
    coverSearchResults,
    steamScreenshotSearchQuery,
    screenshotSearchResults,
    steamScreenshotsData,
    steamSummaryPreview,
    steamSummarySearchQuery,
    steamSummarySearchResults,
    toggleCoverSelection,
    toggleSteamScreenshot,
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
    sgdbAvailable,
  }
}
