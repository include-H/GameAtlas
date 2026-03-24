import type { Game } from './types'

const FAVORITES_KEY = 'game-library-favorites'

export interface GameListApiItem {
  id: number
  title: string
  title_alt: string | null
  visibility: 'public' | 'private'
  summary: string | null
  release_date: string | null
  engine: string | null
  cover_image: string | null
  banner_image: string | null
  needs_review: boolean
  downloads: number
  primary_screenshot?: string | null
  screenshot_count?: number
  file_count?: number
  developer_count?: number
  publisher_count?: number
  platform_count?: number
  created_at: string
  updated_at: string
}

export function readFavorites(): string[] {
  if (typeof window === 'undefined') return []
  try {
    const raw = window.localStorage.getItem(FAVORITES_KEY)
    if (!raw) return []
    const ids = JSON.parse(raw)
    return Array.isArray(ids) ? ids.map(String) : []
  } catch {
    return []
  }
}

export function writeFavorites(ids: string[]) {
  if (typeof window === 'undefined') return
  window.localStorage.setItem(FAVORITES_KEY, JSON.stringify(ids))
}

export function getFavoriteCount(): number {
  return readFavorites().length
}

export function applyFavorite(game: Game, favoriteIds?: Set<string>): Game {
  const favorites = favoriteIds ?? new Set(readFavorites())
  return {
    ...game,
    isFavorite: favorites.has(String(game.id)),
  }
}

export function mapGameListItem(item: GameListApiItem, favoriteIds?: Set<string>): Game {
  return applyFavorite({
    id: item.id,
    title: item.title,
    title_alt: item.title_alt,
    visibility: item.visibility,
    summary: item.summary,
    release_date: item.release_date,
    engine: item.engine,
    cover_image: item.cover_image,
    banner_image: item.banner_image,
    needs_review: item.needs_review,
    primary_screenshot: item.primary_screenshot ?? null,
    screenshot_count: item.screenshot_count ?? 0,
    file_count: item.file_count ?? 0,
    developer_count: item.developer_count ?? 0,
    publisher_count: item.publisher_count ?? 0,
    platform_count: item.platform_count ?? 0,
    downloads: item.downloads,
    created_at: item.created_at,
    updated_at: item.updated_at,
    screenshots: [],
    file_paths: [],
  }, favoriteIds)
}
