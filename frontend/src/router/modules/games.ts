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
    locale: 'menu.games',
    requiresAuth: true,
    icon: IconTrophy,
    roles: ['*'],
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
    locale: 'menu.games.timeline',
    requiresAuth: true,
    icon: IconCalendarClock,
    roles: ['*'],
  },
} as RouteRecordRaw

/**
 * Game detail route
 */
export const gameDetailRoute = {
  path: '/games/:id',
  name: 'game-detail',
  component: () => import('@/views/GameDetailView.vue'),
  meta: {
    locale: 'menu.game.detail',
    requiresAuth: true,
    roles: ['*'],
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
    locale: 'menu.pending.center',
    requiresAuth: true,
    requiresAdmin: true,
    roles: ['*'],
    icon: IconExclamationCircle,
  },
} as RouteRecordRaw

/**
 * Wiki edit route
 */
export const wikiEditRoute = {
  path: '/wiki/:gameId/edit',
  name: 'wiki-edit',
  component: () => import('@/views/WikiEditView.vue'),
  meta: {
    locale: 'menu.wiki.edit',
    requiresAuth: true,
    requiresAdmin: true,
    roles: ['*'],
    hideInMenu: true,
    activeMenu: 'games',
  },
} as RouteRecordRaw
