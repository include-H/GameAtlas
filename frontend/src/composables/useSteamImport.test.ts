import { ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { EditGameForm } from '@/utils/edit-game-form'
import { useSteamImport } from './useSteamImport'

const {
  searchSteamGridDBMock,
  getHeroesByGameIdMock,
  isSteamGridDBAvailableMock,
} = vi.hoisted(() => ({
  searchSteamGridDBMock: vi.fn(),
  getHeroesByGameIdMock: vi.fn(),
  isSteamGridDBAvailableMock: vi.fn(),
}))

vi.mock('@/services/steam.service', () => ({
  default: {
    searchGames: vi.fn(),
    getGameDetails: vi.fn(),
  },
  proxySteamAssetUrl: (url: string) => url,
}))

vi.mock('@/services/steamgriddb.service', () => ({
  default: {
    isAvailable: isSteamGridDBAvailableMock,
    search: searchSteamGridDBMock,
    getGridsByGameId: vi.fn(),
    getHeroesByGameId: getHeroesByGameIdMock,
    getLogosBySteamAppId: vi.fn(),
    getLogosByGameId: vi.fn(),
  },
}))

const buildForm = () => ref<EditGameForm>({
  title: 'Example Game',
  title_alt: '',
  visibility: 'public',
  developer_ids: [],
  publisher_ids: [],
  release_date: undefined,
  series_id: null,
  summary: '',
  covers: [],
  banners: [],
  logo: null,
  logo_visible: true,
  preview_videos: [],
  screenshots: [],
  file_paths: [],
})

describe('useSteamImport', () => {
  beforeEach(() => {
    isSteamGridDBAvailableMock.mockReset()
    isSteamGridDBAvailableMock.mockResolvedValue(true)
    searchSteamGridDBMock.mockReset()
    getHeroesByGameIdMock.mockReset()
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

    await steamImport.selectScreenshotGame(steamImport.screenshotSearchResults.value[0])

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
      0,
    )
    expect(uploadAssetFromUrl).toHaveBeenNthCalledWith(
      2,
      'https://cdn.example.com/hero-2.jpg',
      'screenshot',
      1,
    )
    expect(form.value.screenshots.map((item) => item.path)).toEqual([
      '/assets/shot-1.jpg',
      '/assets/shot-2.jpg',
    ])
  })
})
