import { beforeEach, describe, expect, it, vi } from 'vitest'

const { delMock, getMock, postMock, putMock } = vi.hoisted(() => ({
  delMock: vi.fn(),
  getMock: vi.fn(),
  postMock: vi.fn(),
  putMock: vi.fn(),
}))

vi.mock('./api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('./api')>()
  return {
    ...actual,
    del: delMock,
    get: getMock,
    post: postMock,
    put: putMock,
  }
})

import gamesService from './games.service'

const baseGame = {
  id: 1,
  public_id: 'game-1',
  title: 'Game One',
  title_alt: null,
  visibility: 'public' as const,
  summary: null,
  release_date: null,
  engine: null,
  cover_image: null,
  banner_image: null,
  wiki_content: null,
  wiki_content_html: null,
  needs_review: false,
  primary_screenshot: null,
  screenshot_count: 0,
  file_count: 0,
  developer_count: 0,
  publisher_count: 0,
  platform_count: 0,
  downloads: 0,
  created_at: '2026-03-25T00:00:00Z',
  updated_at: '2026-03-25T00:00:00Z',
}

describe('games service', () => {
  beforeEach(() => {
    delMock.mockReset()
    getMock.mockReset()
    postMock.mockReset()
    putMock.mockReset()
    vi.unstubAllEnvs()
    window.localStorage.clear()
  })

  it('loads games, builds query params and marks favorites', async () => {
    window.localStorage.setItem('game-library-favorites', JSON.stringify(['game-1']))
    getMock.mockResolvedValue({
      data: [
        baseGame,
        {
          ...baseGame,
          id: 2,
          public_id: 'game-2',
          title: 'Game Two',
        },
      ],
      pagination: {
        page: 2,
        limit: 20,
        total: 2,
        totalPages: 1,
      },
    })

    const result = await gamesService.getGames({
      query: {
        page: 2,
        limit: 20,
        search: 'halo',
        series: 'fps',
        platform: 'pc',
        needs_review: true,
        pending: false,
        tag: [3, 7],
      },
      sort: {
        field: 'updated_at',
        order: 'desc',
        seed: 9,
      },
    })

    expect(result.data.map((item) => ({ id: item.public_id, isFavorite: item.isFavorite }))).toEqual([
      { id: 'game-1', isFavorite: true },
      { id: 'game-2', isFavorite: false },
    ])

    const params = getMock.mock.calls[0]?.[1]?.params as URLSearchParams
    expect(getMock.mock.calls[0]?.[0]).toBe('/games')
    expect(params.get('page')).toBe('2')
    expect(params.get('limit')).toBe('20')
    expect(params.get('search')).toBe('halo')
    expect(params.get('series')).toBe('fps')
    expect(params.get('platform')).toBe('pc')
    expect(params.get('needs_review')).toBe('true')
    expect(params.get('pending')).toBe('false')
    expect(params.getAll('tag')).toEqual(['3', '7'])
    expect(params.get('sort')).toBe('updated_at')
    expect(params.get('order')).toBe('desc')
    expect(params.get('seed')).toBe('9')
  })

  it('filters favorites when requested', async () => {
    window.localStorage.setItem('game-library-favorites', JSON.stringify(['game-2']))
    getMock.mockResolvedValue({
      data: [
        baseGame,
        {
          ...baseGame,
          id: 2,
          public_id: 'game-2',
        },
      ],
      pagination: {
        page: 1,
        limit: 20,
        total: 2,
        totalPages: 1,
      },
    })

    const result = await gamesService.getGames({
      query: {
        favorite: true,
      },
    })

    expect(result.data).toHaveLength(1)
    expect(result.data[0]?.public_id).toBe('game-2')
  })

  it('loads all pages through getAllGames', async () => {
    getMock
      .mockResolvedValueOnce({
        data: [{ ...baseGame, id: 1, public_id: 'game-1' }],
        pagination: { page: 1, limit: 1, total: 2, totalPages: 2 },
      })
      .mockResolvedValueOnce({
        data: [{ ...baseGame, id: 2, public_id: 'game-2' }],
        pagination: { page: 2, limit: 1, total: 2, totalPages: 2 },
      })

    const result = await gamesService.getAllGames({ limit: 1 })

    expect(result.map((item) => item.public_id)).toEqual(['game-1', 'game-2'])
    expect(getMock).toHaveBeenCalledTimes(2)
  })

  it('maps game files to version metadata', async () => {
    getMock.mockResolvedValue({
      data: {
        ...baseGame,
        preview_video: null,
        preview_videos: [],
        screenshots: [],
        series: null,
        platforms: [],
        developers: [],
        publishers: [],
        tags: [],
        tag_groups: [],
        files: [
          {
            id: 10,
            game_id: 1,
            file_name: 'Alpha.vhdx',
            file_path: '/roms/Alpha.vhdx',
            label: '',
            notes: 'latest build',
            size_bytes: 123,
            sort_order: 2,
            source_created_at: '2026-03-25T00:00:00Z',
            created_at: '2026-03-24T00:00:00Z',
            updated_at: '2026-03-25T00:00:00Z',
          },
          {
            id: 9,
            game_id: 1,
            file_name: 'Legacy.iso',
            file_path: '/roms/Legacy.iso',
            label: 'Legacy',
            notes: null,
            size_bytes: 99,
            sort_order: 1,
            source_created_at: '2026-03-20T00:00:00Z',
            created_at: '2026-03-20T00:00:00Z',
            updated_at: '2026-03-20T00:00:00Z',
          },
        ],
      },
    })

    const result = await gamesService.getGameVersions('game-1')

    expect(result).toEqual([
      {
        id: '9',
        gameId: 'game-1',
        version: 'Legacy',
        releaseDate: '2026-03-20T00:00:00Z',
        size: 99,
        isLatest: false,
        canLaunch: false,
        downloadUrl: '/api/games/game-1/files/9/download',
        launchScriptUrl: '/api/games/game-1/files/9/launch-script',
        changelog: undefined,
      },
      {
        id: '10',
        gameId: 'game-1',
        version: 'Alpha',
        releaseDate: '2026-03-25T00:00:00Z',
        size: 123,
        isLatest: true,
        canLaunch: true,
        downloadUrl: '/api/games/game-1/files/10/download',
        launchScriptUrl: '/api/games/game-1/files/10/launch-script',
        changelog: 'latest build',
      },
    ])
  })

  it('normalizes preview video to the first sorted asset', async () => {
    getMock.mockResolvedValue({
      data: {
        ...baseGame,
        preview_video: {
          id: 12,
          asset_uid: 'video-primary',
          path: '/assets/video-primary.mp4',
          sort_order: 5,
        },
        preview_videos: [
          {
            id: 11,
            asset_uid: 'video-first',
            path: '/assets/video-first.mp4',
            sort_order: 0,
          },
          {
            id: 12,
            asset_uid: 'video-primary',
            path: '/assets/video-primary.mp4',
            sort_order: 5,
          },
        ],
        screenshots: [],
        series: null,
        platforms: [],
        developers: [],
        publishers: [],
        tags: [],
        tag_groups: [],
        files: [],
      },
    })

    const result = await gamesService.getGame('game-1')

    expect(result.preview_video?.asset_uid).toBe('video-first')
    expect(result.preview_videos.map((item) => item.asset_uid)).toEqual(['video-first', 'video-primary'])
  })

  it('keeps preview video empty when there are no videos', async () => {
    getMock.mockResolvedValue({
      data: {
        ...baseGame,
        preview_video: null,
        preview_videos: [],
        screenshots: [],
        series: null,
        platforms: [],
        developers: [],
        publishers: [],
        tags: [],
        tag_groups: [],
        files: [],
      },
    })

    const result = await gamesService.getGame('game-1')

    expect(result.preview_video).toBeNull()
  })

  it('toggles favorites in localStorage', async () => {
    expect(await gamesService.toggleFavorite('game-1')).toEqual({ isFavorite: true })
    expect(window.localStorage.getItem('game-library-favorites')).toBe(JSON.stringify(['game-1']))

    expect(await gamesService.toggleFavorite('game-1')).toEqual({ isFavorite: false })
    expect(window.localStorage.getItem('game-library-favorites')).toBe(JSON.stringify([]))
  })

  it('maps stats and uses local favorite count', async () => {
    window.localStorage.setItem('game-library-favorites', JSON.stringify(['game-1', 'game-9']))
    getMock.mockResolvedValue({
      data: {
        total_games: 3,
        total_downloads: 7,
        recent_games: [baseGame],
        popular_games: [{ ...baseGame, public_id: 'game-9' }],
        pending_reviews: 2,
      },
    })

    const result = await gamesService.getStats()

    expect(result.total_games).toBe(3)
    expect(result.favorite_count).toBe(2)
    expect(result.recent_games[0]?.isFavorite).toBe(true)
    expect(result.popular_games[0]?.isFavorite).toBe(true)
    expect(result.pending_reviews).toBe(2)
  })

  it('serializes aggregate relation fields only when present', async () => {
    putMock.mockResolvedValue({
      data: {
        game: baseGame,
      },
    })

    await gamesService.updateGameAggregate('game-1', {
      game: {
        title: 'Game One',
        visibility: 'public',
      },
      assets: {
        files: [],
        delete_assets: [],
        screenshot_order_asset_uids: [],
        video_order_asset_uids: [],
      },
    })

    expect(putMock).toHaveBeenCalledTimes(1)
    expect(putMock.mock.calls[0]?.[0]).toBe('/games/game-1/aggregate')
    expect(putMock.mock.calls[0]?.[1]).toEqual({
      game: {
        title: 'Game One',
        title_alt: null,
        visibility: 'public',
        summary: null,
        release_date: null,
        engine: null,
        cover_image: null,
        banner_image: null,
        needs_review: false,
      },
      assets: {
        files: [],
        delete_assets: [],
        screenshot_order_asset_uids: [],
        video_order_asset_uids: [],
      },
    })
    expect(putMock.mock.calls[0]?.[1]?.game).not.toHaveProperty('series_id')
    expect(putMock.mock.calls[0]?.[1]?.game).not.toHaveProperty('platform_ids')
    expect(putMock.mock.calls[0]?.[1]?.game).not.toHaveProperty('developer_ids')
    expect(putMock.mock.calls[0]?.[1]?.game).not.toHaveProperty('publisher_ids')
    expect(putMock.mock.calls[0]?.[1]?.game).not.toHaveProperty('tag_ids')
  })

  it('preserves explicit aggregate clear semantics for relation fields', async () => {
    putMock.mockResolvedValue({
      data: {
        game: baseGame,
      },
    })

    await gamesService.updateGameAggregate('game-1', {
      game: {
        title: 'Game One',
        series_id: null,
        developer_ids: [],
      },
      assets: {
        files: [],
        delete_assets: [],
        screenshot_order_asset_uids: [],
        video_order_asset_uids: [],
      },
    })

    expect(putMock.mock.calls[0]?.[1]).toMatchObject({
      game: {
        title: 'Game One',
        series_id: null,
        developer_ids: [],
      },
    })
  })
})
