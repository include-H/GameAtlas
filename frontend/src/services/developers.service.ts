import { get, post } from './api'
import type { ApiEnvelope, Developer } from './types'

interface ListDevelopersOptions {
  query?: string
  limit?: number
}

export const developersService = {
  async listDevelopers(options: ListDevelopersOptions = {}): Promise<Developer[]> {
    const queryParams = new URLSearchParams()
    if (options.query?.trim()) queryParams.append('search', options.query.trim())
    if (options.limit) queryParams.append('limit', String(options.limit))
    const response = await get<ApiEnvelope<Developer[]>>('/developers', { params: queryParams })
    return response.data || []
  },

  async getPopularDevelopers(limit?: number): Promise<(Developer & { game_count: number })[]> {
    const items = await this.listDevelopers({ limit })
    return items.map((item) => ({ ...item, game_count: 0 }))
  },

  async searchDevelopers(query: string, limit?: number): Promise<Developer[]> {
    return this.listDevelopers({ query, limit })
  },

  async createDeveloper(data: {
    name: string
    slug?: string
    sort_order?: number
  }): Promise<Developer> {
    const response = await post<ApiEnvelope<Developer>>('/developers', data)
    return response.data
  },
}
