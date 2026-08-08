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
import { getAssetFileExtension } from '@/utils/asset-file-extension'
import { uploadAsset } from '@/services/assets'
import { useSteamImportMetadata } from '@/composables/useSteamImportMetadata'
import { useSteamImportDownload, type ImportSource } from '@/composables/useSteamImportDownload'
export type { WikiMetadataCandidateSelection } from '@/composables/useSteamImportMetadata'
export type { ImportSource } from '@/composables/useSteamImportDownload'

type AlertType = 'success' | 'warning' | 'error'

interface UploadedAssetLike {
  id?: number
  asset_uid?: string
  path: string
}

interface ScreenshotCandidatesData {
  name: string
  screenshots: string[]
  usedFallbackAssets: boolean
}

interface UseSteamImportOptions {
  form: Ref<Pick<EditGameForm, 'summary' | 'title' | 'title_alt' | 'release_date' | 'developer_ids' | 'publisher_ids' | 'covers' | 'logos' | 'banners' | 'screenshots'>>
  gameId: Ref<number | undefined>
  getWikiContent: () => string
  uploadAssetFromUrl?: (
    url: string,
    assetType: 'cover' | 'banner' | 'screenshot' | 'logo',
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
  ensureDeveloperNames: (names: string[]) => Promise<number[]>
  ensurePublisherNames: (names: string[]) => Promise<number[]>
  addAlert: (message: string, type: AlertType) => void
  onAssetPersisted?: () => Promise<void> | void
}

export const useSteamImport = (options: UseSteamImportOptions) => {
  const rememberedSteamGame = ref<SteamGameSearchResult | null>(null)

  watch(options.gameId, () => {
    rememberedSteamGame.value = null
  })

  const showScreenshotSelector = ref(false)
  const screenshotSearchUrl = ref('')
  const screenshotPreviewUrl = ref('')
  const isDownloadingScreenshot = ref(false)
  const screenshotCandidatesData = ref<ScreenshotCandidatesData | null>(null)
  const selectedRemoteScreenshots = ref<Set<number>>(new Set())
  const isDownloadingRemoteScreenshots = ref(false)
  const screenshotSource = ref<ImportSource>('steam')
  const sgdbScreenshotSearchResults = ref<SteamGameSearchResult[]>([])
  const sgdbScreenshotThumbs = ref<Record<string, string>>({})
  const isSearchingSgdbScreenshots = ref(false)

  const showLogoSelector = ref(false)
  const logoSearchUrl = ref('')
  const logoPreviewUrl = ref('')
  const isDownloadingLogo = ref(false)
  const logoImages = ref<string[]>([])
  const selectedLogos = ref<Set<number>>(new Set())
  const isDownloadingLogos = ref(false)
  const sgdbLogoSearchResults = ref<SteamGameSearchResult[]>([])
  const sgdbLogoThumbs = ref<Record<string, string>>({})
  const isSearchingSgdbLogos = ref(false)

  const pickSteamSearchQuery = () => {
    const preferred = options.form.value.title_alt?.trim()
    if (preferred) return preferred
    return options.form.value.title?.trim() || ''
  }

  const defaultUploadAssetFromUrl = async (
    url: string,
    assetType: 'cover' | 'banner' | 'screenshot' | 'logo',
  ): Promise<UploadedAssetLike> => {
    const gameId = options.gameId.value
    if (!gameId) {
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

    return uploadAsset(assetType, gameId, file)
  }

  const uploadAssetFromUrl = options.uploadAssetFromUrl ?? defaultUploadAssetFromUrl

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
    ensureDeveloperNames: options.ensureDeveloperNames,
    ensurePublisherNames: options.ensurePublisherNames,
    addAlert: options.addAlert,
    rememberedSteamGame,
    onGameSelected: (game) => {
      rememberedSteamGame.value = game
    },
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
    resetDownloadState,
  } = useSteamImportDownload({
    form: options.form,
    gameId: options.gameId,
    pickSteamSearchQuery,
    uploadAssetFromUrl,
    createEditableCover: options.createEditableCover,
    createEditableBanner: options.createEditableBanner,
    addAlert: options.addAlert,
    onAssetPersisted: options.onAssetPersisted,
    rememberedSteamGame,
    onGameSelected: (game) => {
      rememberedSteamGame.value = game
    },
  })

  const screenshotSteamPicker = useSteamPicker<ScreenshotCandidatesData>({
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
        screenshots: finalAssets,
        usedFallbackAssets: screenshotCandidates.length === 0 && finalAssets.length > 0,
      }
      screenshotCandidatesData.value = data
      selectedRemoteScreenshots.value.clear()
      rememberedSteamGame.value = game
      return data
    },
    onError: (message) => {
      options.addAlert('Steam 截图处理失败：' + message, 'error')
    },
  })

  const screenshotSearchQuery = screenshotSteamPicker.query
  const steamScreenshotSearchResults = screenshotSteamPicker.results
  const selectedScreenshotGame = screenshotSteamPicker.selectedGame
  const isSearchingSteamScreenshots = screenshotSteamPicker.isSearching

  const logoSearchQuery = ref('')
  const selectedLogoGame = ref<SteamGameSearchResult | null>(null)

  const mergeSgdbScreenshotThumbs = (results: SteamGameSearchResult[]) => {
    const thumbs = sgdbScreenshotThumbs.value
    return results.map((result) => (
      thumbs[result.id] ? { ...result, tinyImage: thumbs[result.id] } : result
    ))
  }

  const mergeSgdbLogoThumbs = (results: SteamGameSearchResult[]) => {
    const thumbs = sgdbLogoThumbs.value
    return results.map((result) => (
      thumbs[result.id] ? { ...result, tinyImage: thumbs[result.id] } : result
    ))
  }

  const searchSteamGridDBForLogos = async () => {
    const games = await steamGridDBService.search(logoSearchQuery.value)
    const results = games.map((game) => ({
      id: String(game.id),
      name: game.name,
      releaseDate: game.release_date
        ? new Date(game.release_date * 1000).getFullYear().toString()
        : undefined,
    }))
    sgdbLogoSearchResults.value = results
    sgdbLogoThumbs.value = {}

    void Promise.allSettled(
      results.map(async (game) => {
        const logos = await steamGridDBService.getLogosByGameId(Number(game.id))
        const thumbnail = logos[0]?.thumb
        if (thumbnail) {
          sgdbLogoThumbs.value = {
            ...sgdbLogoThumbs.value,
            [game.id]: thumbnail,
          }
        }
      }),
    )
  }

  const searchSteamGridDBForScreenshots = async () => {
    const games = await steamGridDBService.search(screenshotSearchQuery.value)
    const results = games.map((game) => ({
      id: String(game.id),
      name: game.name,
      releaseDate: game.release_date
        ? new Date(game.release_date * 1000).getFullYear().toString()
        : undefined,
    }))
    sgdbScreenshotSearchResults.value = results
    sgdbScreenshotThumbs.value = {}

    void Promise.allSettled(
      results.map(async (game) => {
        const heroes = await steamGridDBService.getHeroesByGameId(Number(game.id))
        const thumbnail = heroes[0]?.thumb
        if (thumbnail) {
          sgdbScreenshotThumbs.value = {
            ...sgdbScreenshotThumbs.value,
            [game.id]: thumbnail,
          }
        }
      }),
    )
  }

  const handleScreenshotSearchClear = () => {
    screenshotSteamPicker.clear()
    screenshotCandidatesData.value = null
    selectedRemoteScreenshots.value.clear()
    sgdbScreenshotSearchResults.value = []
    sgdbScreenshotThumbs.value = {}
  }

  const searchScreenshots = async () => {
    screenshotCandidatesData.value = null
    selectedRemoteScreenshots.value.clear()
    if (screenshotSource.value === 'steamgriddb') {
      isSearchingSgdbScreenshots.value = true
      try {
        await searchSteamGridDBForScreenshots()
        screenshotSteamPicker.clearResults()
        screenshotSteamPicker.back()
      } catch (error) {
        options.addAlert('SteamGridDB 搜索失败：' + getHttpErrorMessage(error), 'error')
      } finally {
        isSearchingSgdbScreenshots.value = false
      }
      return
    }

    sgdbScreenshotSearchResults.value = []
    sgdbScreenshotThumbs.value = {}
    await screenshotSteamPicker.search()
  }

  const selectScreenshotGame = async (game: SteamGameSearchResult) => {
    if (screenshotSource.value === 'steamgriddb') {
      screenshotSteamPicker.setSelectedGame(game)
      isSearchingSgdbScreenshots.value = true
      try {
        const heroes = await steamGridDBService.getHeroesByGameId(Number(game.id))
        screenshotCandidatesData.value = {
          name: game.name,
          screenshots: heroes.map((hero) => hero.url),
          usedFallbackAssets: false,
        }
        selectedRemoteScreenshots.value = new Set()
        rememberedSteamGame.value = game
      } catch (error) {
        options.addAlert('SteamGridDB 获取横幅失败：' + getHttpErrorMessage(error), 'error')
        screenshotSteamPicker.back()
      } finally {
        isSearchingSgdbScreenshots.value = false
      }
      return
    }

    await screenshotSteamPicker.select(game)
  }

  const backToScreenshotGameSearch = () => {
    screenshotSteamPicker.back()
    screenshotCandidatesData.value = null
    selectedRemoteScreenshots.value.clear()
  }

  const toggleScreenshot = (index: number) => {
    if (selectedRemoteScreenshots.value.has(index)) {
      selectedRemoteScreenshots.value.delete(index)
    } else {
      selectedRemoteScreenshots.value.add(index)
    }
  }

  const selectAllScreenshots = () => {
    if (!screenshotCandidatesData.value) return
    selectedRemoteScreenshots.value = new Set(
      screenshotCandidatesData.value.screenshots.map((_, index) => index),
    )
  }

  const invertSelectionScreenshots = () => {
    if (!screenshotCandidatesData.value) return
    const next = new Set<number>()
    for (let index = 0; index < screenshotCandidatesData.value.screenshots.length; index++) {
      if (!selectedRemoteScreenshots.value.has(index)) {
        next.add(index)
      }
    }
    selectedRemoteScreenshots.value = next
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
      const uploaded = await uploadAssetFromUrl(
        screenshotPreviewUrl.value,
        'screenshot',
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

  const downloadSelectedScreenshots = async () => {
    if (!screenshotCandidatesData.value || !options.gameId.value) return

    const indices = Array.from(selectedRemoteScreenshots.value).sort((a, b) => a - b)
    if (indices.length === 0) return

    isDownloadingRemoteScreenshots.value = true
    try {
      for (let i = 0; i < indices.length; i++) {
        const index = indices[i]
        const screenshotUrl = screenshotCandidatesData.value.screenshots[index]
        const currentIndex = options.form.value.screenshots.length
        const uploaded = await uploadAssetFromUrl(screenshotUrl, 'screenshot')
        options.form.value.screenshots.push(options.createEditableScreenshot(uploaded, currentIndex))
      }

      await options.onAssetPersisted?.()
      showScreenshotSelector.value = false
      backToScreenshotGameSearch()
      screenshotSearchQuery.value = ''
      screenshotSteamPicker.clearResults()
      options.addAlert(`成功添加 ${indices.length} 张截图`, 'success')
    } catch (error) {
      options.addAlert('下载失败：' + getHttpErrorMessage(error), 'error')
    } finally {
      isDownloadingRemoteScreenshots.value = false
    }
  }

  const handleLogoSearchClear = () => {
    logoSearchQuery.value = ''
    logoImages.value = []
    selectedLogos.value = new Set()
    selectedLogoGame.value = null
    sgdbLogoSearchResults.value = []
    sgdbLogoThumbs.value = {}
  }

  const searchLogos = async () => {
    logoImages.value = []
    selectedLogos.value = new Set()
    isSearchingSgdbLogos.value = true
    try {
      await searchSteamGridDBForLogos()
      selectedLogoGame.value = null
    } catch (error) {
      options.addAlert('SteamGridDB 搜索失败：' + getHttpErrorMessage(error), 'error')
    } finally {
      isSearchingSgdbLogos.value = false
    }
  }

  const selectLogoGame = async (game: SteamGameSearchResult) => {
    selectedLogoGame.value = game
    isSearchingSgdbLogos.value = true
    try {
      const logos = await steamGridDBService.getLogosByGameId(Number(game.id))
      logoImages.value = logos.map((logo) => logo.url)
      selectedLogos.value = new Set()
      rememberedSteamGame.value = game
    } catch (error) {
      options.addAlert('SteamGridDB 获取 Logo 失败：' + getHttpErrorMessage(error), 'error')
      selectedLogoGame.value = null
    } finally {
      isSearchingSgdbLogos.value = false
    }
  }

  const backToLogoGameSearch = () => {
    selectedLogoGame.value = null
    logoImages.value = []
    selectedLogos.value = new Set()
  }

  const toggleLogoSelection = (index: number) => {
    if (selectedLogos.value.has(index)) {
      selectedLogos.value.delete(index)
    } else {
      selectedLogos.value.add(index)
    }
  }

  const selectAllLogos = () => {
    selectedLogos.value = new Set(logoImages.value.map((_, index) => index))
  }

  const invertSelectionLogos = () => {
    const next = new Set<number>()
    for (let index = 0; index < logoImages.value.length; index++) {
      if (!selectedLogos.value.has(index)) {
        next.add(index)
      }
    }
    selectedLogos.value = next
  }

  const downloadSelectedLogos = async () => {
    if (!logoImages.value.length || !options.gameId.value) return

    const indices = Array.from(selectedLogos.value).sort((a, b) => a - b)
    if (indices.length === 0) return

    isDownloadingLogos.value = true
    try {
      for (const index of indices) {
        const uploaded = await uploadAssetFromUrl(logoImages.value[index], 'logo')
        options.form.value.logos.push(options.createEditableLogo(uploaded))
      }
      await options.onAssetPersisted?.()
      showLogoSelector.value = false
      backToLogoGameSearch()
      logoSearchQuery.value = ''
      sgdbLogoSearchResults.value = []
      options.addAlert(`成功添加 ${indices.length} 张 Logo`, 'success')
    } catch (error) {
      options.addAlert('下载失败：' + getHttpErrorMessage(error), 'error')
    } finally {
      isDownloadingLogos.value = false
    }
  }

  const loadLogoFromUrl = async () => {
    if (!logoSearchUrl.value.trim()) return

    isDownloadingLogo.value = true
    try {
      const uploaded = await uploadAssetFromUrl(logoSearchUrl.value, 'logo')
      options.form.value.logos.push(options.createEditableLogo(uploaded))
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
    const remembered = rememberedSteamGame.value
    if (remembered) {
      void selectScreenshotGame(remembered)
      return
    }
    const query = pickSteamSearchQuery()
    if (!query) return
    screenshotSearchQuery.value = query
    searchScreenshots()
  })

  watch(screenshotSource, () => {
    if (!showScreenshotSelector.value) return
    const query = screenshotSearchQuery.value.trim()
    if (!query) return
    void searchScreenshots()
  })

  watch(showLogoSelector, (isOpen) => {
    if (!isOpen) return
    const remembered = rememberedSteamGame.value
    if (remembered) {
      void selectLogoGame(remembered)
      return
    }
    const query = pickSteamSearchQuery()
    if (!query) return
    logoSearchQuery.value = query
    void searchLogos()
  })

  const screenshotSearchResults = computed(() => (
    screenshotSource.value === 'steamgriddb'
      ? mergeSgdbScreenshotThumbs(sgdbScreenshotSearchResults.value)
      : [...steamScreenshotSearchResults.value]
  ))
  const isSearchingScreenshots = computed(() => (
    screenshotSource.value === 'steamgriddb'
      ? isSearchingSgdbScreenshots.value
      : isSearchingSteamScreenshots.value
  ))
  const logoSearchResults = computed(() => mergeSgdbLogoThumbs(sgdbLogoSearchResults.value))
  const isSearchingLogo = computed(() => isSearchingSgdbLogos.value)

  const resetSteamImportState = () => {
    showSummarySelector.value = false
    showScreenshotSelector.value = false
    showLogoSelector.value = false

    resetMetadataImportState()
    resetDownloadState()

    screenshotSource.value = 'steam'
    screenshotSteamPicker.clear()
    screenshotCandidatesData.value = null
    selectedRemoteScreenshots.value = new Set()
    sgdbScreenshotSearchResults.value = []
    sgdbScreenshotThumbs.value = {}
    screenshotSearchUrl.value = ''
    screenshotPreviewUrl.value = ''

    logoSearchQuery.value = ''
    selectedLogoGame.value = null
    logoImages.value = []
    selectedLogos.value = new Set()
    sgdbLogoSearchResults.value = []
    sgdbLogoThumbs.value = {}
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
  }
}
