import { get, post } from './api'
import type { ApiResponse, SteamFetchImagesResponse, SteamGameDetails, SteamGameSearchResult } from './types'

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
  release_date: string
  developers: string[]
  publishers: string[]
  preview_videos?: Array<{ url: string; name: string; is_dash: boolean }>
  preview_video_url: string | null
  preview_video_name?: string | null
  preview_video_debug?: string[]
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

const steamService = {
  async searchGames(query: string): Promise<SteamGameSearchResult[]> {
    if (!query || query.trim().length === 0) return []
    const response = await get<ApiResponse<SteamSearchApiItem[]>>('/steam/search', {
      params: {
        q: query.trim(),
      },
    })
    return (response.data || []).map(mapSearchResult)
  },

  async getGameDetails(appId: string): Promise<SteamGameDetails> {
    const response = await get<ApiResponse<SteamAssetsApiItem>>(`/steam/${appId}/assets`)
    const data = response.data
    return {
      name: data.name,
      description: data.description || '',
      releaseDate: data.release_date || '',
      developers: data.developers || [],
      publishers: data.publishers || [],
      previewVideos: (data.preview_videos || []).map((item) => ({
        url: item.url,
        name: item.name,
        isDash: !!item.is_dash,
      })),
      previewVideoUrl: data.preview_video_url || undefined,
      previewVideoName: data.preview_video_name || undefined,
      previewVideoDebug: data.preview_video_debug || [],
      genres: [],
      tags: [],
      platforms: [],
      screenshots: data.screenshot_urls || [],
      headerImage: data.cover_url || '',
      libraryHero: data.banner_url || undefined,
      background: data.banner_url || undefined,
    }
  },

  async applyAssets(appId: string, payload: { game_id: number; cover_url?: string; banner_url?: string; preview_video_url?: string; screenshot_urls: string[] }): Promise<SteamFetchImagesResponse> {
    const response = await post<ApiResponse<SteamAssetsApiItem>>(`/steam/${appId}/apply-assets`, payload, {
      // DASH trailer import can take longer than default API timeout.
      timeout: 5 * 60 * 1000,
    })
    return {
      coverImage: response.data.cover_url || '',
      bannerImage: response.data.banner_url || '',
      previewVideo: response.data.preview_video_url || '',
      screenshots: response.data.screenshot_urls || [],
    }
  },
}

export default steamService
