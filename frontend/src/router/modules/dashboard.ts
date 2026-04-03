import type { RouteRecordRaw } from 'vue-router'
import { IconHome } from '@arco-design/web-vue/es/icon'

/**
 * Dashboard routes
 */
export default {
  path: '/',
  name: 'dashboard',
  component: () => import('@/views/DashboardView.vue'),
  meta: {
    title: '首页',
    icon: IconHome,
  },
} as RouteRecordRaw
