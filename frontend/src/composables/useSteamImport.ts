import { computed, ref, watch, type Ref } from 'vue'
import type {
  EditGameEditableBanner,
  EditGameEditableCover,
  EditGameEditableLogo,
  EditGameEditableScreenshot,
  EditGameForm,
} from '@/utils/edit-game-form'
import steamService, { proxySteamAssetUrl } from '@/services/steam.service'
import steamGridDBService from '@/services/steamgriddb.service'
import { useSteamPicker } from '@/composables/useSteamPicker'
import type { SteamGameSearchResult } from '@/services/types'
import { getHttpErrorMessage } from '@/utils/http-error'
import { useSteamImportMetadata } from '@/composables/useSteamImportMetadata'
import { useSteamImportDownload } from '@/composables/useSteamImportDownload'
export type { WikiMetadataCandidateSelection } from '@/composables/useSteamImportMetadata'
export type { ImportSource } from '@/composables/useSteamImportDownload'

type AlertType = 'success' | 'warning' | 'error'

interface UploadedAssetLike {
  id?: number
  asset_uid?: string
  path: string
}

interface SteamScreenshotsData {
  name: string
  cover: string
  screenshots: string[]
  appId: string
  usedFallbackAssets: boolean
}

interface UseSteamImportOptions {
  form: Ref<Pick<EditGameForm, 'summary' | 'title' | 'title_alt' | 'release_date' | 'developer_ids' | 'publisher_ids' | 'covers' | 'logo' | 'banners' | 'screenshots'>>
  gameId: Ref<number | undefined>
  getWikiContent: () => string
  uploadAssetFromUrl: (
    url: string,
    assetType: 'cover' | 'banner' | 'screenshot' | 'logo',
    sortOrder?: number,
  ) => Promise<UploadedAssetLike>
  createEditableCover: (
    asset: UploadedAssetLike | string,
  ) => EditGameEditableCover
  createEditableBanner: (
    asset: UploadedAssetLike | string,
  ) => EditGameEditableBanner
  createEditableLogo: (
    asset: UploadedAssetLike | string,
  ) => EditGameEditableLogo
  createEditableScreenshot: (
    asset: UploadedAssetLike | string,
    index: number,
  ) => EditGameEditableScreenshot
  addAlert: (message: string, type: AlertType) => void
  onAssetPersisted?: () => Promise<void> | void
}

