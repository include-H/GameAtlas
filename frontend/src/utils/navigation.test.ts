import { afterEach, describe, expect, it, vi } from 'vitest'

import { hasHistoryBack, navigateBackOrFallback } from './navigation'

describe('navigation helpers', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('detects when browser history can go back', () => {
    expect(hasHistoryBack(2)).toBe(true)
    expect(hasHistoryBack(1)).toBe(false)
  })

  it('uses browser back when there is navigation history', () => {
    const back = vi.fn()
    const push = vi.fn()

    vi.stubGlobal('window', {
      history: {
        length: 2,
      },
    })

    navigateBackOrFallback(
      {
        back,
        push,
      } as never,
      { name: 'games' },
    )

    expect(back).toHaveBeenCalledTimes(1)
    expect(push).not.toHaveBeenCalled()
  })

  it('falls back when there is no navigation history', () => {
    const back = vi.fn()
    const push = vi.fn()

    vi.stubGlobal('window', {
      history: {
        length: 1,
      },
    })

    navigateBackOrFallback(
      {
        back,
        push,
      } as never,
      { name: 'games' },
    )

    expect(back).not.toHaveBeenCalled()
    expect(push).toHaveBeenCalledWith({ name: 'games' })
  })
})
