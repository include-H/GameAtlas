import { afterEach, describe, expect, it, vi } from 'vitest'
import type { Router } from 'vue-router'

import {
  hasHistoryBack,
  navigateBackOrFallback,
  navigateToExplicitReturnOrFallback,
  readInternalReturnPath,
} from './navigation'

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

    const router = {
      back,
      push,
    } as unknown as Router
    navigateBackOrFallback(router, { name: 'games' })

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

    const router = {
      back,
      push,
    } as unknown as Router
    navigateBackOrFallback(router, { name: 'games' })

    expect(back).not.toHaveBeenCalled()
    expect(push).toHaveBeenCalledWith({ name: 'games' })
  })

  it('accepts only a single internal return path', () => {
    expect(readInternalReturnPath('/games?sort=title')).toBe('/games?sort=title')
    expect(readInternalReturnPath('//example.com/games')).toBeUndefined()
    expect(readInternalReturnPath([' /games', '/dashboard'])).toBeUndefined()
    expect(readInternalReturnPath(undefined)).toBeUndefined()
  })

  it('uses an explicit return path before browser history', () => {
    const back = vi.fn()
    const push = vi.fn()
    const replace = vi.fn()

    vi.stubGlobal('window', {
      history: {
        length: 2,
      },
    })

    const router = {
      back,
      push,
      replace,
    } as unknown as Router
    navigateToExplicitReturnOrFallback(router, '/games', { name: 'dashboard' })

    expect(replace).toHaveBeenCalledWith('/games')
    expect(back).not.toHaveBeenCalled()
    expect(push).not.toHaveBeenCalled()
  })
})
