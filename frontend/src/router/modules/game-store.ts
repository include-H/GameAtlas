import type { RouteRecordRaw } from 'vue-router'
import { IconFire } from '@arco-design/web-vue/es/icon'

/**
 * 游戏店：第二种浏览方式（随机发现 / 随便逛逛）。
 * 独立全屏场景页，不套用后台管理布局。
 */
export default {
  path: '/store',
  name: 'game-store',
  component: () => import('@/views/GameStoreView.vue'),
  meta: {
    title: '游戏店',
    icon: IconFire,
    fullscreen: true,
  },
} as RouteRecordRaw
