import type { GameDetailDto, GameListItemDto } from '@/services/types'
import { resolveAssetUrl } from '@/utils/asset-url'

export interface AmbientBackgroundPool {
  screenshots: string[]
  banners: string[]
}

type ImageCandidateGame = {
  banner_image?: GameListItemDto['banner_image']
  primary_screenshot?: GameListItemDto['primary_screenshot']
}

type ImageCandidateGameDetail = {
  banner_image?: GameDetailDto['banner_image']
  screenshots?: GameDetailDto['screenshots']
}

export const createAmbientBackgroundPool = (): AmbientBackgroundPool => ({
  screenshots: [],
  banners: [],
})

const pushResolved = (target: string[], value: string | null | undefined) => {
  const resolvedUrl = resolveAssetUrl(value)
  if (resolvedUrl && !target.includes(resolvedUrl)) {
    target.push(resolvedUrl)
  }
}

export const getAmbientBackgroundPoolFromGameListItem = (game?: ImageCandidateGame | null) => {
  const pool = createAmbientBackgroundPool()
  if (!game) {
    return pool
  }

  pushResolved(pool.screenshots, game.primary_screenshot)
  pushResolved(pool.banners, game.banner_image)
  return pool
}

export const getAmbientBackgroundPoolFromGameDetail = (game?: ImageCandidateGameDetail | null) => {
  const pool = createAmbientBackgroundPool()
  if (!game) {
    return pool
  }

  for (const screenshot of game.screenshots || []) {
    pushResolved(pool.screenshots, screenshot.path)
  }
  pushResolved(pool.banners, game.banner_image)
  return pool
}

export const mergeAmbientBackgroundPools = (pools: Array<AmbientBackgroundPool | null | undefined>) => {
  const merged = createAmbientBackgroundPool()
  for (const pool of pools) {
    for (const screenshot of pool?.screenshots || []) {
      pushResolved(merged.screenshots, screenshot)
    }
    for (const banner of pool?.banners || []) {
      pushResolved(merged.banners, banner)
    }
  }
  return merged
}

export const getAmbientBackgroundPoolFromGames = (games: Array<ImageCandidateGame | null | undefined>) => {
  const pools: AmbientBackgroundPool[] = []
  for (const game of games) {
    pools.push(getAmbientBackgroundPoolFromGameListItem(game))
  }
  return mergeAmbientBackgroundPools(pools)
}

export const getPrioritizedAmbientBackgroundUrls = (pool?: AmbientBackgroundPool | null) => {
  if (!pool) {
    return []
  }

  if (pool.screenshots.length > 0) {
    return pool.screenshots
  }

  return pool.banners
}

export const hasAmbientBackgroundPoolImages = (pool?: AmbientBackgroundPool | null) => {
  return getPrioritizedAmbientBackgroundUrls(pool).length > 0
}
