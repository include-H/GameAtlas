import { get, post } from './api'
import type { ApiEnvelope, ApiPageEnvelope, Developer } from './types'

interface ListDevelopersOptions {
  query?: string
  limit?: number
}

export const developersService = {
  async listDevelopers(options: ListDevelopersOptions = {}): Promise<Developer[]> {
    const queryParams = new URLSearchParams()
    if (options.query?.trim()) queryParams.append('search', options.query.trim())
    if (options.limit) queryParams.append('limit', String(options.limit))
    else queryParams.append('limit', '100')
    const response = await get<ApiPageEnvelope<Developer>>('/developers', { params: queryParams })
    return response.data
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
