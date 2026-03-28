import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { applyFavorite, getFavoriteCount, readFavorites, writeFavorites } from './game-list-helpers'

const FAVORITES_KEY = 'game-library-favorites'

describe('game-list-helpers', () => {
  beforeEach(() => {
    window.localStorage.clear()
  })

  afterEach(() => {
    window.localStorage.clear()
  })

  it('reads and writes favorites from localStorage', () => {
    writeFavorites(['1', '2'])

    expect(readFavorites()).toEqual(['1', '2'])
    expect(getFavoriteCount()).toBe(2)
  })

  it('returns an empty list for invalid stored payloads', () => {
    window.localStorage.setItem(FAVORITES_KEY, '{bad json')
    expect(readFavorites()).toEqual([])

    window.localStorage.setItem(FAVORITES_KEY, JSON.stringify({ id: 1 }))
    expect(readFavorites()).toEqual([])
  })

  it('marks games as favorite based on provided ids or persisted favorites', () => {
    writeFavorites(['game-1'])

    expect(applyFavorite({ public_id: 'game-1', title: 'A' })).toEqual({
      public_id: 'game-1',
      title: 'A',
      isFavorite: true,
    })

    expect(applyFavorite({ public_id: 'game-2', title: 'B' }, new Set(['game-2']))).toEqual({
      public_id: 'game-2',
      title: 'B',
      isFavorite: true,
    })

    expect(applyFavorite({ public_id: '', title: 'C' })).toEqual({
      public_id: '',
      title: 'C',
      isFavorite: false,
    })
  })

  it('returns an empty list when localStorage access throws', () => {
    const getItemMock = vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new DOMException('Blocked', 'SecurityError')
    })

    expect(readFavorites()).toEqual([])

    getItemMock.mockRestore()
  })

  it('ignores localStorage write failures', () => {
    const setItemMock = vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new DOMException('Blocked', 'SecurityError')
    })

    expect(() => writeFavorites(['1', '2'])).not.toThrow()

    setItemMock.mockRestore()
  })
})
