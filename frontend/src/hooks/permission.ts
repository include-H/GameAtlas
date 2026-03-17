import { RouteLocationNormalized, RouteRecordRaw } from 'vue-router'

/**
 * Permission hook for Arco Design Vue Pro
 * Provides route access control based on user roles
 * (Authentication removed - all routes are accessible)
 */
export default function usePermission() {
  /**
   * Check if the current user has access to a route
   * @param route - Route to check access for
   * @returns true (always - no auth required)
   */
  const accessRouter = (_route: RouteLocationNormalized | RouteRecordRaw) => {
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
   * @returns true (always - no auth required)
   */
  const hasRole = (_role: string) => {
    return true
  }

  /**
   * Check if user has any of the specified roles
   * @param roles - Array of roles to check
   * @returns true (always - no auth required)
   */
  const hasAnyRole = (_roles: string[]) => {
    return true
  }

  return {
    accessRouter,
    findFirstPermissionRoute,
    hasRole,
    hasAnyRole,
  }
}