import { get, post } from './api'
import type { ApiEnvelope, Publisher } from './types'

async function listPublishers(): Promise<Publisher[]> {
  const response = await get<ApiEnvelope<Publisher[]>>('/publishers')
  return response.data
}

export const publishersService = {
  async getPopularPublishers(limit?: number): Promise<(Publisher & { game_count: number })[]> {
    const all = await listPublishers()
    return all.slice(0, limit || all.length).map((item) => ({ ...item, game_count: 0 }))
  },

  async searchPublishers(query: string, limit?: number): Promise<Publisher[]> {
    const all = await listPublishers()
    const keyword = query.trim().toLowerCase()
    return all
      .filter((item) => item.name.toLowerCase().includes(keyword))
      .slice(0, limit || all.length)
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
