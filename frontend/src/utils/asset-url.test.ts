import { describe, expect, it } from 'vitest'

import { resolveAssetCandidates, resolveAssetUrl } from './asset-url'

describe('asset-url helpers', () => {
  it('returns the original path for non-asset paths', () => {
    expect(resolveAssetUrl('/images/logo.png')).toBe('/images/logo.png')
    expect(resolveAssetCandidates('/images/logo.png')).toEqual(['/images/logo.png'])
  })

  it('keeps asset paths relative', () => {
    expect(resolveAssetUrl('/assets/cover.png')).toBe('/assets/cover.png')
    expect(resolveAssetCandidates('/assets/cover.png')).toEqual(['/assets/cover.png'])
  })
})
