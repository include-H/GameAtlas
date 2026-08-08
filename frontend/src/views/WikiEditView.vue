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
          class="app-text-action-btn"
          type="text"
          :loading="isHistoryLoading"
          :disabled="historyEntries.length === 0"
          @click="openHistoryDialog"
        >
          历史记录
        </a-button>
        <a-button
          class="app-text-action-btn"
          type="text"
          :disabled="isSaving"
          @click="handleCancel"
        >
          取消
        </a-button>
        <a-button
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
      <a-col :xs="24" :sm="24" :md="24" :lg="20" :xl="18" :xxl="16">
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
      </a-col>
    </a-row>

    <a-modal
      :visible="historyPreviewVisible"
      :footer="false"
      :mask-closable="true"
      modal-class="wiki-edit-history-modal"
      @cancel="historyPreviewVisible = false"
    >
      <template #title>
        <div class="wiki-edit-side-title">历史记录</div>
      </template>

      <section v-if="isHistoryLoading" class="wiki-edit-history-empty wiki-edit-history-empty--dialog">
        <a-spin :size="20" />
      </section>

      <section v-else-if="hasHistoryLoadFailure" class="wiki-edit-history-empty wiki-edit-history-empty--dialog">
        历史记录加载失败，请稍后重试
      </section>

      <section v-else-if="historyEntries.length === 0" class="wiki-edit-history-empty wiki-edit-history-empty--dialog">
        还没有历史记录
      </section>

      <template v-else-if="selectedHistory">
        <section class="wiki-edit-history-preview">
          <aside class="wiki-edit-history-list">
            <a-button
              v-for="entry in historyEntries"
              :key="entry.id"
              class="app-text-action-btn app-glass-surface wiki-edit-history-item"
              :class="{ 'wiki-edit-history-item--active': selectedHistory?.id === entry.id }"
              type="text"
              @click="openHistoryPreview(entry)"
            >
              <span class="wiki-edit-history-item-content">
                <strong class="wiki-edit-history-summary">{{ entry.change_summary || '未填写修改说明' }}</strong>
                <time class="wiki-edit-history-label" :datetime="entry.created_at">
                  {{ formatDateTime(entry.created_at) || '时间未知' }}
                </time>
              </span>
            </a-button>
          </aside>

          <div class="wiki-edit-history-preview-main">
            <div class="wiki-edit-history-preview-header">
              <div class="wiki-edit-history-preview-meta">
                <strong class="wiki-edit-history-preview-summary">{{ selectedHistory.change_summary || '未填写修改说明' }}</strong>
                <time :datetime="selectedHistory.created_at">
                  {{ formatDateTime(selectedHistory.created_at) || '时间未知' }}
                </time>
              </div>

              <div class="wiki-edit-history-preview-actions">
                <a-button class="app-text-action-btn" type="text" size="small" @click="previewHistoryContent = !previewHistoryContent">
                  {{ previewHistoryContent ? '查看源码' : '预览渲染' }}
                </a-button>
                <a-button type="primary" size="small" @click="restoreHistory">
                  恢复到编辑器
                </a-button>
              </div>
            </div>

            <div class="app-glass-surface wiki-edit-history-preview-panel">
              <div
                v-if="previewHistoryContent"
                class="wiki-edit-history-preview-surface wiki-edit-history-preview-rendered"
              >
                <markdown-renderer :content="selectedHistory.content" />
              </div>
              <pre
                v-else
                class="wiki-edit-history-preview-surface wiki-edit-history-preview-source"
              >{{ selectedHistory.content }}</pre>
            </div>
          </div>
        </section>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useWikiEditDocument } from '@/composables/useWikiEditDocument'
import { useWikiEditHistory } from '@/composables/useWikiEditHistory'
import { useGamesStore } from '@/stores/games'
import { useUiStore } from '@/stores/ui'
import { navigateBackOrFallback } from '@/utils/navigation'
import { formatDisplayDateTime } from '@/utils/date'
import { getAmbientBackgroundPoolFromGameDetail, hasAmbientBackgroundPoolImages } from '@/utils/ambient-background'
import {
  IconSave
} from '@arco-design/web-vue/es/icon'
import WikiEditor from '@/components/WikiEditor.vue'

const route = useRoute()
const router = useRouter()
const gamesStore = useGamesStore()
const uiStore = useUiStore()

const MarkdownRenderer = defineAsyncComponent(() => import('@/components/MarkdownRenderer.vue'))

const AMBIENT_BACKGROUND_OWNER = 'wiki-edit'

