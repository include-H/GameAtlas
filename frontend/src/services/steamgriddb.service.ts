import { get } from './api'
import type { ApiEnvelope } from './types'

export interface SteamGridDBImage {
  id: number
  score: number
  style?: string
  notes: string
  language: string
  url: string
  thumb: string
}

export interface SteamGridDBGame {
  id: number
  name: string
  release_date: number
  types: string[]
  verified: boolean
}

const steamGridDBService = {
  async isAvailable(): Promise<boolean> {
    try {
      const response = await get<ApiEnvelope<boolean>>('/steamgriddb/available')
      return response.data === true
    } catch {
      return false
    }
  },

  async search(query: string): Promise<SteamGridDBGame[]> {
    if (!query || query.trim().length === 0) return []
    const response = await get<ApiEnvelope<SteamGridDBGame[]>>('/steamgriddb/search', {
      params: { q: query.trim() },
    })
    return response.data ?? []
  },

  async getGridsBySteamAppId(appId: string): Promise<SteamGridDBImage[]> {
    const response = await get<ApiEnvelope<SteamGridDBImage[]>>(`/steamgriddb/${appId}/grids`)
    return response.data ?? []
  },

  async getHeroesBySteamAppId(appId: string): Promise<SteamGridDBImage[]> {
    const response = await get<ApiEnvelope<SteamGridDBImage[]>>(`/steamgriddb/${appId}/heroes`)
    return response.data ?? []
  },

  async getLogosBySteamAppId(appId: string): Promise<SteamGridDBImage[]> {
    const response = await get<ApiEnvelope<SteamGridDBImage[]>>(`/steamgriddb/${appId}/logos`)
    return response.data ?? []
  },

  async getGridsByGameId(gameId: number): Promise<SteamGridDBImage[]> {
    const response = await get<ApiEnvelope<SteamGridDBImage[]>>(`/steamgriddb/game/${gameId}/grids`)
    return response.data ?? []
  },

  async getHeroesByGameId(gameId: number): Promise<SteamGridDBImage[]> {
    const response = await get<ApiEnvelope<SteamGridDBImage[]>>(`/steamgriddb/game/${gameId}/heroes`)
    return response.data ?? []
  },

  async getLogosByGameId(gameId: number): Promise<SteamGridDBImage[]> {
    const response = await get<ApiEnvelope<SteamGridDBImage[]>>(`/steamgriddb/game/${gameId}/logos`)
    return response.data ?? []
  },
}

export default steamGridDBService
