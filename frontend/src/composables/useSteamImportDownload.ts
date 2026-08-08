import { computed, ref, watch, type Ref } from 'vue'
import type {
  EditGameEditableBanner,
  EditGameEditableCover,
  EditGameForm,
} from '@/utils/edit-game-form'
import steamService, { proxySteamAssetUrl } from '@/services/steam.service'
import steamGridDBService from '@/services/steamgriddb.service'
import { useSteamPicker } from '@/composables/useSteamPicker'
import type { SteamGameSearchResult } from '@/services/types'
import { getHttpErrorMessage } from '@/utils/http-error'

type AlertType = 'success' | 'warning' | 'error'

export type ImportSource = 'steam' | 'steamgriddb'

interface UploadedAssetLike {
  id?: number
  asset_uid?: string
  path: string
}

interface UseSteamImportDownloadOptions {
  form: Ref<Pick<EditGameForm, 'covers' | 'banners' | 'title' | 'title_alt'>>
  gameId: Ref<number | undefined>
  pickSteamSearchQuery: () => string
  uploadAssetFromUrl: (
    url: string,
    assetType: 'cover' | 'banner',
  ) => Promise<UploadedAssetLike>
  createEditableCover: (
    asset: UploadedAssetLike | string,
  ) => EditGameEditableCover
  createEditableBanner: (
    asset: UploadedAssetLike | string,
  ) => EditGameEditableBanner
  addAlert: (message: string, type: AlertType) => void
  onAssetPersisted?: () => Promise<void> | void
  rememberedSteamGame?: Ref<SteamGameSearchResult | null>
  rememberedSgdbGame?: Ref<SteamGameSearchResult | null>
  onSteamGameSelected?: (game: SteamGameSearchResult) => void
  onSgdbGameSelected?: (game: SteamGameSearchResult) => void
}

