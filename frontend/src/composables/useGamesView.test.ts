import { describe, expect, it } from 'vitest'

import {
  buildGamesListRequest,
  buildGamesRouteQuery,
  normalizeGamesSortValue,
  parsePositiveQueryNumber,
  parseRouteTagIds,
  readSingleQueryValue,
} from './useGamesView'

describe('useGamesView helpers', () => {
  it('reads the first string query value', () => {
    expect(readSingleQueryValue(['', 'halo', 'ignored'])).toBe('halo')
    expect(readSingleQueryValue('steam')).toBe('steam')
    expect(readSingleQueryValue(undefined)).toBeUndefined()
  })

  it('parses positive query numbers and tag ids safely', () => {
    expect(parsePositiveQueryNumber('24', 12)).toBe(24)
    expect(parsePositiveQueryNumber('0', 12)).toBe(12)
    expect(parseRouteTagIds(['1', 'x', '2', '-3'])).toEqual([1, 2])
  })

  it('normalizes legacy sort aliases', () => {
    expect(normalizeGamesSortValue('newest')).toBe('created_desc')
    expect(normalizeGamesSortValue('downloads')).toBe('downloads_desc')
    expect(normalizeGamesSortValue('random')).toBe('random_desc')
    expect(normalizeGamesSortValue('unexpected')).toBe('created_desc')
  })

  it('builds a cleaned route query and resets page for filter changes', () => {
    const result = buildGamesRouteQuery(
      {
        page: '3',
        limit: '48',
        needs: 'legacy',
        filter: 'favorites',
      },
      {
        search: 'halo',
        filter: undefined,
      },
    )

    expect(result).toEqual({
      page: '1',
      limit: '48',
      search: 'halo',
    })
  })

  it('builds list request params from route query', () => {
    const result = buildGamesListRequest({
      routeQuery: {
        page: '2',
        search: 'halo',
        platform: '3',
        tag: ['1', '2', 'oops'],
        needs_review: 'true',
        seed: '99',
      },
      itemsPerPage: 48,
      filterFavorites: true,
      sortBy: 'random_desc',
    })

    expect(result).toEqual({
      query: {
        page: 2,
        limit: 48,
        search: 'halo',
        platform: '3',
        tag: [1, 2],
        favorite: true,
        needs_review: true,
      },
      sort: {
        field: 'random',
        order: 'desc',
        seed: 99,
      },
    })
  })
})
