import { del, get, post, put } from './api'
import type {
  ApiEnvelope,
  ApiPageEnvelope,
  Developer,
  GameDetail,
  GameDetailDto,
  GameFileEntry,
  GameListItem,
  GameListItemDto,
  GameListQuery,
  GameSort,
  GameStats,
  GameVersion,
  GameWriteRequest,
  Platform,
  Publisher,
  Series,
  TimelineGame,
  TimelineGameResponse,
} from './types'
import {
  applyFavorite,
  getFavoriteCount,
  readFavorites,
  writeFavorites,
} from './game-list-helpers'

interface MetadataApiItem {
  id: number
  name: string
  slug: string
  sort_order: number
  created_at: string
}

interface GameStatsApiResponse {
  total_games: number
  total_downloads: number
  recent_games: GameListItemDto[]
  popular_games: GameListItemDto[]
  pending_reviews: number
}

interface TimelinePaginationApi {
  limit: number
  from: string
  to: string
  hasMore: boolean
  nextCursor: string
}

interface TimelineGamesApiResponse {
  data: TimelineGameResponse[]
  pagination: TimelinePaginationApi
}

interface AggregateFileInput {
  id?: number
  file_path: string
  label?: string | null
  notes?: string | null
  sort_order: number
}

interface AggregateDeleteAssetInput {
  asset_type: 'cover' | 'banner' | 'screenshot' | 'video'
  path: string
  asset_id?: number
  asset_uid?: string
}

interface AggregateUpdateInput {
  game: Partial<GameWriteRequest>
  files: AggregateFileInput[]
  delete_assets: AggregateDeleteAssetInput[]
  screenshot_order_asset_uids: string[]
  video_order_asset_uids: string[]
}

interface AggregateUpdateApiResponse {
  game: GameListItemDto
  warnings?: {
    asset_delete_paths?: string[]
  }
}

type TimelineGamesResult = {
  data: TimelineGame[]
  hasMore: boolean
  nextCursor: string | null
  from: string | null
  to: string | null
}

type GameFileReleaseSource = Pick<GameFileEntry, 'source_created_at' | 'created_at'>

function annotateFavorite<T extends { public_id: string }>(item: T, favoriteIds?: Set<string>) {
  return applyFavorite(item, favoriteIds)
}

function sortByOrder<T extends { sort_order: number }>(items: T[]): T[] {
  return [...items].sort((a, b) => a.sort_order - b.sort_order)
}

function normalizeGameDetail(item: GameDetailDto): GameDetail {
  const favoriteIds = new Set(readFavorites())
  return annotateFavorite({
    ...item,
    preview_videos: sortByOrder(item.preview_videos || []),
    screenshots: sortByOrder(item.screenshots || []),
    files: sortByOrder(item.files || []),
    series: item.series || null,
    platforms: item.platforms || [],
    developers: item.developers || [],
    publishers: item.publishers || [],
    tags: item.tags || [],
    tag_groups: item.tag_groups || [],
  }, favoriteIds)
}

function getFileName(filePath?: string | null): string {
  const normalized = (filePath || '').trim()
  if (!normalized) return ''
  const segments = normalized.split(/[\\/]/)
  const fileName = segments[segments.length - 1] || normalized
  return fileName.replace(/\.[^./\\]+$/, '')
}

function canLaunchFromFileName(fileName?: string | null): boolean {
  const normalized = (fileName || '').trim().toLowerCase()
  return normalized.endsWith('.vhd') || normalized.endsWith('.vhdx')
}

function getGameFileReleaseDate(file: GameFileReleaseSource): string {
  return file.source_created_at || file.created_at || ''
}

function getLatestGameFileTimestamp(files: GameFileReleaseSource[]): number {
  return files.reduce((latest, file) => {
    const timestamp = Date.parse(getGameFileReleaseDate(file))
    if (Number.isNaN(timestamp)) {
      return latest
    }
    return Math.max(latest, timestamp)
  }, Number.NEGATIVE_INFINITY)
}

async function listMetadata<T extends Series | Platform | Developer | Publisher>(resource: string): Promise<T[]> {
  const response = await get<ApiEnvelope<MetadataApiItem[]>>(`/${resource}`)
  return response.data.map((item) => item as T)
}

