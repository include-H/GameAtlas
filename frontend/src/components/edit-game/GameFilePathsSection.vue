<template>
  <a-form-item label="游戏文件路径">
    <div class="file-paths-container">
      <div v-for="(item, index) in filePaths" :key="index" class="file-path-item">
        <div class="file-path-row">
          <a-input
            :model-value="item.path"
            placeholder="游戏文件路径"
            class="file-path-input"
            @update:model-value="handlePathUpdate(index, $event)"
          >
            <template #prepend>
              <span class="path-index">{{ index + 1 }}</span>
            </template>
          </a-input>
          <a-input
            :model-value="item.label"
            placeholder="版本备注"
            class="file-label-input"
            @update:model-value="handleLabelUpdate(index, $event)"
          />
          <a-button type="text" html-type="button" @click="emit('browse', index)">
            <template #icon>
              <icon-folder />
            </template>
            浏览
          </a-button>
          <a-button
            type="text"
            status="danger"
            html-type="button"
            @click="emit('remove', index)"
          >
            <icon-minus />
          </a-button>
        </div>
      </div>

      <a-button
        type="text"
        long
        html-type="button"
        :style="{ marginTop: '8px' }"
        @click="emit('add')"
      >
        <template #icon>
          <icon-plus />
        </template>
        添加文件路径
      </a-button>
    </div>
  </a-form-item>
</template>

<script setup lang="ts">
import { IconFolder, IconMinus, IconPlus } from '@arco-design/web-vue/es/icon'
import type { FilePathItem } from '@/composables/useGameFilePaths'

defineProps<{
  filePaths: FilePathItem[]
}>()

const emit = defineEmits<{
  add: []
  remove: [index: number]
  browse: [index: number]
  'update-item': [payload: { index: number; field: 'path' | 'label'; value: string }]
}>()

const updateItem = (index: number, field: 'path' | 'label', value: string | number | undefined) => {
  emit('update-item', {
    index,
    field,
    value: String(value ?? ''),
  })
}

const handlePathUpdate = (index: number, value: string | number | undefined) => {
  updateItem(index, 'path', typeof value === 'string' || typeof value === 'number' ? value : '')
}

const handleLabelUpdate = (index: number, value: string | number | undefined) => {
  updateItem(index, 'label', typeof value === 'string' || typeof value === 'number' ? value : '')
}
</script>

<style scoped>
.file-paths-container {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
}

.file-path-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.file-path-row {
  display: flex;
  gap: 8px;
  align-items: center;
  width: 100%;
}

.file-path-input {
  flex: 5;
  min-width: 0;
}

.file-label-input {
  flex: 4;
  min-width: 0;
}

.file-path-item :deep(.arco-input-prepend) {
  background: var(--color-fill-2);
  border-right: 1px solid var(--color-border-2);
  padding: 0 8px;
}

.path-index {
  font-size: 12px;
  color: var(--color-text-3);
  font-weight: 600;
}
</style>
