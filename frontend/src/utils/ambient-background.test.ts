import { describe, expect, it } from 'vitest'

import {
  getAmbientBackgroundPoolFromGameDetail,
  getAmbientBackgroundPoolFromGameListItem,
  getAmbientBackgroundPoolFromGames,
  getPrioritizedAmbientBackgroundUrls,
  hasAmbientBackgroundPoolImages,
  mergeAmbientBackgroundPools,
} from './ambient-background'

describe('ambient-background helpers', () => {
  it('collects screenshot and banner into separate list item pools', () => {
    expect(
      getAmbientBackgroundPoolFromGameListItem({
        primary_screenshot: '/assets/game/shot-1.jpg',
        banner_image: '/assets/game/banner.jpg',
      }),
    ).toEqual({
      screenshots: ['/assets/game/shot-1.jpg'],
      banners: ['/assets/game/banner.jpg'],
    })
  })

  it('prioritizes screenshots over banners when both exist', () => {
    const pool = getAmbientBackgroundPoolFromGameListItem({
      primary_screenshot: '/assets/game/shot-1.jpg',
      banner_image: '/assets/game/banner.jpg',
    })

    expect(getPrioritizedAmbientBackgroundUrls(pool)).toEqual(['/assets/game/shot-1.jpg'])
  })

  it('falls back to banners when a list item has no screenshot', () => {
    expect(
      getPrioritizedAmbientBackgroundUrls(getAmbientBackgroundPoolFromGameListItem({
        primary_screenshot: null,
        banner_image: '/assets/game/banner.jpg',
      })),
    ).toEqual(['/assets/game/banner.jpg'])
  })

  it('returns empty pool when a list item has neither screenshot nor banner', () => {
    expect(
      hasAmbientBackgroundPoolImages(getAmbientBackgroundPoolFromGameListItem({
        primary_screenshot: null,
        banner_image: null,
      })),
    ).toBe(false)
  })

  it('collects screenshots and banner into separate detail pools', () => {
    expect(
      getAmbientBackgroundPoolFromGameDetail({
        screenshots: [
          { id: 1, asset_uid: 'shot-1', path: '/assets/game/shot-1.jpg', sort_order: 0 },
          { id: 2, asset_uid: 'shot-2', path: '/assets/game/shot-2.jpg', sort_order: 1 },
        ],
        banner_image: '/assets/game/banner.jpg',
      }),
    ).toEqual({
      screenshots: ['/assets/game/shot-1.jpg', '/assets/game/shot-2.jpg'],
      banners: ['/assets/game/banner.jpg'],
    })
  })

  it('falls back to banner for detail backgrounds when screenshots are missing', () => {
    expect(
      getPrioritizedAmbientBackgroundUrls(getAmbientBackgroundPoolFromGameDetail({
        screenshots: [],
        banner_image: '/assets/game/banner.jpg',
      })),
    ).toEqual(['/assets/game/banner.jpg'])
  })

  it('keeps screenshots and banners separate when aggregating multiple games', () => {
    expect(
      getAmbientBackgroundPoolFromGames([
        {
          primary_screenshot: '/assets/game-a/shot.jpg',
          banner_image: '/assets/game-a/banner.jpg',
        },
        {
          primary_screenshot: null,
          banner_image: '/assets/game-b/banner.jpg',
        },
      ]),
    ).toEqual({
      screenshots: ['/assets/game-a/shot.jpg'],
      banners: ['/assets/game-a/banner.jpg', '/assets/game-b/banner.jpg'],
    })
  })

  it('skips games with no suitable ambient images', () => {
    expect(
      getAmbientBackgroundPoolFromGames([
        {
          primary_screenshot: '/assets/game-a/shot.jpg',
          banner_image: null,
        },
        {
          primary_screenshot: null,
          banner_image: null,
        },
      ]),
    ).toEqual({
      screenshots: ['/assets/game-a/shot.jpg'],
      banners: [],
    })
  })

  it('deduplicates screenshots and banners while merging pools', () => {
    expect(
      mergeAmbientBackgroundPools([
        {
          screenshots: ['/assets/game-a/shot.jpg', '/assets/game-a/shot.jpg'],
          banners: ['/assets/game-a/banner.jpg'],
        },
        {
          screenshots: ['/assets/game-a/shot.jpg', '/assets/game-b/shot.jpg'],
          banners: ['/assets/game-a/banner.jpg', '/assets/game-b/banner.jpg'],
        },
      ]),
    ).toEqual({
      screenshots: ['/assets/game-a/shot.jpg', '/assets/game-b/shot.jpg'],
      banners: ['/assets/game-a/banner.jpg', '/assets/game-b/banner.jpg'],
    })
  })
})
