import { describe, expect, it } from 'vitest'

import {
  buildGamesSortOptionValue,
  buildGamesListRequest,
  buildGamesRouteQuery,
  hasGamesActiveFilters,
  normalizeGamesFavoriteRouteQuery,
  normalizeGamesPaginationResponseQuery,
  normalizeGamesPaginationRouteQuery,
  normalizeGamesSortRouteQuery,
  parseGamesSortField,
  parseGamesSortOrder,
  readSingleQueryValue,
} from './useGamesView'

describe('useGamesView helpers', () => {
  it('reads the first string query value', () => {
    expect(readSingleQueryValue(['', 'halo', 'ignored'])).toBe('halo')
    expect(readSingleQueryValue('steam')).toBe('steam')
    expect(readSingleQueryValue(undefined)).toBeUndefined()
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
        favorite: undefined,
      },
    })
  })

  it('forwards unsupported sort values so the backend transport layer can reject them', () => {
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
        favorite: undefined,
      },
      sort: {
        field: 'legacy_default',
        order: 'desc',
        seed: undefined,
      },
    })
  })

  it('preserves invalid sort query so the backend can return a transport error', () => {
    const result = normalizeGamesSortRouteQuery(
      {
        page: '2',
        sort: 'legacy_default',
        order: 'desc',
        seed: '123',
        search: 'halo',
      },
    )

    expect(result).toBeNull()
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

  it('preserves favorite=false so the backend transport layer can reject it', () => {
    const result = normalizeGamesFavoriteRouteQuery({
      page: '2',
      favorite: 'false',
      search: 'halo',
    })

    expect(result).toBeNull()
  })

  it('preserves invalid favorite query values so the backend can reject them', () => {
    const result = normalizeGamesFavoriteRouteQuery({
      page: '2',
      favorite: 'favorites',
      search: 'halo',
    })

    expect(result).toBeNull()
  })

  it('preserves favorite=true as the only valid favorite route value', () => {
    const result = normalizeGamesFavoriteRouteQuery({
      page: '2',
      favorite: 'true',
      search: 'halo',
    })

    expect(result).toBeNull()
  })

  it('drops invalid pagination query values before request building', () => {
    const result = normalizeGamesPaginationRouteQuery({
      page: 'oops',
      limit: '999',
      search: 'halo',
    })

    expect(result).toEqual({
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
        favorite: true,
      },
      sort: {
        field: 'random',
        order: 'desc',
        seed: 99,
      },
    })
  })

  it('falls back to the supported default page size when route limit is unsupported', () => {
    const result = buildGamesListRequest({
      routeQuery: {
        page: '2',
        limit: '13',
        search: 'halo',
      },
      itemsPerPage: 24,
    })

    expect(result).toEqual({
      query: {
        page: 2,
        limit: 24,
        search: 'halo',
        favorite: undefined,
      },
    })
  })

  it('treats whitespace-only search as no committed backend filter', () => {
    const result = buildGamesListRequest({
      routeQuery: {
        page: '2',
        search: '   ',
      },
      itemsPerPage: 24,
    })

    expect(result).toEqual({
      query: {
        page: 2,
        limit: 24,
        search: undefined,
        favorite: undefined,
      },
    })
    expect(hasGamesActiveFilters({ search: '   ' })).toBe(false)
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
        favorite: undefined,
      },
      sort: {
        field: 'random',
        order: undefined,
        seed: undefined,
      },
    })
  })

  it('preserves invalid native order so the backend can reject it', () => {
    const result = normalizeGamesSortRouteQuery({
      page: '2',
      sort: 'title',
      order: 'sideways',
      search: 'halo',
    })

    expect(result).toBeNull()
  })

  it('rewrites out-of-range pages to the backend last page', () => {
    const result = normalizeGamesPaginationResponseQuery(
      {
        page: '9',
        limit: '24',
        search: 'halo',
      },
      {
        page: 9,
        limit: 24,
        totalPages: 3,
      },
    )

    expect(result).toEqual({
      page: '3',
      limit: '24',
      search: 'halo',
    })
  })

  it('clears stale page query when the result set becomes empty', () => {
    const result = normalizeGamesPaginationResponseQuery(
      {
        page: '3',
        search: 'halo',
      },
      {
        page: 3,
        limit: 24,
        totalPages: 0,
      },
    )

    expect(result).toEqual({
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
        favorite: true,
      },
    })
  })

  it('forwards favorite=false so the backend can reject the invalid query', () => {
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
        favorite: false,
      },
    })
  })

  it('forwards invalid favorite strings so the backend can reject them', () => {
    const result = buildGamesListRequest({
      routeQuery: {
        page: '2',
        favorite: 'favorites',
      },
      itemsPerPage: 24,
    })

    expect(result).toEqual({
      query: {
        page: 2,
        limit: 24,
        search: undefined,
        favorite: undefined,
        favorite_raw: 'favorites',
      },
    })
  })

  it('treats only committed route filters as active filters', () => {
    expect(hasGamesActiveFilters({})).toBe(false)
    expect(hasGamesActiveFilters({ search: 'halo' })).toBe(true)
    expect(hasGamesActiveFilters({ favorite: 'true' })).toBe(true)
    expect(hasGamesActiveFilters({ favorite: 'false' })).toBe(false)
  })
})
