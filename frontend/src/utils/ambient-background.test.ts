import { describe, expect, it } from 'vitest'

import {
  getAmbientBackgroundUrlsFromGameDetail,
  getAmbientBackgroundUrlsFromGameListItem,
  getAmbientBackgroundUrlsFromGames,
} from './ambient-background'

describe('ambient-background helpers', () => {
  it('collects screenshot and banner for list items', () => {
    expect(
      getAmbientBackgroundUrlsFromGameListItem({
        primary_screenshot: '/assets/game/shot-1.jpg',
        banner_image: '/assets/game/banner.jpg',
      }),
    ).toEqual(['/assets/game/shot-1.jpg', '/assets/game/banner.jpg'])
  })

  it('returns only banner when list item has no screenshot', () => {
    expect(
      getAmbientBackgroundUrlsFromGameListItem({
        primary_screenshot: null,
        banner_image: '/assets/game/banner.jpg',
      }),
    ).toEqual(['/assets/game/banner.jpg'])
  })

  it('returns empty when list item has neither screenshot nor banner', () => {
    expect(
      getAmbientBackgroundUrlsFromGameListItem({
        primary_screenshot: null,
        banner_image: null,
      }),
    ).toEqual([])
  })

  it('collects screenshots and banner for detail backgrounds', () => {
    expect(
      getAmbientBackgroundUrlsFromGameDetail({
        screenshots: [
          { id: 1, asset_uid: 'shot-1', path: '/assets/game/shot-1.jpg', sort_order: 0 },
          { id: 2, asset_uid: 'shot-2', path: '/assets/game/shot-2.jpg', sort_order: 1 },
        ],
        banner_image: '/assets/game/banner.jpg',
      }),
    ).toEqual(['/assets/game/shot-1.jpg', '/assets/game/shot-2.jpg', '/assets/game/banner.jpg'])
  })

  it('falls back to banner for detail backgrounds when screenshots are missing', () => {
    expect(
      getAmbientBackgroundUrlsFromGameDetail({
        screenshots: [],
        banner_image: '/assets/game/banner.jpg',
      }),
    ).toEqual(['/assets/game/banner.jpg'])
  })

  it('collects all suitable images when aggregating multiple games', () => {
    expect(
      getAmbientBackgroundUrlsFromGames([
        {
          primary_screenshot: '/assets/game-a/shot.jpg',
          banner_image: '/assets/game-a/banner.jpg',
        },
        {
          primary_screenshot: null,
          banner_image: '/assets/game-b/banner.jpg',
        },
      ]),
    ).toEqual(['/assets/game-a/shot.jpg', '/assets/game-a/banner.jpg', '/assets/game-b/banner.jpg'])
  })

  it('skips games with no suitable ambient images', () => {
    expect(
      getAmbientBackgroundUrlsFromGames([
        {
          primary_screenshot: '/assets/game-a/shot.jpg',
          banner_image: null,
        },
        {
          primary_screenshot: null,
          banner_image: null,
        },
      ]),
    ).toEqual(['/assets/game-a/shot.jpg'])
  })
})
