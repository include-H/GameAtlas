import { beforeEach, describe, expect, it, vi } from 'vitest'

import useLocale from './useLocale'

describe('useLocale', () => {
  beforeEach(() => {
    window.localStorage.clear()
  })

  it('defaults to zh-CN and resolves nested translations', () => {
    const { currentLocale, t } = useLocale()

    expect(currentLocale.value).toBe('zh-CN')
    expect(t('menu.dashboard')).toBe('首页')
  })

  it('supports flat translation keys inside nested objects', () => {
    window.localStorage.setItem('locale', 'en-US')

    const { currentLocale, t } = useLocale()

    expect(currentLocale.value).toBe('en-US')
    expect(t('menu.games.timeline')).toBe('Timeline')
    expect(t('menu.pending.center')).toBe('Pending Workbench')
    expect(t('menu.unknown')).toBe('menu.unknown')
  })

  it('persists locale changes and reloads the page', () => {
    const reloadMock = vi.fn()
    const originalLocation = window.location

    Object.defineProperty(window, 'location', {
      value: {
        ...originalLocation,
        reload: reloadMock,
      },
      configurable: true,
    })

    const { setLocale } = useLocale()
    setLocale('en-US')

    expect(window.localStorage.getItem('locale')).toBe('en-US')
    expect(reloadMock).toHaveBeenCalledTimes(1)

    Object.defineProperty(window, 'location', {
      value: originalLocation,
      configurable: true,
    })
  })

  it('falls back when localStorage read throws', () => {
    const getItemMock = vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new DOMException('Blocked', 'SecurityError')
    })

    const { currentLocale } = useLocale()

    expect(currentLocale.value).toBe('zh-CN')
    getItemMock.mockRestore()
  })

  it('does not throw when localStorage write throws', () => {
    const reloadMock = vi.fn()
    const originalLocation = window.location
    const setItemMock = vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new DOMException('Blocked', 'SecurityError')
    })

    Object.defineProperty(window, 'location', {
      value: {
        ...originalLocation,
        reload: reloadMock,
      },
      configurable: true,
    })

    const { setLocale } = useLocale()

    expect(() => setLocale('en-US')).not.toThrow()
    expect(reloadMock).toHaveBeenCalledTimes(1)

    setItemMock.mockRestore()
    Object.defineProperty(window, 'location', {
      value: originalLocation,
      configurable: true,
    })
  })
})
