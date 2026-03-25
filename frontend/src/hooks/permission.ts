import { RouteLocationNormalized, RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

/**
 * Permission hook for Arco Design Vue Pro
 * Provides route access control based on route meta and current auth state.
 * Routes marked with `meta.requiresAdmin` are only accessible to admins.
 */
export default function usePermission() {
  const authStore = useAuthStore()
  /**
   * Check if the current user has access to a route
   * @param route - Route to check access for
   * @returns true when current auth state permits access
   */
  const accessRouter = (route: RouteLocationNormalized | RouteRecordRaw) => {
    if (route.meta?.requiresAdmin) {
      return authStore.isAdmin
    }
    return true
  }

  return {
    accessRouter,
  }
}
