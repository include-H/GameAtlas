<script setup lang="ts">
// 云串流根组件：路由式切换 主机列表 → 播放视图。
import { ref } from 'vue'
import type { Host } from './client/host-store'
import type { AppEntry } from './client/nvhttp'
import HostListView from './components/HostListView.vue'
import StreamPlayer from './components/StreamPlayer.vue'

interface StreamTarget {
  host: Host
  app: AppEntry
}

const target = ref<StreamTarget | null>(null)

function onLaunch(host: Host, app: AppEntry) {
  console.log('[debug-onLaunch]', host, app)
  target.value = { host, app }
}

function onExit() {
  target.value = null
}
</script>

<template>
  <StreamPlayer v-if="target" :host="target.host" :app="target.app" @exit="onExit" />
  <HostListView v-else @launch="onLaunch" />
</template>
