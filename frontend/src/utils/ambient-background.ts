import type { GameDetailDto, GameListItemDto, Series } from '@/services/types'
import { resolveAssetUrl } from '@/utils/asset-url'

type ImageCandidateGame = {
  banner_image?: GameListItemDto['banner_image']
  primary_screenshot?: GameListItemDto['primary_screenshot']
  cover_image?: GameListItemDto['cover_image']
}

type ImageCandidateGameDetail = {
  banner_image?: GameDetailDto['banner_image']
  cover_image?: GameDetailDto['cover_image']
  screenshots?: GameDetailDto['screenshots']
}

type ImageCandidateSeries = {
  cover_image?: Series['cover_image']
  cover_candidates?: Series['cover_candidates']
}

const pushResolved = (target: string[], value: string | null | undefined) => {
  const resolvedUrl = resolveAssetUrl(value)
  if (resolvedUrl && !target.includes(resolvedUrl)) {
    target.push(resolvedUrl)
  }
}

export const getAmbientBackgroundUrlsFromGameListItem = (game?: ImageCandidateGame | null) => {
  if (!game) {
    return []
  }

  const urls: string[] = []
  pushResolved(urls, game.banner_image)
  pushResolved(urls, game.primary_screenshot)
  pushResolved(urls, game.cover_image)
  return urls
}

export const getAmbientBackgroundUrlsFromGameDetail = (game?: ImageCandidateGameDetail | null) => {
  if (!game) {
    return []
  }

  const urls: string[] = []
  for (const screenshot of game.screenshots || []) {
    pushResolved(urls, screenshot.path)
  }
  pushResolved(urls, game.banner_image)
  pushResolved(urls, game.cover_image)
  return urls
}

export const getAmbientBackgroundUrlsFromGames = (games: Array<ImageCandidateGame | null | undefined>) => {
  const urls: string[] = []
  for (const game of games) {
    const gameUrls = getAmbientBackgroundUrlsFromGameListItem(game)
    for (const url of gameUrls) {
      if (!urls.includes(url)) {
        urls.push(url)
      }
    }
  }
  return urls
}

export const getAmbientBackgroundUrlsFromSeries = (series?: ImageCandidateSeries | null) => {
  if (!series) {
    return []
  }

  const urls: string[] = []
  for (const candidate of series.cover_candidates || []) {
    pushResolved(urls, candidate)
  }
  pushResolved(urls, series.cover_image)
  return urls
}
