<script setup lang="ts">
// 应用选择弹窗：GET /api/applist → 列表 → 点应用进入串流。
import { onMounted, ref, watch } from 'vue'
import type { Host } from '../client/host-store'
import { fetchAppList, type AppEntry } from '../client/nvhttp'

const props = defineProps<{
  visible: boolean
  host: Host
}>()

const emit = defineEmits<{
  'update:visible': [visible: boolean]
  select: [app: AppEntry]
}>()

const apps = ref<AppEntry[]>([])
const loading = ref(false)
const error = ref<string | null>(null)
const loaded = ref(false)

async function load() {
  loading.value = true
  error.value = null
  try {
    apps.value = await fetchAppList(props.host)
    loaded.value = true
  } catch (err) {
    error.value = (err as Error).message ?? String(err)
  } finally {
    loading.value = false
  }
}

watch(
  () => props.visible,
  (visible) => {
    if (visible && !loaded.value) void load()
  },
)

onMounted(() => {
  if (props.visible && !loaded.value) void load()
})

function close() {
  emit('update:visible', false)
}

function pick(app: AppEntry) {
  // 必须先 emit select 再 close：close 会同步触发父组件把 pickerTarget
  // 置 null（update:visible=false），onPickApp 依赖它取主机。
  console.log('[debug-pick]', app)
  emit('select', app)
  close()
}
</script>

<template>
  <a-modal
    :visible="visible"
    :title="`选择应用 · ${host.address}`"
    :footer="false"
    width="480px"
    @cancel="close"
  >
    <div v-if="loading" class="empty-hint">正在获取应用列表…</div>
    <a-alert v-else-if="error" type="error" :message="error" @close="close" />
    <div v-else class="app-picker-list">
      <div
        v-for="app in apps"
        :key="app.id"
        class="app-picker-item glass-surface glass-surface--interactive"
        @click="pick(app)"
      >
        <span class="app-title">{{ app.title }}</span>
        <a-tag v-if="app.hdrSupported" color="gold" size="small">HDR</a-tag>
      </div>
      <div v-if="apps.length === 0" class="empty-hint">主机上没有可串流的应用</div>
    </div>
  </a-modal>
</template>
