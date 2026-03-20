<template>
  <div class="wiki-edit">
    <!-- Header -->
    <div class="wiki-edit-header">
      <div class="wiki-edit-header-left">
        <h1 class="wiki-edit-title">
          {{ isExisting ? '编辑 Wiki' : '创建 Wiki' }}
        </h1>
        <p v-if="game" class="wiki-edit-subtitle">
          {{ game.title }}
        </p>
      </div>

      <div class="wiki-edit-actions">
        <a-button
          class="app-secondary-cta"
          type="secondary"
          :disabled="isSaving"
          @click="router.back()"
        >
          取消
        </a-button>
        <a-button
          class="app-primary-cta"
          type="primary"
          :loading="isSaving"
          @click="handleSave"
        >
          <template #icon>
            <icon-save />
          </template>
          保存
        </a-button>
      </div>
    </div>

    <!-- Wiki Form -->
    <a-row :gutter="16" justify="center" class="wiki-edit-row">
      <a-col :xs="24" :sm="24" :md="24" :lg="15" :xl="14" :xxl="13">
        <a-card class="wiki-edit-card">
          <div class="wiki-edit-main">
            <wiki-editor v-model="wikiData.content" />

            <div class="wiki-edit-summary">
              <div class="wiki-edit-summary__label">修改说明</div>
              <a-input
                v-model="wikiData.change_summary"
                :max-length="120"
                allow-clear
                placeholder="例如：补充角色介绍、修正发售日期、重写剧情简介"
              />
            </div>
          </div>
        </a-card>
      </a-col>

      <a-col :xs="24" :sm="24" :md="24" :lg="7" :xl="6" :xxl="5">
        <a-card class="wiki-edit-history-card">
          <template #title>
            <div class="wiki-edit-side-title">历史记录</div>
          </template>

          <div v-if="isHistoryLoading" class="wiki-edit-history-empty">
            <a-spin :size="18" />
          </div>

          <div v-else-if="historyEntries.length === 0" class="wiki-edit-history-empty">
            还没有历史记录
          </div>

          <div v-else class="wiki-edit-history">
            <button
              v-for="entry in historyEntries"
              :key="entry.id"
              class="wiki-edit-history-item"
              :class="{ 'wiki-edit-history-item--active': selectedHistory?.id === entry.id }"
              type="button"
              @click="selectedHistory = entry"
            >
              <strong>{{ entry.change_summary || '未填写修改说明' }}</strong>
              <span class="wiki-edit-history-label">{{ formatDateTime(entry.created_at) }}</span>
            </button>
          </div>
        </a-card>

        <a-card v-if="selectedHistory" class="wiki-edit-history-card wiki-edit-history-preview-card">
          <template #title>
            <div class="wiki-edit-side-title">历史预览</div>
          </template>

          <div class="wiki-edit-history-preview-meta">
            <strong>{{ selectedHistory.change_summary || '未填写修改说明' }}</strong>
            <span>{{ formatDateTime(selectedHistory.created_at) }}</span>
          </div>

          <div class="wiki-edit-history-preview-actions">
            <a-button type="secondary" size="small" @click="previewHistoryContent = !previewHistoryContent">
              {{ previewHistoryContent ? '查看源码' : '预览渲染' }}
            </a-button>
            <a-button type="primary" size="small" @click="restoreHistory">
              恢复到编辑器
            </a-button>
          </div>

          <div v-if="previewHistoryContent" class="wiki-edit-history-preview-rendered">
            <markdown-renderer :content="selectedHistory.content" />
          </div>
          <pre v-else class="wiki-edit-history-preview-source">{{ selectedHistory.content }}</pre>
        </a-card>
      </a-col>
    </a-row>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useGamesStore } from '@/stores/games'
import { useUiStore } from '@/stores/ui'
import wikiService, { type WikiContent, type WikiHistoryEntry } from '@/services/wiki.service'
import {
  IconSave
} from '@arco-design/web-vue/es/icon'
import WikiEditor from '@/components/WikiEditor.vue'
import MarkdownRenderer from '@/components/MarkdownRenderer.vue'

const route = useRoute()
const router = useRouter()
const gamesStore = useGamesStore()
const uiStore = useUiStore()

const game = computed(() => gamesStore.currentGame)
const wiki = ref<WikiContent | null>(null)
const historyEntries = ref<WikiHistoryEntry[]>([])
const selectedHistory = ref<WikiHistoryEntry | null>(null)
const isHistoryLoading = ref(false)
const previewHistoryContent = ref(true)

const isSaving = ref(false)

const wikiData = ref({
  content: '',
  change_summary: '',
})

const isExisting = computed(() => Boolean(wiki.value?.content))

const handleSave = async () => {
  if (!game.value) return

  isSaving.value = true

  try {
    const wikiDataToSend = {
      content: wikiData.value.content,
      change_summary: wikiData.value.change_summary.trim() || undefined,
    }

    await wikiService.updateWikiPage(String(game.value.id), wikiDataToSend)
    uiStore.addAlert(isExisting.value ? 'Wiki 已更新' : 'Wiki 已创建', 'success')
    wikiData.value.change_summary = ''
    await loadHistory(String(game.value.id))

    router.push({ name: 'game-detail', params: { id: String(game.value.id) } })
  } catch (error: any) {
    const errorMessage = error?.response?.data?.error || error?.message || '保存 Wiki 失败'
    uiStore.addAlert(errorMessage, 'error')
    console.error('Failed to save wiki:', error)
  } finally {
    isSaving.value = false
  }
}

