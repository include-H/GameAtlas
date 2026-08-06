import { get, post } from './api'
import type { ApiEnvelope, ApiPageEnvelope, GameListItem, GameListItemDto, MetadataPagination, Series, SeriesDetail } from './types'

async function listSeriesWithParams(params?: {
  search?: string
  limit?: number
  sort?: 'name' | 'popular'
}): Promise<Series[]> {
  const queryParams = new URLSearchParams()
  // 2026-04-06: series search uses the same trimmed transport semantics as
  // developers/publishers, so metadata search inputs stop drifting by resource.
  if (params?.search?.trim()) queryParams.append('search', params.search.trim())
  if (params?.limit) queryParams.append('limit', String(params.limit))
  if (params?.sort) queryParams.append('sort', params.sort)
  const response = await get<ApiPageEnvelope<Series>>('/series', { params: queryParams })
  return response.data
}

export const seriesService = {
  async getSeriesPage(params: {
    page: number
    limit: number
    search?: string
    sort?: 'name' | 'popular'
  }): Promise<ApiPageEnvelope<Series>> {
    const queryParams = new URLSearchParams({
      page: String(params.page),
      limit: String(params.limit),
    })
    if (params.search?.trim()) queryParams.append('search', params.search.trim())
    if (params.sort) queryParams.append('sort', params.sort)
    return get<ApiPageEnvelope<Series>>('/series', { params: queryParams })
  },

  async getSeriesDetail(id: number | string, options?: { page?: number; limit?: number }): Promise<SeriesDetail> {
    const queryParams = new URLSearchParams()
    if (options?.page) queryParams.append('page', String(options.page))
    if (options?.limit) queryParams.append('limit', String(options.limit))
    const response = await get<ApiEnvelope<{ series: Series; games: GameListItemDto[]; pagination: MetadataPagination }>>(`/series/${id}`, {
      params: queryParams,
    })
    return {
      series: response.data.series,
      games: response.data.games.map((item): GameListItem => {
        const { is_favorite, ...game } = item
        return {
          ...game,
          isFavorite: is_favorite,
        }
      }),
      pagination: response.data.pagination,
    }
  },

  async getPopularSeries(limit = 100): Promise<(Series & { game_count: number })[]> {
    const all = await listSeriesWithParams({ limit, sort: 'popular' })
    return all.map((item) => ({ ...item, game_count: item.game_count || 0 }))
  },

  async searchSeries(query: string, limit = 100): Promise<Series[]> {
    return listSeriesWithParams({ search: query, limit, sort: 'popular' })
  },

  async createSeries(data: {
    name: string
    slug?: string
    sort_order?: number
  }): Promise<Series> {
    const response = await post<ApiEnvelope<Series>>('/series', data)
    return response.data
  },
}
