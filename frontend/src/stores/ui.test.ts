import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useUiStore } from './ui'

const { getItemMock, setItemMock } = vi.hoisted(() => ({
  getItemMock: vi.fn(),
  setItemMock: vi.fn(),
}))

vi.mock('@/utils/safe-local-storage', () => ({
  safeLocalStorageGetItem: getItemMock,
  safeLocalStorageSetItem: setItemMock,
}))

describe('useUiStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    getItemMock.mockReset()
    setItemMock.mockReset()
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it('persists and restores the games view mode', () => {
    const store = useUiStore()
    store.setGamesViewMode('list')

    expect(store.gamesViewMode).toBe('list')
    expect(setItemMock).toHaveBeenCalledWith('gamesViewMode', 'list')

    getItemMock.mockReturnValue('grid')
    store.initializeViewMode()
    expect(store.gamesViewMode).toBe('grid')

    getItemMock.mockReturnValue('not-a-mode')
    store.initializeViewMode()
    expect(store.gamesViewMode).toBe('grid')
  })

  it('persists and restores the sidebar collapse state', () => {
    const store = useUiStore()
    store.setSidebarCollapsed(true)

    expect(store.sidebarCollapsed).toBe(true)
    expect(setItemMock).toHaveBeenCalledWith('sidebarCollapsed', 'true')

    getItemMock.mockReturnValue('false')
    store.initializeSidebarCollapsed()
    expect(store.sidebarCollapsed).toBe(false)
  })

  it('clears ambient background only for the owning page', () => {
    const store = useUiStore()
    const pool = {
      owner: 'game-detail',
      key: 'game-1',
      pool: {
        screenshots: ['/assets/screenshot.jpg'],
        banners: ['/assets/banner.jpg'],
      },
    }
    store.setAmbientBackgroundSource(pool)

    store.clearAmbientBackgroundSource('other-page')
    expect(store.ambientBackgroundSource).toEqual(pool)

    store.clearAmbientBackgroundSource('game-detail')
    expect(store.ambientBackgroundSource).toBeNull()
  })

  it('adds alerts and auto-dismisses them', () => {
    vi.useFakeTimers()
    const store = useUiStore()

    store.addAlert('保存成功', 'success')
    expect(store.alerts).toHaveLength(1)

    vi.advanceTimersByTime(5001)
    expect(store.alerts).toHaveLength(0)
  })

  it('checks custom background availability with HEAD requests', async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true })
    vi.stubGlobal('fetch', fetchMock)

    const store = useUiStore()
    await store.initializeSharedBackgroundAvailability()

    expect(fetchMock).toHaveBeenCalledWith('/api/data/bg.jpg', {
      method: 'HEAD',
      cache: 'no-store',
    })
    expect(store.sharedBackgroundAvailability).toBe('available')

    fetchMock.mockResolvedValue({ ok: false })
    await store.refreshSharedBackgroundAvailability()
    expect(store.sharedBackgroundAvailability).toBe('missing')

    fetchMock.mockRejectedValue(new Error('offline'))
    await store.refreshSharedBackgroundAvailability()
    expect(store.sharedBackgroundAvailability).toBe('missing')
  })
})