const loadHistory = async (gameId: string) => {
  isHistoryLoading.value = true
  try {
    historyEntries.value = await wikiService.getWikiHistory(gameId)
    selectedHistory.value = historyEntries.value[0] || null
  } catch {
    historyEntries.value = []
    selectedHistory.value = null
  } finally {
    isHistoryLoading.value = false
  }
}

const restoreHistory = () => {
  if (!selectedHistory.value) return
  wikiData.value.content = selectedHistory.value.content
  wikiData.value.change_summary = `恢复历史版本：${selectedHistory.value.change_summary || formatDateTime(selectedHistory.value.created_at)}`
  uiStore.addAlert('已将历史版本内容恢复到编辑器', 'success')
}

const formatDateTime = (value?: string) => {
  if (!value) return ''
  const date = new Date(value.replace(' ', 'T'))
  if (Number.isNaN(date.getTime())) return value
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
}

const loadWikiEditorData = async (gameId: string) => {
  try {
    await gamesStore.fetchGame(gameId)
    wiki.value = null
    wikiData.value = {
      content: '',
      change_summary: '',
    }
    historyEntries.value = []
    selectedHistory.value = null

    // Try to load existing wiki
    try {
      const wikiContent = await wikiService.getWikiPage(gameId)
      if (wikiContent && wikiContent.content) {
        wiki.value = wikiContent
        wikiData.value = {
          content: wikiContent.content,
          change_summary: '',
        }
      }
    } catch {
      // Wiki doesn't exist yet
    }

    await loadHistory(gameId)
  } catch {
    uiStore.addAlert('Failed to load game', 'error')
    router.push({ name: 'games' })
  }
}

watch(
  () => route.params.gameId,
  async (gameId) => {
    if (!gameId || typeof gameId !== 'string') return
    await loadWikiEditorData(gameId)
  },
  { immediate: true },
)
</script>

<style scoped>
.wiki-edit {
  animation: fadeIn 0.3s ease;
  min-height: calc(100vh - 88px);
  display: flex;
  flex-direction: column;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.wiki-edit-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 24px;
  gap: 16px;
}

.wiki-edit-header-left {
  flex: 1;
}

.wiki-edit-title {
  font-size: 24px;
  font-weight: 600;
  margin: 0 0 4px;
  color: var(--color-text-1);
}

.wiki-edit-subtitle {
  color: var(--color-text-3);
  margin: 0;
}

.wiki-edit-actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}

.wiki-edit-row {
  flex: 1;
  min-height: 0;
}

.wiki-edit-card {
  width: 100%;
  height: calc(100vh - 220px);
}

.wiki-edit-card :deep(.arco-card-body) {
  padding: 20px;
  height: 100%;
  box-sizing: border-box;
}

.wiki-edit-main {
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.wiki-edit-main :deep(.wiki-editor) {
  flex: 1;
}

.wiki-edit-summary {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.wiki-edit-summary__label,
.wiki-edit-side-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-1);
}

.wiki-edit-title-input {
  margin-bottom: 16px;
}

.wiki-edit-info-card,
.wiki-edit-preview-card,
.wiki-edit-history-card {
  margin-bottom: 16px;
}

.wiki-edit-history-card :deep(.arco-card-body) {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.wiki-edit-help {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.wiki-edit-help-item {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.wiki-edit-help-title {
  font-weight: 600;
  font-size: 13px;
  color: var(--color-text-1);
}

.wiki-edit-help-code {
  font-family: monospace;
  font-size: 12px;
  color: var(--color-text-3);
  background: var(--color-fill-2);
  padding: 2px 6px;
  border-radius: 3px;
}

.wiki-edit-preview-empty {
  text-align: center;
  color: var(--color-text-3);
  padding: 16px;
}

.wiki-edit-preview-icon {
  font-size: 32px;
  margin-bottom: 8px;
}

.wiki-edit-preview-empty p {
  margin: 0;
}

.wiki-edit-history {
  display: flex;
  flex-direction: column;
  gap: 12px;
  max-height: 320px;
  overflow-y: auto;
}

.wiki-edit-history-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 12px;
  text-align: left;
  padding: 12px;
  border: 1px solid var(--color-border-2);
  border-radius: 10px;
  background: var(--color-fill-1);
  color: var(--color-text-1);
  cursor: pointer;
  transition: border-color 0.2s ease, background 0.2s ease;
}

.wiki-edit-history-item:hover,
.wiki-edit-history-item--active {
  border-color: rgb(var(--primary-6));
  background: rgba(var(--primary-6), 0.08);
}

.wiki-edit-history-label {
  color: var(--color-text-3);
}

.wiki-edit-history-empty {
  min-height: 120px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text-3);
}

.wiki-edit-history-preview-card {
  max-height: calc(100vh - 420px);
}

.wiki-edit-history-preview-meta {
  display: flex;
  flex-direction: column;
  gap: 4px;
  color: var(--color-text-2);
  font-size: 12px;
}

.wiki-edit-history-preview-actions {
  display: flex;
  gap: 8px;
}

.wiki-edit-history-preview-rendered,
.wiki-edit-history-preview-source {
  overflow: auto;
  max-height: 320px;
  margin: 0;
  padding: 12px;
  border-radius: 10px;
  background: var(--color-fill-1);
}

.wiki-edit-history-preview-source {
  white-space: pre-wrap;
  word-break: break-word;
  color: var(--color-text-2);
  font-size: 12px;
  font-family: 'Fira Code', 'Consolas', monospace;
}

@media (max-width: 1200px) {
  .wiki-edit-card {
    height: auto;
    min-height: 520px;
  }

  .wiki-edit-history-preview-card {
    max-height: none;
  }
}
</style>
