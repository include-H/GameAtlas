import { computed, reactive, ref, watch, type Ref } from 'vue'
import type {
  EditGameEditableBanner,
  EditGameEditableCover,
  EditGameEditableLogo,
  EditGameEditableScreenshot,
  EditGameForm,
} from '@/utils/edit-game-form'
import steamService, { proxySteamAssetUrl } from '@/services/steam.service'
import steamGridDBService from '@/services/steamgriddb.service'
import { useSteamPicker, type SteamPickerRequest } from '@/composables/useSteamPicker'
import type { SteamGameSearchResult } from '@/services/types'
import { getHttpErrorMessage } from '@/utils/http-error'
import { getAssetFileExtension } from '@/utils/asset-file-extension'
import { uploadAsset } from '@/services/assets'
import { useSteamImportMetadata } from '@/composables/useSteamImportMetadata'
import { useSteamImportDownload, type ImportSource } from '@/composables/useSteamImportDownload'
import { createRequestGeneration } from '@/utils/request-generation'
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

// 模块级共享 + 按 gameId 隔离：编辑弹窗与媒体页各自实例化
// useSteamImport，组件级 ref 会导致摘要选中后素材侧无法复用。
// 必须用 reactive Map：普通 Map 的 set 不触发响应，computed 会缓存旧值。
// Steam 与 SGDB 各自维护记忆——Steam AppID 与 SGDB game id 不是同一套
// 编号体系，SGDB 源首次必须走自己的搜索，不能硬复用 AppID。
const steamGameMemoryByGame = reactive(new Map<number, SteamGameSearchResult>())
const steamGameMemoryForNewGame = ref<SteamGameSearchResult | null>(null)
const sgdbGameMemoryByGame = reactive(new Map<number, SteamGameSearchResult>())
const sgdbGameMemoryForNewGame = ref<SteamGameSearchResult | null>(null)

export const resetSteamGameMemory = () => {
  steamGameMemoryByGame.clear()
  steamGameMemoryForNewGame.value = null
  sgdbGameMemoryByGame.clear()
  sgdbGameMemoryForNewGame.value = null
}

