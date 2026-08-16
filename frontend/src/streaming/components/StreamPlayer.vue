<script setup lang="ts">
// 播放视图：<video> + MediaStreamTrackGenerator 装配 + 输入捕获 +
// 延迟/帧率统计浮层 + 退出按钮（Ctrl+Alt+Shift+Q 快捷键在上游
// keyboard.ts 内保留，会触发 client.disconnect → 本组件退出）。
import { onMounted, onUnmounted, ref } from 'vue'
import type { Host } from '../client/host-store'
import type { AppEntry } from '../client/nvhttp'
import { loadSettings } from '../client/stream-settings'
import { detectCapabilities, type Capabilities } from '../capabilities'
import { useStreamSession } from '../composables/useStreamSession'

const props = defineProps<{
  host: Host
  app: AppEntry
}>()

const emit = defineEmits<{
  exit: []
}>()

const root = ref<HTMLElement | null>(null)
const caps = ref<Capabilities | null>(null)

const {
  phase,
  status,
  error,
  isActive,
  start,
  stop,
} = useStreamSession()

onMounted(async () => {
  caps.value = await detectCapabilities()
  if (root.value) {
    await start(root.value, props.host, props.app, loadSettings(), caps.value)
  }
})

onUnmounted(() => {
  void stop()
})

async function onExit() {
  await stop()
  emit('exit')
}
</script>

<template>
  <div ref="root" class="stream-player" tabindex="0">
    <div class="video-container">
      <div class="stream-overlay" :class="{ hidden: !isActive }">{{ status }}</div>
      <a-button
        v-if="isActive"
        class="stream-exit-btn"
        type="primary"
        status="danger"
        @click="onExit"
      >
        <template #icon><icon-close /></template>
        退出串流
      </a-button>
      <div v-else class="stream-exit-hint">正在连接 {{ host.address }} · {{ app.title }}…</div>
    </div>

    <a-modal
      :visible="phase === 'failed'"
      title="串流启动失败"
      @cancel="emit('exit')"
    >
      <p>{{ error }}</p>
      <template #footer>
        <a-button type="primary" @click="emit('exit')">返回主机列表</a-button>
      </template>
    </a-modal>
  </div>
</template>
