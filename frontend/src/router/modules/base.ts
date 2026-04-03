import type { RouteRecordRaw } from 'vue-router'
const LoginView = () => import('@/views/LoginView.vue')

export default {
  path: '/login',
  name: 'login',
  component: LoginView,
  meta: {
    hideInMenu: true,
  },
} as RouteRecordRaw
