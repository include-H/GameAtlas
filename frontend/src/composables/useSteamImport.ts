import { computed, ref, watch, type Ref } from 'vue'
import type {
  EditGameEditableBanner,
  EditGameEditableCover,
  EditGameEditableLogo,
  EditGameEditableScreenshot,
  EditGameForm,
} from '@/composables/edit-game-form'
import steamService, { proxySteamAssetUrl } from '@/services/steam.service'
import steamGridDBService from '@/services/steamgriddb.service'
import { useSteamPicker } from '@/composables/useSteamPicker'
import type { SteamGameSearchResult } from '@/services/types'
import { getHttpErrorMessage } from '@/utils/http-error'
import { useSteamImportMetadata } from '@/composables/useSteamImportMetadata'
export type { WikiMetadataCandidateSelection } from '@/composables/useSteamImportMetadata'

export type ImportSource = 'steam' | 'steamgriddb'

type AlertType = 'success' | 'warning' | 'error'
type AssetType = 'cover' | 'banner' | 'screenshot' | 'video' | 'logo'

interface UploadedAssetLike {
  id?: number
  asset_id?: number
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
  form: Ref<Pick<EditGameForm, 'summary' | 'title' | 'title_alt' | 'release_date' | 'developer_ids' | 'publisher_ids' | 'covers' | 'logo' | 'banner_image' | 'banners' | 'screenshots'>>
  gameId: Ref<number | undefined>
  getWikiContent: () => string
  uploadAssetFromUrl: (
    url: string,
    assetType: 'cover' | 'banner' | 'screenshot' | 'logo',
    sortOrder?: number,
  ) => Promise<UploadedAssetLike>
  queueAssetDeletion: (
    type: AssetType,
    path: string,
    assetId?: number,
    assetUid?: string,
  ) => void
  deleteAsset?: (type: AssetType, gameId: number, assetUid: string) => Promise<void>
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
  const showCoverSelector = ref(false)
  const coverSearchUrl = ref('')
  const coverPreviewUrl = ref('')
  const isDownloadingCover = ref(false)
  const steamCoverImages = ref<string[]>([])
  const selectedCoverImage = ref('')
  const selectedCovers = ref<Set<number>>(new Set())
  const isDownloadingSteamCovers = ref(false)

  const showBannerSelector = ref(false)
  const bannerSearchUrl = ref('')
  const bannerPreviewUrl = ref('')
  const isDownloadingBanner = ref(false)
  const steamBannerImages = ref<string[]>([])
  const selectedBanners = ref<Set<number>>(new Set())
  const isDownloadingSteamBanners = ref(false)

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

  // Data source toggle: 'steam' | 'steamgriddb'
  const coverSource = ref<ImportSource>('steam')
  const bannerSource = ref<ImportSource>('steam')

