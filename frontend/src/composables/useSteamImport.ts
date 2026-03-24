import { ref, watch, type Ref } from 'vue'
import steamService, { proxySteamAssetUrl } from '@/services/steam.service'
import { useSteamPicker } from '@/composables/useSteamPicker'
import type { SteamGameDetails, SteamGameSearchResult } from '@/services/types'
import { getHttpErrorMessage } from '@/utils/http-error'

type AlertType = 'success' | 'warning' | 'error'
type AssetType = 'cover' | 'banner' | 'screenshot' | 'video'

interface UploadedAssetLike {
  id?: number
  asset_id?: number
  asset_uid?: string
  path: string
  sort_order?: number
}

interface EditableScreenshotLike {
  id?: number
  asset_uid?: string
  path: string
  sort_order?: number
  client_key: string
}

interface SteamImportFormBridge {
  summary: string
  title: string
  title_alt: string
  release_date: string | undefined
  developers: Array<string | number>
  publishers: Array<string | number>
  cover_image: string
  banner_image: string
  screenshots: EditableScreenshotLike[]
}

interface SteamScreenshotsData {
  name: string
  cover: string
  screenshots: string[]
  appId: string
  usedFallbackAssets: boolean
}

interface UseSteamImportOptions {
  form: Ref<SteamImportFormBridge>
  releaseDate: Ref<Date | null>
  gameId: Ref<number | undefined>
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
  ) => EditableScreenshotLike
  addAlert: (message: string, type: AlertType) => void
}

const stripHtmlToText = (html: string) => {
  if (!html.trim()) return ''

  if (typeof window !== 'undefined' && typeof DOMParser !== 'undefined') {
    const doc = new DOMParser().parseFromString(html, 'text/html')
    return (doc.body.textContent || '')
      .replace(/\u00a0/g, ' ')
      .replace(/\s+\n/g, '\n')
      .replace(/\n{3,}/g, '\n\n')
      .trim()
  }

  return html
    .replace(/<br\s*\/?>/gi, '\n')
    .replace(/<\/p>/gi, '\n\n')
    .replace(/<[^>]+>/g, ' ')
    .replace(/&nbsp;/gi, ' ')
    .replace(/\s+/g, ' ')
    .trim()
}

export const useSteamImport = (options: UseSteamImportOptions) => {
  const showSummarySelector = ref(false)
  const steamSummaryPreview = ref('')
  const steamSummaryDetails = ref<SteamGameDetails | null>(null)

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

  const applySteamMetadataToForm = (details: SteamGameDetails) => {
    if (details.releaseDate) {
      options.form.value.release_date = details.releaseDate
      options.releaseDate.value = new Date(`${details.releaseDate}T00:00:00`)
    }
    if (details.developers && details.developers.length > 0) {
      const merged = new Set<string | number>(options.form.value.developers)
      for (const name of details.developers) {
        if (name.trim()) merged.add(name.trim())
      }
      options.form.value.developers = Array.from(merged)
    }
    if (details.publishers && details.publishers.length > 0) {
      const merged = new Set<string | number>(options.form.value.publishers)
      for (const name of details.publishers) {
        if (name.trim()) merged.add(name.trim())
      }
      options.form.value.publishers = Array.from(merged)
    }
  }

  const summarySteamPicker = useSteamPicker<SteamGameDetails>({
    onSelect: async (game) => {
      const details = await steamService.getGameDetails(game.id)
      steamSummaryDetails.value = details
      steamSummaryPreview.value = stripHtmlToText(details.description || '')
      return details
    },
    onError: (message) => {
      options.addAlert('Steam 简介处理失败：' + message, 'error')
    },
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

  const steamSummarySearchQuery = summarySteamPicker.query
  const steamSummarySearchResults = summarySteamPicker.results
  const selectedSteamSummaryGame = summarySteamPicker.selectedGame
  const isSearchingSteamSummary = summarySteamPicker.isSearching

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

  const handleSummarySearchClear = () => {
    summarySteamPicker.clear()
    steamSummaryPreview.value = ''
    steamSummaryDetails.value = null
  }

  const searchSteamForSummary = async () => {
    steamSummaryPreview.value = ''
    steamSummaryDetails.value = null
    await summarySteamPicker.search()
  }

  const selectSteamSummaryGame = async (game: SteamGameSearchResult) => {
    steamSummaryPreview.value = ''
    steamSummaryDetails.value = null
    await summarySteamPicker.select(game)
  }

  const backToSummarySearch = () => {
    summarySteamPicker.back()
    steamSummaryPreview.value = ''
    steamSummaryDetails.value = null
  }

  const confirmSummaryImport = () => {
    const details = steamSummaryDetails.value
    const hasImportableMetadata = !!details?.releaseDate || !!details?.developers?.[0] || !!details?.publishers?.[0]
    if (!steamSummaryPreview.value && !hasImportableMetadata) {
      options.addAlert('当前没有可导入的 Steam 信息', 'warning')
      return
    }

    if (steamSummaryPreview.value) {
      options.form.value.summary = steamSummaryPreview.value
    }
    if (details) {
      applySteamMetadataToForm(details)
    }
    showSummarySelector.value = false
    options.addAlert(
      `已导入 Steam 信息：${selectedSteamSummaryGame.value?.name || 'Steam 游戏'}`,
      'success',
    )
  }

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

  watch(showSummarySelector, (isOpen) => {
    if (!isOpen) return
    const query = pickSteamSearchQuery()
    if (!query) return
    steamSummarySearchQuery.value = query
    searchSteamForSummary()
  })

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

    steamSummarySearchQuery.value = ''
    steamSummarySearchResults.value = []
    selectedSteamSummaryGame.value = null
    steamSummaryPreview.value = ''
    steamSummaryDetails.value = null

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
  }
}
