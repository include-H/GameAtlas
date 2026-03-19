import { get, post } from './api'
import type { ApiResponse, Series } from './types'

async function listSeries(): Promise<Series[]> {
  const response = await get<ApiResponse<Series[]>>('/series')
  return response.data || []
}

export const seriesService = {
  async getAllSeries(): Promise<Series[]> {
    return listSeries()
  },

  async getPopularSeries(limit?: number): Promise<(Series & { game_count: number })[]> {
    const all = await listSeries()
    return all.slice(0, limit || all.length).map((item) => ({ ...item, game_count: 0 }))
  },

  async searchSeries(query: string, limit?: number): Promise<Series[]> {
    const all = await listSeries()
    const keyword = query.trim().toLowerCase()
    return all
      .filter((item) => item.name.toLowerCase().includes(keyword))
      .slice(0, limit || all.length)
  },

  async createSeries(data: {
    name: string
    slug?: string
    sort_order?: number
  }): Promise<Series> {
    const response = await post<ApiResponse<Series>>('/series', data)
    return response.data
  },
}
