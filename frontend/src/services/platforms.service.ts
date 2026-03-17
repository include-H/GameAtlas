import { get, post } from './api'
import type { ApiResponse, Platform } from './types'

export const platformService = {
  async getAllPlatforms(): Promise<Platform[]> {
    const response = await get<ApiResponse<Platform[]>>('/platforms')
    return response.data || []
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
    const response = await post<ApiResponse<Platform>>('/platforms', data)
    return response.data
  },
}

export default platformService
