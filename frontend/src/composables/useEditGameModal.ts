import { computed, onMounted, onUnmounted, ref, watch, type Ref } from 'vue'
import {
  createEditableBanner,
  createEditableCover,
  createEditableLogo,
  createEditableScreenshot,
  createEditableVideo,
  createEmptyEditGameForm,
  formatEditGameReleaseDate,
  parseEditGameReleaseDate,
  type EditGameForm,
} from '@/utils/edit-game-form'
import { directoryService } from '@/services/directory.service'
import { useGameFilePaths } from '@/composables/useGameFilePaths'
import { useSteamImport } from '@/composables/useSteamImport'
import { useEditGameWorkflow } from '@/composables/useEditGameWorkflow'
import { useEditGameAssets } from '@/composables/useEditGameAssets'
import { useEditGameFormBootstrap } from '@/composables/useEditGameFormBootstrap'
import { useEditGameMediaState } from '@/composables/useEditGameMediaState'
import { useEditGameMetadataPickers } from '@/composables/useEditGameMetadataPickers'
import { useEditGameUploadUrls } from '@/composables/useEditGameUploadUrls'
import { useUiStore } from '@/stores/ui'
import { useGamesStore } from '@/stores/games'
import type { AdminGameDetail } from '@/services/types'

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

  const addAlert = (message: string, type: 'success' | 'warning' | 'error') => {
    uiStore.addAlert(message, type)
  }

  const gamesStore = useGamesStore()

  const {
    CREATE_DEVELOPER_OPTION_VALUE,
    CREATE_PUBLISHER_OPTION_VALUE,
    CREATE_SERIES_OPTION_VALUE,
    seriesPicker,
    developerPicker,
    publisherPicker,
    seriesOptions,
    developerOptions,
    publisherOptions,
    isSearchingSeries,
    isSearchingDevelopers,
    isSearchingPublishers,
    isCreatingSeries,
    isCreatingDevelopers,
    isCreatingPublishers,
    canCreateSeriesOption,
    canCreateDeveloperOption,
    canCreatePublisherOption,
    seriesSearchQuery,
    developerSearchQuery,
    publisherSearchQuery,
    filteredSeriesOptions,
    filteredDeveloperOptions,
    filteredPublisherOptions,
    handleSeriesSearch,
    handleDeveloperSearch,
    handlePublisherSearch,
    handleSeriesEnter,
    handleSeriesSelection,
    handleDeveloperSelection,
    handlePublisherSelection,
  } = useEditGameMetadataPickers({
    form,
    addAlert,
  })

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

  const {
    uploadAction,
    uploadData,
    bannerUploadAction,
    bannerUploadData,
    screenshotUploadAction,
    screenshotUploadData,
    logoUploadAction,
    logoUploadData,
    uploadHeaders,
  } = useEditGameUploadUrls({
    gameId: currentGameId,
  })

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
    invalidateSave,
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
    // 2026-08-08: 保存成功后把最新 GameListItem 原地写入 games 列表，
    // keep-alive 恢复的游戏库无需重拉即可显示新素材状态。
    onGameSaved: (game) => {
      gamesStore.applyAggregateListItem(game)
    },
  })

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
    invalidateSave()
    seriesPicker.reset()
    developerPicker.reset()
    publisherPicker.reset()
    resetFileBrowserState()
    resetSteamImportState()
    resetVideoUploadState()
  }

  watch(() => props.game, async (game, previousGame) => {
    if ((game?.id ?? null) !== (previousGame?.id ?? null)) {
      invalidateSave()
    }
    const initialized = await initializeOptions(game)
    if (!initialized) return
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
      const initialized = await initializeOptions(props.game)
      if (!initialized) return
      hydrateFormFromGame(props.game)
    }
  })

  onUnmounted(invalidateSave)

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
    developerSearchQuery,
    publisherSearchQuery,
    seriesSearchQuery,
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
