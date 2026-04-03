import { afterEach, describe, expect, it, vi } from 'vitest'

describe('proxySteamAssetUrl', () => {
  afterEach(() => {
    vi.unstubAllEnvs()
    vi.resetModules()
  })

  it('builds steam proxy urls from the shared api-url helper', async () => {
    vi.stubEnv('VITE_API_BASE_URL', 'https://api.example.com/base/')

    const { proxySteamAssetUrl } = await import('./steam.service')

    expect(proxySteamAssetUrl('https://cdn.example.com/image.jpg')).toBe(
      'https://api.example.com/base/steam/proxy?url=https%3A%2F%2Fcdn.example.com%2Fimage.jpg'
    )
  })

  it('does not proxy urls that are already using the steam proxy endpoint', async () => {
    vi.stubEnv('VITE_API_BASE_URL', 'https://api.example.com/base/')

    const { proxySteamAssetUrl } = await import('./steam.service')

    expect(
      proxySteamAssetUrl('https://api.example.com/base/steam/proxy?url=https%3A%2F%2Fcdn.example.com%2Fimage.jpg')
    ).toBe('https://api.example.com/base/steam/proxy?url=https%3A%2F%2Fcdn.example.com%2Fimage.jpg')
  })
})
