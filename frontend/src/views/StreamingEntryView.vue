<script setup lang="ts">
// 云串流入口页：展示串流地址（:47999）+ 新窗口打开 + 自签证书信任说明。
import { computed } from 'vue'

const streamPort = 47999

const streamUrl = computed(() => {
  const hostname = window.location.hostname || '127.0.0.1'
  return `https://${hostname}:${streamPort}/`
})

function openStream() {
  window.open(streamUrl.value, '_blank', 'noopener,noreferrer')
}
</script>

<template>
  <div class="streaming-entry">
    <div class="streaming-entry__card app-glass-surface">
      <div class="streaming-entry__header">
        <icon-thunderbolt :size="36" class="streaming-entry__icon" />
        <h2>云串流</h2>
        <p class="streaming-entry__desc">
          通过 Moonlight Web 客户端在浏览器中串流局域网内的游戏主机（Sunshine / GameStream）
        </p>
      </div>

      <a-descriptions :column="1" bordered size="small" class="streaming-entry__info">
        <a-descriptions-item label="串流地址">
          <code class="streaming-entry__url">{{ streamUrl }}</code>
        </a-descriptions-item>
        <a-descriptions-item label="串流端口">47999（HTTPS）</a-descriptions-item>
      </a-descriptions>

      <div class="streaming-entry__actions">
        <a-button type="primary" size="large" @click="openStream">
          <template #icon><icon-external-link /></template>
          新窗口打开
        </a-button>
      </div>

      <a-alert class="streaming-entry__tip" type="warning">
        <template #icon><icon-exclamation-circle /></template>
        首次访问需在浏览器中信任串流代理的自签证书：
        打开串流地址 → 点击「高级」→「继续前往」（Chrome/Edge）后刷新页面即可。
        证书由本地代理生成，仅用于局域网内加密传输。
      </a-alert>

      <p class="streaming-entry__note">
        串流页为独立文档（<code>https://&lt;NAS IP&gt;:47999/</code>），需要跨源隔离
        （COOP/COEP）以启用 WASM 多线程解码，因此不在主站内嵌。
      </p>
    </div>
  </div>
</template>

<style scoped>
.streaming-entry {
  display: flex;
  justify-content: center;
  padding: 48px 20px;
}

.streaming-entry__card {
  width: 100%;
  max-width: 620px;
  padding: 32px;
  border-radius: 16px;
}

.streaming-entry__header {
  text-align: center;
  margin-bottom: 24px;
}

.streaming-entry__icon {
  color: var(--app-accent, #4c8dff);
}

.streaming-entry__desc {
  color: var(--color-text-3);
  margin: 8px 0 0;
  line-height: 1.6;
}

.streaming-entry__url {
  font-family: 'SF Mono', Consolas, monospace;
  font-size: 13px;
}

.streaming-entry__actions {
  margin-top: 24px;
  text-align: center;
}

.streaming-entry__tip {
  margin-top: 20px;
}

.streaming-entry__note {
  margin-top: 16px;
  font-size: 12px;
  color: var(--color-text-3);
  line-height: 1.6;
  text-align: center;
}
</style>
