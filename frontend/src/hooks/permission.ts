import { RouteLocationNormalized, RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

/**
 * Permission hook for route and menu access control.
 * The current frontend only distinguishes between public routes and admin-only routes.
 */
export default function usePermission() {
  const authStore = useAuthStore()

  /**
   * Check if the current user has access to a route
   * @param route - Route to check access for
   * @returns true when current auth state permits access
   */
  const accessRouter = (route: RouteLocationNormalized | RouteRecordRaw) => {
    return !route.meta?.requiresAdmin || authStore.isAdmin
  }

  return {
    accessRouter,
  }
}
