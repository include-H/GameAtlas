import { get, post } from './api'
import { buildApiUrl, buildSteamProxyUrl } from './api-url'
import type { ApiEnvelope, SteamFetchImagesResponse, SteamGameDetails, SteamGameSearchResult } from './types'

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
  cover_url: string | null
  banner_url: string | null
  screenshot_urls: string[]
}

function mapSearchResult(item: SteamSearchApiItem): SteamGameSearchResult {
  return {
    id: String(item.app_id),
    name: item.name,
    releaseDate: item.release_date || undefined,
    tinyImage: proxySteamAssetUrl(item.tiny_image) || undefined,
  }
}

const STEAM_PROXY_PATH = '/steam/proxy'
const STEAM_PROXY_URL_PREFIX = `${buildApiUrl(STEAM_PROXY_PATH)}?`

function isSteamProxyUrl(rawUrl: string): boolean {
  return rawUrl.startsWith(STEAM_PROXY_URL_PREFIX)
}

export function proxySteamAssetUrl(rawUrl?: string | null): string {
  const value = rawUrl?.trim()
  if (!value) return ''
  if (isSteamProxyUrl(value)) return value

  try {
    const parsed = new URL(value)
    if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') {
      return value
    }
  } catch {
    return value
  }

  return buildSteamProxyUrl(value)
}

const steamService = {
  async searchGames(query: string): Promise<SteamGameSearchResult[]> {
    if (!query || query.trim().length === 0) return []
    const response = await get<ApiEnvelope<SteamSearchApiItem[]>>('/steam/search', {
      params: {
        q: query.trim(),
      },
    })
    return (response.data || []).map(mapSearchResult)
  },

  async getGameDetails(appId: string): Promise<SteamGameDetails> {
    const response = await get<ApiEnvelope<SteamAssetsApiItem>>(`/steam/${appId}/assets`)
    const data = response.data
    return {
      name: data.name,
      description: data.description || '',
      releaseDate: data.release_date || '',
      developers: data.developers || [],
      publishers: data.publishers || [],
      previewVideos: [],
      genres: [],
      tags: [],
      platforms: [],
      screenshots: (data.screenshot_urls || []).map((url) => proxySteamAssetUrl(url)),
      headerImage: proxySteamAssetUrl(data.cover_url),
      libraryHero: proxySteamAssetUrl(data.banner_url) || undefined,
      background: proxySteamAssetUrl(data.banner_url) || undefined,
    }
  },

  async applyAssets(appId: string, payload: { game_id: number; cover_url?: string; banner_url?: string; screenshot_urls: string[] }): Promise<SteamFetchImagesResponse> {
    const response = await post<ApiEnvelope<SteamAssetsApiItem>>(`/steam/${appId}/apply-assets`, payload, {
      // DASH trailer import can take longer than default API timeout.
      timeout: 5 * 60 * 1000,
    })
    return {
      coverImage: response.data.cover_url || '',
      bannerImage: response.data.banner_url || '',
      screenshots: response.data.screenshot_urls || [],
    }
  },
}

export default steamService