const syncAmbientBackground = () => {
  const pool = getAmbientBackgroundPoolFromGameDetail(game.value)
  if (!game.value?.public_id || !hasAmbientBackgroundPoolImages(pool)) {
    uiStore.clearAmbientBackgroundSource(AMBIENT_BACKGROUND_OWNER)
    return
  }

  uiStore.setAmbientBackgroundSource({
    owner: AMBIENT_BACKGROUND_OWNER,
    key: game.value.public_id,
    pool,
  })
}

onUnmounted(() => {
  uiStore.clearAmbientBackgroundSource(AMBIENT_BACKGROUND_OWNER)
})

const requestedGameId = computed(() => {
  const rawValue = route.params.publicId
  return typeof rawValue === 'string' ? rawValue.trim() : Array.isArray(rawValue) ? String(rawValue[0] || '').trim() : ''
})

const getGameDetailRoute = () => {
  if (!game.value?.public_id) {
    return { name: 'games' as const }
  }

  return {
    name: 'game-detail' as const,
    params: { publicId: game.value.public_id },
  }
}

const handleCancel = () => {
  navigateBackOrFallback(router, getGameDetailRoute())
}

const formatDateTime = (value?: string) => formatDisplayDateTime(value)

const {
  game,
  wikiData,
  isSaving,
  isExisting,
  loadWikiEditorData,
  handleSave,
} = useWikiEditDocument({
  gamesStore,
  uiStore,
  requestedGameId,
  onLoadGameFailed: () => {
    router.push({ name: 'games' })
  },
  onSaveSuccess: async () => {
    if (game.value?.public_id) {
      await loadHistory(game.value.public_id)
    }
    navigateBackOrFallback(router, getGameDetailRoute())
  },
})

const {
  historyEntries,
  selectedHistory,
  isHistoryLoading,
  hasHistoryLoadFailure,
  previewHistoryContent,
  historyPreviewVisible,
  resetHistoryState,
  loadHistory,
  restoreHistory,
  openHistoryDialog,
  openHistoryPreview,
} = useWikiEditHistory({
  wikiData,
  addAlert: (message, type) => uiStore.addAlert(message, type),
  formatDateTime,
})

watch(
  requestedGameId,
  async (gameId) => {
    if (!gameId) {
      uiStore.clearAmbientBackgroundSource(AMBIENT_BACKGROUND_OWNER)
      return
    }
    uiStore.clearAmbientBackgroundSource(AMBIENT_BACKGROUND_OWNER)
    const loaded = await loadWikiEditorData(gameId)
    if (!loaded) {
      return
    }
    syncAmbientBackground()
    resetHistoryState()
    await loadHistory(gameId)
  },
  { immediate: true },
)
</script>

<style scoped>
.wiki-edit {
  min-height: calc(100vh - 88px);
  display: flex;
  flex-direction: column;
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

.wiki-edit-main {
  width: 100%;
  height: calc(100vh - 220px);
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

.wiki-edit-summary :deep(.arco-input-wrapper) {
  border-color: rgb(var(--primary-6));
  background: rgba(var(--primary-6), 0.08);
}

.wiki-edit-summary :deep(.arco-input-wrapper:hover),
.wiki-edit-summary :deep(.arco-input-wrapper.arco-input-focus) {
  border-color: rgb(var(--primary-6));
  background: rgba(var(--primary-6), 0.08);
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
.wiki-edit-preview-card {
  margin-bottom: 16px;
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
}

.wiki-edit-history-item {
  font-size: 12px;
  text-align: left;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  justify-content: flex-start;
  width: 100%;
  box-sizing: border-box;
  padding: 12px;
  border-radius: 10px;
  background: color-mix(in srgb, var(--app-card-surface) 86%, transparent);
  color: var(--color-text-1);
  cursor: pointer;
  transition: border-color 0.2s ease, background 0.2s ease;
  white-space: normal;
  min-width: 0;
}

.wiki-edit-history-item.arco-btn {
  display: flex;
  align-items: stretch;
  justify-content: flex-start;
  width: 100%;
  height: auto;
  min-height: 64px;
  gap: 4px;
  white-space: normal;
}

.wiki-edit-history-item-content {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 4px;
  width: 100%;
  min-width: 0;
  text-align: left;
}

.wiki-edit-history-summary,
.wiki-edit-history-label {
  display: block;
  width: 100%;
  max-width: 100%;
  min-width: 0;
}

.wiki-edit-history-summary {
  line-height: 1.5;
  overflow-wrap: anywhere;
  word-break: break-word;
}

.wiki-edit-history-label {
  color: var(--color-text-3);
  font-size: 11px;
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex-shrink: 0;
}

.wiki-edit-history-item:hover,
.wiki-edit-history-item--active {
  border-color: rgb(var(--primary-6));
  background: rgba(var(--primary-6), 0.08);
}

.wiki-edit-history-empty {
  min-height: 120px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text-3);
}

.wiki-edit-history-empty--dialog {
  min-height: 320px;
}

.wiki-edit-history-preview-meta {
  display: flex;
  flex-direction: column;
  gap: 4px;
  flex: 1;
  min-width: 0;
  color: var(--color-text-2);
  font-size: 12px;
}

.wiki-edit-history-preview-summary {
  font-size: 18px;
  line-height: 1.5;
  color: var(--color-text-1);
  max-width: 100%;
  overflow-wrap: anywhere;
  word-break: break-word;
}

.wiki-edit-history-preview {
  display: grid;
  grid-template-columns: 280px minmax(0, 1fr);
  gap: 16px;
  width: 100%;
  max-width: 100%;
  box-sizing: border-box;
  min-height: 0;
}

.wiki-edit-history-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  width: 100%;
  box-sizing: border-box;
  max-height: min(70vh, 720px);
  overflow-y: auto;
  overflow-x: hidden;
  padding-right: 4px;
  min-width: 0;
}

.wiki-edit-history-preview-main {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}

.wiki-edit-history-preview-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
}

