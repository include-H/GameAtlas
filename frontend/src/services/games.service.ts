import { get, post, put, del } from './api'
import type {
  ApiResponse,
  Developer,
  Game,
  GameFileEntry,
  GameFilter,
  GameInput,
  GameTagGroup,
  GameSort,
  GameStats,
  GameVersion,
  PaginatedResponse,
  Platform,
  Publisher,
  Series,
  Tag,
} from './types'
import {
  applyFavorite,
  getFavoriteCount,
  mapGameListItem,
  readFavorites,
  type GameListApiItem,
  writeFavorites,
} from './game-list-helpers'

interface MetadataApiItem {
  id: number
  name: string
  slug: string
  sort_order: number
  created_at: string
}

interface GameFileApiItem {
  id: number
  game_id: number
  file_name: string
  file_path?: string
  label: string | null
  notes: string | null
  size_bytes: number | null
  sort_order: number
  source_created_at: string | null
  created_at: string
  updated_at: string
}

interface ScreenshotApiItem {
  id: number
  asset_uid: string
  path: string
  sort_order: number
}

interface GameDetailApiItem extends GameListApiItem {
  wiki_content: string | null
  wiki_content_html: string | null
  preview_videos: ScreenshotApiItem[]
  screenshots: ScreenshotApiItem[]
  series: MetadataApiItem[]
  platforms: MetadataApiItem[]
  developers: MetadataApiItem[]
  publishers: MetadataApiItem[]
  tags: Tag[]
  tag_groups: GameTagGroup[]
  files: GameFileApiItem[]
}

interface GameStatsApiResponse {
  total_games: number
  total_downloads: number
  total_size: number
  recent_games: GameListApiItem[]
  popular_games: GameListApiItem[]
  pending_reviews: number
}

interface TimelineGameApiItem {
  id: number
  title: string
  release_date: string | null
  cover_image: string | null
}

interface TimelinePaginationApi {
  limit: number
  from: string
  to: string
  hasMore: boolean
  nextCursor: string
}

interface TimelineGamesApiResponse {
  data: TimelineGameApiItem[]
  pagination: TimelinePaginationApi
}

type TimelineGamesResult = {
  data: Game[]
  hasMore: boolean
  nextCursor: string | null
  from: string | null
  to: string | null
}

function mapMetadataItem<T extends Series | Platform | Developer | Publisher>(item: MetadataApiItem): T {
  return item as T
}

function mapFile(file: GameFileApiItem): GameFileEntry {
  return { ...file }
}

function mapGameDetail(item: GameDetailApiItem): Game {
  const screenshots = [...item.screenshots].sort((a, b) => a.sort_order - b.sort_order)
  const files = [...item.files].sort((a, b) => a.sort_order - b.sort_order)
  const favoriteIds = new Set(readFavorites())

  return applyFavorite({
    ...mapGameListItem(item, favoriteIds),
    series: item.series.map((value) => mapMetadataItem<Series>(value)),
    developers: item.developers.map((value) => mapMetadataItem<Developer>(value)),
    publishers: item.publishers.map((value) => mapMetadataItem<Publisher>(value)),
    tags: item.tags || [],
    tag_groups: item.tag_groups || [],
    platforms: item.platforms.map((value) => value.name),
    platform: item.platforms[0]?.name,
    wiki_content: item.wiki_content,
    wiki_content_html: item.wiki_content_html,
    preview_videos: [...(item.preview_videos || [])].sort((a, b) => a.sort_order - b.sort_order),
    screenshot_items: screenshots,
    files: files.map(mapFile),
  }, favoriteIds)
}

