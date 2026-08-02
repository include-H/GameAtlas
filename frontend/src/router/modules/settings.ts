import type { RouteRecordRaw } from 'vue-router'
import { IconSettings } from '@arco-design/web-vue/es/icon'

export default {
  path: '/settings',
  name: 'settings',
  component: () => import('@/views/SettingsView.vue'),
  meta: {
    title: '设置',
    icon: IconSettings,
    requiresAdmin: true,
  },
} as RouteRecordRaw
