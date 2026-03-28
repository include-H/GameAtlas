import { describe, expect, it } from 'vitest'

import { createDetailRouteQuery, resolveReturnRoute } from './navigation'

describe('navigation helpers', () => {
  it('creates a return query from route path name and string query values', () => {
    const result = createDetailRouteQuery({
      path: '/games',
      name: 'games',
      query: {
        page: '2',
        keyword: ['halo', 'ignored'],
        empty: '',
        invalid: 123,
      },
    } as never)

    expect(result).toEqual({
      returnPath: '/games',
      returnName: 'games',
      returnQuery_page: '2',
      returnQuery_keyword: 'halo',
    })
  })

  it('resolves a named return route when returnName exists', () => {
    const result = resolveReturnRoute(
      {
        query: {
          returnPath: '/games',
          returnName: 'games',
          returnQuery_page: '3',
        },
      } as never,
      { name: 'dashboard' },
    )

    expect(result).toEqual({
      name: 'games',
      query: {
        page: '3',
      },
    })
  })

  it('falls back when no return route is present', () => {
    const fallback = { name: 'dashboard' }

    expect(resolveReturnRoute({ query: {} } as never, fallback)).toBe(fallback)
  })
})
