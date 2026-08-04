<template>
  <Teleport to="body">
    <div class="alert-banners">
      <a-alert
        v-for="alert in alerts"
        :key="alert.id"
        :type="getAlertType(alert.type)"
        :closable="alert.dismissible"
        @close="removeAlert(alert.id)"
        class="mb-2"
      >
        {{ alert.message }}
      </a-alert>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useUiStore } from '@/stores/ui'

const uiStore = useUiStore()

const alerts = computed(() => uiStore.alerts)

const removeAlert = (id: string) => {
  uiStore.removeAlert(id)
}

const getAlertType = (type: string) => {
  const types: Record<string, 'info' | 'success' | 'warning' | 'error'> = {
    info: 'info',
    success: 'success',
    warning: 'warning',
    error: 'error',
  }
  return types[type] || 'info'
}

defineOptions({
  name: 'AlertBanner',
})
</script>

<!-- Teleport 到 body 后 scoped 样式不会命中，这里使用全局样式。
     层级/穿透属性必须保持在这里，避免再被页面内堆叠上下文（如 .content）困住。 -->
<style>
.alert-banners {
  position: fixed;
  top: 80px;
  right: 24px;
  z-index: 2000;
  max-width: 400px;
  /* 空白区域不拦截鼠标，避免遮挡右上角按钮；告警卡片本身仍可点击关闭 */
  pointer-events: none;
}

.alert-banners .arco-alert {
  pointer-events: auto;
}

.alert-banners .mb-2 {
  margin-bottom: 8px;
}

@media (max-width: 576px) {
  .alert-banners {
    top: 68px;
    right: 12px;
    left: 12px;
    max-width: none;
  }
}
</style>