export const useSteamImportDownload = (options: UseSteamImportDownloadOptions) => {
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

  // Data source toggle: 'steam' | 'steamgriddb'
  const coverSource = ref<ImportSource>('steam')
  const bannerSource = ref<ImportSource>('steam')

  const sgdbAvailable = ref(false)
  steamGridDBService.isAvailable().then((v) => { sgdbAvailable.value = v })

  const coverSteamPicker = useSteamPicker<string[]>({
    onSelect: async (game) => {
      const details = await steamService.getGameDetails(game.id)
      const images = [details.coverImage, details.bannerImage].filter(
        (value): value is string => !!value,
      )
      steamCoverImages.value = images
      selectedCoverImage.value = ''
      selectedCovers.value = new Set()
      options.onSteamGameSelected?.(game)
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
      options.onSteamGameSelected?.(game)
      return images
    },
    onError: (message) => {
      options.addAlert('Steam 横幅处理失败：' + message, 'error')
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

  // SteamGridDB search helpers
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
        coverSteamPicker.clearResults()
        coverSteamPicker.back()
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
      try {
        await coverSteamPicker.selectExternal(game, async () => {
          const grids = await steamGridDBService.getGridsByGameId(Number(game.id))
          const images = grids.map((g) => g.url)
          steamCoverImages.value = images
          selectedCoverImage.value = ''
          selectedCovers.value = new Set()
        })
        options.onSgdbGameSelected?.(game)
      } catch (e) {
        options.addAlert('SteamGridDB 获取封面失败：' + getHttpErrorMessage(e), 'error')
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

    coverSteamPicker.setSearching(true)
    try {
      const uploaded = await options.uploadAssetFromUrl(selectedCoverImage.value, 'cover')
      options.form.value.covers.push(options.createEditableCover(uploaded))
      await options.onAssetPersisted?.()
      showCoverSelector.value = false
      backToCoverGameSearch()
      coverSteamPicker.setQuery('')
      coverSteamPicker.clearResults()
      options.addAlert('封面下载成功', 'success')
    } catch (error) {
      options.addAlert('下载失败：' + getHttpErrorMessage(error), 'error')
    } finally {
      coverSteamPicker.setSearching(false)
    }
  }

  const downloadSelectedSteamCovers = async () => {
    if (!steamCoverImages.value.length || !options.gameId.value) return

    const indices = Array.from(selectedCovers.value).sort((a, b) => a - b)
    if (indices.length === 0) return

    isDownloadingSteamCovers.value = true
    let succeeded = 0
    const failedIndices: number[] = []
    let firstError = ''
    try {
      for (const index of indices) {
        try {
          const coverUrl = steamCoverImages.value[index]
          const uploaded = await options.uploadAssetFromUrl(coverUrl, 'cover')
          options.form.value.covers.push(options.createEditableCover(uploaded))
          succeeded += 1
        } catch (error) {
          failedIndices.push(index)
          if (!firstError) firstError = getHttpErrorMessage(error)
        }
      }
      if (succeeded > 0) {
        await options.onAssetPersisted?.()
      }
      if (failedIndices.length === 0) {
        showCoverSelector.value = false
        backToCoverGameSearch()
        coverSteamPicker.setQuery('')
        coverSteamPicker.clearResults()
        options.addAlert(`成功添加 ${succeeded} 张封面`, 'success')
      } else if (succeeded > 0) {
        selectedCovers.value = new Set(failedIndices)
        options.addAlert(`成功添加 ${succeeded} 张封面，${failedIndices.length} 张失败`, 'warning')
      } else {
        options.addAlert(`下载失败 ${failedIndices.length} 张：${firstError}`, 'error')
      }
    } catch (error) {
      options.addAlert('保存失败：' + getHttpErrorMessage(error), 'error')
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
        bannerSteamPicker.clearResults()
        bannerSteamPicker.back()
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
      try {
        await bannerSteamPicker.selectExternal(game, async () => {
          const heroes = await steamGridDBService.getHeroesByGameId(Number(game.id))
          const images = heroes.map((g) => g.url)
          steamBannerImages.value = images
          selectedBanners.value = new Set()
        })
        options.onSgdbGameSelected?.(game)
      } catch (e) {
        options.addAlert('SteamGridDB 获取横幅失败：' + getHttpErrorMessage(e), 'error')
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
    let succeeded = 0
    const failedIndices: number[] = []
    let firstError = ''
    try {
      for (const index of indices) {
        try {
          const bannerUrl = steamBannerImages.value[index]
          const uploaded = await options.uploadAssetFromUrl(bannerUrl, 'banner')
          options.form.value.banners.push(options.createEditableBanner(uploaded))
          succeeded += 1
        } catch (error) {
          failedIndices.push(index)
          if (!firstError) firstError = getHttpErrorMessage(error)
        }
      }
      if (succeeded > 0) {
        await options.onAssetPersisted?.()
      }
      if (failedIndices.length === 0) {
        showBannerSelector.value = false
        backToBannerGameSearch()
        bannerSteamPicker.setQuery('')
        bannerSteamPicker.clearResults()
        bannerSearchUrl.value = ''
        bannerPreviewUrl.value = ''
        options.addAlert(`成功添加 ${succeeded} 张横幅`, 'success')
      } else if (succeeded > 0) {
        selectedBanners.value = new Set(failedIndices)
        options.addAlert(`成功添加 ${succeeded} 张横幅，${failedIndices.length} 张失败`, 'warning')
      } else {
        options.addAlert(`下载失败 ${failedIndices.length} 张：${firstError}`, 'error')
      }
    } catch (error) {
      options.addAlert('保存失败：' + getHttpErrorMessage(error), 'error')
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

  const toggleCoverSelection = (index: number) => {
    if (selectedCovers.value.has(index)) {
      selectedCovers.value.delete(index)
    } else {
      selectedCovers.value.add(index)
    }
  }

  const selectAllCovers = () => {
    selectedCovers.value = new Set(steamCoverImages.value.map((_, index) => index))
  }

  const invertSelectionCovers = () => {
    const next = new Set<number>()
    for (let index = 0; index < steamCoverImages.value.length; index++) {
      if (!selectedCovers.value.has(index)) {
        next.add(index)
      }
    }
    selectedCovers.value = next
  }

  const selectAllBanners = () => {
    selectedBanners.value = new Set(steamBannerImages.value.map((_, index) => index))
  }

  const invertSelectionBanners = () => {
    const next = new Set<number>()
    for (let index = 0; index < steamBannerImages.value.length; index++) {
      if (!selectedBanners.value.has(index)) {
        next.add(index)
      }
    }
    selectedBanners.value = next
  }

  // Watch selectors opening — auto-populate search query or reuse remembered game
  watch(showCoverSelector, (isOpen) => {
    if (!isOpen) return
    const remembered = coverSource.value === 'steamgriddb'
      ? options.rememberedSgdbGame?.value
      : options.rememberedSteamGame?.value
    if (remembered) {
      coverSteamPicker.setQuery(remembered.id)
      void selectSteamCoverGame(remembered)
      return
    }
    const query = options.pickSteamSearchQuery()
    if (!query) return
    coverSteamPicker.setQuery(query)
    searchSteamForCover()
  })

  watch(showBannerSelector, (isOpen) => {
    if (!isOpen) return
    const remembered = bannerSource.value === 'steamgriddb'
      ? options.rememberedSgdbGame?.value
      : options.rememberedSteamGame?.value
    if (remembered) {
      bannerSteamPicker.setQuery(remembered.id)
      void selectSteamBannerGame(remembered)
      return
    }
    const query = options.pickSteamSearchQuery()
    if (!query) return
    bannerSteamPicker.setQuery(query)
    searchSteamForBanner()
  })

  watch(coverSource, () => {
    if (!showCoverSelector.value) return
    const remembered = coverSource.value === 'steamgriddb'
      ? options.rememberedSgdbGame?.value
      : options.rememberedSteamGame?.value
    if (remembered) {
      coverSteamPicker.setQuery(remembered.id)
      void selectSteamCoverGame(remembered)
      return
    }
    const query = steamCoverSearchQuery.value.trim()
    if (!query) return
    searchSteamForCover()
  })

  watch(bannerSource, () => {
    if (!showBannerSelector.value) return
    const remembered = bannerSource.value === 'steamgriddb'
      ? options.rememberedSgdbGame?.value
      : options.rememberedSteamGame?.value
    if (remembered) {
      bannerSteamPicker.setQuery(remembered.id)
      void selectSteamBannerGame(remembered)
      return
    }
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
      : [...steamCoverSearchResults.value],
  )
  const isSearchingCover = computed(() =>
    coverSource.value === 'steamgriddb' ? sgdbSearching.value : isSearchingSteamCover.value,
  )
  const bannerSearchResults = computed(() =>
    bannerSource.value === 'steamgriddb'
      ? mergeSGDBThumbs(sgdbSearchResults.value)
      : [...steamBannerSearchResults.value],
  )
  const isSearchingBanner = computed(() =>
    bannerSource.value === 'steamgriddb' ? sgdbSearching.value : isSearchingSteamBanner.value,
  )

  const resetDownloadState = () => {
    showCoverSelector.value = false
    showBannerSelector.value = false

    sgdbSearchResults.value = []
    sgdbThumbs.value = {}

    coverSteamPicker.clear()
    steamCoverImages.value = []
    selectedCoverImage.value = ''
    selectedCovers.value = new Set()
    coverSearchUrl.value = ''
    coverPreviewUrl.value = ''

    bannerSteamPicker.clear()
    steamBannerImages.value = []
    selectedBanners.value = new Set()
    bannerSearchUrl.value = ''
    bannerPreviewUrl.value = ''
  }

  return {
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
  }
}
