import type { RouteRecordRaw } from 'vue-router'

/**
 * Base routes - redirect login to dashboard
 */
export default {
  path: '/login',
  name: 'login',
  redirect: '/',
  meta: {
    hideInMenu: true,
  },
} as RouteRecordRaw