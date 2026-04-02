<template>
  <a-modal
    v-model:visible="visible"
    title="选择游戏文件"
    :width="600"
    :footer="false"
    @cancel="handleCancel"
  >
    <div class="file-browser">
      <!-- Path Navigation -->
      <div class="file-browser-header">
        <a-space>
          <a-button 
            class="app-text-action-btn app-secondary-compact"
            type="text"
            size="small" 
            :disabled="!canGoUp" 
            @click="goToParent"
          >
            <template #icon>
              <icon-arrow-up />
            </template>
            上级
          </a-button>
          <span class="current-path">{{ currentPath }}</span>
        </a-space>
      </div>

      <!-- File List -->
      <a-list
        class="file-list"
        :bordered="false"
        :data="directoryItems"
      >
        <template #item="{ item }">
          <a-list-item
            :class="['file-item', item.type]"
            @click="handleItemClick(item)"
          >
            <a-list-item-meta>
              <template #avatar>
                <icon-folder v-if="item.type === 'directory'" class="file-icon folder" />
                <icon-file v-else class="file-icon file" />
              </template>
              <template #title>
                <span class="file-name">{{ item.name }}</span>
              </template>
              <template #description>
                <span v-if="item.type === 'file'" class="file-size">
                  {{ formatSize(item.size) }}
                </span>
              </template>
            </a-list-item-meta>
            <template #actions>
              <a-button 
                v-if="item.type === 'file'" 
                class="app-text-action-btn app-secondary-compact"
                type="text" 
                size="small"
                @click.stop="selectFile(item)"
              >
                选择
              </a-button>
            </template>
          </a-list-item>
        </template>
      </a-list>
    </div>
  </a-modal>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { IconFolder, IconFile, IconArrowUp } from '@arco-design/web-vue/es/icon'
import { directoryService, type DirectoryItem } from '@/services/directory.service'

interface Props {
  visible: boolean
  initialPath?: string
}

const props = withDefaults(defineProps<Props>(), {
  initialPath: ''
})

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'select': [path: string]
}>()

const visible = computed({
  get: () => props.visible,
  set: (value) => emit('update:visible', value)
})

const currentPath = ref('')
const parentPath = ref<string | null>(null)
const directoryItems = ref<DirectoryItem[]>([])

const canGoUp = computed(() => parentPath.value !== null)

// Load directory content
const loadDirectory = async (path?: string) => {
  try {
    const data = await directoryService.listDirectory(path)
    currentPath.value = data.currentPath
    parentPath.value = data.parentPath
    directoryItems.value = data.items
  } catch (error) {
    console.error('Failed to load directory:', error)
  }
}

// Handle item click
const handleItemClick = async (item: DirectoryItem) => {
  if (item.type === 'directory') {
    await loadDirectory(item.path)
  } else {
    selectFile(item)
  }
}

// Select file
const selectFile = (item: DirectoryItem) => {
  emit('select', item.path)
  visible.value = false
}

// Go to parent directory
const goToParent = async () => {
  if (parentPath.value) {
    await loadDirectory(parentPath.value)
  }
}

// Cancel
const handleCancel = () => {
  visible.value = false
}

// Format file size
const formatSize = (bytes?: number) => {
  if (!bytes) return ''
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let size = bytes
  let unitIndex = 0
  
  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024
    unitIndex++
  }
  
  return `${size.toFixed(1)} ${units[unitIndex]}`
}

// Initialize when modal opens
watch(visible, async (newVal) => {
  if (newVal) {
    await loadDirectory(props.initialPath || undefined)
  }
}, { immediate: true })
</script>

<style scoped>
.file-browser {
  max-height: 500px;
  display: flex;
  flex-direction: column;
}

.file-browser-header {
  padding: 12px;
  border: 1px solid var(--app-card-border);
  background: var(--app-card-surface);
  border-radius: 8px;
  backdrop-filter: blur(var(--app-card-backdrop-blur));
  -webkit-backdrop-filter: blur(var(--app-card-backdrop-blur));
  margin-bottom: 12px;
}

.current-path {
  font-size: 13px;
  color: var(--color-text-2);
  word-break: break-all;
  font-family: monospace;
}

.file-list {
  max-height: 400px;
  overflow-y: auto;
}

.file-item {
  cursor: pointer;
  transition: background-color 0.2s, border-color 0.2s;
  border-radius: 8px;
  border: 1px solid transparent;
  margin-bottom: 4px;
}

.file-item:hover {
  background-color: color-mix(in srgb, var(--app-card-surface) 82%, transparent);
  border-color: var(--app-card-border);
}

.file-item.directory {
  background-color: color-mix(in srgb, var(--app-card-surface) 74%, transparent);
  border-color: var(--app-card-border);
}

.file-name {
  font-size: 14px;
}

.file-size {
  font-size: 12px;
  color: var(--color-text-3);
}

.file-icon {
  font-size: 20px;
}

.file-icon.folder {
  color: rgb(var(--warning-6));
}

.file-icon.file {
  color: var(--color-text-3);
}
</style>
