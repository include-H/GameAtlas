import type { RouteRecordRaw } from 'vue-router'
import { IconApps } from '@arco-design/web-vue/es/icon'

export default {
  path: '/series',
  name: 'series-library',
  component: () => import('@/views/SeriesLibraryView.vue'),
  meta: {
    title: '系列库',
    icon: IconApps,
  },
} as RouteRecordRaw

export const seriesDetailRoute = {
  path: '/series/:id',
  name: 'series-detail',
  component: () => import('@/views/SeriesDetailView.vue'),
  meta: {
    title: '系列详情',
    hideInMenu: true,
    activeMenu: 'series-library',
  },
} as RouteRecordRaw
