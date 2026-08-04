import type { RouteRecordRaw } from 'vue-router'
import { IconTag } from '@arco-design/web-vue/es/icon'

export default {
  path: '/publishers',
  name: 'publisher-library',
  component: () => import('@/views/PublisherLibraryView.vue'),
  meta: {
    title: '发行商库',
    icon: IconTag,
  },
} as RouteRecordRaw

export const publisherDetailRoute = {
  path: '/publishers/:id',
  name: 'publisher-detail',
  component: () => import('@/views/PublisherDetailView.vue'),
  meta: {
    title: '发行商详情',
    hideInMenu: true,
    activeMenu: 'publisher-library',
  },
} as RouteRecordRaw
