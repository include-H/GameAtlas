import type { Favoritable } from './types'
import { safeLocalStorageGetItem, safeLocalStorageSetItem } from '@/utils/safe-local-storage'

const FAVORITES_KEY = 'game-library-favorites'

export function readFavorites(): string[] {
  try {
    const raw = safeLocalStorageGetItem(FAVORITES_KEY)
    if (!raw) return []
    const ids = JSON.parse(raw)
    return Array.isArray(ids) ? ids.map(String) : []
  } catch {
    return []
  }
}

export function writeFavorites(ids: string[]) {
  safeLocalStorageSetItem(FAVORITES_KEY, JSON.stringify(ids))
}

export function getFavoriteCount(): number {
  return readFavorites().length
}

export function applyFavorite<T extends { public_id?: string } & Favoritable>(game: T, favoriteIds?: Set<string>): T {
  const favorites = favoriteIds ?? new Set(readFavorites())
  return {
    ...game,
    isFavorite: Boolean(game.public_id) && favorites.has(String(game.public_id)),
  }
}
