import type { RouteRecordRaw } from 'vue-router'

/**
 * 404 Not Found route
 */
export default {
  path: '/:pathMatch(.*)*',
  name: 'not-found',
  component: () => import('@/views/NotFoundView.vue'),
  meta: {
    locale: 'menu.notFound',
    hideInMenu: true,
    keepAlive: false,
  },
} as RouteRecordRaw
