import { get, put } from './api'
import type { ApiResponse } from './types'

export interface WikiContent {
  content?: string
  html?: string
  updated_at?: string
}

export interface WikiHistoryEntry {
  id: number
  content: string
  change_summary?: string
  created_at: string
}

export const wikiService = {
  async getWikiPage(gameId: string): Promise<WikiContent | null> {
    try {
      const response = await get<ApiResponse<{
        content: string | null
        content_html: string | null
        updated_at: string
      }>>(`/games/${gameId}/wiki`)
      return {
        content: response.data.content || '',
        html: response.data.content_html || '',
        updated_at: response.data.updated_at,
      }
    } catch {
      return null
    }
  },

  async updateWikiPage(gameId: string, data: {
    content: string
    change_summary?: string
  }): Promise<WikiContent | null> {
    const response = await put<ApiResponse<{
      content: string | null
      content_html: string | null
      updated_at: string
    }>>(`/games/${gameId}/wiki`, data)
    return {
      content: response.data.content || '',
      html: response.data.content_html || '',
      updated_at: response.data.updated_at,
    }
  },

  async getWikiHistory(gameId: string, limit = 50): Promise<WikiHistoryEntry[]> {
    const response = await get<ApiResponse<WikiHistoryEntry[]>>(`/games/${gameId}/wiki/history`, {
      params: { limit },
    })
    return response.data || []
  },
}

export default wikiService