  const sgdbAvailable = ref(false)
  steamGridDBService.isAvailable().then((v) => { sgdbAvailable.value = v })

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
      const details = await steamService.getGameDetails(game.id)
      const images = [details.coverImage, details.bannerImage].filter(
        (value): value is string => !!value,
      )
      steamCoverImages.value = images
      selectedCoverImage.value = ''
      selectedCovers.value = new Set()
      return images
    },
    onError: (message) => {
      options.addAlert('Steam 封面处理失败：' + message, 'error')
    },
  })

  const bannerSteamPicker = useSteamPicker<string[]>({
    onSelect: async (game) => {
      const details = await steamService.getGameDetails(game.id)
      const images = Array.from(new Set([details.bannerImage, details.coverImage].filter(Boolean) as string[]))
      steamBannerImages.value = images
      selectedBanners.value = new Set()
      return images
    },
    onError: (message) => {
      options.addAlert('Steam 横幅处理失败：' + message, 'error')
    },
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
      const logos = await steamGridDBService.getLogosBySteamAppId(game.id)
      const images = logos.map((g) => g.url)
      steamLogoImages.value = images
      selectedLogoImage.value = ''
      return images
    },
    onError: (message) => {
      options.addAlert('Logo 处理失败：' + message, 'error')
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

  const steamLogoSearchQuery = logoSteamPicker.query
  const logoSearchResults = logoSteamPicker.results
  const selectedSteamLogoGame = logoSteamPicker.selectedGame
  const isSearchingLogo = logoSteamPicker.isSearching

  // SteamGridDB search helpers — maps results to SteamGameSearchResult shape for UI reuse
  const sgdbSearchResults = ref<SteamGameSearchResult[]>([])
  const sgdbSearching = ref(false)
  const sgdbThumbs = ref<Record<string, string>>({})

  const searchSGDB = async (query: string): Promise<SteamGameSearchResult[]> => {
    const games = await steamGridDBService.search(query)
    const results: SteamGameSearchResult[] = games.map((g) => ({
      id: String(g.id),
      name: g.name,
      releaseDate: g.release_date ? new Date(g.release_date * 1000).getFullYear().toString() : undefined,
    }))
    // SGDB search API doesn't return thumbnails — fetch the first grid for each result in background
    sgdbThumbs.value = {}
    void Promise.allSettled(
      results.map(async (r) => {
        try {
          const grids = await steamGridDBService.getGridsByGameId(Number(r.id))
          if (grids.length > 0) {
            sgdbThumbs.value = { ...sgdbThumbs.value, [r.id]: grids[0].thumb }
          }
        } catch {
          // thumbnail is cosmetic — ignore failures
        }
      }),
    )
    return results
  }

  const handleCoverSearchClear = () => {
    coverSteamPicker.clear()
    sgdbSearchResults.value = []
    sgdbThumbs.value = {}
    steamCoverImages.value = []
    selectedCoverImage.value = ''
    selectedCovers.value = new Set()
  }

  const searchSteamForCover = async () => {
    steamCoverImages.value = []
    selectedCoverImage.value = ''
    selectedCovers.value = new Set()
    if (coverSource.value === 'steamgriddb') {
      sgdbSearching.value = true
      try {
        sgdbSearchResults.value = await searchSGDB(coverSteamPicker.query.value)
        coverSteamPicker.results.value = []
        coverSteamPicker.selectedGame.value = null
      } catch (e) {
        options.addAlert('SteamGridDB 搜索失败：' + getHttpErrorMessage(e), 'error')
      } finally {
        sgdbSearching.value = false
      }
    } else {
      sgdbSearchResults.value = []
      await coverSteamPicker.search()
    }
  }

  const selectSteamCoverGame = async (game: SteamGameSearchResult) => {
    if (coverSource.value === 'steamgriddb') {
      coverSteamPicker.selectedGame.value = game
      coverSteamPicker.isSearching.value = true
      try {
        const grids = await steamGridDBService.getGridsByGameId(Number(game.id))
        const images = grids.map((g) => g.url)
        steamCoverImages.value = images
        selectedCoverImage.value = ''
        selectedCovers.value = new Set()
      } catch (e) {
        options.addAlert('SteamGridDB 获取封面失败：' + getHttpErrorMessage(e), 'error')
        coverSteamPicker.selectedGame.value = null
      } finally {
        coverSteamPicker.isSearching.value = false
      }
    } else {
      await coverSteamPicker.select(game)
    }
  }

  const backToCoverGameSearch = () => {
    coverSteamPicker.back()
    steamCoverImages.value = []
    selectedCoverImage.value = ''
    selectedCovers.value = new Set()
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
      options.form.value.covers.push(options.createEditableCover(uploaded))
      await options.onAssetPersisted?.()
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
      options.form.value.covers.push(options.createEditableCover(uploaded))
      await options.onAssetPersisted?.()
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

  const downloadSelectedSteamCovers = async () => {
    if (!steamCoverImages.value.length || !options.gameId.value) return

    const indices = Array.from(selectedCovers.value).sort((a, b) => a - b)
    if (indices.length === 0) return

    isDownloadingSteamCovers.value = true
    try {
      for (const index of indices) {
        const coverUrl = steamCoverImages.value[index]
        const uploaded = await options.uploadAssetFromUrl(coverUrl, 'cover')
        options.form.value.covers.push(options.createEditableCover(uploaded))
      }
      await options.onAssetPersisted?.()
      showCoverSelector.value = false
      backToCoverGameSearch()
      steamCoverSearchQuery.value = ''
      steamCoverSearchResults.value = []
      options.addAlert(`成功添加 ${indices.length} 张封面`, 'success')
    } catch (error) {
      options.addAlert('下载失败：' + getHttpErrorMessage(error), 'error')
    } finally {
      isDownloadingSteamCovers.value = false
    }
  }

  const handleBannerSearchClear = () => {
    bannerSteamPicker.clear()
    sgdbSearchResults.value = []
    sgdbThumbs.value = {}
    steamBannerImages.value = []
    selectedBanners.value = new Set()
  }

  const searchSteamForBanner = async () => {
    steamBannerImages.value = []
    selectedBanners.value = new Set()
    if (bannerSource.value === 'steamgriddb') {
      sgdbSearching.value = true
      try {
        sgdbSearchResults.value = await searchSGDB(bannerSteamPicker.query.value)
        bannerSteamPicker.results.value = []
        bannerSteamPicker.selectedGame.value = null
      } catch (e) {
        options.addAlert('SteamGridDB 搜索失败：' + getHttpErrorMessage(e), 'error')
      } finally {
        sgdbSearching.value = false
      }
    } else {
      sgdbSearchResults.value = []
      await bannerSteamPicker.search()
    }
  }

  const selectSteamBannerGame = async (game: SteamGameSearchResult) => {
    if (bannerSource.value === 'steamgriddb') {
      bannerSteamPicker.selectedGame.value = game
      bannerSteamPicker.isSearching.value = true
      try {
        const heroes = await steamGridDBService.getHeroesByGameId(Number(game.id))
        const images = heroes.map((g) => g.url)
        steamBannerImages.value = images
        selectedBanners.value = new Set()
      } catch (e) {
        options.addAlert('SteamGridDB 获取横幅失败：' + getHttpErrorMessage(e), 'error')
        bannerSteamPicker.selectedGame.value = null
      } finally {
        bannerSteamPicker.isSearching.value = false
      }
    } else {
      await bannerSteamPicker.select(game)
    }
  }

  const backToBannerGameSearch = () => {
    bannerSteamPicker.back()
    steamBannerImages.value = []
    selectedBanners.value = new Set()
  }

  const loadBannerFromUrl = async () => {
    if (!bannerSearchUrl.value.trim()) return

    isDownloadingBanner.value = true
    try {
      const uploaded = await options.uploadAssetFromUrl(bannerSearchUrl.value, 'banner')
      options.form.value.banners.push(options.createEditableBanner(uploaded))
      options.form.value.banner_image = options.form.value.banners[0]?.path || ''
      await options.onAssetPersisted?.()
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
    if (!selectedBanners.value.size || !options.gameId.value) return

    const indices = Array.from(selectedBanners.value).sort((a, b) => a - b)
    if (indices.length === 0) return

    isDownloadingSteamBanners.value = true
    try {
      for (const index of indices) {
        const bannerUrl = steamBannerImages.value[index]
        const uploaded = await options.uploadAssetFromUrl(bannerUrl, 'banner')
        options.form.value.banners.push(options.createEditableBanner(uploaded))
      }
      options.form.value.banner_image = options.form.value.banners[0]?.path || ''
      await options.onAssetPersisted?.()
      showBannerSelector.value = false
      backToBannerGameSearch()
      steamBannerSearchQuery.value = ''
      steamBannerSearchResults.value = []
      bannerSearchUrl.value = ''
      bannerPreviewUrl.value = ''
      options.addAlert(`成功添加 ${indices.length} 张横幅`, 'success')
    } catch (error) {
      options.addAlert('下载失败：' + getHttpErrorMessage(error), 'error')
    } finally {
      isDownloadingSteamBanners.value = false
    }
  }

  const toggleBannerSelection = (index: number) => {
    if (selectedBanners.value.has(index)) {
      selectedBanners.value.delete(index)
    } else {
      selectedBanners.value.add(index)
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

  const toggleCoverSelection = (index: number) => {
    if (selectedCovers.value.has(index)) {
      selectedCovers.value.delete(index)
    } else {
      selectedCovers.value.add(index)
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
      const oldLogo = options.form.value.logo
      if (oldLogo?.asset_uid && options.deleteAsset) {
        void options.deleteAsset('logo', options.gameId.value, oldLogo.asset_uid)
      }
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
      const oldLogo = options.form.value.logo
      if (oldLogo?.asset_uid && options.deleteAsset) {
        void options.deleteAsset('logo', options.gameId.value!, oldLogo.asset_uid)
      }
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

  watch(showLogoSelector, (isOpen) => {
    if (!isOpen) return
    const query = pickSteamSearchQuery()
    if (!query) return
    steamLogoSearchQuery.value = query
    searchSteamForLogo()
  })

  watch(coverSource, () => {
    if (!showCoverSelector.value) return
    const query = steamCoverSearchQuery.value.trim()
    if (!query) return
    searchSteamForCover()
  })

  watch(bannerSource, () => {
    if (!showBannerSelector.value) return
    const query = steamBannerSearchQuery.value.trim()
    if (!query) return
    searchSteamForBanner()
  })

  // Merge SGDB thumbnails into search results reactively
  const mergeSGDBThumbs = (results: SteamGameSearchResult[]): SteamGameSearchResult[] => {
    const thumbs = sgdbThumbs.value
    return results.map((r) => (thumbs[r.id] ? { ...r, tinyImage: thumbs[r.id] } : r))
  }

  // Computed search results/loading — switches between Steam and SteamGridDB based on source
  const coverSearchResults = computed(() =>
    coverSource.value === 'steamgriddb'
      ? mergeSGDBThumbs(sgdbSearchResults.value)
      : steamCoverSearchResults.value,
  )
  const isSearchingCover = computed(() =>
    coverSource.value === 'steamgriddb' ? sgdbSearching.value : isSearchingSteamCover.value,
  )
  const bannerSearchResults = computed(() =>
    bannerSource.value === 'steamgriddb'
      ? mergeSGDBThumbs(sgdbSearchResults.value)
      : steamBannerSearchResults.value,
  )
  const isSearchingBanner = computed(() =>
    bannerSource.value === 'steamgriddb' ? sgdbSearching.value : isSearchingSteamBanner.value,
  )
  const screenshotSearchResults = computed(() => steamScreenshotSearchResults.value)
  const isSearchingScreenshots = computed(() => isSearchingSteamScreenshots.value)

  const resetSteamImportState = () => {
    showSummarySelector.value = false
    showCoverSelector.value = false
    showBannerSelector.value = false
    showScreenshotSelector.value = false
    showLogoSelector.value = false

    resetMetadataImportState()

    sgdbSearchResults.value = []
    sgdbThumbs.value = {}

    steamCoverSearchQuery.value = ''
    steamCoverSearchResults.value = []
    selectedSteamGame.value = null
    steamCoverImages.value = []
    selectedCoverImage.value = ''
    selectedCovers.value = new Set()
    coverSearchUrl.value = ''
    coverPreviewUrl.value = ''

    steamBannerSearchQuery.value = ''
    steamBannerSearchResults.value = []
    selectedSteamBannerGame.value = null
    steamBannerImages.value = []
    selectedBanners.value = new Set()
    bannerSearchUrl.value = ''
    bannerPreviewUrl.value = ''

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