function mapPlatformValues(values?: Array<number | string>): number[] {
  if (!values || values.length === 0) return []
  return values
    .map((value) => Number(value))
    .filter((value) => !Number.isNaN(value))
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

type GameFileReleaseSource = Pick<GameFileEntry, 'source_created_at' | 'created_at'>

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

function mapTimelineItem(item: TimelineGameApiItem, favoriteIds?: Set<string>): Game {
  return applyFavorite({
    id: item.id,
    title: item.title,
    release_date: item.release_date,
    cover_image: item.cover_image,
    downloads: 0,
    created_at: '',
    updated_at: '',
    preview_videos: [],
    screenshot_items: [],
    files: [],
  }, favoriteIds)
}

async function listMetadata<T extends Series | Platform | Developer | Publisher>(resource: string): Promise<T[]> {
  const response = await get<ApiResponse<MetadataApiItem[]>>(`/${resource}`)
  return (response.data || []).map((item) => mapMetadataItem<T>(item))
}

const gamesService = {
  async getGames(params?: {
    page?: number
    pageSize?: number
    filter?: GameFilter
    sort?: GameSort
  }): Promise<PaginatedResponse<Game>> {
    const queryParams = new URLSearchParams()
    if (params?.page) queryParams.append('page', String(params.page))
    if (params?.pageSize) queryParams.append('limit', String(params.pageSize))
    if (params?.filter?.search) queryParams.append('search', params.filter.search)
    if (params?.filter?.series) queryParams.append('series', params.filter.series)
    if (params?.filter?.platform) queryParams.append('platform', params.filter.platform)
    if (params?.filter?.status === 'pending-review') queryParams.append('needs_review', 'true')
    if (params?.filter?.tag_ids?.length) {
      params.filter.tag_ids.forEach((tagId) => {
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

    const response = await get<PaginatedResponse<GameListApiItem>>('/games', { params: queryParams })
    let games = response.data.map((item) => mapGameListItem(item))

    if (params?.filter?.favorite) {
      const favoriteIds = new Set(readFavorites())
      games = games.filter((game) => favoriteIds.has(String(game.id)))
    }

    return {
      data: games,
      pagination: response.pagination,
    }
  },

  async getAllGames(params?: {
    filter?: GameFilter
    sort?: GameSort
    pageSize?: number
  }): Promise<Game[]> {
    const pageSize = Math.max(1, Math.min(params?.pageSize || 100, 200))
    const allGames: Game[] = []
    let page = 1

    while (true) {
      const response = await this.getGames({
        page,
        pageSize,
        filter: params?.filter,
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
      data: (response.data || []).map((item) => mapTimelineItem(item, favoriteIds)),
      hasMore: Boolean(response.pagination?.hasMore),
      nextCursor: response.pagination?.nextCursor || null,
      from: response.pagination?.from || null,
      to: response.pagination?.to || null,
    }
  },

  async getGame(id: string): Promise<Game> {
    const response = await get<ApiResponse<GameDetailApiItem>>(`/games/${id}`)
    return mapGameDetail(response.data)
  },

  async createGame(data: { title: string; visibility?: 'public' | 'private'; file_path?: string }): Promise<Game> {
    const payload = {
      title: data.title,
      title_alt: null,
      visibility: data.visibility ?? 'public',
      summary: null,
      release_date: null,
      engine: null,
      cover_image: null,
      banner_image: null,
      needs_review: false,
      series_ids: [],
      platform_ids: [],
      developer_ids: [],
      publisher_ids: [],
      tag_ids: [],
    }
    const response = await post<ApiResponse<GameListApiItem>>('/games', payload)
    return mapGameListItem(response.data)
  },

  async updateGame(id: string, data: Partial<GameInput>): Promise<Game> {
    const payload = {
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
      series_ids: data.series || [],
      platform_ids: mapPlatformValues(data.platforms),
      developer_ids: data.developers || [],
      publisher_ids: data.publishers || [],
      tag_ids: data.tag_ids || [],
    }
    const response = await put<ApiResponse<GameListApiItem>>(`/games/${id}`, payload)
    return mapGameListItem(response.data)
  },

  async createGameFile(gameId: string, data: {
    file_path: string
    label?: string | null
    notes?: string | null
    sort_order: number
  }): Promise<GameFileEntry> {
    const response = await post<ApiResponse<GameFileApiItem>>(`/games/${gameId}/files`, data)
    return mapFile(response.data)
  },

  async updateGameFile(gameId: string, fileId: string, data: {
    file_path: string
    label?: string | null
    notes?: string | null
    sort_order: number
  }): Promise<GameFileEntry> {
    const response = await put<ApiResponse<GameFileApiItem>>(`/games/${gameId}/files/${fileId}`, data)
    return mapFile(response.data)
  },

  async deleteGameFile(gameId: string, fileId: string): Promise<void> {
    await del<ApiResponse<void>>(`/games/${gameId}/files/${fileId}`)
  },

  async deleteGame(id: string): Promise<void> {
    await del<ApiResponse<void>>(`/games/${id}`)
  },

  async getGameVersions(gameId: string): Promise<GameVersion[]> {
    const game = await this.getGame(gameId)
    const files = [...(game.files || [])].sort((a, b) => a.sort_order - b.sort_order)
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
    const response = await get<ApiResponse<GameStatsApiResponse>>('/games/stats')
    return {
      total_games: response.data.total_games,
      total_downloads: response.data.total_downloads,
      total_size: response.data.total_size,
      recent_games: response.data.recent_games.map((item) => mapGameListItem(item)),
      popular_games: response.data.popular_games.map((item) => mapGameListItem(item)),
      favorite_count: getFavoriteCount(),
      pending_reviews: response.data.pending_reviews,
    }
  },

  async addGameFromFile(filePath: string): Promise<{ success: Game[]; failed: never[] }> {
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
