import { ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { EditGameForm } from '@/utils/edit-game-form'
import { useSteamImport } from './useSteamImport'

const {
  searchGamesMock,
  getGameDetailsMock,
  searchSteamGridDBMock,
  getHeroesByGameIdMock,
  getLogosByGameIdMock,
  isSteamGridDBAvailableMock,
} = vi.hoisted(() => ({
  searchGamesMock: vi.fn(),
  getGameDetailsMock: vi.fn(),
  searchSteamGridDBMock: vi.fn(),
  getHeroesByGameIdMock: vi.fn(),
  getLogosByGameIdMock: vi.fn(),
  isSteamGridDBAvailableMock: vi.fn(),
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
    getGridsByGameId: vi.fn(),
    getHeroesByGameId: getHeroesByGameIdMock,
    getLogosByGameId: getLogosByGameIdMock,
  },
}))

const flushPromises = () => new Promise((resolve) => setTimeout(resolve, 0))

const buildForm = () => ref<EditGameForm>({
  title: 'Example Game',
  title_alt: '',
  visibility: 'public',
  developer_ids: [],
  publisher_ids: [],
  release_date: undefined,
  series_id: null,
  summary: '',
  save_path_template: '',
  covers: [],
  banners: [],
  logos: [],
  logo_visible: true,
  preview_videos: [],
  screenshots: [],
  file_paths: [],
})