export const useSteamImport = (options: UseSteamImportOptions) => {
  const showScreenshotSelector = ref(false)
  const screenshotSearchUrl = ref('')
  const screenshotPreviewUrl = ref('')
  const isDownloadingScreenshot = ref(false)
  const steamScreenshotsData = ref<SteamScreenshotsData | null>(null)
  const selectedSteamScreenshots = ref<Set<number>>(new Set())
  const isDownloadingSteamScreenshots = ref(false)

  const showLogoSelector = ref(false)
  const logoSearchUrl = ref('')
  const logoPreviewUrl = ref('')
  const isDownloadingLogo = ref(false)
  const steamLogoImages = ref<string[]>([])
  const selectedLogoImage = ref('')
  const isDownloadingSteamLogos = ref(false)

  const pickSteamSearchQuery = () => {
    const preferred = options.form.value.title_alt?.trim()
    if (preferred) return preferred
    return options.form.value.title?.trim() || ''
  }

  const {
    applySelectedWikiMetadata,
    backToSummarySearch,
    confirmSummaryImport,
    handleSummarySearchClear,
    handleWikiMetadataCandidateSelectionChange,
    importMetadataFromWiki,
    isApplyingWikiMetadata,
    isPreparingWikiMetadataCandidates,
    isSearchingSteamSummary,
    resetMetadataImportState,
    searchSteamForSummary,
    selectSteamSummaryGame,
    selectedSteamSummaryGame,
    showSummarySelector,
    steamSummaryPreview,
    steamSummarySearchQuery,
    steamSummarySearchResults,
    wikiMetadataCandidates,
    wikiMetadataPickerVisible,
  } = useSteamImportMetadata({
    form: options.form,
    getWikiContent: options.getWikiContent,
    addAlert: options.addAlert,
  })

  // Cover & Banner download logic (extracted)
  const {
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
    steamCoverSearchQuery,
    coverSearchResults,
    selectedSteamGame,
    isSearchingCover,
    steamBannerSearchQuery,
    bannerSearchResults,
    selectedSteamBannerGame,
    isSearchingBanner,
    coverSource,
    bannerSource,
    sgdbAvailable,
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
    resetDownloadState,
  } = useSteamImportDownload({
    form: options.form,
    gameId: options.gameId,
    pickSteamSearchQuery,
    uploadAssetFromUrl: options.uploadAssetFromUrl,
    createEditableCover: options.createEditableCover,
    createEditableBanner: options.createEditableBanner,
    addAlert: options.addAlert,
    onAssetPersisted: options.onAssetPersisted,
  })

  const screenshotSteamPicker = useSteamPicker<SteamScreenshotsData>({
    onSelect: async (game) => {
      const details = await steamService.getGameDetails(game.id)
      const screenshotCandidates = (details.screenshots || []).filter(Boolean)
      // 2026-04-06: Steam import falls back only to backend-native asset classes.
      // Impact: screenshot fallback now draws from banner/cover instead of frontend-invented aliases.
      const fallbackAssets = [details.bannerImage, details.coverImage].filter(
        (value): value is string => !!value,
      )
      const finalAssets =
        screenshotCandidates.length > 0
          ? screenshotCandidates
          : Array.from(new Set(fallbackAssets))

      const data = {
        name: game.name,
        cover: game.tinyImage || '',
        screenshots: finalAssets,
        appId: game.id,
        usedFallbackAssets: screenshotCandidates.length === 0 && finalAssets.length > 0,
      }
      steamScreenshotsData.value = data
      selectedSteamScreenshots.value.clear()
      return data
    },
    onError: (message) => {
      options.addAlert('Steam 截图处理失败：' + message, 'error')
    },
  })

  const logoSteamPicker = useSteamPicker<string[]>({
    onSelect: async (game) => {
      const logos = await steamGridDBService.getLogosBySteamAppId(Number(game.id))
      const images = logos.map((g: { url: string }) => g.url)
      steamLogoImages.value = images
      selectedLogoImage.value = ''
      return images
    },
    onError: (message) => {
      options.addAlert('Logo 处理失败：' + message, 'error')
    },
  })

  const steamScreenshotSearchQuery = screenshotSteamPicker.query
  const steamScreenshotSearchResults = screenshotSteamPicker.results
  const selectedSteamScreenshotGame = screenshotSteamPicker.selectedGame
  const isSearchingSteamScreenshots = screenshotSteamPicker.isSearching

  const steamLogoSearchQuery = logoSteamPicker.query
  const logoSearchResults = logoSteamPicker.results
  const selectedSteamLogoGame = logoSteamPicker.selectedGame
  const isSearchingLogo = logoSteamPicker.isSearching

  const handleScreenshotSearchClear = () => {
    screenshotSteamPicker.clear()
    steamScreenshotsData.value = null
    selectedSteamScreenshots.value.clear()
  }

  const searchSteamForScreenshots = async () => {
    steamScreenshotsData.value = null
    selectedSteamScreenshots.value.clear()
    await screenshotSteamPicker.search()
  }

  const selectSteamScreenshotGame = async (game: SteamGameSearchResult) => {
    await screenshotSteamPicker.select(game)
  }

  const backToScreenshotGameSearch = () => {
    screenshotSteamPicker.back()
    steamScreenshotsData.value = null
    selectedSteamScreenshots.value.clear()
  }

  const toggleSteamScreenshot = (index: number) => {
    if (selectedSteamScreenshots.value.has(index)) {
      selectedSteamScreenshots.value.delete(index)
    } else {
      selectedSteamScreenshots.value.add(index)
    }
  }

  const loadScreenshotPreview = () => {
    if (screenshotSearchUrl.value.trim()) {
      screenshotPreviewUrl.value = proxySteamAssetUrl(screenshotSearchUrl.value.trim())
    }
  }

  const confirmScreenshotSelection = async () => {
    if (!screenshotPreviewUrl.value) return
    isDownloadingScreenshot.value = true
    try {
      const uploaded = await options.uploadAssetFromUrl(
        screenshotPreviewUrl.value,
        'screenshot',
        options.form.value.screenshots.length,
      )
      options.form.value.screenshots.push(
        options.createEditableScreenshot(uploaded, options.form.value.screenshots.length),
      )
      await options.onAssetPersisted?.()
      showScreenshotSelector.value = false
      screenshotSearchUrl.value = ''
      screenshotPreviewUrl.value = ''
      options.addAlert('截图下载成功', 'success')
    } catch (error) {
      options.addAlert('截图下载失败：' + getHttpErrorMessage(error), 'error')
    } finally {
      isDownloadingScreenshot.value = false
    }
  }

  const downloadSelectedSteamScreenshots = async () => {
    if (!steamScreenshotsData.value || !options.gameId.value) return

    const indices = Array.from(selectedSteamScreenshots.value).sort((a, b) => a - b)
    if (indices.length === 0) return

    isDownloadingSteamScreenshots.value = true
    try {
      for (let i = 0; i < indices.length; i++) {
        const index = indices[i]
        const screenshotUrl = steamScreenshotsData.value.screenshots[index]
        const currentIndex = options.form.value.screenshots.length
        const uploaded = await options.uploadAssetFromUrl(screenshotUrl, 'screenshot', currentIndex)
        options.form.value.screenshots.push(options.createEditableScreenshot(uploaded, currentIndex))
      }

      await options.onAssetPersisted?.()
      showScreenshotSelector.value = false
      backToScreenshotGameSearch()
      steamScreenshotSearchQuery.value = ''
      steamScreenshotSearchResults.value = []
      options.addAlert(`成功添加 ${indices.length} 张截图`, 'success')
    } catch (error) {
      options.addAlert('下载失败：' + getHttpErrorMessage(error), 'error')
    } finally {
      isDownloadingSteamScreenshots.value = false
    }
  }

  const handleLogoSearchClear = () => {
    logoSteamPicker.clear()
    steamLogoImages.value = []
    selectedLogoImage.value = ''
  }

  const searchSteamForLogo = async () => {
    steamLogoImages.value = []
    selectedLogoImage.value = ''
    await logoSteamPicker.search()
  }

  const selectSteamLogoGame = async (game: SteamGameSearchResult) => {
    await logoSteamPicker.select(game)
  }

  const backToLogoGameSearch = () => {
    logoSteamPicker.back()
    steamLogoImages.value = []
    selectedLogoImage.value = ''
  }

  const downloadSelectedSteamLogo = async () => {
    if (!selectedLogoImage.value || !options.gameId.value) return

    isDownloadingSteamLogos.value = true
    try {
      const uploaded = await options.uploadAssetFromUrl(selectedLogoImage.value, 'logo')
      options.form.value.logo = options.createEditableLogo(uploaded)
      await options.onAssetPersisted?.()
      showLogoSelector.value = false
      backToLogoGameSearch()
      steamLogoSearchQuery.value = ''
      logoSearchResults.value = []
      options.addAlert('Logo 下载成功', 'success')
    } catch (error) {
      options.addAlert('下载失败：' + getHttpErrorMessage(error), 'error')
    } finally {
      isDownloadingSteamLogos.value = false
    }
  }

  const loadLogoFromUrl = async () => {
    if (!logoSearchUrl.value.trim()) return

    isDownloadingLogo.value = true
    try {
      const uploaded = await options.uploadAssetFromUrl(logoSearchUrl.value, 'logo')
      options.form.value.logo = options.createEditableLogo(uploaded)
      await options.onAssetPersisted?.()
      showLogoSelector.value = false
      logoSearchUrl.value = ''
      logoPreviewUrl.value = ''
      options.addAlert('Logo 下载成功', 'success')
    } catch (error) {
      options.addAlert('Logo 下载失败：' + getHttpErrorMessage(error), 'error')
    } finally {
      isDownloadingLogo.value = false
    }
  }

  const confirmLogoSelection = async () => {
    if (logoSearchUrl.value) {
      await loadLogoFromUrl()
    }
  }

  watch(showScreenshotSelector, (isOpen) => {
    if (!isOpen) return
    const query = pickSteamSearchQuery()
    if (!query) return
    steamScreenshotSearchQuery.value = query
    searchSteamForScreenshots()
  })

  watch(showLogoSelector, (isOpen) => {
    if (!isOpen) return
    const query = pickSteamSearchQuery()
    if (!query) return
    steamLogoSearchQuery.value = query
    searchSteamForLogo()
  })

  const screenshotSearchResults = computed(() => steamScreenshotSearchResults.value)
  const isSearchingScreenshots = computed(() => isSearchingSteamScreenshots.value)

  const resetSteamImportState = () => {
    showSummarySelector.value = false
    showScreenshotSelector.value = false
    showLogoSelector.value = false

    resetMetadataImportState()
    resetDownloadState()

    steamScreenshotSearchQuery.value = ''
    steamScreenshotSearchResults.value = []
    selectedSteamScreenshotGame.value = null
    steamScreenshotsData.value = null
    selectedSteamScreenshots.value = new Set()
    screenshotSearchUrl.value = ''
    screenshotPreviewUrl.value = ''

    steamLogoSearchQuery.value = ''
    logoSearchResults.value = []
    selectedSteamLogoGame.value = null
    steamLogoImages.value = []
    selectedLogoImage.value = ''
    logoSearchUrl.value = ''
    logoPreviewUrl.value = ''
  }

  return {
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
  }
}
