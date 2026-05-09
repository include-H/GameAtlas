import { get } from './api'
import { buildApiUrl, buildSteamProxyUrl } from './api-url'
import type { ApiEnvelope, SteamGameDetails, SteamGameSearchResult } from './types'

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
  developers: string[] | null
  publishers: string[] | null
  cover_url: string | null
  banner_url: string | null
  screenshot_urls: string[] | null
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
    // 2026-04-09: keep blank-query short-circuit in the UI request layer so the picker
    // does not fire meaningless admin search requests before the backend transport contract applies.
    if (!query || query.trim().length === 0) return []
    const params: Record<string, string> = { q: query.trim() }
    const response = await get<ApiEnvelope<SteamSearchApiItem[]>>('/steam/search', { params })
    return response.data.map(mapSearchResult)
  },

  async getGameDetails(appId: string): Promise<SteamGameDetails> {
    const response = await get<ApiEnvelope<SteamAssetsApiItem>>(`/steam/${appId}/assets`)
    const data = response.data
    return {
      name: data.name,
      description: data.description || '',
      releaseDate: data.release_date || '',
      developers: data.developers ?? [],
      publishers: data.publishers ?? [],
      previewVideos: [],
      genres: [],
      tags: [],
      screenshots: (data.screenshot_urls ?? []).map((url) => proxySteamAssetUrl(url)),
      // 2026-04-06: keep Steam asset semantics aligned with the backend preview contract.
      // Impact: the frontend only carries cover/banner/screenshots here and does not invent
      // extra asset aliases from the same banner url.
      coverImage: proxySteamAssetUrl(data.cover_url),
      bannerImage: proxySteamAssetUrl(data.banner_url) || undefined,
    }
  },
}

export default steamService
