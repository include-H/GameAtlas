import { ref, watch, type Ref } from 'vue'
import type {
  EditGameEditableScreenshot,
  EditGameForm,
} from '@/composables/edit-game-form'
import steamService, { proxySteamAssetUrl } from '@/services/steam.service'
import { useSteamPicker } from '@/composables/useSteamPicker'
import type { SteamGameSearchResult } from '@/services/types'
import { getHttpErrorMessage } from '@/utils/http-error'
import { useSteamImportMetadata } from '@/composables/useSteamImportMetadata'
export type { WikiMetadataCandidateSelection } from '@/composables/useSteamImportMetadata'

type AlertType = 'success' | 'warning' | 'error'
type AssetType = 'cover' | 'banner' | 'screenshot' | 'video'

interface UploadedAssetLike {
  id?: number
  asset_id?: number
  asset_uid?: string
  path: string
  sort_order?: number
}

interface SteamScreenshotsData {
  name: string
  cover: string
  screenshots: string[]
  appId: string
  usedFallbackAssets: boolean
}

interface UseSteamImportOptions {
  form: Ref<Pick<EditGameForm, 'summary' | 'title' | 'title_alt' | 'release_date' | 'engine' | 'developer_ids' | 'publisher_ids' | 'platform_ids' | 'cover_image' | 'banner_image' | 'screenshots'>>
  gameId: Ref<number | undefined>
  getWikiContent: () => string
  uploadAssetFromUrl: (
    url: string,
    assetType: 'cover' | 'banner' | 'screenshot',
    sortOrder?: number,
  ) => Promise<UploadedAssetLike>
  queueAssetDeletion: (
    type: AssetType,
    path: string,
    assetId?: number,
    assetUid?: string,
  ) => void
  createEditableScreenshot: (
    asset: UploadedAssetLike | string,
    index: number,
  ) => EditGameEditableScreenshot
  addAlert: (message: string, type: AlertType) => void
}

