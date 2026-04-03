import { get, post } from './api'
import type { ApiEnvelope, Publisher } from './types'

interface ListPublishersOptions {
  query?: string
  limit?: number
}

export const publishersService = {
  async listPublishers(options: ListPublishersOptions = {}): Promise<Publisher[]> {
    const queryParams = new URLSearchParams()
    if (options.query?.trim()) queryParams.append('search', options.query.trim())
    if (options.limit) queryParams.append('limit', String(options.limit))
    const response = await get<ApiEnvelope<Publisher[]>>('/publishers', { params: queryParams })
    return response.data || []
  },

  async getPopularPublishers(limit?: number): Promise<(Publisher & { game_count: number })[]> {
    const items = await this.listPublishers({ limit })
    return items.map((item) => ({ ...item, game_count: 0 }))
  },

  async searchPublishers(query: string, limit?: number): Promise<Publisher[]> {
    return this.listPublishers({ query, limit })
  },

  async createPublisher(data: {
    name: string
    slug?: string
    sort_order?: number
  }): Promise<Publisher> {
    const response = await post<ApiEnvelope<Publisher>>('/publishers', data)
    return response.data
  },
}
