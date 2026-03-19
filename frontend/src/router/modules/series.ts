import type { RouteRecordRaw } from 'vue-router'
import { IconApps } from '@arco-design/web-vue/es/icon'

export default {
  path: '/series',
  name: 'series-library',
  component: () => import('@/views/SeriesLibraryView.vue'),
  meta: {
    locale: 'menu.series',
    requiresAuth: true,
    icon: IconApps,
    roles: ['*'],
  },
} as RouteRecordRaw

export const seriesDetailRoute = {
  path: '/series/:id',
  name: 'series-detail',
  component: () => import('@/views/SeriesDetailView.vue'),
  meta: {
    locale: 'menu.series.detail',
    requiresAuth: true,
    roles: ['*'],
    hideInMenu: true,
    activeMenu: 'series-library',
  },
} as RouteRecordRaw