const gamesService = {
  async getGames(params?: {
    query?: GameListQuery
    sort?: GameSort
  }): Promise<ApiPageEnvelope<GameListItem>> {
    const queryParams = new URLSearchParams()
    if (params?.query?.page) queryParams.append('page', String(params.query.page))
    if (params?.query?.limit) queryParams.append('limit', String(params.query.limit))
    if (params?.query?.search) queryParams.append('search', params.query.search)
    if (params?.query?.series) queryParams.append('series', params.query.series)
    if (params?.query?.platform) queryParams.append('platform', params.query.platform)
    if (typeof params?.query?.needs_review === 'boolean') queryParams.append('needs_review', String(params.query.needs_review))
    if (typeof params?.query?.pending === 'boolean') queryParams.append('pending', String(params.query.pending))
    if (params?.query?.tag?.length) {
      params.query.tag.forEach((tagId) => {
        queryParams.append('tag', String(tagId))
      })
    }
    if (params?.sort) {
      queryParams.append('sort', params.sort.field)
      queryParams.append('order', params.sort.order)
      if (typeof params.sort.seed === 'number') {
        queryParams.append('seed', String(params.sort.seed))
      }
    }

    const response = await get<ApiPageEnvelope<GameListItemDto>>('/games', { params: queryParams })
    const favoriteIds = new Set(readFavorites())
    let games = response.data.map((item) => annotateFavorite(item, favoriteIds))

    if (params?.query?.favorite) {
      games = games.filter((game) => favoriteIds.has(String(game.public_id)))
    }

    return {
      data: games,
      pagination: response.pagination,
    }
  },

  async getAllGames(params?: {
    query?: Omit<GameListQuery, 'page' | 'limit'>
    sort?: GameSort
    limit?: number
  }): Promise<GameListItem[]> {
    const limit = Math.max(1, Math.min(params?.limit || 100, 200))
    const allGames: GameListItem[] = []
    let page = 1

    while (true) {
      const response = await this.getGames({
        query: {
          ...params?.query,
          page,
          limit,
        },
        sort: params?.sort,
      })

      allGames.push(...response.data)
      const totalPages = response.pagination?.totalPages || 1
      if (page >= totalPages) {
        break
      }
      page += 1
    }

    return allGames
  },

  async getTimelineGames(params?: {
    years?: number
    limit?: number
    cursor?: string | null
    from?: string | null
    to?: string | null
  }): Promise<TimelineGamesResult> {
    const queryParams = new URLSearchParams()
    const years = Math.max(1, Math.min(params?.years || 2, 10))
    const limit = Math.max(1, Math.min(params?.limit || 60, 100))
    queryParams.append('years', String(years))
    queryParams.append('limit', String(limit))
    if (params?.cursor) queryParams.append('cursor', params.cursor)
    if (params?.from) queryParams.append('from', params.from)
    if (params?.to) queryParams.append('to', params.to)
    const response = await get<TimelineGamesApiResponse>('/games/timeline', { params: queryParams })
    const favoriteIds = new Set(readFavorites())

    return {
      data: response.data.map((item) => annotateFavorite(item, favoriteIds)),
      hasMore: Boolean(response.pagination?.hasMore),
      nextCursor: response.pagination?.nextCursor || null,
      from: response.pagination?.from || null,
      to: response.pagination?.to || null,
    }
  },

  async getGame(id: string): Promise<GameDetail> {
    const response = await get<ApiEnvelope<GameDetailDto>>(`/games/${id}`)
    return normalizeGameDetail(response.data)
  },

  async createGame(data: {
    title: string
    visibility?: 'public' | 'private'
    file_path?: string
  }): Promise<GameListItem> {
    const payload: GameWriteRequest = {
      title: data.title,
      title_alt: null,
      visibility: data.visibility ?? 'public',
      summary: null,
      release_date: null,
      engine: null,
      cover_image: null,
      banner_image: null,
      needs_review: false,
      series_id: null,
      platform_ids: [],
      developer_ids: [],
      publisher_ids: [],
      tag_ids: [],
    }
    const response = await post<ApiEnvelope<GameListItemDto>>('/games', payload)
    return annotateFavorite(response.data)
  },

  async updateGame(id: string, data: Partial<GameWriteRequest>): Promise<GameListItem> {
    const payload: GameWriteRequest = {
      title: data.title || '',
      title_alt: data.title_alt ?? null,
      visibility: data.visibility ?? 'public',
      summary: data.summary ?? null,
      release_date: data.release_date ?? null,
      engine: data.engine ?? null,
      cover_image: data.cover_image ?? null,
      banner_image: data.banner_image ?? null,
      needs_review: data.needs_review ?? false,
      preview_video_asset_uid: data.preview_video_asset_uid ?? null,
      series_id: data.series_id ?? null,
      platform_ids: data.platform_ids || [],
      developer_ids: data.developer_ids || [],
      publisher_ids: data.publisher_ids || [],
      tag_ids: data.tag_ids || [],
    }
    const response = await put<ApiEnvelope<GameListItemDto>>(`/games/${id}`, payload)
    return annotateFavorite(response.data)
  },

  async updateGameAggregate(id: string, data: AggregateUpdateInput): Promise<{ game: GameListItem; warnings: string[] }> {
    const payload = {
      title: data.game.title || '',
      title_alt: data.game.title_alt ?? null,
      visibility: data.game.visibility ?? 'public',
      summary: data.game.summary ?? null,
      release_date: data.game.release_date ?? null,
      engine: data.game.engine ?? null,
      cover_image: data.game.cover_image ?? null,
      banner_image: data.game.banner_image ?? null,
      needs_review: data.game.needs_review ?? false,
      preview_video_asset_uid: data.game.preview_video_asset_uid ?? null,
      series_id: data.game.series_id ?? null,
      platform_ids: data.game.platform_ids || [],
      developer_ids: data.game.developer_ids || [],
      publisher_ids: data.game.publisher_ids || [],
      tag_ids: data.game.tag_ids || [],
      files: data.files.map((item) => ({
        id: item.id,
        file_path: item.file_path,
        label: item.label ?? null,
        notes: item.notes ?? null,
        sort_order: item.sort_order,
      })),
      delete_assets: data.delete_assets.map((item) => ({
        asset_type: item.asset_type,
        path: item.path,
        asset_id: item.asset_id,
        asset_uid: item.asset_uid,
      })),
      screenshot_order_asset_uids: data.screenshot_order_asset_uids,
      video_order_asset_uids: data.video_order_asset_uids,
    }

    const response = await put<ApiEnvelope<AggregateUpdateApiResponse>>(`/games/${id}/aggregate`, payload)
    const warnings = response.data.warnings?.asset_delete_paths || []
    return {
      game: annotateFavorite(response.data.game),
      warnings,
    }
  },

  async createGameFile(gameId: string, data: {
    file_path: string
    label?: string | null
    notes?: string | null
    sort_order: number
  }): Promise<GameFileEntry> {
    const response = await post<ApiEnvelope<GameFileEntry>>(`/games/${gameId}/files`, data)
    return response.data
  },

  async updateGameFile(gameId: string, fileId: string, data: {
    file_path: string
    label?: string | null
    notes?: string | null
    sort_order: number
  }): Promise<GameFileEntry> {
    const response = await put<ApiEnvelope<GameFileEntry>>(`/games/${gameId}/files/${fileId}`, data)
    return response.data
  },

  async deleteGameFile(gameId: string, fileId: string): Promise<void> {
    await del<ApiEnvelope<void>>(`/games/${gameId}/files/${fileId}`)
  },

  async deleteGame(id: string): Promise<void> {
    await del<ApiEnvelope<void>>(`/games/${id}`)
  },

  async getGameVersions(gameId: string): Promise<GameVersion[]> {
    const game = await this.getGame(gameId)
    const files = [...game.files].sort((a, b) => a.sort_order - b.sort_order)
    const latestTimestamp = getLatestGameFileTimestamp(files)

    return files.map((file, index) => ({
      id: String(file.id),
      gameId,
      version: file.label?.trim() || getFileName(file.file_name) || `文件 ${index + 1}`,
      releaseDate: getGameFileReleaseDate(file),
      size: file.size_bytes ?? 0,
      isLatest: !Number.isNaN(Date.parse(getGameFileReleaseDate(file))) && Date.parse(getGameFileReleaseDate(file)) === latestTimestamp,
      canLaunch: canLaunchFromFileName(file.file_name),
      downloadUrl: `/api/games/${gameId}/files/${file.id}/download`,
      launchScriptUrl: `/api/games/${gameId}/files/${file.id}/launch-script`,
      changelog: file.notes || undefined,
    }))
  },

  async toggleFavorite(gameId: string): Promise<{ isFavorite: boolean }> {
    const favorites = new Set(readFavorites())
    if (favorites.has(gameId)) {
      favorites.delete(gameId)
    } else {
      favorites.add(gameId)
    }
    const ids = Array.from(favorites)
    writeFavorites(ids)
    return { isFavorite: ids.includes(gameId) }
  },

  async getStats(): Promise<GameStats> {
    const response = await get<ApiEnvelope<GameStatsApiResponse>>('/games/stats')
    const favoriteIds = new Set(readFavorites())
    return {
      total_games: response.data.total_games,
      total_downloads: response.data.total_downloads,
      recent_games: response.data.recent_games.map((item) => annotateFavorite(item, favoriteIds)),
      popular_games: response.data.popular_games.map((item) => annotateFavorite(item, favoriteIds)),
      favorite_count: getFavoriteCount(),
      pending_reviews: response.data.pending_reviews,
    }
  },

  async addGameFromFile(filePath: string): Promise<{ success: GameListItem[]; failed: never[] }> {
    const fileName = filePath.split('/').pop() || filePath.split('\\').pop() || 'Unknown Game'
    const title = fileName.replace(/\.[^/.]+$/, '')
    const game = await this.createGame({ title, file_path: filePath })
    return { success: [game], failed: [] }
  },

  async getAllSeries(): Promise<Series[]> {
    return listMetadata<Series>('series')
  },

  async getAllPlatforms(): Promise<Platform[]> {
    return listMetadata<Platform>('platforms')
  },

  async getAllDevelopers(): Promise<Developer[]> {
    return listMetadata<Developer>('developers')
  },

  async getAllPublishers(): Promise<Publisher[]> {
    return listMetadata<Publisher>('publishers')
  },
}

export default gamesService
