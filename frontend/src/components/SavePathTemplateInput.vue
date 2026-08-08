<template>
  <div class="save-path-template-input">
    <div class="save-path-template-input__row">
      <a-select
        class="save-path-template-input__select"
        :model-value="baseKey"
        placeholder="选择基础目录"
        @update:model-value="handleBaseChange"
      >
        <a-option
          v-for="option in SAVE_PATH_BASE_OPTIONS"
          :key="option.key"
          :value="option.key"
          :label="option.label"
        >
          {{ option.label }}
        </a-option>
      </a-select>
      <a-input
        :model-value="subpath"
        :placeholder="inputPlaceholder"
        allow-clear
        @update:model-value="handleInput"
      />
    </div>
    <div class="save-path-template-input__hint">
      <template v-if="finalTemplate">
        最终模板：<code class="save-path-template-input__code">{{ finalTemplate }}</code>
      </template>
      <template v-else>
        留空则不提供「打开存档目录」
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import {
  SAVE_PATH_BASE_OPTIONS,
  joinSavePathTemplate,
  normalizeWindowsPath,
  splitSavePathTemplate,
  type SavePathTemplateMode,
} from '@/utils/save-path-template'

interface Props {
  modelValue: string
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const mode = ref<SavePathTemplateMode>('base')
const baseKey = ref('userprofile')
const subpath = ref('')

// 外部值变化时重新解析；仅当解析结果与当前内部状态不一致才写入，避免回写循环。
watch(
  () => props.modelValue,
  (value) => {
    const parts = splitSavePathTemplate(value ?? '')
    if (parts.mode !== mode.value || parts.baseKey !== baseKey.value || parts.subpath !== subpath.value) {
      mode.value = parts.mode
      baseKey.value = parts.baseKey
      subpath.value = parts.subpath
    }
  },
  { immediate: true },
)

const inputPlaceholder = computed(() =>
  mode.value === 'custom' ? '如 C:\\XboxGames' : "如 Sid Meier's Civilization 5",
)

const finalTemplate = computed(() => buildTemplate())

const buildTemplate = (): string =>
  joinSavePathTemplate({ mode: mode.value, baseKey: baseKey.value, subpath: subpath.value })

const handleBaseChange = (key: unknown) => {
  if (typeof key !== 'string') return
  if (key === 'custom') {
    // base → custom：输入框内容变为当前拼接出的完整模板
    subpath.value = buildTemplate()
    mode.value = 'custom'
  } else {
    if (mode.value === 'custom') {
      // custom → base：重新解析完整模板，去掉基础变量前缀得到子路径
      subpath.value = splitSavePathTemplate(subpath.value).subpath
    }
    mode.value = 'base'
    baseKey.value = key
  }
  emit('update:modelValue', buildTemplate())
}

const handleInput = (value: string) => {
  const normalized = normalizeWindowsPath(value)
  subpath.value = normalized
  emit('update:modelValue', buildTemplate())
}
</script>

<style scoped>
.save-path-template-input__row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.save-path-template-input__select {
  width: 220px;
  flex-shrink: 0;
}

.save-path-template-input__hint {
  margin-top: 4px;
  font-size: 12px;
  line-height: 1.5;
  color: var(--color-text-3);
}

.save-path-template-input__code {
  word-break: break-all;
}
</style>
