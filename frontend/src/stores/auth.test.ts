import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const { getMock, postMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
  postMock: vi.fn(),
}))

vi.mock('@/services/api', () => ({
  get: getMock,
  post: postMock,
}))

import { useAuthStore } from './auth'

describe('auth store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    getMock.mockReset()
    postMock.mockReset()
  })

  it('fetches the current admin user', async () => {
    getMock.mockResolvedValue({
      data: {
        is_admin: true,
        role: 'admin',
        admin_display_name: 'Boss',
      },
    })

    const store = useAuthStore()
    const result = await store.fetchMe()

    expect(getMock).toHaveBeenCalledWith('/auth/me')
    expect(store.isAdmin).toBe(true)
    expect(store.adminDisplayName).toBe('Boss')
    expect(store.initialized).toBe(true)
    expect(store.user).toEqual({
      username: 'Boss',
      role: 'admin',
    })
    expect(result).toEqual({
      user: {
        username: 'Boss',
        role: 'admin',
      },
      isAdmin: true,
    })
  })

  it('falls back to guest state when fetchMe fails', async () => {
    getMock.mockRejectedValue(new Error('network failed'))

    const store = useAuthStore()
    store.isAdmin = true
    store.adminDisplayName = 'Someone'

    const result = await store.fetchMe()

    expect(store.isAdmin).toBe(false)
    expect(store.adminDisplayName).toBe('Admin')
    expect(store.initialized).toBe(true)
    expect(result).toEqual({
      user: {
        username: 'Guest',
        role: 'guest',
      },
      isAdmin: false,
    })
  })

  it('logs in via post and refreshes auth state', async () => {
    postMock.mockResolvedValue({ data: { is_admin: true } })
    getMock.mockResolvedValue({
      data: {
        is_admin: true,
        role: 'admin',
        admin_display_name: 'Lead Admin',
      },
    })

    const store = useAuthStore()
    const result = await store.login('secret')

    expect(postMock).toHaveBeenCalledWith('/auth/login', { password: 'secret' })
    expect(getMock).toHaveBeenCalledWith('/auth/me')
    expect(result).toEqual({
      user: {
        username: 'Lead Admin',
        role: 'admin',
      },
      isAdmin: true,
    })
  })

  it('logs out and resets store state', async () => {
    postMock.mockResolvedValue({ data: { logged_out: true } })

    const store = useAuthStore()
    store.isAdmin = true
    store.adminDisplayName = 'Boss'
    store.initialized = false

    await store.logout()

    expect(postMock).toHaveBeenCalledWith('/auth/logout')
    expect(store.isAdmin).toBe(false)
    expect(store.adminDisplayName).toBe('Admin')
    expect(store.initialized).toBe(true)
  })
})
