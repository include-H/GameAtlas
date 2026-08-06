import { ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { EditGameForm } from '@/utils/edit-game-form'
import type { SteamGameDetails, SteamGameSearchResult } from '@/services/types'
import { useSteamImportDownload } from './useSteamImportDownload'

const {
  searchGamesMock,
  getGameDetailsMock,
  isSteamGridDBAvailableMock,
  searchSteamGridDBMock,
  getGridsByGameIdMock,
  getHeroesByGameIdMock,
} = vi.hoisted(() => ({
  searchGamesMock: vi.fn(),
  getGameDetailsMock: vi.fn(),
  isSteamGridDBAvailableMock: vi.fn(),
  searchSteamGridDBMock: vi.fn(),
  getGridsByGameIdMock: vi.fn(),
  getHeroesByGameIdMock: vi.fn(),
}))

vi.mock('@/services/steam.service', () => ({
  default: {
    searchGames: searchGamesMock,
    getGameDetails: getGameDetailsMock,
  },
  proxySteamAssetUrl: (url: string) => url,
}))

vi.mock('@/services/steamgriddb.service', () => ({
  default: {
    isAvailable: isSteamGridDBAvailableMock,
    search: searchSteamGridDBMock,
    getGridsByGameId: getGridsByGameIdMock,
    getHeroesByGameId: getHeroesByGameIdMock,
  },
}))

const buildForm = () => ref<Pick<EditGameForm, 'covers' | 'banners' | 'title' | 'title_alt'>>({
  title: 'Example Game',
  title_alt: '',
  covers: [],
  banners: [],
})

const buildDownload = () => {
  const form = buildForm()
  const gameId = ref(42)
  const uploadAssetFromUrl = vi.fn()
  const addAlert = vi.fn()
  const onAssetPersisted = vi.fn()
  const createEditableCover = (asset: unknown) => {
    const path = typeof asset === 'string' ? asset : (asset as { path: string }).path
    const assetUid = typeof asset === 'string' ? undefined : (asset as { asset_uid?: string }).asset_uid
    return { path, asset_uid: assetUid }
  }
  const createEditableBanner = createEditableCover

  const download = useSteamImportDownload({
    form,
    gameId,
    pickSteamSearchQuery: () => form.value.title,
    uploadAssetFromUrl,
    createEditableCover,
    createEditableBanner,
    addAlert,
    onAssetPersisted,
  })

  return { download, form, gameId, uploadAssetFromUrl, addAlert, onAssetPersisted }
}

const steamDetails = (overrides: Partial<SteamGameDetails> = {}): SteamGameDetails => ({
  name: 'Steam Game',
  description: '',
  releaseDate: '2024',
  developers: [],
  publishers: [],
  screenshots: [],
  coverImage: 'https://cdn.example.com/cover.jpg',
  bannerImage: 'https://cdn.example.com/banner.jpg',
  ...overrides,
})

const gameResult = (id: string, name: string): SteamGameSearchResult => ({
  id,
  name,
})

describe('useSteamImportDownload', () => {
  beforeEach(() => {
    searchGamesMock.mockReset()
    getGameDetailsMock.mockReset()
    isSteamGridDBAvailableMock.mockReset()
    isSteamGridDBAvailableMock.mockResolvedValue(true)
    searchSteamGridDBMock.mockReset()
    getGridsByGameIdMock.mockReset()
    getHeroesByGameIdMock.mockReset()
  })

  it('searches and selects SteamGridDB cover candidates', async () => {
    searchSteamGridDBMock.mockResolvedValue([
      { id: 7, name: 'Grid Cover', release_date: 1_704_067_200 },
    ])
    getGridsByGameIdMock.mockResolvedValue([
      { url: 'https://cdn.example.com/grid-cover.jpg', thumb: 'https://cdn.example.com/grid-thumb.jpg' },
    ])

    const { download } = buildDownload()
    download.coverSource.value = 'steamgriddb'
    download.steamCoverSearchQuery.value = 'Grid Cover'
    await download.searchSteamForCover()

    await vi.waitFor(() => {
      expect(download.coverSearchResults.value[0]?.tinyImage).toBe('https://cdn.example.com/grid-thumb.jpg')
    })
    expect(download.coverSearchResults.value[0]).toMatchObject({
      id: '7',
      name: 'Grid Cover',
    })

    await download.selectSteamCoverGame(download.coverSearchResults.value[0]!)
    expect(download.steamCoverImages.value).toEqual(['https://cdn.example.com/grid-cover.jpg'])
  })

  it('searches Steam and downloads a selected cover', async () => {
    searchGamesMock.mockResolvedValue([gameResult('77', 'Steam Cover')])
    getGameDetailsMock.mockResolvedValue(steamDetails())

    const { download, form, uploadAssetFromUrl, addAlert } = buildDownload()
    download.steamCoverSearchQuery.value = 'Steam Cover'
    await download.searchSteamForCover()

    expect(download.coverSearchResults.value[0]).toMatchObject({ id: '77' })
    await download.selectSteamCoverGame(download.coverSearchResults.value[0]!)
    expect(download.steamCoverImages.value).toEqual([
      'https://cdn.example.com/cover.jpg',
      'https://cdn.example.com/banner.jpg',
    ])

    download.selectedCoverImage.value = download.steamCoverImages.value[0]!
    uploadAssetFromUrl.mockResolvedValue({
      path: '/assets/steam-cover.jpg',
      asset_uid: 'steam-cover',
    })
    await download.downloadSelectedSteamCover()

    expect(form.value.covers.map((item) => item.path)).toEqual(['/assets/steam-cover.jpg'])
    expect(addAlert).toHaveBeenCalledWith('封面下载成功', 'success')
  })

  it('downloads multiple selected Steam covers', async () => {
    const { download, form, uploadAssetFromUrl, addAlert } = buildDownload()
    download.steamCoverImages.value = [
      'https://cdn.example.com/cover-1.jpg',
      'https://cdn.example.com/cover-2.jpg',
    ]
    download.selectedCovers.value = new Set([0, 1])
    uploadAssetFromUrl
      .mockResolvedValueOnce({ path: '/assets/cover-1.jpg', asset_uid: 'cover-1' })
      .mockResolvedValueOnce({ path: '/assets/cover-2.jpg', asset_uid: 'cover-2' })

    await download.downloadSelectedSteamCovers()

    expect(form.value.covers.map((item) => item.path)).toEqual([
      '/assets/cover-1.jpg',
      '/assets/cover-2.jpg',
    ])
    expect(addAlert).toHaveBeenCalledWith('成功添加 2 张封面', 'success')
  })

  it('searches and downloads SteamGridDB banner candidates', async () => {
    searchSteamGridDBMock.mockResolvedValue([{ id: 8, name: 'Grid Banner' }])
    getHeroesByGameIdMock.mockResolvedValue([
      { url: 'https://cdn.example.com/grid-banner.jpg', thumb: 'https://cdn.example.com/grid-banner-thumb.jpg' },
    ])

    const { download, form, uploadAssetFromUrl, addAlert } = buildDownload()
    download.bannerSource.value = 'steamgriddb'
    download.steamBannerSearchQuery.value = 'Grid Banner'
    await download.searchSteamForBanner()
    await download.selectSteamBannerGame(download.bannerSearchResults.value[0]!)

    expect(download.steamBannerImages.value).toEqual(['https://cdn.example.com/grid-banner.jpg'])
    download.selectedBanners.value = new Set([0])
    uploadAssetFromUrl.mockResolvedValue({
      path: '/assets/grid-banner.jpg',
      asset_uid: 'grid-banner',
    })
    await download.downloadSelectedSteamBanner()

    expect(form.value.banners.map((item) => item.path)).toEqual(['/assets/grid-banner.jpg'])
    expect(addAlert).toHaveBeenCalledWith('成功添加 1 张横幅', 'success')
  })

  it('resets cover and banner download state', () => {
    const { download } = buildDownload()
    download.showCoverSelector.value = true
    download.coverSource.value = 'steamgriddb'
    download.steamCoverSearchQuery.value = 'query'
    download.steamCoverImages.value = ['https://cdn.example.com/cover.jpg']
    download.selectedCovers.value = new Set([0])
    download.showBannerSelector.value = true
    download.steamBannerImages.value = ['https://cdn.example.com/banner.jpg']
    download.selectedBanners.value = new Set([0])

    download.resetDownloadState()

    expect(download.showCoverSelector.value).toBe(false)
    expect(download.steamCoverSearchQuery.value).toBe('')
    expect(download.steamCoverImages.value).toEqual([])
    expect(download.selectedCovers.value.size).toBe(0)
    expect(download.showBannerSelector.value).toBe(false)
    expect(download.steamBannerSearchQuery.value).toBe('')
    expect(download.steamBannerImages.value).toEqual([])
    expect(download.selectedBanners.value.size).toBe(0)
  })
})
