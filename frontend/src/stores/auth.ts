import { defineStore } from 'pinia'
import { ref } from 'vue'

/**
 * Auth store (simplified - no authentication required)
 */
export const useAuthStore = defineStore('auth', () => {
  const user = ref<{ username: string; role: string }>({
    username: 'User',
    role: 'admin',
  })

  const login = async () => {
    return { token: '', user: user.value }
  }

  const register = async () => {
    return { token: '', user: user.value }
  }

  return {
    user,
    login,
    register,
  }
})
