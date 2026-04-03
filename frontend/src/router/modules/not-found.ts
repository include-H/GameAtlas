import type { RouteRecordRaw } from 'vue-router'

/**
 * 404 Not Found route
 */
export default {
  path: '/:pathMatch(.*)*',
  name: 'not-found',
  component: () => import('@/views/NotFoundView.vue'),
  meta: {
    title: '未找到页面',
    hideInMenu: true,
  },
} as RouteRecordRaw
