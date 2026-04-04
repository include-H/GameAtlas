import { describe, expect, it } from 'vitest'

import {
  buildGamesListRequest,
  buildGamesRouteQuery,
  hasGamesActiveFilters,
  normalizeGamesFavoriteRouteQuery,
  normalizeGamesSortRouteQuery,
  parseGamesSortValue,
  parsePositiveQueryNumber,
  parseRouteBoolean,
  parsePositiveRouteNumber,
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
    expect(parseRouteBoolean('true')).toBe(true)
    expect(parseRouteBoolean('false')).toBe(false)
    expect(parseRouteBoolean('favorites')).toBeUndefined()
    expect(parsePositiveRouteNumber('3')).toBe(3)
    expect(parsePositiveRouteNumber('pc')).toBeUndefined()
    expect(parseRouteTagIds(['1', 'x', '2', '-3'])).toEqual([1, 2])
  })

  it('parses only supported sort values', () => {
    expect(parseGamesSortValue('updated_desc')).toBe('updated_desc')
    expect(parseGamesSortValue('created_desc')).toBe('created_desc')
    expect(parseGamesSortValue('downloads_desc')).toBe('downloads_desc')
    expect(parseGamesSortValue('random_desc')).toBe('random_desc')
    expect(parseGamesSortValue('unexpected')).toBeUndefined()
  })

  it('uses backend default sort when route does not declare one', () => {
    const result = buildGamesListRequest({
      routeQuery: {
        page: '2',
        search: 'halo',
      },
      itemsPerPage: 24,
    })

    expect(result).toEqual({
      query: {
        page: 2,
        limit: 24,
        search: 'halo',
        platform: undefined,
        tag: [],
        favorite: undefined,
      },
    })
  })

  it('uses backend default sort when route declares an unsupported value', () => {
    const result = buildGamesListRequest({
      routeQuery: {
        page: '2',
        search: 'halo',
        sort: 'legacy_default',
      },
      itemsPerPage: 24,
    })

    expect(result).toEqual({
      query: {
        page: 2,
        limit: 24,
        search: 'halo',
        platform: undefined,
        tag: [],
        favorite: undefined,
      },
    })
  })

  it('drops invalid sort and seed from route query', () => {
    const result = normalizeGamesSortRouteQuery(
      {
        page: '2',
        sort: 'legacy_default',
        seed: '123',
        search: 'halo',
      },
    )

    expect(result).toEqual({
      page: '2',
      search: 'halo',
    })
  })

  it('drops stale seed when sort is no longer random', () => {
    const result = normalizeGamesSortRouteQuery({
      page: '2',
      sort: 'updated_desc',
      seed: '123',
      search: 'halo',
    })

    expect(result).toEqual({
      page: '2',
      sort: 'updated_desc',
      search: 'halo',
    })
  })

  it('drops favorite=false because backend only defines favorite=true as a filter', () => {
    const result = normalizeGamesFavoriteRouteQuery({
      page: '2',
      favorite: 'false',
      search: 'halo',
    })

    expect(result).toEqual({
      page: '2',
      search: 'halo',
    })
  })

  it('drops invalid favorite query values', () => {
    const result = normalizeGamesFavoriteRouteQuery({
      page: '2',
      favorite: 'favorites',
      search: 'halo',
    })

    expect(result).toEqual({
      page: '2',
      search: 'halo',
    })
  })

  it('adds route seed when random sort is missing one', () => {
    const result = normalizeGamesSortRouteQuery({
      page: '2',
      sort: 'random_desc',
      search: 'halo',
    })

    expect(result).toMatchObject({
      page: '2',
      sort: 'random_desc',
      search: 'halo',
    })
    expect(Number(result?.seed)).toBeGreaterThan(0)
  })

  it('builds a cleaned route query and resets page for filter changes', () => {
    const result = buildGamesRouteQuery(
      {
        page: '3',
        limit: '48',
        favorite: 'true',
      },
      {
        search: 'halo',
        favorite: undefined,
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
        sort: 'random_desc',
        seed: '99',
        favorite: 'true',
      },
      itemsPerPage: 48,
    })

    expect(result).toEqual({
      query: {
        page: 2,
        limit: 48,
        search: 'halo',
        platform: 3,
        tag: [1, 2],
        favorite: true,
      },
      sort: {
        field: 'random',
        order: 'desc',
        seed: 99,
      },
    })
  })

  it('does not invent random seed when route has not committed one', () => {
    const result = buildGamesListRequest({
      routeQuery: {
        page: '2',
        sort: 'random_desc',
      },
      itemsPerPage: 48,
    })

    expect(result).toEqual({
      query: {
        page: 2,
        limit: 48,
        search: undefined,
        platform: undefined,
        tag: [],
        favorite: undefined,
      },
      sort: {
        field: 'random',
        order: 'desc',
        seed: undefined,
      },
    })
  })

  it('passes native favorite route semantics through to the backend request', () => {
    const result = buildGamesListRequest({
      routeQuery: {
        page: '2',
        favorite: 'true',
      },
      itemsPerPage: 24,
    })

    expect(result).toEqual({
      query: {
        page: 2,
        limit: 24,
        search: undefined,
        platform: undefined,
        tag: [],
        favorite: true,
      },
    })
  })

  it('drops favorite=false before building the backend request', () => {
    const result = buildGamesListRequest({
      routeQuery: {
        page: '2',
        favorite: 'false',
      },
      itemsPerPage: 24,
    })

    expect(result).toEqual({
      query: {
        page: 2,
        limit: 24,
        search: undefined,
        platform: undefined,
        tag: [],
        favorite: undefined,
      },
    })
  })

  it('treats only committed route filters as active filters', () => {
    expect(hasGamesActiveFilters({})).toBe(false)
    expect(hasGamesActiveFilters({ search: 'halo' })).toBe(true)
    expect(hasGamesActiveFilters({ platform: '3' })).toBe(true)
    expect(hasGamesActiveFilters({ tag: ['1'] })).toBe(true)
    expect(hasGamesActiveFilters({ favorite: 'true' })).toBe(true)
    expect(hasGamesActiveFilters({ favorite: 'false' })).toBe(false)
  })
})