.wiki-edit-history-preview-actions {
  display: flex;
  flex-direction: row;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
}

.wiki-edit-history-preview-panel {
  min-height: min(70vh, 720px);
  border-radius: 12px;
  overflow: hidden;
  background: color-mix(in srgb, var(--app-card-surface) 92%, transparent);
}

.wiki-edit-history-preview-surface {
  overflow: auto;
  min-height: min(70vh, 720px);
  max-height: min(70vh, 720px);
  margin: 0;
  padding: 16px 18px;
  box-sizing: border-box;
  background: transparent;
}

.wiki-edit-history-preview-rendered,
.wiki-edit-history-preview-source {
  margin: 0;
}

.wiki-edit-history-preview-rendered {
  min-height: 100%;
}

.wiki-edit-history-preview-source {
  white-space: pre-wrap;
  word-break: break-word;
  color: var(--color-text-1);
  font-size: 14px;
  line-height: 1.6;
  font-family: 'Fira Code', 'Consolas', monospace;
}

@media (max-width: 1200px) {
  .wiki-edit-main {
    height: auto;
    min-height: 520px;
  }
}

@media (max-width: 992px) {
  .wiki-edit {
    min-height: auto;
  }

  .wiki-edit-header {
    flex-direction: column;
    align-items: stretch;
    gap: 12px;
    margin-bottom: 16px;
  }

  .wiki-edit-actions {
    width: 100%;
  }

  .wiki-edit-main {
    min-height: 460px;
  }

  .wiki-edit-history-preview-actions {
    flex-wrap: wrap;
  }

  .wiki-edit-history-preview-panel,
  .wiki-edit-history-preview-surface {
    min-height: 420px;
    max-height: 420px;
  }
}

@media (max-width: 768px) {
  .wiki-edit-title {
    font-size: 22px;
  }

  .wiki-edit-actions {
    flex-direction: column;
  }

  .wiki-edit-history-preview {
    grid-template-columns: 1fr;
    gap: 12px;
  }

  .wiki-edit-history-preview-header {
    flex-direction: column;
    gap: 8px;
  }

  .wiki-edit-history-list {
    max-height: 220px;
    padding-right: 0;
  }

  .wiki-edit-history-preview-panel,
  .wiki-edit-history-preview-rendered,
  .wiki-edit-history-preview-source {
    min-height: 240px;
    max-height: 240px;
  }

  .wiki-edit-history-preview-surface {
    padding: 12px;
  }
}
</style>

<style>
/* 仅约束历史弹窗的布局，底色、边框和阴影沿用全局 .arco-modal，避免编辑器内容透出。 */
.wiki-edit-history-modal.arco-modal {
  width: min(1040px, calc(100vw - 24px));
  max-width: calc(100vw - 24px);
  box-sizing: border-box;
  overflow: hidden;
}

.wiki-edit-history-modal .arco-modal-header {
  padding: 18px 20px 0;
  border-bottom: 0;
  background: transparent;
}

.wiki-edit-history-modal .arco-modal-body {
  box-sizing: border-box;
  min-width: 0;
  padding: 16px 20px 20px;
  background: transparent;
}

.wiki-edit-history-modal .arco-modal-close-btn {
  top: 16px;
  right: 16px;
}

@media (max-width: 768px) {
  .wiki-edit-history-modal .arco-modal-header {
    padding: 16px 16px 0;
  }

  .wiki-edit-history-modal .arco-modal-body {
    padding: 12px 16px 16px;
  }
}
</style>
