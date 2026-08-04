import { get, post } from './api'
import type { ApiEnvelope, GameListItem, GameListItemDto, Publisher, PublisherDetail } from './types'

interface ListPublishersOptions {
  query?: string
  limit?: number
  sort?: 'name' | 'popular'
}

async function listPublishersWithParams(options: ListPublishersOptions = {}): Promise<Publisher[]> {
  const queryParams = new URLSearchParams()
  if (options.query?.trim()) queryParams.append('search', options.query.trim())
  if (options.limit) queryParams.append('limit', String(options.limit))
  if (options.sort) queryParams.append('sort', options.sort)
  const response = await get<ApiEnvelope<Publisher[]>>('/publishers', { params: queryParams })
  return response.data
}

export const publishersService = {
  async listPublishers(options: ListPublishersOptions = {}): Promise<Publisher[]> {
    return listPublishersWithParams(options)
  },

  async getAllPublishers(options: ListPublishersOptions = {}): Promise<Publisher[]> {
    return listPublishersWithParams(options)
  },

  async getPublisherDetail(id: number | string): Promise<PublisherDetail> {
    const response = await get<ApiEnvelope<{ publisher: Publisher; games: GameListItemDto[] }>>(`/publishers/${id}`)
    return {
      publisher: response.data.publisher,
      games: response.data.games.map((item): GameListItem => {
        const { is_favorite, ...game } = item
        return {
          ...game,
          isFavorite: is_favorite,
        }
      }),
    }
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
