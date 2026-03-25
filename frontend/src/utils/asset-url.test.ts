import { afterEach, describe, expect, it, vi } from 'vitest'

import { resolveAssetCandidates, resolveAssetUrl } from './asset-url'

describe('asset-url helpers', () => {
  afterEach(() => {
    vi.unstubAllEnvs()
  })

  it('returns the original path for non-asset paths', () => {
    expect(resolveAssetUrl('/images/logo.png')).toBe('/images/logo.png')
    expect(resolveAssetCandidates('/images/logo.png')).toEqual(['/images/logo.png'])
  })

  it('resolves asset urls against an absolute api base', () => {
    vi.stubEnv('VITE_API_BASE_URL', 'https://api.example.com/v1')

    expect(resolveAssetUrl('/assets/cover.png')).toBe('https://api.example.com/assets/cover.png')
    expect(resolveAssetCandidates('/assets/cover.png')).toEqual([
      'https://api.example.com/assets/cover.png',
      '/assets/cover.png',
    ])
  })

  it('falls back to backend host derived from the browser location', () => {
    vi.stubEnv('VITE_API_BASE_URL', '/api')
    vi.stubEnv('VITE_BACKEND_PORT', '4567')

    expect(resolveAssetUrl('/assets/banner.png')).toBe('http://localhost:4567/assets/banner.png')
  })
})
