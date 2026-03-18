import { get, post, put, del } from './api'
import type {
  ApiResponse,
  Developer,
  Game,
  GameFileEntry,
  GameFilter,
  GameInput,
  GameSort,
  GameStats,
  GameVersion,
  PaginatedResponse,
  Platform,
  Publisher,
  Series,
} from './types'

const FAVORITES_KEY = 'game-library-favorites'

interface GameListApiItem {
  id: number
  title: string
  title_alt: string | null
  summary: string | null
  release_date: string | null
  engine: string | null
  cover_image: string | null
  banner_image: string | null
  needs_review: boolean
  views: number
  downloads: number
  created_at: string
  updated_at: string
}

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
  file_path: string
  label: string | null
  notes: string | null
  size_bytes: number | null
  sort_order: number
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
  preview_video: ScreenshotApiItem | null
  preview_videos: ScreenshotApiItem[]
  screenshots: ScreenshotApiItem[]
  series: MetadataApiItem[]
  platforms: MetadataApiItem[]
  developers: MetadataApiItem[]
  publishers: MetadataApiItem[]
  files: GameFileApiItem[]
}

function readFavorites(): string[] {
  if (typeof window === 'undefined') return []
  try {
    const raw = window.localStorage.getItem(FAVORITES_KEY)
    if (!raw) return []
    const ids = JSON.parse(raw)
    return Array.isArray(ids) ? ids.map(String) : []
  } catch {
    return []
  }
}

function writeFavorites(ids: string[]) {
  if (typeof window === 'undefined') return
  window.localStorage.setItem(FAVORITES_KEY, JSON.stringify(ids))
}

function applyFavorite(game: Game): Game {
  return {
    ...game,
    isFavorite: readFavorites().includes(String(game.id)),
  }
}

function mapMetadataItem<T extends Series | Platform | Developer | Publisher>(item: MetadataApiItem): T {
  return item as T
}

function mapFile(file: GameFileApiItem): GameFileEntry {
  return { ...file }
}

function mapGameListItem(item: GameListApiItem): Game {
  return applyFavorite({
    id: item.id,
    title: item.title,
    title_alt: item.title_alt,
    summary: item.summary,
    release_date: item.release_date,
    engine: item.engine,
    cover_image: item.cover_image,
    banner_image: item.banner_image,
    needs_review: item.needs_review,
    views: item.views,
    downloads: item.downloads,
    created_at: item.created_at,
    updated_at: item.updated_at,
    screenshots: [],
    file_paths: [],
  })
}

function mapGameDetail(item: GameDetailApiItem): Game {
  const screenshots = [...item.screenshots].sort((a, b) => a.sort_order - b.sort_order)
  const files = [...item.files].sort((a, b) => a.sort_order - b.sort_order)

  return applyFavorite({
    ...mapGameListItem(item),
    series: item.series.map((value) => mapMetadataItem<Series>(value)),
    developers: item.developers.map((value) => mapMetadataItem<Developer>(value)),
    publishers: item.publishers.map((value) => mapMetadataItem<Publisher>(value)),
    platforms: item.platforms.map((value) => value.name),
    platform: item.platforms[0]?.name,
    wiki_content: item.wiki_content,
    wiki_content_html: item.wiki_content_html,
    preview_video: item.preview_video,
    preview_videos: [...(item.preview_videos || [])].sort((a, b) => a.sort_order - b.sort_order),
    screenshot_items: screenshots,
    screenshots: screenshots.map((shot) => shot.path),
    files: files.map(mapFile),
    file_path: files[0]?.file_path,
    file_paths: files.map((file) => ({
      id: file.id,
      path: file.file_path,
      label: file.label || '',
      notes: file.notes || '',
      size: file.size_bytes,
      sort_order: file.sort_order,
    })),
  })
}

function mapPlatformValues(values?: Array<number | string>): number[] {
  if (!values || values.length === 0) return []
  return values
    .map((value) => Number(value))
    .filter((value) => !Number.isNaN(value))
}

async function listMetadata<T extends Series | Platform | Developer | Publisher>(resource: string): Promise<T[]> {
  const response = await get<ApiResponse<MetadataApiItem[]>>(`/${resource}`)
  return (response.data || []).map((item) => mapMetadataItem<T>(item))
}

export const gamesService = {
  async getGames(params?: {
    page?: number
    pageSize?: number
    filter?: GameFilter
    sort?: GameSort
  }): Promise<PaginatedResponse<Game>> {
    const queryParams: Record<string, any> = {
      page: params?.page,
      limit: params?.pageSize,
    }

    if (params?.filter?.search) queryParams.search = params.filter.search
    if (params?.filter?.series) queryParams.series = params.filter.series
    if (params?.filter?.platform) queryParams.platform = params.filter.platform
    if (params?.filter?.status === 'pending-review') queryParams.needs_review = true
    if (params?.sort) {
      queryParams.sort = params.sort.field
      queryParams.order = params.sort.order
    }

    Object.keys(queryParams).forEach((key) => {
      if (queryParams[key] === undefined) delete queryParams[key]
    })

    const response = await get<PaginatedResponse<GameListApiItem>>('/games', { params: queryParams })
    let games = response.data.map(mapGameListItem)

    if (params?.filter?.favorite) {
      const favoriteIds = new Set(readFavorites())
      games = games.filter((game) => favoriteIds.has(String(game.id)))
    }

    return {
      data: games,
      pagination: response.pagination,
    }
  },

  async getGame(id: string): Promise<Game> {
    const response = await get<ApiResponse<GameDetailApiItem>>(`/games/${id}`)
    return mapGameDetail(response.data)
  },

  async createGame(data: { title: string; file_path?: string }): Promise<Game> {
    const payload = {
      title: data.title,
      title_alt: null,
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
    }
    const response = await post<ApiResponse<GameListApiItem>>('/games', payload)
    return mapGameListItem(response.data)
  },

  async updateGame(id: string, data: Partial<GameInput>): Promise<Game> {
    const payload = {
      title: data.title || '',
      title_alt: data.title_alt ?? null,
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

    return files.map((file, index) => ({
      id: String(file.id),
      gameId,
      version: file.label || `文件 ${index + 1}`,
      releaseDate: file.updated_at || file.created_at,
      size: file.size_bytes || 0,
      isLatest: index === 0,
      downloadUrl: `/api/games/${gameId}/files/${file.id}/download`,
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
    const response = await this.getGames({
      page: 1,
      pageSize: 200,
      sort: { field: 'updated_at', order: 'desc' },
    })
    const allGames = response.data
    const favorites = new Set(readFavorites())
    const sortedByCreated = [...allGames].sort((a, b) => (b.created_at || '').localeCompare(a.created_at || ''))
    const sortedByDownloads = [...allGames].sort((a, b) => (b.downloads || 0) - (a.downloads || 0))

    return {
      total_games: response.pagination.total,
      total_downloads: allGames.reduce((sum, game) => sum + (game.downloads || 0), 0),
      total_views: allGames.reduce((sum, game) => sum + (game.views || 0), 0),
      total_size: 0,
      recent_games: sortedByCreated.slice(0, 12),
      popular_games: sortedByDownloads.slice(0, 12),
      favorite_games: allGames.filter((game) => favorites.has(String(game.id))),
      pending_reviews: allGames.filter((game) => game.needs_review).length,
    }
  },

  async addGameFromFile(filePath: string): Promise<{ success: any[]; failed: any[] }> {
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
