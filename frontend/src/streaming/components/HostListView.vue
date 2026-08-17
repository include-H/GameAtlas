<script setup lang="ts">
// 主机列表视图：卡片列表 + 添加主机（输入 IP）+ 配对状态徽标 +
// 设置按钮 + 进入串流（应用选择）。
import { computed, onMounted, ref } from 'vue'
import { Message } from '@arco-design/web-vue'
import { loadHosts, removeHost, saveHosts, type Host } from '../client/host-store'
import { detectCapabilities, type Capabilities } from '../capabilities'
import type { AppEntry } from '../client/nvhttp'
import PairDialog from './PairDialog.vue'
import AppPickerDialog from './AppPickerDialog.vue'
import SettingsDialog from './SettingsDialog.vue'

const emit = defineEmits<{
  launch: [host: Host, app: AppEntry]
}>()

const hosts = ref<Host[]>([])
const newAddress = ref('')
const adding = ref(false)
const caps = ref<Capabilities | null>(null)

const pairTarget = ref<Host | null>(null)
const pickerTarget = ref<Host | null>(null)
const settingsVisible = ref(false)

const capabilityWarnings = computed(() => {
  const warnings: string[] = [];
  if (!caps.value) return warnings;
  if (!caps.value.crossOriginIsolated || !caps.value.sharedArrayBuffer) {
    warnings.push('页面缺少跨源隔离（COOP/COEP），WASM 多线程将不可用——请通过 Go 串流代理（https://<NAS>:47999）打开本页。');
  }
  if (!caps.value.webCodecs) {
    warnings.push('浏览器不支持 WebCodecs，无法解码视频流。');
  }
  if (!caps.value.gamepad) {
    warnings.push('浏览器不支持 Gamepad API，手柄输入不可用。');
  }
  return warnings;
});

function formatLastSeen(ts: number): string {
  if (!ts) return '从未连接';
  const diff = Date.now() - ts;
  if (diff < 60_000) return '刚刚';
  if (diff < 3_600_000) return `${Math.floor(diff / 60_000)} 分钟前`;
  if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)} 小时前`;
  return new Date(ts).toLocaleDateString();
}

function addHost() {
  const address = newAddress.value.trim();
  if (!address || adding.value) return;
  if (hosts.value.some((h) => h.address === address)) {
    newAddress.value = '';
    return;
  }
  adding.value = true;
  // id 留空由后端生成；paired 由后端按证书文件判断。
  void saveHosts([{ id: '', name: address, address, paired: false, lastSeen: 0 }])
    .then((fresh) => {
      hosts.value = fresh;
      newAddress.value = '';
    })
    .catch((err: unknown) => {
      Message.error(`添加主机失败：${(err as Error).message ?? String(err)}`);
    })
    .finally(() => {
      adding.value = false;
    });
}

function onPaired(host: Host) {
  // 配对成功后把主机落库并刷新（paired 由后端重新计算）。
  void saveHosts([host])
    .then((fresh) => {
      hosts.value = fresh;
    })
    .catch((err: unknown) => {
      Message.error(`保存配对状态失败：${(err as Error).message ?? String(err)}`);
    })
    .finally(() => {
      pairTarget.value = null;
    });
}

function onRemove(host: Host) {
  void removeHost(host.id)
    .then((fresh) => {
      hosts.value = fresh;
    })
    .catch((err: unknown) => {
      Message.error(`移除主机失败：${(err as Error).message ?? String(err)}`);
    });
}

function onPickApp(app: AppEntry) {
  console.log('[debug-onPickApp]', pickerTarget.value, app)
  const host = pickerTarget.value;
  if (!host) return;
  pickerTarget.value = null;
  emit('launch', host, app);
}

onMounted(async () => {
  caps.value = await detectCapabilities();
  try {
    hosts.value = await loadHosts();
  } catch (err) {
    Message.error(`加载主机列表失败：${(err as Error).message ?? String(err)}`);
  }
});
</script>

<template>
  <div class="streaming-root">
    <header class="streaming-header">
      <icon-thunderbolt :size="26" style="color: var(--stream-accent)" />
      <div>
        <div class="title">云串流</div>
        <div class="subtitle">局域网游戏串流 · Moonlight Web 客户端</div>
      </div>
      <div style="flex: 1" />
      <a-button @click="settingsVisible = true">
        <template #icon><icon-settings /></template>
        串流设置
      </a-button>
    </header>

    <div class="streaming-body">
      <div
        v-for="warning in capabilityWarnings"
        :key="warning"
        class="capability-warning"
      >
        <icon-exclamation-circle />
        {{ warning }}
      </div>

      <div v-if="hosts.length" class="section-title">主机</div>
      <div v-if="hosts.length" class="host-grid">
        <div v-for="host in hosts" :key="host.id" class="host-card glass-surface">
          <div class="host-name">
            <span class="host-name-text">{{ host.name }}</span>
            <a-tag
              :color="host.paired ? 'green' : 'orangered'"
              size="small"
            >
              {{ host.paired ? '已配对' : '未配对' }}
            </a-tag>
          </div>
          <div class="host-address">{{ host.address }}</div>
          <div class="host-meta">
            <span>上次连接 {{ formatLastSeen(host.lastSeen) }}</span>
          </div>
          <div class="host-actions">
            <a-button
              type="primary"
              :disabled="!host.paired"
              @click="pickerTarget = host"
            >
              <template #icon><icon-play-arrow /></template>
              进入
            </a-button>
            <a-button :disabled="host.paired" @click="pairTarget = host">
              {{ host.paired ? '重新配对' : '配对' }}
            </a-button>
            <div class="spacer" />
            <a-button type="text" class="app-text-action-btn" @click="onRemove(host)">
              <template #icon><icon-delete /></template>
              移除
            </a-button>
          </div>
        </div>
      </div>

      <div class="add-host-row glass-surface">
        <div style="flex: 1">
          <a-input
            v-model="newAddress"
            placeholder="输入主机 IP，如 192.168.1.100"
            @press-enter="addHost"
          >
            <template #prefix><icon-link /></template>
          </a-input>
          <div class="hint">主机需运行 Sunshine 或 NVIDIA GameStream 服务</div>
        </div>
        <a-button type="outline" :loading="adding" @click="addHost">添加主机</a-button>
      </div>

      <div v-if="!hosts.length" class="empty-hint">
        还没有主机——输入 IP 添加你的第一台串流主机
      </div>

      <div class="footer-note">
        首次使用请先信任 Go 串流代理的自签证书（通过 https://&lt;NAS IP&gt;:47999 打开）<br />
        Ctrl+Alt+Shift+Q 可随时退出串流
      </div>
    </div>

    <PairDialog
      v-if="pairTarget"
      :visible="true"
      :host="pairTarget"
      @update:visible="(v: boolean) => { if (!v) pairTarget = null }"
      @paired="onPaired"
    />
    <AppPickerDialog
      v-if="pickerTarget"
      :visible="true"
      :host="pickerTarget"
      @update:visible="(v: boolean) => { if (!v) pickerTarget = null }"
      @select="onPickApp"
    />
    <SettingsDialog v-model:visible="settingsVisible" />
  </div>
</template>
