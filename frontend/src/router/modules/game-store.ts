import type { RouteRecordRaw } from 'vue-router'
import { IconFire } from '@arco-design/web-vue/es/icon'

/**
 * 游戏店：第二种浏览方式（随机发现 / 随便逛逛）。
 * 保留后台框架（顶部/侧边/底部），在内容区全屏铺满场景。
 */
export default {
  path: '/store',
  name: 'game-store',
  component: () => import('@/views/GameStoreView.vue'),
  meta: {
    title: '游戏店',
    icon: IconFire,
  },
} as RouteRecordRaw
