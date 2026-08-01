<template>
  <a-modal
    v-model:visible="visible"
    title="选择游戏文件"
    :width="900"
    :footer="false"
    :body-style="{ padding: '0' }"
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
      <div class="file-browser-header app-glass-surface">
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

      <!-- File List - Table View -->
      <div class="file-table-container">
        <table class="file-table" v-if="!hasLoadFailure">
          <thead>
            <tr>
              <th class="col-name">名称</th>
              <th class="col-size">大小</th>
              <th class="col-type">类型</th>
              <th class="col-action">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="item in directoryItems"
              :key="item.path"
              :class="['file-row', item.type]"
              @click="handleItemClick(item)"
            >
              <td class="col-name">
                <div class="file-name-cell">
                  <icon-folder v-if="item.type === 'directory'" class="file-icon folder" />
                  <icon-file v-else class="file-icon file" />
                  <span class="file-name">{{ item.name }}</span>
                </div>
              </td>
              <td class="col-size">
                <span v-if="item.type === 'file'" class="file-size">{{ formatSize(item.size ?? undefined) }}</span>
                <span v-else class="file-size dim">—</span>
              </td>
              <td class="col-type">
                <span v-if="item.type === 'directory'" class="file-type">文件夹</span>
                <span v-else class="file-type">{{ getFileType(item.name) }}</span>
              </td>
              <td class="col-action">
                <a-button 
                  v-if="item.type === 'file'" 
                  class="app-text-action-btn app-secondary-compact"
                  type="text" 
                  size="mini"
                  @click.stop="selectFile(item)"
                >
                  选择
                </a-button>
                <span v-else class="action-hint">打开</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </a-modal>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { IconFolder, IconFile, IconArrowUp } from '@arco-design/web-vue/es/icon'
import { directoryService, type DirectoryItem } from '@/services/directory.service'
import { useUiStore } from '@/stores/ui'

const uiStore = useUiStore()

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
    directoryListIncomplete.value = data.incomplete
    skippedCount.value = data.skippedCount
  } catch {
    if (currentPath.value || directoryItems.value.length > 0) {
      loadFailedWithStaleData.value = true
      uiStore.addAlert('目录刷新失败，当前显示的是上次成功加载的内容', 'warning')
      return
    }
    currentPath.value = ''
    parentPath.value = null
    directoryItems.value = []
    directoryListIncomplete.value = false
    skippedCount.value = 0
    hasLoadFailure.value = true
    uiStore.addAlert('目录加载失败，请稍后重试', 'error')
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

// Get file type from extension
const getFileType = (filename: string) => {
  const ext = filename.split('.').pop()?.toUpperCase()
  if (!ext) return '文件'
  return `${ext} 文件`
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
  display: flex;
  flex-direction: column;
  min-height: 500px;
}

.file-browser-status {
  margin: 12px;
  padding: 10px 12px;
  border-radius: 8px;
  font-size: 13px;
  line-height: 1.5;
}

.file-browser-status--warning {
  color: var(--color-status-warning-text);
  background: var(--color-status-warning-bg);
  border: 1px solid var(--color-status-warning-border);
}

.file-browser-status--error {
  color: var(--color-status-error-text);
  background: var(--color-status-error-bg);
  border: 1px solid var(--color-status-error-border);
}

.file-browser-header {
  padding: 12px 16px;
  border-bottom: 1px solid var(--color-border-2);
}

.current-path {
  font-size: 13px;
  color: var(--color-text-2);
  word-break: break-all;
  font-family: monospace;
}

.file-table-container {
  flex: 1;
  overflow-y: auto;
  max-height: 500px;
}

.file-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}

.file-table thead {
  position: sticky;
  top: 0;
  z-index: 1;
  background: var(--color-bg-2);
}

.file-table th {
  padding: 8px 12px;
  text-align: left;
  font-weight: 500;
  color: var(--color-text-3);
  border-bottom: 1px solid var(--color-border-2);
  font-size: 12px;
  white-space: nowrap;
}

.file-table td {
  padding: 6px 12px;
  border-bottom: 1px solid var(--color-border-1);
}

.file-row {
  cursor: pointer;
  transition: background-color 0.15s;
}

.file-row:hover {
  background-color: color-mix(in srgb, var(--color-primary-6) 8%, transparent);
}

.file-row.directory {
  font-weight: 500;
}

.col-name {
  min-width: 300px;
}

.col-size {
  width: 100px;
  text-align: right;
}

.col-type {
  width: 120px;
}

.col-action {
  width: 80px;
  text-align: center;
}

.file-name-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.file-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-size {
  color: var(--color-text-2);
  font-family: monospace;
  font-size: 12px;
}

.file-size.dim {
  color: var(--color-text-4);
}

.file-type {
  color: var(--color-text-3);
  font-size: 12px;
}

.file-icon {
  font-size: 18px;
  flex-shrink: 0;
}

.file-icon.folder {
  color: rgb(var(--warning-6));
}

.file-icon.file {
  color: var(--color-text-4);
}

.action-hint {
  color: var(--color-text-4);
  font-size: 12px;
}
</style>
