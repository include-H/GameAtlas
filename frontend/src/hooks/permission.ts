import { RouteLocationNormalized, RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

/**
 * Permission hook for Arco Design Vue Pro
 * Provides route access control based on user roles
 * (Authentication removed - all routes are accessible)
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

  /**
   * Find the first route the user has permission to access
   * @param routes - Array of routes to search
   * @returns First accessible route or null
   */
  const findFirstPermissionRoute = (routes: RouteRecordRaw[]): RouteRecordRaw | null => {
    for (const route of routes) {
      if (accessRouter(route)) {
        return route
      }
      if (route.children) {
        const childRoute = findFirstPermissionRoute(route.children)
        if (childRoute) {
          return childRoute
        }
      }
    }
    return null
  }

  /**
   * Check if user has a specific role
   * @param role - Role to check
   * @returns true if current user has role
   */
  const hasRole = (role: string) => {
    if (role === 'admin') {
      return authStore.isAdmin
    }
    return false
  }

  /**
   * Check if user has any of the specified roles
   * @param roles - Array of roles to check
   * @returns true if current user matches any role
   */
  const hasAnyRole = (roles: string[]) => {
    return roles.some((role) => hasRole(role))
  }

  return {
    accessRouter,
    findFirstPermissionRoute,
    hasRole,
    hasAnyRole,
  }
}
