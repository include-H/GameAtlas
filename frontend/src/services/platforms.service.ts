import { get, post } from './api'
import type { ApiEnvelope, Platform } from './types'

interface ListPlatformsOptions {
  query?: string
  limit?: number
}

const platformService = {
  async listPlatforms(options: ListPlatformsOptions = {}): Promise<Platform[]> {
    const queryParams = new URLSearchParams()
    if (options.query?.trim()) queryParams.append('search', options.query.trim())
    if (options.limit) queryParams.append('limit', String(options.limit))
    const response = await get<ApiEnvelope<Platform[]>>('/platforms', { params: queryParams })
    return response.data
  },

  async getAllPlatforms(): Promise<Platform[]> {
    return this.listPlatforms()
  },

  async searchPlatforms(query: string, limit?: number): Promise<Platform[]> {
    return this.listPlatforms({ query, limit })
  },

  async createPlatform(data: {
    name: string
    slug?: string
    sort_order?: number
  }): Promise<Platform> {
    const response = await post<ApiEnvelope<Platform>>('/platforms', data)
    return response.data
  },
}

export default platformService
