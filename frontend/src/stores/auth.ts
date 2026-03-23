import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { get, post } from '@/services/api'
import type { ApiResponse } from '@/services/types'

export const useAuthStore = defineStore('auth', () => {
  const isAdmin = ref(false)
  const initialized = ref(false)
  const adminDisplayName = ref('Admin')

  const user = computed(() => ({
    username: isAdmin.value ? adminDisplayName.value : 'Guest',
    role: isAdmin.value ? 'admin' : 'guest',
  }))

  const fetchMe = async () => {
    try {
      const response = await get<ApiResponse<{ is_admin: boolean; role: string; admin_display_name?: string }>>('/auth/me')
      isAdmin.value = !!response.data?.is_admin
      adminDisplayName.value = response.data?.admin_display_name?.trim() || 'Admin'
    } catch {
      isAdmin.value = false
      adminDisplayName.value = 'Admin'
    } finally {
      initialized.value = true
    }
    return { user: user.value, isAdmin: isAdmin.value }
  }

  const login = async (password: string) => {
    await post<ApiResponse<{ is_admin: boolean }>>('/auth/login', { password })
    return fetchMe()
  }

  const logout = async () => {
    await post<ApiResponse<{ logged_out: boolean }>>('/auth/logout')
    isAdmin.value = false
    adminDisplayName.value = 'Admin'
    initialized.value = true
  }

  return {
    user,
    isAdmin,
    adminDisplayName,
    initialized,
    fetchMe,
    login,
    logout,
  }
})