describe('useSteamImport', () => {
  beforeEach(() => {
    searchGamesMock.mockReset()
    getGameDetailsMock.mockReset()
    isSteamGridDBAvailableMock.mockReset()
    isSteamGridDBAvailableMock.mockResolvedValue(true)
    searchSteamGridDBMock.mockReset()
    getHeroesByGameIdMock.mockReset()
    getLogosByGameIdMock.mockReset()
  })

  it('imports selected SteamGridDB heroes as screenshot assets', async () => {
    searchSteamGridDBMock.mockResolvedValue([
      { id: 77, name: 'Grid Only Game', release_date: 1_704_067_200 },
    ])
    getHeroesByGameIdMock.mockResolvedValue([
      { url: 'https://cdn.example.com/hero-1.jpg', thumb: 'https://cdn.example.com/hero-1-thumb.jpg' },
      { url: 'https://cdn.example.com/hero-2.jpg', thumb: 'https://cdn.example.com/hero-2-thumb.jpg' },
    ])
    const uploadAssetFromUrl = vi.fn()
      .mockResolvedValueOnce({ path: '/assets/shot-1.jpg', asset_uid: 'shot-1' })
      .mockResolvedValueOnce({ path: '/assets/shot-2.jpg', asset_uid: 'shot-2' })
    const form = buildForm()
    const steamImport = useSteamImport({
      form,
      gameId: ref(42),
      getWikiContent: () => '',
      uploadAssetFromUrl,
      createEditableCover: (asset) => ({
        asset_uid: typeof asset === 'string' ? undefined : asset.asset_uid,
        path: typeof asset === 'string' ? asset : asset.path,
      }),
      createEditableBanner: (asset) => ({
        asset_uid: typeof asset === 'string' ? undefined : asset.asset_uid,
        path: typeof asset === 'string' ? asset : asset.path,
      }),
      createEditableLogo: (asset) => ({
        asset_uid: typeof asset === 'string' ? undefined : asset.asset_uid,
        path: typeof asset === 'string' ? asset : asset.path,
        position_x: null,
        position_y: null,
        width_pct: null,
      }),
      createEditableScreenshot: (asset, index) => ({
        asset_uid: typeof asset === 'string' ? undefined : asset.asset_uid,
        path: typeof asset === 'string' ? asset : asset.path,
        client_key: `screenshot-${index}`,
      }),
      ensureDeveloperNames: vi.fn().mockResolvedValue([]),
      ensurePublisherNames: vi.fn().mockResolvedValue([]),
      addAlert: vi.fn(),
    })

    steamImport.screenshotSource.value = 'steamgriddb'
    steamImport.screenshotSearchQuery.value = 'Grid Only Game'
    await steamImport.searchScreenshots()

    expect(steamImport.screenshotSearchResults.value).toEqual([
      expect.objectContaining({
        id: '77',
        name: 'Grid Only Game',
        releaseDate: '2024',
      }),
    ])

    await steamImport.selectScreenshotGame(steamImport.screenshotSearchResults.value[0]!)

    expect(steamImport.screenshotCandidatesData.value?.screenshots).toEqual([
      'https://cdn.example.com/hero-1.jpg',
      'https://cdn.example.com/hero-2.jpg',
    ])

    steamImport.selectAllScreenshots()
    await steamImport.downloadSelectedScreenshots()

    expect(uploadAssetFromUrl).toHaveBeenNthCalledWith(
      1,
      'https://cdn.example.com/hero-1.jpg',
      'screenshot',
    )
    expect(uploadAssetFromUrl).toHaveBeenNthCalledWith(
      2,
      'https://cdn.example.com/hero-2.jpg',
      'screenshot',
    )
    expect(form.value.screenshots.map((item) => item.path)).toEqual([
      '/assets/shot-1.jpg',
      '/assets/shot-2.jpg',
    ])
  })

  it('searches SteamGridDB directly for logo candidates', async () => {
    searchSteamGridDBMock.mockResolvedValue([
      { id: 88, name: 'Grid Only Logo Game', release_date: 1_704_067_200 },
    ])
    getLogosByGameIdMock.mockResolvedValue([
      { url: 'https://cdn.example.com/logo-1.png', thumb: 'https://cdn.example.com/logo-1-thumb.png' },
      { url: 'https://cdn.example.com/logo-2.png', thumb: 'https://cdn.example.com/logo-2-thumb.png' },
    ])
    const uploadAssetFromUrl = vi.fn()
      .mockResolvedValueOnce({ path: '/assets/logo-1.png', asset_uid: 'logo-1' })
      .mockResolvedValueOnce({ path: '/assets/logo-2.png', asset_uid: 'logo-2' })
    const form = buildForm()
    const steamImport = useSteamImport({
      form,
      gameId: ref(42),
      getWikiContent: () => '',
      uploadAssetFromUrl,
      createEditableCover: (asset) => ({
        asset_uid: typeof asset === 'string' ? undefined : asset.asset_uid,
        path: typeof asset === 'string' ? asset : asset.path,
      }),
      createEditableBanner: (asset) => ({
        asset_uid: typeof asset === 'string' ? undefined : asset.asset_uid,
        path: typeof asset === 'string' ? asset : asset.path,
      }),
      createEditableLogo: (asset) => ({
        asset_uid: typeof asset === 'string' ? undefined : asset.asset_uid,
        path: typeof asset === 'string' ? asset : asset.path,
        position_x: null,
        position_y: null,
        width_pct: null,
      }),
      createEditableScreenshot: (asset, index) => ({
        asset_uid: typeof asset === 'string' ? undefined : asset.asset_uid,
        path: typeof asset === 'string' ? asset : asset.path,
        client_key: `screenshot-${index}`,
      }),
      ensureDeveloperNames: vi.fn().mockResolvedValue([]),
      ensurePublisherNames: vi.fn().mockResolvedValue([]),
      addAlert: vi.fn(),
    })

    steamImport.logoSearchQuery.value = 'Grid Only Logo Game'
    await steamImport.searchLogos()

    expect(searchSteamGridDBMock).toHaveBeenCalledWith('Grid Only Logo Game')
    expect(steamImport.logoSearchResults.value).toEqual([
      expect.objectContaining({ id: '88', name: 'Grid Only Logo Game' }),
    ])

    await steamImport.selectLogoGame(steamImport.logoSearchResults.value[0])
    expect(steamImport.logoImages.value).toEqual([
      'https://cdn.example.com/logo-1.png',
      'https://cdn.example.com/logo-2.png',
    ])

    steamImport.selectAllLogos()
    await steamImport.downloadSelectedLogos()

    expect(uploadAssetFromUrl).toHaveBeenNthCalledWith(1, 'https://cdn.example.com/logo-1.png', 'logo')
    expect(uploadAssetFromUrl).toHaveBeenNthCalledWith(2, 'https://cdn.example.com/logo-2.png', 'logo')
    expect(form.value.logos.map((item) => item.path)).toEqual([
      '/assets/logo-1.png',
      '/assets/logo-2.png',
    ])
  })

  it('falls back to Steam cover and banner when a game has no screenshots', async () => {
    searchGamesMock.mockResolvedValue([{ id: '55', name: 'Steam Only' }])
    getGameDetailsMock.mockResolvedValue({
      name: 'Steam Only',
      description: '',
      releaseDate: '2024',
      developers: [],
      publishers: [],
      screenshots: [],
      coverImage: 'https://cdn.example.com/cover.jpg',
      bannerImage: 'https://cdn.example.com/banner.jpg',
    })

    const form = buildForm()
    const steamImport = useSteamImport({
      form,
      gameId: ref(42),
      getWikiContent: () => '',
      uploadAssetFromUrl: vi.fn(),
      createEditableCover: vi.fn(),
      createEditableBanner: vi.fn(),
      createEditableLogo: vi.fn(),
      createEditableScreenshot: vi.fn(),
      ensureDeveloperNames: vi.fn().mockResolvedValue([]),
      ensurePublisherNames: vi.fn().mockResolvedValue([]),
      addAlert: vi.fn(),
    })

    steamImport.screenshotSearchQuery.value = 'Steam Only'
    await steamImport.searchScreenshots()
    expect(steamImport.screenshotSearchResults.value[0]).toMatchObject({
      id: '55',
      name: 'Steam Only',
    })

    await steamImport.selectScreenshotGame(steamImport.screenshotSearchResults.value[0]!)
    expect(steamImport.screenshotCandidatesData.value?.screenshots).toEqual([
      'https://cdn.example.com/banner.jpg',
      'https://cdn.example.com/cover.jpg',
    ])
  })

  it('clears SteamGridDB screenshot search state', async () => {
    searchSteamGridDBMock.mockResolvedValue([{ id: 99, name: 'Grid Screenshot' }])
    getHeroesByGameIdMock.mockResolvedValue([])

    const steamImport = useSteamImport({
      form: buildForm(),
      gameId: ref(42),
      getWikiContent: () => '',
      uploadAssetFromUrl: vi.fn(),
      createEditableCover: vi.fn(),
      createEditableBanner: vi.fn(),
      createEditableLogo: vi.fn(),
      createEditableScreenshot: vi.fn(),
      ensureDeveloperNames: vi.fn().mockResolvedValue([]),
      ensurePublisherNames: vi.fn().mockResolvedValue([]),
      addAlert: vi.fn(),
    })
    steamImport.screenshotSource.value = 'steamgriddb'
    steamImport.screenshotSearchQuery.value = 'Grid Screenshot'
    await steamImport.searchScreenshots()

    expect(steamImport.screenshotSearchResults.value).toHaveLength(1)
    steamImport.handleScreenshotSearchClear()

    expect(steamImport.screenshotSearchQuery.value).toBe('')
    expect(steamImport.screenshotSearchResults.value).toEqual([])
    expect(steamImport.screenshotCandidatesData.value).toBeNull()
  })

  it('resets screenshot selector state', async () => {
    searchSteamGridDBMock.mockResolvedValue([{ id: 100, name: 'Grid Reset' }])
    getHeroesByGameIdMock.mockResolvedValue([])

    const steamImport = useSteamImport({
      form: buildForm(),
      gameId: ref(42),
      getWikiContent: () => '',
      uploadAssetFromUrl: vi.fn(),
      createEditableCover: vi.fn(),
      createEditableBanner: vi.fn(),
      createEditableLogo: vi.fn(),
      createEditableScreenshot: vi.fn(),
      ensureDeveloperNames: vi.fn().mockResolvedValue([]),
      ensurePublisherNames: vi.fn().mockResolvedValue([]),
      addAlert: vi.fn(),
    })
    steamImport.screenshotSource.value = 'steamgriddb'
    steamImport.screenshotSearchQuery.value = 'Grid Reset'
    await steamImport.searchScreenshots()

    steamImport.resetSteamImportState()

    expect(steamImport.showScreenshotSelector.value).toBe(false)
    expect(steamImport.screenshotSource.value).toBe('steam')
    expect(steamImport.screenshotSearchQuery.value).toBe('')
    expect(steamImport.screenshotSearchResults.value).toEqual([])
    expect(steamImport.screenshotCandidatesData.value).toBeNull()
  })

  it('reuses the last selected Steam game for later asset pickers', async () => {
    searchGamesMock.mockResolvedValue([{ id: '2285650', name: 'Picked Game' }])
    getGameDetailsMock.mockResolvedValue({
      name: 'Picked Game',
      description: '',
      releaseDate: '2024',
      developers: [],
      publishers: [],
      screenshots: [],
      coverImage: 'https://cdn.example.com/cover.jpg',
      bannerImage: 'https://cdn.example.com/banner.jpg',
    })

    const steamImport = useSteamImport({
      form: buildForm(),
      gameId: ref(42),
      getWikiContent: () => '',
      uploadAssetFromUrl: vi.fn(),
      createEditableCover: vi.fn(),
      createEditableBanner: vi.fn(),
      createEditableLogo: vi.fn(),
      createEditableScreenshot: vi.fn(),
      ensureDeveloperNames: vi.fn().mockResolvedValue([]),
      ensurePublisherNames: vi.fn().mockResolvedValue([]),
      addAlert: vi.fn(),
    })

    steamImport.showCoverSelector.value = true
    await flushPromises()
    expect(searchGamesMock).toHaveBeenCalledTimes(1)

    await steamImport.selectSteamCoverGame(steamImport.coverSearchResults.value[0]!)
    expect(getGameDetailsMock).toHaveBeenCalledTimes(1)

    steamImport.showBannerSelector.value = true
    await flushPromises()
    expect(searchGamesMock).toHaveBeenCalledTimes(1)
    expect(getGameDetailsMock).toHaveBeenCalledTimes(2)
  })

  it('clears the remembered Steam game when switching edited game', async () => {
    searchGamesMock.mockResolvedValue([{ id: '2285650', name: 'Picked Game' }])
    getGameDetailsMock.mockResolvedValue({
      name: 'Picked Game',
      description: '',
      releaseDate: '2024',
      developers: [],
      publishers: [],
      screenshots: [],
      coverImage: 'https://cdn.example.com/cover.jpg',
      bannerImage: 'https://cdn.example.com/banner.jpg',
    })

    const gameId = ref(42)
    const steamImport = useSteamImport({
      form: buildForm(),
      gameId,
      getWikiContent: () => '',
      uploadAssetFromUrl: vi.fn(),
      createEditableCover: vi.fn(),
      createEditableBanner: vi.fn(),
      createEditableLogo: vi.fn(),
      createEditableScreenshot: vi.fn(),
      ensureDeveloperNames: vi.fn().mockResolvedValue([]),
      ensurePublisherNames: vi.fn().mockResolvedValue([]),
      addAlert: vi.fn(),
    })

    steamImport.showCoverSelector.value = true
    await flushPromises()
    await steamImport.selectSteamCoverGame(steamImport.coverSearchResults.value[0]!)
    expect(getGameDetailsMock).toHaveBeenCalledTimes(1)

    gameId.value = 99
    await flushPromises()

    steamImport.showBannerSelector.value = true
    await flushPromises()
    expect(searchGamesMock).toHaveBeenCalledTimes(2)
    expect(getGameDetailsMock).toHaveBeenCalledTimes(1)
  })
})
