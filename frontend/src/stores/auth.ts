import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { get, post } from '@/services/api'
import type { ApiEnvelope } from '@/services/types'

export const useAuthStore = defineStore('auth', () => {
  const isAdmin = ref(false)
  const role = ref<'admin' | 'guest'>('guest')
  const initialized = ref(false)
  const authLoadFailed = ref(false)
  const adminDisplayName = ref('')

  const user = computed(() => ({
    username: isAdmin.value ? adminDisplayName.value : 'Guest',
    role: role.value,
  }))

  const fetchMe = async () => {
    try {
      const response = await get<ApiEnvelope<{ is_admin: boolean; role: string; admin_display_name?: string }>>('/auth/me')
      // 2026-04-08: /auth/me already has a native guest/admin response contract.
      // Impact: transport failures stay distinguishable from guest mode instead of being
      // collapsed into the same unauthenticated frontend state.
      isAdmin.value = response.data.is_admin
      // 2026-04-08: auth role now follows the backend response directly.
      // Impact: the frontend no longer rebuilds a parallel "admin/guest" label from is_admin.
      role.value = response.data.role === 'admin' ? 'admin' : 'guest'
      adminDisplayName.value = response.data.admin_display_name?.trim() || ''
      authLoadFailed.value = false
    } catch {
      isAdmin.value = false
      role.value = 'guest'
      adminDisplayName.value = ''
      authLoadFailed.value = true
    } finally {
      initialized.value = true
    }
    return { user: user.value, isAdmin: isAdmin.value, authLoadFailed: authLoadFailed.value }
  }

  const login = async (password: string) => {
    await post<ApiEnvelope<{ is_admin: boolean }>>('/auth/login', { password })
    return fetchMe()
  }

  const logout = async () => {
    await post<ApiEnvelope<{ logged_out: boolean }>>('/auth/logout')
    isAdmin.value = false
    role.value = 'guest'
    adminDisplayName.value = ''
    authLoadFailed.value = false
    initialized.value = true
  }

  return {
    user,
    isAdmin,
    role,
    adminDisplayName,
    authLoadFailed,
    initialized,
    fetchMe,
    login,
    logout,
  }
})
