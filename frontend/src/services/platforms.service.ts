import { get, post } from './api'
import type { ApiEnvelope, Platform } from './types'

const platformService = {
  async getAllPlatforms(): Promise<Platform[]> {
    const response = await get<ApiEnvelope<Platform[]>>('/platforms')
    return response.data
  },

  async searchPlatforms(query: string, limit?: number): Promise<Platform[]> {
    const all = await this.getAllPlatforms()
    const keyword = query.trim().toLowerCase()
    return all
      .filter((item) => item.name.toLowerCase().includes(keyword))
      .slice(0, limit || all.length)
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
