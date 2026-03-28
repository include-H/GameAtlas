import { afterEach, describe, expect, it, vi } from 'vitest'

describe('buildApiUrl', () => {
  afterEach(() => {
    vi.unstubAllEnvs()
    vi.resetModules()
  })

  it('uses the default api base', async () => {
    vi.stubEnv('VITE_API_BASE_URL', '')

    const { buildApiUrl } = await import('./api-url')

    expect(buildApiUrl('/games/game-1/files/9/download')).toBe('/api/games/game-1/files/9/download')
  })

  it('uses a custom api base for browser-driven urls', async () => {
    vi.stubEnv('VITE_API_BASE_URL', 'https://api.example.com/base')

    const { buildApiUrl } = await import('./api-url')

    expect(buildApiUrl('/games/game-1/files/9/download')).toBe('https://api.example.com/base/games/game-1/files/9/download')
  })

  it('normalizes duplicate slashes at the join boundary', async () => {
    vi.stubEnv('VITE_API_BASE_URL', 'https://api.example.com/base/')

    const { buildApiUrl } = await import('./api-url')

    expect(buildApiUrl('games/game-1')).toBe('https://api.example.com/base/games/game-1')
  })
})
