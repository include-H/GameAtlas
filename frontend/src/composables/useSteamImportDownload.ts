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
  form: Ref<Pick<EditGameForm, 'covers' | 'banners' | 'banner_image' | 'title' | 'title_alt'>>
  gameId: Ref<number | undefined>
  pickSteamSearchQuery: () => string
  uploadAssetFromUrl: (
    url: string,
    assetType: 'cover' | 'banner',
    sortOrder?: number,
  ) => Promise<UploadedAssetLike>
  createEditableCover: (
    asset: UploadedAssetLike | string,
  ) => EditGameEditableCover
  createEditableBanner: (
    asset: UploadedAssetLike | string,
  ) => EditGameEditableBanner
  addAlert: (message: string, type: AlertType) => void
  onAssetPersisted?: () => Promise<void> | void
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

  const toggleCoverSelection = (index: number) => {
    if (selectedCovers.value.has(index)) {
      selectedCovers.value.delete(index)
    } else {
      selectedCovers.value.add(index)
    }
  }

  // Watch selectors opening — auto-populate search query
  watch(showCoverSelector, (isOpen) => {
    if (!isOpen) return
    const query = options.pickSteamSearchQuery()
    if (!query) return
    steamCoverSearchQuery.value = query
    searchSteamForCover()
  })

  watch(showBannerSelector, (isOpen) => {
    if (!isOpen) return
    const query = options.pickSteamSearchQuery()
    if (!query) return
    steamBannerSearchQuery.value = query
    searchSteamForBanner()
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

  const resetDownloadState = () => {
    showCoverSelector.value = false
    showBannerSelector.value = false

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
    handleBannerSearchClear,
    searchSteamForBanner,
    selectSteamBannerGame,
    backToBannerGameSearch,
    loadBannerFromUrl,
    confirmBannerSelection,
    downloadSelectedSteamBanner,
    toggleBannerSelection,
    resetDownloadState,
  }
}
