import { get, post } from './api'
import type { ApiResponse, SteamFetchImagesResponse, SteamGameDetails, SteamGameSearchResult } from './types'
import { useUiStore } from '@/stores/ui'

interface SteamSearchApiItem {
  app_id: number
  name: string
  release_date: string | null
  tiny_image: string | null
}

interface SteamAssetsApiItem {
  app_id: number
  name: string
  description: string
  cover_url: string | null
  banner_url: string | null
  screenshot_urls: string[]
}

function mapSearchResult(item: SteamSearchApiItem): SteamGameSearchResult {
  return {
    id: String(item.app_id),
    name: item.name,
    releaseDate: item.release_date || undefined,
    tinyImage: item.tiny_image || undefined,
  }
}

export const steamService = {
  async searchGames(query: string): Promise<SteamGameSearchResult[]> {
    if (!query || query.trim().length === 0) return []
    const uiStore = useUiStore()
    const response = await get<ApiResponse<SteamSearchApiItem[]>>('/steam/search', {
      params: {
        q: query.trim(),
        proxy: uiStore.getProxyUrl() || undefined,
      },
    })
    return (response.data || []).map(mapSearchResult)
  },

  async getGameDetails(appId: string): Promise<SteamGameDetails> {
    const uiStore = useUiStore()
    const response = await get<ApiResponse<SteamAssetsApiItem>>(`/steam/${appId}/assets`, {
      params: {
        proxy: uiStore.getProxyUrl() || undefined,
      },
    })
    const data = response.data
    return {
      name: data.name,
      description: data.description || '',
      releaseDate: '',
      developers: [],
      publishers: [],
      genres: [],
      tags: [],
      platforms: [],
      screenshots: data.screenshot_urls || [],
      headerImage: data.cover_url || '',
      libraryHero: data.banner_url || undefined,
      background: data.banner_url || undefined,
    }
  },

  async applyAssets(appId: string, payload: { game_id: number; cover_url?: string; banner_url?: string; screenshot_urls: string[] }): Promise<SteamFetchImagesResponse> {
    const uiStore = useUiStore()
    const response = await post<ApiResponse<SteamAssetsApiItem>>(`/steam/${appId}/apply-assets`, payload, {
      params: {
        proxy: uiStore.getProxyUrl() || undefined,
      },
    })
    return {
      coverImage: response.data.cover_url || '',
      bannerImage: response.data.banner_url || '',
      screenshots: response.data.screenshot_urls || [],
    }
  },
}

export default steamService
