<script setup lang="ts">
// 播放视图：<video> + MediaStreamTrackGenerator 装配 + 输入捕获 +
// 延迟/帧率统计浮层 + 退出按钮（Ctrl+Alt+Shift+Q 快捷键在上游
// keyboard.ts 内保留，会触发 client.disconnect → 本组件退出）。
import { onMounted, onUnmounted, ref } from 'vue'
import type { Host } from '../client/host-store'
import type { AppEntry } from '../client/nvhttp'
import { loadSettingsFromServer } from '../client/stream-settings'
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
    // 优先后端设置（合并后返回），失败自动回退 localStorage。
    const settings = await loadSettingsFromServer()
    await start(root.value, props.host, props.app, settings, caps.value)
  }
})

onUnmounted(() => {
  void stop()
})

async function onExit() {
  await stop()
  emit('exit')
}

function toggleFullscreen() {
  if (document.fullscreenElement) {
    void document.exitFullscreen().catch(() => {})
  } else {
    void document.documentElement.requestFullscreen().catch(() => {})
  }
}
</script>

<template>
  <div ref="root" class="stream-player" tabindex="0">
    <div class="video-container">
      <div class="stream-overlay" :class="{ hidden: !isActive }">{{ status }}</div>
      <div v-if="isActive" class="stream-actions">
        <a-button
          class="stream-fullscreen-btn"
          type="secondary"
          @click="toggleFullscreen"
        >
          <template #icon><icon-fullscreen /></template>
          全屏
        </a-button>
        <a-button
          class="stream-exit-btn"
          type="primary"
          status="danger"
          @click="onExit"
        >
          <template #icon><icon-close /></template>
          退出串流
        </a-button>
      </div>
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
