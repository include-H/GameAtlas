<script setup lang="ts">
// 串流设置弹窗：分辨率 / 帧率 / 码率 / 编码 / 音频。
// 编码选项按 detectCodecSupport() 过滤不可用的硬件解码。
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { detectCodecSupport, type CodecSupport } from '../capabilities'
import {
  AUDIO_OPTIONS,
  CODEC_OPTIONS,
  RESOLUTION_OPTIONS,
  defaultBitrateMbps,
  defaultSettings,
  isCustomResolution,
  loadSettings,
  saveSettings,
  type StreamSettings,
} from '../client/stream-settings'
import type { AudioConfiguration, VideoCodec } from '../client/moonlight-client'

const props = defineProps<{
  visible: boolean
}>()

const emit = defineEmits<{
  'update:visible': [visible: boolean]
  saved: []
}>()

const settings = reactive<StreamSettings>(defaultSettings())

const codecSupport = ref<CodecSupport | null>(null)
const customResolution = ref(false)

const FPS_OPTIONS: number[] = [30, 60];

const resolutionOptions = RESOLUTION_OPTIONS.map((r) => ({
  label: r.label,
  value: `${r.width}x${r.height}`,
}));

const codecOptions = computed(() =>
  CODEC_OPTIONS.filter((c) => {
    if (!codecSupport.value) return true;
    return c.value === 'h264' ? codecSupport.value.h264
      : c.value === 'hevc' ? codecSupport.value.hevc
      : codecSupport.value.av1;
  }),
);

const currentResolution = computed(() => {
  if (isCustomResolution(settings.width, settings.height)) return 'custom';
  return `${settings.width}x${settings.height}`;
});

function syncFromStorage() {
  const s = loadSettings();
  settings.width = s.width;
  settings.height = s.height;
  settings.fps = s.fps;
  settings.bitrateMbps = s.bitrateMbps;
  settings.codec = s.codec;
  settings.audio = s.audio;
  settings.showStats = s.showStats;
  customResolution.value = isCustomResolution(s.width, s.height);
}

function onResolutionChange(value: string | number) {
  if (value === 'custom') return;
  const preset = RESOLUTION_OPTIONS.find((r) => `${r.width}x${r.height}` === String(value));
  if (!preset) return;
  settings.width = preset.width;
  settings.height = preset.height;
  // 切分辨率时自动建议码率（仅当用户没手动改过码率时？此处直接建议默认值，
  // 用户可再调）。
  settings.bitrateMbps = defaultBitrateMbps(preset.width, preset.height, settings.fps);
}

function onFpsChange() {
  settings.bitrateMbps = defaultBitrateMbps(settings.width, settings.height, settings.fps);
}

function useAutoBitrate() {
  settings.bitrateMbps = defaultBitrateMbps(settings.width, settings.height, settings.fps);
}

function save() {
  saveSettings({ ...settings });
  emit('saved');
  emit('update:visible', false);
}

watch(
  () => props.visible,
  (visible) => {
    if (visible) syncFromStorage();
  },
);

onMounted(async () => {
  codecSupport.value = await detectCodecSupport();
  // 若当前保存的编码不可用，回退到第一个可用项。
  const supported = codecOptions.value.map((c) => c.value as VideoCodec);
  if (codecSupport.value && !supported.includes(settings.codec)) {
    settings.codec = supported[0] ?? 'h264';
  }
});
</script>

<template>
  <a-modal
    :visible="visible"
    title="串流设置"
    :footer="false"
    width="520px"
    @cancel="emit('update:visible', false)"
  >
    <a-form layout="vertical">
      <a-form-item label="分辨率">
        <a-select
          :model-value="currentResolution"
          :options="[
            ...resolutionOptions,
            { label: '自定义', value: 'custom' },
          ]"
          @change="onResolutionChange"
        />
        <a-space v-if="currentResolution === 'custom'" style="margin-top: 8px">
          <a-input-number v-model="settings.width" :min="320" :max="7680" :step="16" />
          <span class="custom-res-x">×</span>
          <a-input-number v-model="settings.height" :min="240" :max="4320" :step="16" />
        </a-space>
      </a-form-item>

      <a-form-item label="帧率">
        <a-select v-model="settings.fps" :options="FPS_OPTIONS" @change="onFpsChange" />
      </a-form-item>

      <a-form-item label="码率">
        <a-space>
          <a-input-number v-model="settings.bitrateMbps" :min="1" :max="400" :step="1" />
          <span class="unit">Mbps</span>
          <a-button size="small" @click="useAutoBitrate">自动</a-button>
        </a-space>
      </a-form-item>

      <a-form-item label="视频编码">
        <a-select
          v-model="settings.codec"
          :options="codecOptions.map((c) => ({ label: c.label, value: c.value }))"
        />
        <div v-if="codecSupport && !codecSupport.hevc && !codecSupport.av1" class="codec-hint">
          当前设备仅硬件支持 H.264，HEVC / AV1 不可用。
        </div>
      </a-form-item>

      <a-form-item label="音频">
        <a-select
          v-model="settings.audio"
          :options="AUDIO_OPTIONS.map((a) => ({ label: a.label, value: a.value as AudioConfiguration }))"
        />
      </a-form-item>

      <a-form-item label="显示统计浮层">
        <a-switch v-model="settings.showStats" />
      </a-form-item>
    </a-form>

    <template #footer>
      <a-space>
        <a-button @click="emit('update:visible', false)">取消</a-button>
        <a-button type="primary" @click="save">保存</a-button>
      </a-space>
    </template>
  </a-modal>
</template>

<style scoped>
.unit {
  color: var(--stream-text-dim);
  font-size: 13px;
}

.custom-res-x {
  color: var(--stream-text-dim);
}

.codec-hint {
  font-size: 12px;
  color: var(--stream-text-dim);
  margin-top: 6px;
}
</style>
