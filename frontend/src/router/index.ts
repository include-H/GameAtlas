import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'

// Import route modules
import base from './modules/base'
import dashboard from './modules/dashboard'
import gameStore from './modules/game-store'
import games, { gameDetailRoute, pendingCenterRoute, timelineRoute, wikiEditRoute } from './modules/games'
import series, { seriesDetailRoute } from './modules/series'
import publishers, { publisherDetailRoute } from './modules/publishers'
import settings from './modules/settings'
import notFound from './modules/not-found'

/**
 * Application routes
 * Organized by feature modules
 */
export const appRoutes: RouteRecordRaw[] = [
  dashboard,
  gameStore,
  games,
  timelineRoute,
  series,
  publishers,
  pendingCenterRoute,
  gameDetailRoute,
  seriesDetailRoute,
  publisherDetailRoute,
  wikiEditRoute,
  settings,
]

/**
 * All routes including public routes
 */
const routes: RouteRecordRaw[] = [
  base,
  ...appRoutes,
  notFound,
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
})

// 2026-05-15: Edge 最小化 bug workaround —— Edge 在 visibilitychange 事件中
// 调用 history.replaceState 会触发内部"页面激活"机制，导致最小化后立刻弹回。
// Vue Router 在 visibilitychange 中调用 replaceState 更新历史状态，
// 其他浏览器正常，唯独 Edge 有此问题。暂时拦截 replaceState 的 state 参数。
// 参考：https://www.cnblogs.com/misillas/p/19614838
if (typeof window !== 'undefined' && /Edg\//.test(navigator.userAgent)) {
  const originalReplaceState = window.history.replaceState.bind(window.history)
  window.history.replaceState = (state: unknown, _title: string, url?: string | URL | null) => {
    originalReplaceState(state, '', url)
  }
}

const isCompactNavigationViewport = () => {
  if (typeof window === 'undefined') return false
  return window.innerWidth < 992
}

router.beforeEach(async (to) => {
  const authStore = useAuthStore()
  const uiStore = useUiStore()

  if (!authStore.initialized) {
    await authStore.fetchMe()
  }

  const requiresAdmin = !!to.meta?.requiresAdmin

  if (requiresAdmin && authStore.authLoadFailed) {
    uiStore.addAlert('认证状态加载失败，请稍后重试', 'error')
    return { name: 'dashboard' }
  }

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
