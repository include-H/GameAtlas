import type { RouteRecordRaw } from 'vue-router'
import { IconTrophy, IconExclamationCircle, IconCalendarClock } from '@arco-design/web-vue/es/icon'

/**
 * Games routes
 */
export default {
  path: '/games',
  name: 'games',
  component: () => import('@/views/GamesView.vue'),
  meta: {
    title: '游戏库',
    icon: IconTrophy,
  },
} as RouteRecordRaw

/**
 * Timeline route
 */
export const timelineRoute = {
  path: '/games/timeline',
  name: 'games-timeline',
  component: () => import('@/views/GamesTimelineView.vue'),
  meta: {
    title: '发售时间线',
    icon: IconCalendarClock,
  },
} as RouteRecordRaw

/**
 * Game detail route
 */
export const gameDetailRoute = {
  path: '/games/:publicId',
  name: 'game-detail',
  component: () => import('@/views/GameDetailView.vue'),
  meta: {
    title: '游戏详情',
    hideInMenu: true,
    activeMenu: 'games',
  },
} as RouteRecordRaw

/**
 * Pending center route
 */
export const pendingCenterRoute = {
  path: '/games/pending',
  name: 'pending-center',
  component: () => import('@/views/PendingCenterView.vue'),
  meta: {
    // PendingCenter compact-layout adaptation is intentionally deferred for now,
    // so we hide this entry on phone/tablet widths by default to avoid exposing a broken workflow.
    hideOnCompactNavigation: true,
    title: '待处理工作台',
    requiresAdmin: true,
    icon: IconExclamationCircle,
  },
} as RouteRecordRaw

/**
 * Wiki edit route
 */
export const wikiEditRoute = {
  path: '/wiki/:publicId/edit',
  name: 'wiki-edit',
  component: () => import('@/views/WikiEditView.vue'),
  meta: {
    title: '编辑Wiki',
    requiresAdmin: true,
    hideInMenu: true,
    activeMenu: 'games',
  },
} as RouteRecordRaw
