import { describe, expect, it } from 'vitest'

import { getAssetThumbUrl, resolveAssetCandidates, resolveAssetUrl } from './asset-url'

describe('asset-url helpers', () => {
  it('returns the original path for non-asset paths', () => {
    expect(resolveAssetUrl('/images/logo.png')).toBe('/images/logo.png')
    expect(resolveAssetCandidates('/images/logo.png')).toEqual(['/images/logo.png'])
  })

  it('keeps asset paths relative', () => {
    expect(resolveAssetUrl('/assets/cover.png')).toBe('/assets/cover.png')
    expect(resolveAssetCandidates('/assets/cover.png')).toEqual(['/assets/cover.png'])
  })

  it('derives thumbnail urls from asset paths', () => {
    expect(getAssetThumbUrl('/assets/game/uid.png')).toBe('/assets/game/uid.thumb.jpg')
    expect(getAssetThumbUrl('/assets/game/uid.webp')).toBe('/assets/game/uid.thumb.jpg')
    expect(getAssetThumbUrl('/assets/game/movie.mp4')).toBe('/assets/game/movie.thumb.jpg')
  })

  it('keeps paths without an extension unchanged', () => {
    expect(getAssetThumbUrl('/assets/game/uid')).toBe('/assets/game/uid')
    expect(getAssetThumbUrl('')).toBe('')
  })
})
