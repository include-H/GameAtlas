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

import gamesService, { mapGameVersions } from './games.service'

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
  primary_screenshot: null,
  screenshot_count: 0,
  file_count: 0,
  developer_count: 0,
  publisher_count: 0,
  platform_count: 0,
  is_favorite: false,
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
  })

  it('loads games, builds query params and maps backend favorite state', async () => {
    getMock.mockResolvedValue({
      data: [
        {
          ...baseGame,
          is_favorite: true,
        },
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
        series: 12,
        platform: 3,
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
    expect(params.get('series')).toBe('12')
    expect(params.get('platform')).toBe('3')
    expect(params.get('pending')).toBe('false')
    expect(params.get('favorite')).toBeNull()
    expect(params.getAll('tag')).toEqual(['3', '7'])
    expect(params.get('sort')).toBe('updated_at')
    expect(params.get('order')).toBe('desc')
    expect(params.get('seed')).toBe('9')
  })

  it('passes favorite filter through to the backend', async () => {
    getMock.mockResolvedValue({
      data: [{ ...baseGame, id: 2, public_id: 'game-2', is_favorite: true }],
      pagination: {
        page: 1,
        limit: 20,
        total: 1,
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
    const params = getMock.mock.calls[0]?.[1]?.params as URLSearchParams
    expect(params.get('favorite')).toBe('true')
  })

  it('does not send favorite=false because backend has no negative favorite filter', async () => {
    getMock.mockResolvedValue({
      data: [{ ...baseGame, id: 2, public_id: 'game-2', is_favorite: true }],
      pagination: {
        page: 1,
        limit: 20,
        total: 1,
        totalPages: 1,
      },
    })

    await gamesService.getGames({
      query: {
        favorite: false,
      },
    })

    const params = getMock.mock.calls[0]?.[1]?.params as URLSearchParams
    expect(params.get('favorite')).toBeNull()
  })

  it('returns delete warnings when game removal leaves cleanup tasks', async () => {
    delMock.mockResolvedValue({
      data: {
        deleted: true,
        warnings: {
          asset_delete_paths: ['/assets/bad-cover.png'],
        },
      },
    })

    const result = await gamesService.deleteGame('game-1')

    expect(delMock).toHaveBeenCalledWith('/games/game-1')
    expect(result).toEqual({
      warnings: ['/assets/bad-cover.png'],
    })
  })

  it('serializes quick-create payload without aggregate-only fields', async () => {
    postMock.mockResolvedValue({
      data: baseGame,
    })

    await gamesService.createGame({
      title: 'Quick Create',
      visibility: 'private',
    })

    expect(postMock).toHaveBeenCalledWith('/games', {
      title: 'Quick Create',
      visibility: 'private',
    })
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

  it('loads all favorite games from backend pagination', async () => {
    getMock
      .mockResolvedValueOnce({
        data: [
          { ...baseGame, id: 2, public_id: 'game-2', is_favorite: true },
        ],
        pagination: { page: 1, limit: 1, total: 2, totalPages: 2 },
      })
      .mockResolvedValueOnce({
        data: [{ ...baseGame, id: 3, public_id: 'game-3', is_favorite: true }],
        pagination: { page: 2, limit: 1, total: 2, totalPages: 2 },
      })

    const result = await gamesService.getAllGames({
      query: { favorite: true },
      limit: 1,
    })

    expect(result.map((item) => item.public_id)).toEqual(['game-2', 'game-3'])
    expect(getMock).toHaveBeenCalledTimes(2)
  })

  it('maps game files to version metadata using backend file order', () => {
    const result = mapGameVersions({
      public_id: 'game-1',
      files: [
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
      ],
    })

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

  it('keeps backend preview video order', async () => {
    getMock.mockResolvedValue({
      data: {
        ...baseGame,
        preview_videos: [
          {
            id: 12,
            asset_uid: 'video-primary',
            path: '/assets/video-primary.mp4',
            sort_order: 5,
          },
          {
            id: 11,
            asset_uid: 'video-first',
            path: '/assets/video-first.mp4',
            sort_order: 0,
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

    expect(result.preview_videos.map((item) => item.asset_uid)).toEqual(['video-primary', 'video-first'])
  })

  it('keeps preview video empty when there are no videos', async () => {
    getMock.mockResolvedValue({
      data: {
        ...baseGame,
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

    expect(result.preview_videos).toEqual([])
  })

  it('sets favorite state through backend endpoints', async () => {
    putMock.mockResolvedValueOnce({
      data: {
        is_favorite: true,
      },
    })
    delMock.mockResolvedValueOnce({
      data: {
        is_favorite: false,
      },
    })

    await expect(gamesService.setFavorite('game-1', true)).resolves.toEqual({ isFavorite: true })
    await expect(gamesService.setFavorite('game-1', false)).resolves.toEqual({ isFavorite: false })

    expect(putMock).toHaveBeenCalledWith('/games/game-1/favorite', {})
    expect(delMock).toHaveBeenCalledWith('/games/game-1/favorite')
  })

  it('maps stats and uses backend favorite count', async () => {
    getMock.mockResolvedValue({
      data: {
        total_games: 3,
        total_downloads: 7,
        recent_games: [{ ...baseGame, is_favorite: true }],
        popular_games: [{ ...baseGame, public_id: 'game-9', is_favorite: true }],
        favorite_count: 2,
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

  it('serializes aggregate relation fields as full replacement payload', async () => {
    putMock.mockResolvedValue({
      data: {
        game: baseGame,
      },
    })

    await gamesService.updateGameAggregate('game-1', {
      game: {
        title: 'Game One',
        visibility: 'public',
        series_id: null,
        platform_ids: [],
        developer_ids: [],
        publisher_ids: [],
        tag_ids: [],
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
        visibility: 'public',
        series_id: null,
        platform_ids: [],
        developer_ids: [],
        publisher_ids: [],
        tag_ids: [],
      },
      assets: {
        files: [],
        delete_assets: [],
        screenshot_order_asset_uids: [],
        video_order_asset_uids: [],
      },
    })
  })
})