export const useSteamImport = (options: UseSteamImportOptions) => {
  const showCoverSelector = ref(false)
  const coverSearchUrl = ref('')
  const coverPreviewUrl = ref('')
  const isDownloadingCover = ref(false)
  const steamCoverImages = ref<string[]>([])
  const selectedCoverImage = ref('')

  const showBannerSelector = ref(false)
  const bannerSearchUrl = ref('')
  const bannerPreviewUrl = ref('')
  const isDownloadingBanner = ref(false)
  const steamBannerImages = ref<string[]>([])
  const selectedBannerImage = ref('')

  const showScreenshotSelector = ref(false)
  const screenshotSearchUrl = ref('')
  const screenshotPreviewUrl = ref('')
  const isDownloadingScreenshot = ref(false)
  const steamScreenshotsData = ref<SteamScreenshotsData | null>(null)
  const selectedSteamScreenshots = ref<Set<number>>(new Set())
  const isDownloadingSteamScreenshots = ref(false)

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

  const coverSteamPicker = useSteamPicker<string[]>({
    onSelect: async (game) => {
      const coverUrl = proxySteamAssetUrl(`https://steamcdn-a.akamaihd.net/steam/apps/${game.id}/library_600x900_2x.jpg`)
      steamCoverImages.value = [coverUrl]
      selectedCoverImage.value = ''
      return [coverUrl]
    },
    onError: (message) => {
      options.addAlert('Steam 封面处理失败：' + message, 'error')
    },
  })

  const bannerSteamPicker = useSteamPicker<string[]>({
    onSelect: async (game) => {
      const details = await steamService.getGameDetails(game.id)
      const libraryHero = details.libraryHero
      const background = details.background
      const headerImage = details.headerImage
      const images = Array.from(new Set([libraryHero, background, headerImage].filter(Boolean) as string[]))
      const finalImages = images.length < 2 && details.screenshots && details.screenshots.length > 0
        ? [...images, ...details.screenshots.slice(0, 5)]
        : images
      steamBannerImages.value = finalImages
      selectedBannerImage.value = ''
      return finalImages
    },
    onError: (message) => {
      options.addAlert('Steam 横幅处理失败：' + message, 'error')
    },
  })

  const screenshotSteamPicker = useSteamPicker<SteamScreenshotsData>({
    onSelect: async (game) => {
      const details = await steamService.getGameDetails(game.id)
      const screenshotCandidates = (details.screenshots || []).filter(Boolean)
      const fallbackAssets = [details.libraryHero, details.background, details.headerImage].filter(
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

  const steamCoverSearchQuery = coverSteamPicker.query
  const steamCoverSearchResults = coverSteamPicker.results
  const selectedSteamGame = coverSteamPicker.selectedGame
  const isSearchingSteamCover = coverSteamPicker.isSearching

  const steamBannerSearchQuery = bannerSteamPicker.query
  const steamBannerSearchResults = bannerSteamPicker.results
  const selectedSteamBannerGame = bannerSteamPicker.selectedGame
  const isSearchingSteamBanner = bannerSteamPicker.isSearching

  const steamScreenshotSearchQuery = screenshotSteamPicker.query
  const steamScreenshotSearchResults = screenshotSteamPicker.results
  const selectedSteamScreenshotGame = screenshotSteamPicker.selectedGame
  const isSearchingSteamScreenshots = screenshotSteamPicker.isSearching

  const handleCoverSearchClear = () => {
    coverSteamPicker.clear()
    steamCoverImages.value = []
    selectedCoverImage.value = ''
  }

  const searchSteamForCover = async () => {
    steamCoverImages.value = []
    selectedCoverImage.value = ''
    await coverSteamPicker.search()
  }

  const selectSteamCoverGame = async (game: SteamGameSearchResult) => {
    await coverSteamPicker.select(game)
  }

  const backToCoverGameSearch = () => {
    coverSteamPicker.back()
    steamCoverImages.value = []
    selectedCoverImage.value = ''
  }

  const loadCoverFromUrl = () => {
    if (coverSearchUrl.value.trim()) {
      coverPreviewUrl.value = proxySteamAssetUrl(coverSearchUrl.value.trim())
    }
  }

  const confirmCoverSelection = async () => {
    if (!coverPreviewUrl.value) return
    isDownloadingCover.value = true
    try {
      const uploaded = await options.uploadAssetFromUrl(coverPreviewUrl.value, 'cover')
      if (options.form.value.cover_image) {
        options.queueAssetDeletion('cover', options.form.value.cover_image)
      }
      options.form.value.cover_image = uploaded.path
      showCoverSelector.value = false
      coverSearchUrl.value = ''
      coverPreviewUrl.value = ''
      options.addAlert('封面下载成功', 'success')
    } catch (error) {
      options.addAlert('封面下载失败：' + getHttpErrorMessage(error), 'error')
    } finally {
      isDownloadingCover.value = false
    }
  }

  const downloadSelectedSteamCover = async () => {
    if (!selectedCoverImage.value || !options.gameId.value) return

    isSearchingSteamCover.value = true
    try {
      const uploaded = await options.uploadAssetFromUrl(selectedCoverImage.value, 'cover')
      if (options.form.value.cover_image) {
        options.queueAssetDeletion('cover', options.form.value.cover_image)
      }
      options.form.value.cover_image = uploaded.path
      showCoverSelector.value = false
      backToCoverGameSearch()
      steamCoverSearchQuery.value = ''
      steamCoverSearchResults.value = []
      options.addAlert('封面下载成功', 'success')
    } catch (error) {
      options.addAlert('下载失败：' + getHttpErrorMessage(error), 'error')
    } finally {
      isSearchingSteamCover.value = false
    }
  }

  const handleBannerSearchClear = () => {
    bannerSteamPicker.clear()
    steamBannerImages.value = []
    selectedBannerImage.value = ''
  }

  const searchSteamForBanner = async () => {
    steamBannerImages.value = []
    selectedBannerImage.value = ''
    await bannerSteamPicker.search()
  }

  const selectSteamBannerGame = async (game: SteamGameSearchResult) => {
    await bannerSteamPicker.select(game)
  }

  const backToBannerGameSearch = () => {
    bannerSteamPicker.back()
    steamBannerImages.value = []
  }

  const loadBannerFromUrl = async () => {
    if (!bannerSearchUrl.value.trim()) return

    isDownloadingBanner.value = true
    try {
      const uploaded = await options.uploadAssetFromUrl(bannerSearchUrl.value, 'banner')
      if (options.form.value.banner_image) {
        options.queueAssetDeletion('banner', options.form.value.banner_image)
      }
      options.form.value.banner_image = uploaded.path
      showBannerSelector.value = false
      bannerSearchUrl.value = ''
      bannerPreviewUrl.value = ''
      options.addAlert('横幅下载成功', 'success')
    } catch (error) {
      options.addAlert('下载失败：' + getHttpErrorMessage(error), 'error')
    } finally {
      isDownloadingBanner.value = false
    }
  }

  const confirmBannerSelection = async () => {
    if (bannerSearchUrl.value) {
      await loadBannerFromUrl()
    }
  }

  const downloadSelectedSteamBanner = async () => {
    if (!selectedBannerImage.value || !options.gameId.value) return

    isDownloadingBanner.value = true
    try {
      const uploaded = await options.uploadAssetFromUrl(selectedBannerImage.value, 'banner')
      if (options.form.value.banner_image) {
        options.queueAssetDeletion('banner', options.form.value.banner_image)
      }
      options.form.value.banner_image = uploaded.path
      showBannerSelector.value = false
      backToBannerGameSearch()
      steamBannerSearchQuery.value = ''
      steamBannerSearchResults.value = []
      bannerSearchUrl.value = ''
      bannerPreviewUrl.value = ''
      options.addAlert('横幅下载成功', 'success')
    } catch (error) {
      options.addAlert('下载失败：' + getHttpErrorMessage(error), 'error')
    } finally {
      isDownloadingBanner.value = false
    }
  }

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

  watch(showCoverSelector, (isOpen) => {
    if (!isOpen) return
    const query = pickSteamSearchQuery()
    if (!query) return
    steamCoverSearchQuery.value = query
    searchSteamForCover()
  })

  watch(showBannerSelector, (isOpen) => {
    if (!isOpen) return
    const query = pickSteamSearchQuery()
    if (!query) return
    steamBannerSearchQuery.value = query
    searchSteamForBanner()
  })

  watch(showScreenshotSelector, (isOpen) => {
    if (!isOpen) return
    const query = pickSteamSearchQuery()
    if (!query) return
    steamScreenshotSearchQuery.value = query
    searchSteamForScreenshots()
  })

  const resetSteamImportState = () => {
    showSummarySelector.value = false
    showCoverSelector.value = false
    showBannerSelector.value = false
    showScreenshotSelector.value = false

    resetMetadataImportState()

    steamCoverSearchQuery.value = ''
    steamCoverSearchResults.value = []
    selectedSteamGame.value = null
    steamCoverImages.value = []
    selectedCoverImage.value = ''
    coverSearchUrl.value = ''
    coverPreviewUrl.value = ''

    steamBannerSearchQuery.value = ''
    steamBannerSearchResults.value = []
    selectedSteamBannerGame.value = null
    steamBannerImages.value = []
    selectedBannerImage.value = ''
    bannerSearchUrl.value = ''
    bannerPreviewUrl.value = ''

    steamScreenshotSearchQuery.value = ''
    steamScreenshotSearchResults.value = []
    selectedSteamScreenshotGame.value = null
    steamScreenshotsData.value = null
    selectedSteamScreenshots.value = new Set()
    screenshotSearchUrl.value = ''
    screenshotPreviewUrl.value = ''
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
  }
}
