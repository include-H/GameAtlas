import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { get, post } from '@/services/api'
import type { ApiResponse } from '@/services/types'

export const useAuthStore = defineStore('auth', () => {
  const isAdmin = ref(false)
  const initialized = ref(false)

  const user = computed(() => ({
    username: isAdmin.value ? 'Admin' : 'Guest',
    role: isAdmin.value ? 'admin' : 'guest',
  }))

  const fetchMe = async () => {
    try {
      const response = await get<ApiResponse<{ is_admin: boolean; role: string }>>('/auth/me')
      isAdmin.value = !!response.data?.is_admin
    } catch {
      isAdmin.value = false
    } finally {
      initialized.value = true
    }
    return { user: user.value, isAdmin: isAdmin.value }
  }

  const login = async (password: string) => {
    await post<ApiResponse<{ is_admin: boolean }>>('/auth/login', { password })
    isAdmin.value = true
    initialized.value = true
    return { user: user.value, isAdmin: true }
  }

  const logout = async () => {
    await post<ApiResponse<{ logged_out: boolean }>>('/auth/logout')
    isAdmin.value = false
    initialized.value = true
  }

  return {
    user,
    isAdmin,
    initialized,
    fetchMe,
    login,
    logout,
  }
})
