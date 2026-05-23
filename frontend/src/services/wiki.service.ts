import { get, put } from './api'
import type { ApiEnvelope } from './types'

export interface WikiDocumentResponse {
  game_id: number
  title: string
  content: string | null
  updated_at: string
  history_count?: number
}

export interface WikiHistoryEntry {
  id: number
  game_id: number
  content: string
  change_summary?: string | null
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
  }): Promise<WikiDocumentResponse> {
    const response = await put<ApiEnvelope<WikiDocumentResponse>>(`/games/${gameId}/wiki`, data)
    return response.data
  },

  async getWikiHistory(gameId: string): Promise<WikiHistoryEntry[]> {
    // 2026-04-06: wiki history size is a backend-owned contract.
    // Impact: the client no longer sends a fake limit parameter that the API never consumed.
    const response = await get<ApiEnvelope<WikiHistoryEntry[]>>(`/games/${gameId}/wiki/history`)
    return response.data
  },
}

export default wikiService
