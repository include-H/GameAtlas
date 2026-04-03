import { get, put } from './api'
import type { ApiEnvelope } from './types'

export interface WikiDocumentResponse {
  content: string | null
  updated_at: string
}

export interface WikiHistoryEntry {
  id: number
  content: string
  change_summary?: string
  created_at: string
}

const wikiService = {
  async getWikiPage(gameId: string): Promise<WikiDocumentResponse> {
    const response = await get<ApiEnvelope<WikiDocumentResponse>>(`/games/${gameId}/wiki`)
    return response.data
  },

  async updateWikiPage(gameId: string, data: {
    content: string
    change_summary?: string
  }): Promise<WikiDocumentResponse | null> {
    const response = await put<ApiEnvelope<WikiDocumentResponse>>(`/games/${gameId}/wiki`, data)
    return response.data
  },

  async getWikiHistory(gameId: string, limit = 50): Promise<WikiHistoryEntry[]> {
    const response = await get<ApiEnvelope<WikiHistoryEntry[]>>(`/games/${gameId}/wiki/history`, {
      params: { limit },
    })
    return response.data
  },
}

export default wikiService
