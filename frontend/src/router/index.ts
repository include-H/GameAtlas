import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

// Import route modules
import base from './modules/base'
import register from './modules/register'
import dashboard from './modules/dashboard'
import games, { gameDetailRoute, pendingCenterRoute, timelineRoute, wikiEditRoute } from './modules/games'
import series, { seriesDetailRoute } from './modules/series'
import notFound from './modules/not-found'

/**
 * Application routes
 * Organized by feature modules
 */
export const appRoutes: RouteRecordRaw[] = [
  dashboard,
  games,
  timelineRoute,
  series,
  pendingCenterRoute,
  gameDetailRoute,
  seriesDetailRoute,
  wikiEditRoute,
]

/**
 * All routes including public routes
 */
const routes: RouteRecordRaw[] = [
  base,
  register,
  ...appRoutes,
  notFound,
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
})

const isCompactNavigationViewport = () => {
  if (typeof window === 'undefined') return false
  return window.innerWidth < 992
}

router.beforeEach(async (to) => {
  const authStore = useAuthStore()

  if (!authStore.initialized) {
    await authStore.fetchMe()
  }

  const requiresAdmin = !!to.meta?.requiresAdmin

  if (requiresAdmin && !authStore.isAdmin) {
    return {
      name: 'login',
      query: { redirect: to.fullPath },
    }
  }

  if (to.meta?.hideOnCompactNavigation && isCompactNavigationViewport()) {
    return { name: 'dashboard' }
  }

  if (to.name === 'login' && authStore.isAdmin) {
    return { name: 'dashboard' }
  }

  return true
})

export default router
