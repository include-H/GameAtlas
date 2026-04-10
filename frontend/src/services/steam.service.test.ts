import { afterEach, describe, expect, it, vi } from 'vitest'

const { getMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
}))

vi.mock('./api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('./api')>()
  return {
    ...actual,
    get: getMock,
  }
})

describe('proxySteamAssetUrl', () => {
  afterEach(() => {
    vi.unstubAllEnvs()
    vi.resetModules()
    getMock.mockReset()
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

  it('maps steam asset preview to backend-native cover, banner, and screenshots only', async () => {
    vi.stubEnv('VITE_API_BASE_URL', 'https://api.example.com/base/')
    getMock.mockResolvedValue({
      data: {
        name: 'Game One',
        description: 'desc',
        release_date: '2024-01-01',
        developers: ['Dev'],
        publishers: ['Pub'],
        cover_url: 'https://cdn.example.com/cover.jpg',
        banner_url: 'https://cdn.example.com/banner.jpg',
        screenshot_urls: ['https://cdn.example.com/shot-1.jpg'],
      },
    })

    const steamService = (await import('./steam.service')).default
    const result = await steamService.getGameDetails('123')

    expect(result.coverImage).toBe(
      'https://api.example.com/base/steam/proxy?url=https%3A%2F%2Fcdn.example.com%2Fcover.jpg'
    )
    expect(result.bannerImage).toBe(
      'https://api.example.com/base/steam/proxy?url=https%3A%2F%2Fcdn.example.com%2Fbanner.jpg'
    )
    expect(result.screenshots).toEqual([
      'https://api.example.com/base/steam/proxy?url=https%3A%2F%2Fcdn.example.com%2Fshot-1.jpg',
    ])
    expect(result).not.toHaveProperty('headerImage')
    expect(result).not.toHaveProperty('libraryHero')
    expect(result).not.toHaveProperty('background')
  })
})
