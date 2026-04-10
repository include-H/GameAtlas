<template>
  <a-modal
    v-model:visible="visible"
    title="选择游戏文件"
    :width="600"
    :footer="false"
    @cancel="handleCancel"
  >
    <div class="file-browser">
      <div v-if="loadFailedWithStaleData" class="file-browser-status file-browser-status--warning">
        目录刷新失败，当前显示的是上次成功加载的内容。
      </div>
      <div v-else-if="hasLoadFailure" class="file-browser-status file-browser-status--error">
        目录加载失败，请稍后重试。
      </div>
      <div v-else-if="directoryListIncomplete" class="file-browser-status file-browser-status--warning">
        本次目录列表不完整，已跳过 {{ skippedCount }} 个无法读取的条目。
      </div>

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
        v-if="!hasLoadFailure"
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
const hasLoadFailure = ref(false)
const loadFailedWithStaleData = ref(false)
const directoryListIncomplete = ref(false)
const skippedCount = ref(0)

const canGoUp = computed(() => parentPath.value !== null)

// Load directory content
const loadDirectory = async (path?: string) => {
  hasLoadFailure.value = false
  loadFailedWithStaleData.value = false
  try {
    const data = await directoryService.listDirectory(path)
    currentPath.value = data.currentPath
    parentPath.value = data.parentPath
    directoryItems.value = data.items
    // 2026-04-09: keep directory browsing best-effort for transient filesystem entry failures,
    // but surface that this response is partial so the picker does not masquerade as a complete listing.
    directoryListIncomplete.value = data.incomplete
    skippedCount.value = data.skippedCount
  } catch (error) {
    // 2026-04-08: directory read failures must not masquerade as a successful empty/current folder state.
    // Impact: initial load shows an explicit failure, while refresh failures keep stale content with a warning.
    if (currentPath.value || directoryItems.value.length > 0) {
      loadFailedWithStaleData.value = true
      return
    }
    currentPath.value = ''
    parentPath.value = null
    directoryItems.value = []
    directoryListIncomplete.value = false
    skippedCount.value = 0
    hasLoadFailure.value = true
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

.file-browser-status {
  margin-bottom: 12px;
  padding: 10px 12px;
  border-radius: 8px;
  font-size: 13px;
  line-height: 1.5;
}

.file-browser-status--warning {
  color: #9a6700;
  background: rgba(255, 196, 92, 0.16);
  border: 1px solid rgba(255, 196, 92, 0.28);
}

.file-browser-status--error {
  color: #b42318;
  background: rgba(217, 45, 32, 0.12);
  border: 1px solid rgba(217, 45, 32, 0.22);
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
