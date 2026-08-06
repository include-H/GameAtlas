import { get, post } from './api'
import type { ApiEnvelope, ApiPageEnvelope, GameListItem, GameListItemDto, MetadataPagination, Publisher, PublisherDetail } from './types'

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
  const response = await get<ApiPageEnvelope<Publisher>>('/publishers', { params: queryParams })
  return response.data
}

export const publishersService = {
  async listPublishers(options: ListPublishersOptions = {}): Promise<Publisher[]> {
    return listPublishersWithParams({ ...options, limit: options.limit ?? 100 })
  },

  async getPublishersPage(params: {
    page: number
    limit: number
    search?: string
    sort?: 'name' | 'popular'
  }): Promise<ApiPageEnvelope<Publisher>> {
    const queryParams = new URLSearchParams({
      page: String(params.page),
      limit: String(params.limit),
    })
    if (params.search?.trim()) queryParams.append('search', params.search.trim())
    if (params.sort) queryParams.append('sort', params.sort)
    return get<ApiPageEnvelope<Publisher>>('/publishers', { params: queryParams })
  },

  async getPublisherDetail(id: number | string, options?: { page?: number; limit?: number }): Promise<PublisherDetail> {
    const queryParams = new URLSearchParams()
    if (options?.page) queryParams.append('page', String(options.page))
    if (options?.limit) queryParams.append('limit', String(options.limit))
    const response = await get<ApiEnvelope<{ publisher: Publisher; games: GameListItemDto[]; pagination: MetadataPagination }>>(`/publishers/${id}`, {
      params: queryParams,
    })
    return {
      publisher: response.data.publisher,
      games: response.data.games.map((item): GameListItem => {
        const { is_favorite, ...game } = item
        return {
          ...game,
          isFavorite: is_favorite,
        }
      }),
      pagination: response.data.pagination,
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
