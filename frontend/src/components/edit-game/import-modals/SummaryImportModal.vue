<template>
  <a-modal
    :visible="visible"
    title="导入 Steam 简介"
    :width="800"
    :footer="false"
    @update:visible="emit('update:visible', $event)"
  >
    <div class="cover-selector-content">
      <steam-search-panel
        :query="steamSummarySearchQuery"
        placeholder="搜索 Steam 游戏..."
        :loading="isSearchingSteamSummary"
        :results="steamSummarySearchResults"
        :selected-game="selectedSteamSummaryGame"
        @update:query="emit('update:steam-summary-search-query', $event)"
        @search="emit('search-summary')"
        @clear="emit('clear-summary')"
        @select="emit('select-summary', $event)"
      >
        <div v-if="selectedSteamSummaryGame" class="steam-summary-section">
          <div class="steam-search-title">
            {{ selectedSteamSummaryGame.name }} 的简介
            <a-button class="app-text-action-btn" type="text" size="mini" html-type="button" @click="emit('back-summary')">返回</a-button>
          </div>

          <div v-if="steamSummaryPreview" class="steam-summary-preview">
            {{ steamSummaryPreview }}
          </div>

          <a-empty
            v-else-if="!isSearchingSteamSummary"
            description="Steam 未返回可用简介"
            class="steam-summary-empty"
          />

          <a-button
            v-if="steamSummaryPreview"
            type="primary"
            long
            html-type="button"
            @click="emit('confirm-summary-import')"
          >
            导入这段简介
          </a-button>
        </div>
      </steam-search-panel>
    </div>
  </a-modal>
</template>

<script setup lang="ts">
import SteamSearchPanel from '@/components/SteamSearchPanel.vue'
import type { SteamGameSearchResult } from '@/services/types'

interface Props {
  visible: boolean
  steamSummarySearchQuery: string
  isSearchingSteamSummary: boolean
  steamSummarySearchResults: SteamGameSearchResult[]
  selectedSteamSummaryGame: SteamGameSearchResult | null
  steamSummaryPreview: string
}

defineProps<Props>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'update:steam-summary-search-query': [value: string]
  'search-summary': []
  'clear-summary': []
  'select-summary': [game: SteamGameSearchResult]
  'back-summary': []
  'confirm-summary-import': []
}>()
</script>

<style scoped>
.cover-selector-content {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.steam-summary-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.steam-summary-preview {
  max-height: 280px;
  overflow-y: auto;
  white-space: pre-wrap;
  line-height: 1.6;
  padding: 12px;
  border-radius: 8px;
  background: var(--color-fill-2);
  color: var(--color-text-2);
}

.steam-summary-empty {
  margin: 8px 0;
}

.steam-search-title {
  font-size: 14px;
  font-weight: 700;
  color: var(--color-text-1);
  display: flex;
  align-items: center;
  justify-content: space-between;
}
</style>
