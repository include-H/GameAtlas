import { describe, expect, it } from 'vitest'

import {
  buildGamesSortOptionValue,
  buildGamesListRequest,
  buildGamesRouteQuery,
  hasGamesActiveFilters,
  normalizeGamesFavoriteRouteQuery,
  normalizeGamesSortRouteQuery,
  parseGamesSortField,
  parseGamesSortOrder,
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

  it('parses only supported native sort fields and orders', () => {
    expect(parseGamesSortField('updated_at')).toBe('updated_at')
    expect(parseGamesSortField('created_at')).toBe('created_at')
    expect(parseGamesSortField('downloads')).toBe('downloads')
    expect(parseGamesSortField('random')).toBe('random')
    expect(parseGamesSortField('unexpected')).toBeUndefined()
    expect(parseGamesSortOrder('asc')).toBe('asc')
    expect(parseGamesSortOrder('desc')).toBe('desc')
    expect(parseGamesSortOrder('sideways')).toBeUndefined()
    expect(buildGamesSortOptionValue('updated_at', 'desc')).toBe('updated_at:desc')
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
        order: 'desc',
      },
      itemsPerPage: 24,
    })

    expect(result).toEqual({
      query: {
        page: 2,
        limit: 24,
        search: 'halo',
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
        order: 'desc',
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
      sort: 'updated_at',
      order: 'desc',
      seed: '123',
      search: 'halo',
    })

    expect(result).toEqual({
      page: '2',
      sort: 'updated_at',
      order: 'desc',
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
      sort: 'random',
      order: 'desc',
      search: 'halo',
    })

    expect(result).toMatchObject({
      page: '2',
      sort: 'random',
      order: 'desc',
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
        tag: ['1', '2', 'oops'],
        sort: 'random',
        order: 'desc',
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
        sort: 'random',
      },
      itemsPerPage: 48,
    })

    expect(result).toEqual({
      query: {
        page: 2,
        limit: 48,
        search: undefined,
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

  it('drops invalid native order while preserving supported sort', () => {
    const result = normalizeGamesSortRouteQuery({
      page: '2',
      sort: 'title',
      order: 'sideways',
      search: 'halo',
    })

    expect(result).toEqual({
      page: '2',
      sort: 'title',
      search: 'halo',
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
        tag: [],
        favorite: undefined,
      },
    })
  })

  it('treats only committed route filters as active filters', () => {
    expect(hasGamesActiveFilters({})).toBe(false)
    expect(hasGamesActiveFilters({ search: 'halo' })).toBe(true)
    expect(hasGamesActiveFilters({ tag: ['1'] })).toBe(true)
    expect(hasGamesActiveFilters({ favorite: 'true' })).toBe(true)
    expect(hasGamesActiveFilters({ favorite: 'false' })).toBe(false)
  })
})
