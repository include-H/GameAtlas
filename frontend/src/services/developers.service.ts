import { get, post } from './api'
import type { ApiResponse, Developer } from './types'

async function listDevelopers(): Promise<Developer[]> {
  const response = await get<ApiResponse<Developer[]>>('/developers')
  return response.data || []
}

export const developersService = {
  async getPopularDevelopers(limit?: number): Promise<(Developer & { game_count: number })[]> {
    const all = await listDevelopers()
    return all.slice(0, limit || all.length).map((item) => ({ ...item, game_count: 0 }))
  },

  async searchDevelopers(query: string, limit?: number): Promise<Developer[]> {
    const all = await listDevelopers()
    const keyword = query.trim().toLowerCase()
    return all
      .filter((item) => item.name.toLowerCase().includes(keyword))
      .slice(0, limit || all.length)
  },

  async createDeveloper(data: {
    name: string
    slug?: string
    sort_order?: number
  }): Promise<Developer> {
    const response = await post<ApiResponse<Developer>>('/developers', data)
    return response.data
  },
}
