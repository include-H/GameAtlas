import type { RouteRecordRaw } from 'vue-router'

/**
 * Register route - redirect to dashboard
 */
export default {
  path: '/register',
  name: 'register',
  redirect: '/',
  meta: {
    hideInMenu: true,
  },
} as RouteRecordRaw