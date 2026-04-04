import { del, get, post, put } from './api'
import { buildApiUrl } from './api-url'
import type {
  ApiEnvelope,
  ApiPageEnvelope,
  Developer,
  GameDetail,
  GameDetailDto,
  GameAggregateUpdateRequest,
  GameFileEntry,
  GameListItem,
  GameListItemDto,
  GameListQuery,
  GameSort,
  GameStats,
  GameVersion,
  Platform,
  Publisher,
  Series,
  TimelineGame,
  TimelineGameResponse,
} from './types'

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
  favorite_count: number
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

interface AggregateUpdateApiResponse {
  game: GameListItemDto
  warnings?: {
    asset_delete_paths?: string[]
  }
}

interface DeleteGameApiResponse {
  deleted: boolean
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

function buildGamesQueryParams(params?: {
  query?: GameListQuery
  sort?: GameSort
}): URLSearchParams {
  const queryParams = new URLSearchParams()
  if (params?.query?.page) queryParams.append('page', String(params.query.page))
  if (params?.query?.limit) queryParams.append('limit', String(params.query.limit))
  if (params?.query?.search) queryParams.append('search', params.query.search)
  if (params?.query?.series) queryParams.append('series', String(params.query.series))
  if (params?.query?.platform) queryParams.append('platform', String(params.query.platform))
  if (params?.query?.favorite === true) queryParams.append('favorite', 'true')
  if (typeof params?.query?.pending === 'boolean') queryParams.append('pending', String(params.query.pending))
  if (params?.query?.pending_issue) queryParams.append('pending_issue', params.query.pending_issue)
  if (typeof params?.query?.pending_include_ignored === 'boolean') queryParams.append('pending_include_ignored', String(params.query.pending_include_ignored))
  if (typeof params?.query?.pending_severe === 'boolean') queryParams.append('pending_severe', String(params.query.pending_severe))
  if (typeof params?.query?.pending_recent_days === 'number' && params.query.pending_recent_days > 0) {
    queryParams.append('pending_recent_days', String(params.query.pending_recent_days))
  }
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
  return queryParams
}

async function fetchGamesPage(params?: {
  query?: GameListQuery
  sort?: GameSort
}): Promise<ApiPageEnvelope<GameListItemDto>> {
  return get<ApiPageEnvelope<GameListItemDto>>('/games', {
    params: buildGamesQueryParams(params),
  })
}

function normalizeGameListItem(item: GameListItemDto): GameListItem {
  return {
    ...item,
    isFavorite: Boolean(item.is_favorite),
  }
}

function normalizeTimelineGame(item: TimelineGameResponse): TimelineGame {
  return { ...item }
}

function normalizeGameDetail(item: GameDetailDto): GameDetail {
  return {
    ...item,
    isFavorite: Boolean(item.is_favorite),
    preview_videos: item.preview_videos,
    screenshots: item.screenshots,
    files: item.files,
    series: item.series,
    platforms: item.platforms,
    developers: item.developers,
    publishers: item.publishers,
    tags: item.tags,
    tag_groups: item.tag_groups,
  }
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

export function mapGameVersions(game: Pick<GameDetail, 'public_id' | 'files'>): GameVersion[] {
  const gameId = game.public_id
  const files = game.files
  const latestTimestamp = getLatestGameFileTimestamp(files)

  return files.map((file, index) => ({
    id: String(file.id),
    gameId,
    version: file.label?.trim() || getFileName(file.file_name) || `文件 ${index + 1}`,
    releaseDate: getGameFileReleaseDate(file),
    size: file.size_bytes ?? 0,
    isLatest: !Number.isNaN(Date.parse(getGameFileReleaseDate(file))) && Date.parse(getGameFileReleaseDate(file)) === latestTimestamp,
    canLaunch: canLaunchFromFileName(file.file_name),
    downloadUrl: buildApiUrl(`/games/${gameId}/files/${file.id}/download`),
    launchScriptUrl: buildApiUrl(`/games/${gameId}/files/${file.id}/launch-script`),
    changelog: file.notes || undefined,
  }))
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
    const response = await fetchGamesPage(params)
    const games = response.data.map((item) => normalizeGameListItem(item))

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
      const totalPages = response.pagination.totalPages
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

    return {
      data: response.data.map((item) => normalizeTimelineGame(item)),
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
  }): Promise<GameListItem> {
    const payload = {
      title: data.title,
      visibility: data.visibility ?? 'public',
    }
    const response = await post<ApiEnvelope<GameListItemDto>>('/games', payload)
    return normalizeGameListItem(response.data)
  },

  async updateGameAggregate(id: string, data: GameAggregateUpdateRequest): Promise<{ game: GameListItem; warnings: string[] }> {
    const response = await put<ApiEnvelope<AggregateUpdateApiResponse>>(`/games/${id}/aggregate`, data)
    const warnings = response.data.warnings?.asset_delete_paths || []
    return {
      game: normalizeGameListItem(response.data.game),
      warnings,
    }
  },

  async deleteGame(id: string): Promise<{ warnings: string[] }> {
    const response = await del<ApiEnvelope<DeleteGameApiResponse>>(`/games/${id}`)
    return {
      warnings: response.data.warnings?.asset_delete_paths || [],
    }
  },

  async setFavorite(gameId: string, isFavorite: boolean): Promise<{ isFavorite: boolean }> {
    const response = isFavorite
      ? await put<ApiEnvelope<{ is_favorite: boolean }>>(`/games/${gameId}/favorite`, {})
      : await del<ApiEnvelope<{ is_favorite: boolean }>>(`/games/${gameId}/favorite`)
    return {
      isFavorite: Boolean(response.data.is_favorite),
    }
  },

  async getStats(): Promise<GameStats> {
    const response = await get<ApiEnvelope<GameStatsApiResponse>>('/games/stats')
    return {
      total_games: response.data.total_games,
      total_downloads: response.data.total_downloads,
      recent_games: response.data.recent_games.map((item) => normalizeGameListItem(item)),
      popular_games: response.data.popular_games.map((item) => normalizeGameListItem(item)),
      favorite_count: response.data.favorite_count,
      pending_reviews: response.data.pending_reviews,
    }
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
