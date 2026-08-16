<script setup lang="ts">
// 配对弹窗：生成 4 位 PIN（可编辑）→ POST /api/pair → 展示结果。
// 配对由 Go 代理执行五步 NvHTTP 握手，浏览器只负责发起与等待。
import { reactive, watch } from 'vue'
import type { Host } from '../client/host-store'
import { pairHost } from '../client/pairing'

const props = defineProps<{
  visible: boolean
  host: Host
}>()

const emit = defineEmits<{
  'update:visible': [visible: boolean]
  paired: [host: Host]
}>()

interface PairState {
  pin: string
  pairing: boolean
  error: string | null
  done: boolean
}

const state = reactive<PairState>({
  pin: '',
  pairing: false,
  error: null,
  done: false,
})

let abort: AbortController | null = null

function newPin(): string {
  return String(Math.floor(1000 + Math.random() * 9000))
}

watch(
  () => props.visible,
  (visible) => {
    if (visible) {
      state.pin = newPin()
      state.pairing = false
      state.error = null
      state.done = false
      abort = null
    }
  },
)

function close() {
  abort?.abort()
  emit('update:visible', false)
}

async function doPair() {
  const pin = state.pin.trim()
  if (!/^\d{4,6}$/.test(pin) || state.pairing) return
  state.pairing = true
  state.error = null
  abort = new AbortController()
  try {
    const updated = await pairHost(props.host, pin, { signal: abort.signal })
    state.done = true
    emit('paired', updated)
  } catch (err) {
    if (abort?.signal.aborted) return
    state.error = (err as Error).message ?? String(err)
  } finally {
    state.pairing = false
  }
}
</script>

<template>
  <a-modal
    :visible="visible"
    :title="`与 ${host.address} 配对`"
    :footer="false"
    width="420px"
    @cancel="close"
  >
    <template v-if="!state.done">
      <p class="pin-hint">
        在主机（Sunshine / GFE）上输入以下 4 位 PIN 完成配对：
      </p>
      <a-input
        v-model="state.pin"
        class="pin-input"
        maxlength="6"
        placeholder="4 位 PIN"
        :disabled="state.pairing"
        @press-enter="doPair"
      />
      <div class="pair-actions">
        <a-button :disabled="state.pairing" @click="state.pin = newPin()">重新生成</a-button>
        <a-button
          type="primary"
          :loading="state.pairing"
          :disabled="!state.pin"
          @click="doPair"
        >
          开始配对
        </a-button>
      </div>
      <a-alert v-if="state.error" type="error" :message="state.error" class="pair-error" />
    </template>
    <template v-else>
      <a-result
        status="success"
        title="配对成功"
        :subtitle="`${host.address} 已与当前设备配对，可以开始串流了`"
      >
        <template #extra>
          <a-button type="primary" @click="close">完成</a-button>
        </template>
      </a-result>
    </template>
  </a-modal>
</template>

<style scoped>
.pin-input {
  text-align: center;
  font-size: 32px;
  font-weight: 700;
  letter-spacing: 8px;
  font-family: 'SF Mono', Consolas, monospace;
  height: 56px;
}

.pair-actions {
  display: flex;
  justify-content: center;
  gap: 12px;
  margin-top: 16px;
}

.pair-error {
  margin-top: 14px;
}
</style>
