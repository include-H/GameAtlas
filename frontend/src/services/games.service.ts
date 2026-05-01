import { del, get, post, put } from './api'
import { buildApiUrl } from './api-url'
import type {
  ApiEnvelope,
  AdminGameDetail,
  AdminGameDetailDto,
  ApiPageEnvelope,
  GameDetail,
  GameDetailDto,
  GameAggregateUpdateRequest,
  GameCreateRequest,
  GameFileEntry,
  GameListItem,
  GameListItemDto,
  GameListQuery,
  GameSortQuery,
  GameStats,
  GameVersion,
  TimelineGame,
  TimelineGameResponse,
} from './types'
import { isAdminGameDetail } from './types'

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
  sort?: GameSortQuery
}): URLSearchParams {
  const queryParams = new URLSearchParams()
  if (params?.query?.page) queryParams.append('page', String(params.query.page))
  if (params?.query?.limit) queryParams.append('limit', String(params.query.limit))
  const search = typeof params?.query?.search === 'string' ? params.query.search.trim() : ''
  if (search) queryParams.append('search', search)
  if (params?.query?.series) queryParams.append('series', String(params.query.series))
  // 2026-05-01: forward route-owned favorite transport values verbatim so the backend
  // decoder decides validity. Do not silently drop favorite=false or unknown strings here,
  // otherwise the client hides a bad query instead of surfacing the backend 400 contract.
  if (typeof params?.query?.favorite_raw === 'string' && params.query.favorite_raw.trim().length > 0) {
    queryParams.append('favorite', params.query.favorite_raw.trim())
  } else if (typeof params?.query?.favorite === 'boolean') {
    queryParams.append('favorite', String(params.query.favorite))
  }
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
    if (typeof params.sort.field === 'string' && params.sort.field.length > 0) {
      queryParams.append('sort', params.sort.field)
    }
    if (typeof params.sort.order === 'string' && params.sort.order.length > 0) {
      queryParams.append('order', params.sort.order)
    }
    if (typeof params.sort.seed === 'number') {
      queryParams.append('seed', String(params.sort.seed))
    }
  }
  return queryParams
}

async function fetchGamesPage(params?: {
  query?: GameListQuery
  sort?: GameSortQuery
}): Promise<ApiPageEnvelope<GameListItemDto>> {
  return get<ApiPageEnvelope<GameListItemDto>>('/games', {
    params: buildGamesQueryParams(params),
  })
}

function normalizeGameListItem(item: GameListItemDto): GameListItem {
  return {
    ...item,
    isFavorite: item.is_favorite,
  }
}

function normalizeTimelineGame(item: TimelineGameResponse): TimelineGame {
  return { ...item }
}

function readTimelinePagination(response: TimelineGamesApiResponse): TimelinePaginationApi {
  const pagination = response.pagination
  if (!pagination) {
    throw new Error('timeline response missing pagination')
  }
  return pagination
}

function normalizeGameDetail(item: GameDetailDto): GameDetail {
  return {
    ...item,
    isFavorite: item.is_favorite,
    preview_videos: item.preview_videos,
    screenshots: item.screenshots,
    files: item.files,
    series: item.series,
    developers: item.developers,
    publishers: item.publishers,
    tags: item.tags,
    tag_groups: item.tag_groups,
  }
}

// Keep a dedicated admin detail normalization path for edit flows:
// getAdminGameDetail uses this guard to fail fast when /games/:publicId
// returns a public payload without resolved file_path values.
function normalizeAdminGameDetail(item: AdminGameDetailDto): AdminGameDetail {
  const detail = normalizeGameDetail(item)
  // 2026-04-06: /games/:publicId still serves public/admin payloads from one route.
  // Impact: edit flows must fail fast if auth/session state no longer yields file_path,
  // instead of silently treating a public payload as editable admin data.
  if (!isAdminGameDetail(detail)) {
    throw new Error('admin game detail requires resolved file_path values')
  }
  return detail
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
  // 2026-04-10: keep the version date fallback as an explicit product rule.
  // Impact: download/version UI shows source_created_at first, then falls back
  // to the file record's created_at when filesystem creation time is unavailable.
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

const gamesService = {
  async getGames(params?: {
    query?: GameListQuery
    sort?: GameSortQuery
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
    sort?: GameSortQuery
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
    // 2026-04-07: timeline route owns a small fixed query surface.
    // Impact: the client keeps sending only supported years/limit ranges,
    // but transport validation now belongs to the backend instead of hidden fallback coercion.
    const queryParams = new URLSearchParams()
    const years = Math.max(1, Math.min(params?.years || 2, 10))
    const limit = Math.max(1, Math.min(params?.limit || 60, 100))
    queryParams.append('years', String(years))
    queryParams.append('limit', String(limit))
    if (params?.cursor) queryParams.append('cursor', params.cursor)
    if (params?.from) queryParams.append('from', params.from)
    if (params?.to) queryParams.append('to', params.to)
    const response = await get<TimelineGamesApiResponse>('/games/timeline', { params: queryParams })
    const pagination = readTimelinePagination(response)

    return {
      data: response.data.map((item) => normalizeTimelineGame(item)),
      // 2026-04-07: /games/timeline always returns pagination metadata with the read window.
      // Impact: missing pagination is a broken backend payload, not a valid "no more data" state.
      hasMore: pagination.hasMore,
      nextCursor: pagination.nextCursor || null,
      from: pagination.from || null,
      to: pagination.to || null,
    }
  },

  // Use the generic detail reader for page/store flows that can legally render either
  // public or admin payloads from /games/:publicId.
  async getGameDetail(id: string): Promise<GameDetail> {
    const response = await get<ApiEnvelope<GameDetailDto>>(`/games/${id}`)
    return normalizeGameDetail(response.data)
  },

  // 2026-05-01: keep an explicit admin detail entry point even though getGameDetail exists.
  // Impact: edit/bootstrap flows can require resolved admin file_path data at the service
  // boundary and fail fast there, instead of carrying a public/admin union deeper into form code.
  async getAdminGameDetail(id: string): Promise<AdminGameDetail> {
    const response = await get<ApiEnvelope<AdminGameDetailDto>>(`/games/${id}`)
    return normalizeAdminGameDetail(response.data)
  },

  async createGame(data: GameCreateRequest): Promise<GameListItem> {
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
}

export default gamesService
