import { del, get, post, put } from './api'
import { buildApiUrl } from './api-url'
import type {
  ApiEnvelope,
  AdminGameDetail,
  AdminGameDetailDto,
  GameDetail,
  GameDetailDto,
  GameAggregateUpdateRequest,
  GameCreateRequest,
  GameFileEntry,
  GameListItem,
  GameListItemDto,
  GameListPagination,
  GameListPageEnvelope,
  GameListQuery,
  GameSortQuery,
  GameStats,
  GameVersion,
  PendingIssueCounts,
  TimelineGame,
  TimelineGameResponse,
  VideoAssetItem,
} from './types'
import { isAdminGameDetail } from './types'

interface GameStatsApiResponse {
  total_games: number
  total_downloads: number
  recent_games: GameListItemDto[]
  recently_updated_games: GameListItemDto[]
  popular_games: GameListItemDto[]
  favorite_games: GameListItemDto[]
  favorite_count: number
  pending_reviews: number
  pending_issue_counts: PendingIssueCounts | null
}

interface TimelinePaginationApi {
  limit: number
  hasMore: boolean
  nextCursor: string
}

interface TimelineGamesApiResponse {
  data: TimelineGameResponse[]
  pagination: TimelinePaginationApi
}

interface GamePreviewVideosApiItem {
  public_id: string
  preview_videos: VideoAssetItem[]
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
  if (params?.query?.visibility) queryParams.append('visibility', params.query.visibility)
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

async function fetchGamesPage<P extends GameListPagination = GameListPagination>(params?: {
  query?: GameListQuery
  sort?: GameSortQuery
}): Promise<GameListPageEnvelope<GameListItemDto, P>> {
  return get<GameListPageEnvelope<GameListItemDto, P>>('/games', {
    params: buildGamesQueryParams(params),
  })
}

function normalizeGameListItem(item: GameListItemDto): GameListItem {
  const { is_favorite, ...rest } = item
  return { ...rest, isFavorite: is_favorite }
}

function readTimelinePagination(response: TimelineGamesApiResponse): TimelinePaginationApi {
  const pagination = response.pagination
  if (!pagination) {
    throw new Error('时间线响应缺少分页信息')
  }
  return pagination
}

function normalizeGameDetail(item: GameDetailDto): GameDetail {
  const { is_favorite, ...rest } = item
  return {
    ...rest,
    isFavorite: is_favorite,
    covers: item.covers,
    logos: item.logos,
    preview_videos: item.preview_videos,
    screenshots: item.screenshots,
    banners: item.banners,
    files: item.files,
    series: item.series,
    developers: item.developers,
    publishers: item.publishers,
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
    throw new Error('管理端游戏详情需要已解析的 file_path 值')
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
    version: file.label?.trim() || getFileName(file.file_name) || `文件 ${index + 1}`,
    releaseDate: getGameFileReleaseDate(file),
    size: file.size_bytes ?? 0,
    isLatest: !Number.isNaN(Date.parse(getGameFileReleaseDate(file))) && Date.parse(getGameFileReleaseDate(file)) === latestTimestamp,
    canLaunch: canLaunchFromFileName(file.file_name),
    downloadUrl: buildApiUrl(`/games/${gameId}/files/${file.id}/download`),
    launchScriptUrl: buildApiUrl(`/games/${gameId}/files/${file.id}/launch-script`),
  }))
}

const gamesService = {
  // P 由调用方按请求模式选择：默认 GameListPagination（无限滚动，无 total/limit）；
  // pending=true 的工作台传 PendingWorkbenchPagination 以获得 total。
  async getGames<P extends GameListPagination = GameListPagination>(params?: {
    query?: GameListQuery
    sort?: GameSortQuery
  }): Promise<GameListPageEnvelope<GameListItem, P>> {
    const response = await fetchGamesPage<P>(params)
    const games = response.data.map((item) => normalizeGameListItem(item))

    return {
      data: games,
      pagination: response.pagination,
    }
  },

  async getTimelineGames(params?: {
    limit?: number
    cursor?: string | null
  }): Promise<TimelineGamesResult> {
    const queryParams = new URLSearchParams()
    const limit = Math.max(1, Math.min(params?.limit || 60, 100))
    queryParams.append('limit', String(limit))
    if (params?.cursor) queryParams.append('cursor', params.cursor)
    const response = await get<TimelineGamesApiResponse>('/games/timeline', { params: queryParams })
    const pagination = readTimelinePagination(response)

    return {
      data: response.data,
      hasMore: pagination.hasMore,
      nextCursor: pagination.nextCursor || null,
    }
  },

  // 游戏店会话批量拉取预告片：一次请求返回多个游戏的 preview_videos，
  // 避免对每个游戏再发一次详情请求。
  async getPreviewVideos(publicIds: string[]): Promise<GamePreviewVideosApiItem[]> {
    const ids = Array.from(new Set(publicIds.map((id) => id.trim()).filter(Boolean)))
    if (ids.length === 0) return []
    const queryParams = new URLSearchParams({ public_ids: ids.join(',') })
    const response = await get<ApiEnvelope<GamePreviewVideosApiItem[]>>('/games/preview-videos', {
      params: queryParams,
    })
    return response.data ?? []
  },

  // Use the generic detail reader for page/store flows that can legally render either
  // public or admin payloads from /games/:publicId.
  async getGameDetail(id: string, signal?: AbortSignal): Promise<GameDetail> {
    const response = await get<ApiEnvelope<GameDetailDto>>(`/games/${id}`, { signal })
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
      recently_updated_games: response.data.recently_updated_games.map((item) => normalizeGameListItem(item)),
      popular_games: response.data.popular_games.map((item) => normalizeGameListItem(item)),
      favorite_games: response.data.favorite_games.map((item) => normalizeGameListItem(item)),
      favorite_count: response.data.favorite_count,
      pending_reviews: response.data.pending_reviews,
      pending_issue_counts: response.data.pending_issue_counts ?? null,
    }
  },

  async refreshFileSizes(): Promise<{ updated: number; errors: number }> {
    const response = await post<ApiEnvelope<{ updated: number; errors: number }>>('/games/refresh-sizes', {})
    return response.data
  },
}

export default gamesService
