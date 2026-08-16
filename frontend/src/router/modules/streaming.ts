import type { RouteRecordRaw } from 'vue-router'
import { IconThunderbolt } from '@arco-design/web-vue/es/icon'

/**
 * 云串流入口（独立文档，由 Go 串流代理托管于 :47999）
 */
export default {
  path: '/stream',
  name: 'stream',
  component: () => import('@/views/StreamingEntryView.vue'),
  meta: {
    title: '云串流',
    icon: IconThunderbolt,
  },
} as RouteRecordRaw