export const useSteamImport = (options: UseSteamImportOptions) => {
  const rememberedSteamGame = computed<SteamGameSearchResult | null>(() => {
    const gameId = options.gameId.value
    if (gameId == null) return steamGameMemoryForNewGame.value
    return steamGameMemoryByGame.get(gameId) ?? null
  })

  const rememberedSgdbGame = computed<SteamGameSearchResult | null>(() => {
    const gameId = options.gameId.value
    if (gameId == null) return sgdbGameMemoryForNewGame.value
    return sgdbGameMemoryByGame.get(gameId) ?? null
  })

  const rememberSteamGame = (game: SteamGameSearchResult) => {
    const gameId = options.gameId.value
    if (gameId == null) {
      steamGameMemoryForNewGame.value = game
    } else {
      steamGameMemoryByGame.set(gameId, game)
    }
  }

  const rememberSgdbGame = (game: SteamGameSearchResult) => {
    const gameId = options.gameId.value
    if (gameId == null) {
      sgdbGameMemoryForNewGame.value = game
    } else {
      sgdbGameMemoryByGame.set(gameId, game)
    }
  }

  const forgetSteamGame = () => {
    const gameId = options.gameId.value
    if (gameId == null) {
      steamGameMemoryForNewGame.value = null
    } else {
      steamGameMemoryByGame.delete(gameId)
    }
  }

  const forgetSgdbGame = () => {
    const gameId = options.gameId.value
    if (gameId == null) {
      sgdbGameMemoryForNewGame.value = null
    } else {
      sgdbGameMemoryByGame.delete(gameId)
    }
  }

  const forgetGameForSource = (source: ImportSource) => {
    if (source === 'steamgriddb') {
      forgetSgdbGame()
    } else {
      forgetSteamGame()
    }
  }

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
  const screenshotRequests = createRequestGeneration()
  const logoRequests = createRequestGeneration()

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
    onGameSelected: rememberSteamGame,
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
    rememberedSgdbGame,
    onSteamGameSelected: rememberSteamGame,
    onSgdbGameSelected: rememberSgdbGame,
    forgetGame: forgetGameForSource,
  })

  const screenshotSteamPicker = useSteamPicker<ScreenshotCandidatesData>({
    onSelect: async (game, request) => {
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
      if (!request.isCurrent()) return data
      screenshotCandidatesData.value = data
      selectedRemoteScreenshots.value.clear()
      rememberSteamGame(game)
      return data
    },    onError: (message) => {
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

  const mapSgdbSearchResults = (games: Awaited<ReturnType<typeof steamGridDBService.search>>) => games.map((game) => ({
      id: String(game.id),
      name: game.name,
      releaseDate: game.release_date
        ? new Date(game.release_date * 1000).getFullYear().toString()
        : undefined,
    }))

  const searchSteamGridDBForLogos = async (query: string, request: ReturnType<typeof logoRequests.begin>) => {
    const games = await steamGridDBService.search(query)
    const results = mapSgdbSearchResults(games)
    if (!request.isCurrent()) return results
    sgdbLogoSearchResults.value = results
    sgdbLogoThumbs.value = {}

    void Promise.allSettled(
      results.map(async (game) => {
        const logos = await steamGridDBService.getLogosByGameId(Number(game.id))
        const thumbnail = logos[0]?.thumb
        if (request.isCurrent() && thumbnail) {
          sgdbLogoThumbs.value = {
            ...sgdbLogoThumbs.value,
            [game.id]: thumbnail,
          }
        }
      }),
    )
  }

  const searchSteamGridDBForScreenshots = async (
    query: string,
    request: ReturnType<typeof screenshotRequests.begin>,
  ) => {
    const games = await steamGridDBService.search(query)
    const results = mapSgdbSearchResults(games)
    if (!request.isCurrent()) return results
    sgdbScreenshotSearchResults.value = results
    sgdbScreenshotThumbs.value = {}

    void Promise.allSettled(
      results.map(async (game) => {
        const heroes = await steamGridDBService.getHeroesByGameId(Number(game.id))
        const thumbnail = heroes[0]?.thumb
        if (request.isCurrent() && thumbnail) {
          sgdbScreenshotThumbs.value = {
            ...sgdbScreenshotThumbs.value,
            [game.id]: thumbnail,
          }
        }
      }),
    )
  }

  const clearScreenshotSelectionState = () => {
    screenshotRequests.invalidate()
    isSearchingSgdbScreenshots.value = false
    screenshotSteamPicker.back()
    screenshotCandidatesData.value = null
    selectedRemoteScreenshots.value = new Set()
  }

  const clearScreenshotSearchState = () => {
    screenshotSteamPicker.clear()
    clearScreenshotSelectionState()
    sgdbScreenshotSearchResults.value = []
    sgdbScreenshotThumbs.value = {}
  }

  const handleScreenshotSearchClear = () => {
    forgetGameForSource(screenshotSource.value)
    clearScreenshotSearchState()
  }

  const searchScreenshots = async () => {
    const request = screenshotRequests.begin()
    screenshotCandidatesData.value = null
    selectedRemoteScreenshots.value.clear()
    if (screenshotSource.value === 'steamgriddb') {
      isSearchingSgdbScreenshots.value = true
      try {
        await searchSteamGridDBForScreenshots(screenshotSearchQuery.value, request)
        if (!request.isCurrent()) return
        screenshotSteamPicker.clearResults()
        screenshotSteamPicker.back()
      } catch (error) {
        if (request.isCurrent()) {
          options.addAlert('SteamGridDB 搜索失败：' + getHttpErrorMessage(error), 'error')
        }
      } finally {
        if (request.isCurrent()) isSearchingSgdbScreenshots.value = false
      }
      return
    }

    sgdbScreenshotSearchResults.value = []
    sgdbScreenshotThumbs.value = {}
    await screenshotSteamPicker.search()
  }

  const selectScreenshotGame = async (game: SteamGameSearchResult) => {
    if (screenshotSource.value === 'steamgriddb') {
      const request = screenshotRequests.begin()
      isSearchingSgdbScreenshots.value = true
      try {
        await screenshotSteamPicker.selectExternal(game, async (pickerRequest: SteamPickerRequest) => {
          const heroes = await steamGridDBService.getHeroesByGameId(Number(game.id))
          const data = {
            name: game.name,
            screenshots: heroes.map((hero) => hero.url),
            usedFallbackAssets: false,
          }
          if (!request.isCurrent() || !pickerRequest.isCurrent()) return
          screenshotCandidatesData.value = data
          selectedRemoteScreenshots.value = new Set()
          rememberSgdbGame(game)
        })
      } catch (error) {
        if (request.isCurrent()) {
          options.addAlert('SteamGridDB 获取横幅失败：' + getHttpErrorMessage(error), 'error')
        }
      } finally {
        if (request.isCurrent()) isSearchingSgdbScreenshots.value = false
      }
      return
    }

    await screenshotSteamPicker.select(game)
  }

  const backToScreenshotGameSearch = () => {
    forgetGameForSource(screenshotSource.value)
    clearScreenshotSearchState()
    const query = pickSteamSearchQuery()
    screenshotSearchQuery.value = query
    if (query) void searchScreenshots()
  }

  const resetScreenshotAfterDownload = () => {
    clearScreenshotSelectionState()
    screenshotSearchQuery.value = ''
    screenshotSteamPicker.clearResults()
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
      resetScreenshotAfterDownload()
      options.addAlert(`成功添加 ${indices.length} 张截图`, 'success')
    } catch (error) {
      options.addAlert('下载失败：' + getHttpErrorMessage(error), 'error')
    } finally {
      isDownloadingRemoteScreenshots.value = false
    }
  }

  const clearLogoSelectionState = () => {
    logoRequests.invalidate()
    isSearchingSgdbLogos.value = false
    selectedLogoGame.value = null
    logoImages.value = []
    selectedLogos.value = new Set()
  }

  const handleLogoSearchClear = () => {
    forgetGameForSource('steamgriddb')
    logoSearchQuery.value = ''
    clearLogoSelectionState()
    sgdbLogoSearchResults.value = []
    sgdbLogoThumbs.value = {}
  }

  const searchLogos = async () => {
    const request = logoRequests.begin()
    logoImages.value = []
    selectedLogos.value = new Set()
    isSearchingSgdbLogos.value = true
    try {
      await searchSteamGridDBForLogos(logoSearchQuery.value, request)
      if (request.isCurrent()) selectedLogoGame.value = null
    } catch (error) {
      if (request.isCurrent()) {
        options.addAlert('SteamGridDB 搜索失败：' + getHttpErrorMessage(error), 'error')
      }
    } finally {
      if (request.isCurrent()) isSearchingSgdbLogos.value = false
    }
  }

  const selectLogoGame = async (game: SteamGameSearchResult) => {
    const request = logoRequests.begin()
    selectedLogoGame.value = game
    isSearchingSgdbLogos.value = true
    try {
      const logos = await steamGridDBService.getLogosByGameId(Number(game.id))
      if (!request.isCurrent()) return
      logoImages.value = logos.map((logo) => logo.url)
      selectedLogos.value = new Set()
      rememberSgdbGame(game)
    } catch (error) {
      if (request.isCurrent()) {
        options.addAlert('SteamGridDB 获取 Logo 失败：' + getHttpErrorMessage(error), 'error')
        selectedLogoGame.value = null
      }
    } finally {
      if (request.isCurrent()) isSearchingSgdbLogos.value = false
    }
  }

  const backToLogoGameSearch = () => {
    forgetGameForSource('steamgriddb')
    clearLogoSelectionState()
    sgdbLogoSearchResults.value = []
    sgdbLogoThumbs.value = {}
    const query = pickSteamSearchQuery()
    logoSearchQuery.value = query
    if (query) void searchLogos()
  }

  const resetLogoAfterDownload = () => {
    clearLogoSelectionState()
    logoSearchQuery.value = ''
    sgdbLogoSearchResults.value = []
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
      resetLogoAfterDownload()
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

  const prepareScreenshotSource = () => {
    clearScreenshotSearchState()

    const remembered = screenshotSource.value === 'steamgriddb'
      ? rememberedSgdbGame.value
      : rememberedSteamGame.value
    if (remembered) {
      screenshotSearchQuery.value = remembered.id
      void selectScreenshotGame(remembered)
      return
    }

    const query = pickSteamSearchQuery()
    if (!query) return
    screenshotSearchQuery.value = query
    void searchScreenshots()
  }

  watch(showScreenshotSelector, (isOpen) => {
    if (!isOpen) return
    prepareScreenshotSource()
  })

  watch(screenshotSource, () => {
    if (!showScreenshotSelector.value) return
    prepareScreenshotSource()
  })

  const prepareLogoSearch = () => {
    clearLogoSelectionState()
    sgdbLogoSearchResults.value = []
    sgdbLogoThumbs.value = {}

    const remembered = rememberedSgdbGame.value
    if (remembered) {
      logoSearchQuery.value = remembered.id
      void selectLogoGame(remembered)
      return
    }

    const query = pickSteamSearchQuery()
    if (!query) return
    logoSearchQuery.value = query
    void searchLogos()
  }

  watch(showLogoSelector, (isOpen) => {
    if (!isOpen) return
    prepareLogoSearch()
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
