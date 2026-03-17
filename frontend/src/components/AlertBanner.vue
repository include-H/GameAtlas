<template>
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

<style scoped>
.alert-banners {
  position: fixed;
  top: 80px;
  right: 24px;
  z-index: 1000;
  max-width: 400px;
}

.mb-2 {
  margin-bottom: 8px;
}
</style>
