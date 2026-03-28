import type { RouteRecordRaw } from 'vue-router'

export default {
  path: '/register',
  name: 'register',
  redirect: '/login',
  meta: {
    hideInMenu: true,
  },
} as RouteRecordRaw
