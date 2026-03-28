import type { RouteRecordRaw } from 'vue-router'
import { IconApps } from '@arco-design/web-vue/es/icon'

export default {
  path: '/series',
  name: 'series-library',
  component: () => import('@/views/SeriesLibraryView.vue'),
  meta: {
    locale: 'menu.series',
    keepAlive: true,
    icon: IconApps,
  },
} as RouteRecordRaw

export const seriesDetailRoute = {
  path: '/series/:id',
  name: 'series-detail',
  component: () => import('@/views/SeriesDetailView.vue'),
  meta: {
    locale: 'menu.series.detail',
    keepAlive: true,
    hideInMenu: true,
    activeMenu: 'series-library',
  },
} as RouteRecordRaw
