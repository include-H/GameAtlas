import { describe, expect, it, vi } from 'vitest'
import type { RouteRecordRaw } from 'vue-router'

const authState = vi.hoisted(() => ({ isAdmin: false }))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authState,
}))

import usePermission from './usePermission'

const publicRoute = {
  path: '/public',
  component: { render: () => null },
  meta: {},
} as unknown as RouteRecordRaw

const adminRoute = {
  path: '/admin',
  component: { render: () => null },
  meta: { requiresAdmin: true },
} as unknown as RouteRecordRaw

describe('usePermission', () => {
  it('allows public routes regardless of auth state', () => {
    authState.isAdmin = false
    expect(usePermission().accessRouter(publicRoute)).toBe(true)

    authState.isAdmin = true
    expect(usePermission().accessRouter(publicRoute)).toBe(true)
  })

  it('blocks admin-only routes for guests', () => {
    authState.isAdmin = false
    expect(usePermission().accessRouter(adminRoute)).toBe(false)
  })

  it('allows admin-only routes for admins', () => {
    authState.isAdmin = true
    expect(usePermission().accessRouter(adminRoute)).toBe(true)
  })
})
