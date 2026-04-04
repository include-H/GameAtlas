import { describe, expect, it } from 'vitest'

import {
  getAmbientBackgroundUrlsFromGameDetail,
  getAmbientBackgroundUrlsFromGameListItem,
  getAmbientBackgroundUrlsFromGames,
} from './ambient-background'

describe('ambient-background helpers', () => {
  it('prefers list item screenshots over banner and cover', () => {
    expect(
      getAmbientBackgroundUrlsFromGameListItem({
        primary_screenshot: '/assets/game/shot-1.jpg',
        banner_image: '/assets/game/banner.jpg',
        cover_image: '/assets/game/cover.jpg',
      }),
    ).toEqual(['/assets/game/shot-1.jpg'])
  })

  it('falls back to banner and then cover when list item has no screenshot', () => {
    expect(
      getAmbientBackgroundUrlsFromGameListItem({
        primary_screenshot: null,
        banner_image: '/assets/game/banner.jpg',
        cover_image: '/assets/game/cover.jpg',
      }),
    ).toEqual(['/assets/game/banner.jpg'])

    expect(
      getAmbientBackgroundUrlsFromGameListItem({
        primary_screenshot: null,
        banner_image: null,
        cover_image: '/assets/game/cover.jpg',
      }),
    ).toEqual(['/assets/game/cover.jpg'])
  })

  it('returns only screenshots for detail backgrounds when screenshots exist', () => {
    expect(
      getAmbientBackgroundUrlsFromGameDetail({
        screenshots: [
          { id: 1, asset_uid: 'shot-1', path: '/assets/game/shot-1.jpg', sort_order: 0 },
          { id: 2, asset_uid: 'shot-2', path: '/assets/game/shot-2.jpg', sort_order: 1 },
        ],
        banner_image: '/assets/game/banner.jpg',
        cover_image: '/assets/game/cover.jpg',
      }),
    ).toEqual(['/assets/game/shot-1.jpg', '/assets/game/shot-2.jpg'])
  })

  it('falls back to banner and cover for detail backgrounds when screenshots are missing', () => {
    expect(
      getAmbientBackgroundUrlsFromGameDetail({
        screenshots: [],
        banner_image: '/assets/game/banner.jpg',
        cover_image: '/assets/game/cover.jpg',
      }),
    ).toEqual(['/assets/game/banner.jpg'])
  })

  it('keeps only screenshot-first urls when aggregating multiple games', () => {
    expect(
      getAmbientBackgroundUrlsFromGames([
        {
          primary_screenshot: '/assets/game-a/shot.jpg',
          banner_image: '/assets/game-a/banner.jpg',
          cover_image: '/assets/game-a/cover.jpg',
        },
        {
          primary_screenshot: null,
          banner_image: '/assets/game-b/banner.jpg',
          cover_image: '/assets/game-b/cover.jpg',
        },
      ]),
    ).toEqual(['/assets/game-a/shot.jpg', '/assets/game-b/banner.jpg'])
  })
})
