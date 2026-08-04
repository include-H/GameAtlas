<template>
  <section class="pending-overview app-glass-surface">
    <div class="pending-overview__header">
      <icon-exclamation-circle
        v-if="pendingReviews > 0"
        class="pending-overview__icon pending-overview__icon--warning"
      />
      <icon-check-circle
        v-else
        class="pending-overview__icon pending-overview__icon--success"
      />
      <div class="pending-overview__copy">
        <h3 class="pending-overview__title">待处理工作台</h3>
        <p class="pending-overview__desc">
          {{ pendingReviews > 0 ? `有 ${pendingReviews} 款游戏存在待补齐内容` : '当前没有待处理事项' }}
        </p>
      </div>
      <a-button
        v-if="pendingReviews > 0"
        class="pending-overview__action"
        type="primary"
        @click="emit('open-pending')"
      >
        去处理
      </a-button>
    </div>

    <div v-if="groupList.length > 0" class="pending-overview__groups">
      <a-tag
        v-for="group in groupList"
        :key="group.key"
        class="pending-overview__tag"
        color="orangered"
      >
        {{ group.label }} {{ group.count }}
      </a-tag>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { IconCheckCircle, IconExclamationCircle } from '@arco-design/web-vue/es/icon'
import type { PendingIssueCounts } from '@/services/types'

const props = defineProps<{
  pendingReviews: number
  groups?: PendingIssueCounts | null
}>()

const emit = defineEmits<{
  'open-pending': []
}>()

const GROUP_ORDER = ['missing-assets', 'missing-wiki', 'missing-files', 'missing-metadata'] as const
const GROUP_LABELS: Record<string, string> = {
  'missing-assets': '缺图片',
  'missing-wiki': '缺 Wiki',
  'missing-files': '缺文件',
  'missing-metadata': '缺信息',
}

const groupList = computed(() => {
  if (!props.groups) return []
  return GROUP_ORDER
    .filter((key) => (props.groups?.groups[key] ?? 0) > 0)
    .map((key) => ({
      key,
      label: GROUP_LABELS[key] ?? key,
      count: props.groups?.groups[key] ?? 0,
    }))
})
</script>

<style scoped>
.pending-overview {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
  padding: 16px 20px;
  border-radius: var(--radius-lg);
  margin-bottom: 24px;
}

.pending-overview__header {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.pending-overview__icon {
  font-size: 24px;
}

.pending-overview__icon--warning {
  color: rgb(var(--warning-6));
}

.pending-overview__icon--success {
  color: rgb(var(--success-6));
}

.pending-overview__copy {
  min-width: 0;
}

.pending-overview__title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-1);
}

.pending-overview__desc {
  margin: 2px 0 0;
  font-size: 13px;
  color: var(--color-text-3);
}

.pending-overview__groups {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.pending-overview__tag {
  margin-right: 0;
}
</style>
