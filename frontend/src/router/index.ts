import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'

// Import route modules
import base from './modules/base'
import register from './modules/register'
import dashboard from './modules/dashboard'
import games, { gameDetailRoute, wikiEditRoute } from './modules/games'
import notFound from './modules/not-found'

/**
 * Application routes
 * Organized by feature modules
 */
export const appRoutes: RouteRecordRaw[] = [
  dashboard,
  games,
  gameDetailRoute,
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

export default router