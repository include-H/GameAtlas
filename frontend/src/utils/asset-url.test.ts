import { describe, expect, it } from 'vitest'

import { resolveAssetCandidates, resolveAssetUrl, withAssetWidth } from './asset-url'

describe('asset-url helpers', () => {
  it('returns the original path for non-asset paths', () => {
    expect(resolveAssetUrl('/images/logo.png')).toBe('/images/logo.png')
    expect(resolveAssetCandidates('/images/logo.png')).toEqual(['/images/logo.png'])
  })

  it('keeps asset paths relative', () => {
    expect(resolveAssetUrl('/assets/cover.png')).toBe('/assets/cover.png')
    expect(resolveAssetCandidates('/assets/cover.png')).toEqual(['/assets/cover.png'])
  })

  it('appends width query to /assets paths only', () => {
    expect(withAssetWidth('/assets/game/cover.jpg', 480)).toBe('/assets/game/cover.jpg?w=480')
    expect(withAssetWidth('/assets/game/cover.jpg')).toBe('/assets/game/cover.jpg')
    expect(withAssetWidth('/images/cover.jpg', 480)).toBe('/images/cover.jpg')
    expect(withAssetWidth('data:image/png;base64,abc', 480)).toBe('data:image/png;base64,abc')
    expect(withAssetWidth('/assets/game/cover.jpg', 0)).toBe('/assets/game/cover.jpg')
  })

})
