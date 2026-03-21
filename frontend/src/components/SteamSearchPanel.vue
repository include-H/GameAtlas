<template>
  <div class="steam-search-section">
    <div class="app-input-action-row">
      <a-input
        :model-value="query"
        class="app-input-action-row__field"
        :placeholder="placeholder"
        allow-clear
        @update:model-value="emit('update:query', $event)"
        @press-enter="emit('search')"
        @clear="emit('clear')"
      >
        <template #prefix>
          <icon-cloud-download />
        </template>
      </a-input>
      <a-button
        class="app-input-action-row__action"
        type="secondary"
        :loading="loading"
        @click="emit('search')"
      >
        <template #icon>
          <icon-search />
        </template>
        搜索
      </a-button>
    </div>

    <div
      v-if="results.length > 0 && !selectedGame"
      class="steam-search-results"
    >
      <div class="steam-search-title">选择游戏</div>
      <div
        v-for="game in results"
        :key="game.id"
        class="steam-search-result-item"
        @click="emit('select', game)"
      >
        <img :src="game.tinyImage" :alt="game.name" />
        <div class="steam-result-info">
          <div class="steam-result-name">{{ game.name }}</div>
          <div class="steam-result-meta">{{ game.releaseDate || '' }}</div>
        </div>
      </div>
    </div>

    <slot />
  </div>
</template>

<script setup lang="ts">
import { IconCloudDownload, IconSearch } from '@arco-design/web-vue/es/icon'
import type { SteamGameSearchResult } from '@/services/types'

defineProps<{
  query: string
  placeholder?: string
  loading?: boolean
  results: SteamGameSearchResult[]
  selectedGame: SteamGameSearchResult | null
}>()

const emit = defineEmits<{
  'update:query': [value: string]
  search: []
  clear: []
  select: [game: SteamGameSearchResult]
}>()
</script>

<style scoped>
.steam-search-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.steam-search-title {
  font-size: 14px;
  font-weight: 700;
  color: var(--color-text-1);
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.steam-search-results {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 300px;
  overflow-y: auto;
  padding: 8px;
  background: var(--app-card-surface);
  border: 1px solid var(--app-card-border);
  border-radius: 10px;
  backdrop-filter: blur(var(--app-card-backdrop-blur));
  -webkit-backdrop-filter: blur(var(--app-card-backdrop-blur));
}

.steam-search-result-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px;
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.2s ease, border-color 0.2s ease, transform 0.2s ease;
  border: 1px solid var(--app-card-border);
  background: color-mix(in srgb, var(--app-card-surface) 84%, transparent);
}

.steam-search-result-item:hover {
  background: var(--color-fill-2);
  border-color: rgba(var(--primary-6), 0.55);
  transform: translateY(-1px);
}

.steam-search-result-item img {
  width: 60px;
  height: 32px;
  flex-shrink: 0;
  object-fit: cover;
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.04);
}

.steam-result-info {
  flex: 1;
  min-width: 0;
}

.steam-result-name {
  font-size: 14px;
  font-weight: 700;
  color: var(--color-text-1);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.steam-result-meta {
  margin-top: 2px;
  font-size: 12px;
  color: var(--color-text-3);
}

@media (max-width: 576px) {
  .steam-search-results {
    max-height: 240px;
    padding: 6px;
  }

  .steam-search-result-item {
    gap: 10px;
    padding: 6px;
  }

  .steam-search-result-item img {
    width: 52px;
    height: 28px;
  }

  .steam-result-name {
    font-size: 13px;
  }
}
</style>
