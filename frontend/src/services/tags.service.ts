import { get, post } from './api'
import type { ApiResponse, Tag, TagGroup } from './types'

const tagsService = {
  async getTagGroups(): Promise<TagGroup[]> {
    const response = await get<ApiResponse<TagGroup[]>>('/tag-groups')
    return response.data || []
  },

  async getTags(params?: {
    group_id?: number
    group_key?: string
    active?: boolean
  }): Promise<Tag[]> {
    const searchParams = new URLSearchParams()
    if (params?.group_id) searchParams.append('group_id', String(params.group_id))
    if (params?.group_key) searchParams.append('group_key', params.group_key)
    if (typeof params?.active === 'boolean') searchParams.append('active', String(params.active))

    const response = await get<ApiResponse<Tag[]>>('/tags', {
      params: searchParams,
    })
    return response.data || []
  },

  async createTagGroup(data: {
    key: string
    name: string
    description?: string | null
    sort_order?: number
    allow_multiple?: boolean
    is_filterable?: boolean
  }): Promise<TagGroup> {
    const response = await post<ApiResponse<TagGroup>>('/tag-groups', data)
    return response.data
  },

  async createTag(data: {
    group_id: number
    name: string
    slug?: string
    parent_id?: number | null
    sort_order?: number
    is_active?: boolean
  }): Promise<Tag> {
    const response = await post<ApiResponse<Tag>>('/tags', data)
    return response.data
  },
}

export default tagsService
