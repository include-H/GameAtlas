import { describe, expect, it } from 'vitest'

import {
  buildGamesSortOptionValue,
  buildGamesListRequest,
  buildGamesRouteQuery,
  hasGamesActiveFilters,
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
    })

    expect(result).toEqual({
      query: {
        page: 1,
        limit: 24,
        search: 'halo',
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
    })

    expect(result).toEqual({
      query: {
        page: 1,
        limit: 24,
        search: 'halo',
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
      sort: 'updated_at',
      order: 'desc',
      search: 'halo',
    })
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

  it('drops valid page query so infinite scroll starts from the first page', () => {
    const result = normalizeGamesPaginationRouteQuery({
      page: '3',
      limit: '48',
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
      sort: 'random',
      order: 'desc',
      search: 'halo',
    })
    expect(Number(result?.seed)).toBeGreaterThan(0)
  })

  it('drops pagination params when building a cleaned route query', () => {
    const result = buildGamesRouteQuery(
      {
        page: '3',
        limit: '48',
      },
      {
        search: 'halo',
      },
    )

    expect(result).toEqual({
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
      },
    })

    expect(result).toEqual({
      query: {
        page: 1,
        limit: 24,
        search: 'halo',
      },
      sort: {
        field: 'random',
        order: 'desc',
        seed: 99,
      },
    })
  })

  it('ignores route limit and always uses the fixed infinite scroll page size', () => {
    const result = buildGamesListRequest({
      routeQuery: {
        page: '2',
        limit: '13',
        search: 'halo',
      },
    })

    expect(result).toEqual({
      query: {
        page: 1,
        limit: 24,
        search: 'halo',
      },
    })
  })

  it('treats whitespace-only search as no committed backend filter', () => {
    const result = buildGamesListRequest({
      routeQuery: {
        page: '2',
        search: '   ',
      },
    })

    expect(result).toEqual({
      query: {
        page: 1,
        limit: 24,
        search: undefined,
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
    })

    expect(result).toEqual({
      query: {
        page: 1,
        limit: 24,
        search: undefined,
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

  it('treats only committed route filters as active filters', () => {
    expect(hasGamesActiveFilters({})).toBe(false)
    expect(hasGamesActiveFilters({ search: 'halo' })).toBe(true)
  })
})
