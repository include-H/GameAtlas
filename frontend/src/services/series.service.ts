import { get, post } from './api'
import type { ApiEnvelope, GameListItem, GameListItemDto, Series, SeriesDetail } from './types'

async function listSeriesWithParams(params?: {
  search?: string
  limit?: number
  sort?: 'name' | 'popular'
}): Promise<Series[]> {
  const queryParams = new URLSearchParams()
  if (params?.search) queryParams.append('search', params.search)
  if (params?.limit) queryParams.append('limit', String(params.limit))
  if (params?.sort) queryParams.append('sort', params.sort)
  const response = await get<ApiEnvelope<Series[]>>('/series', { params: queryParams })
  return response.data
}

export const seriesService = {
  async getAllSeries(params?: {
    search?: string
    limit?: number
    sort?: 'name' | 'popular'
  }): Promise<Series[]> {
    return listSeriesWithParams(params)
  },

  async getSeriesDetail(id: number | string): Promise<SeriesDetail> {
    const response = await get<ApiEnvelope<{ series: Series; games: GameListItemDto[] }>>(`/series/${id}`)
    return {
      series: response.data.series,
      games: response.data.games.map((item): GameListItem => ({
        ...item,
        isFavorite: Boolean(item.is_favorite),
      })),
    }
  },

  async getPopularSeries(limit?: number): Promise<(Series & { game_count: number })[]> {
    const all = await listSeriesWithParams({ limit, sort: 'popular' })
    return all.map((item) => ({ ...item, game_count: item.game_count || 0 }))
  },

  async searchSeries(query: string, limit?: number): Promise<Series[]> {
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
