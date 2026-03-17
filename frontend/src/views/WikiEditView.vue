<template>
  <div class="wiki-edit">
    <!-- Header -->
    <div class="wiki-edit-header">
      <div class="wiki-edit-header-left">
        <a-breadcrumb class="wiki-edit-breadcrumb">
          <a-breadcrumb-item v-for="(item, index) in breadcrumbs" :key="index">
            <span v-if="item.to" @click="router.push(item.to)">
              {{ item.title }}
            </span>
            <span v-else>{{ item.title }}</span>
          </a-breadcrumb-item>
        </a-breadcrumb>
        <h1 class="wiki-edit-title">
          {{ isExisting ? 'Edit Wiki' : 'Create Wiki' }}
        </h1>
        <p v-if="game" class="wiki-edit-subtitle">
          {{ game.title }}
        </p>
      </div>

      <div class="wiki-edit-actions">
        <a-button
          :disabled="isSaving"
          @click="router.back()"
        >
          Cancel
        </a-button>
        <a-button
          type="primary"
          :loading="isSaving"
          @click="handleSave"
        >
          <template #icon>
            <icon-save />
          </template>
          Save
        </a-button>
      </div>
    </div>

    <!-- Wiki Form -->
    <a-row :gutter="16" justify="center" class="wiki-edit-row">
      <a-col :xs="24" :sm="24" :md="24" :lg="22" :xl="20" :xxl="18">
        <a-card class="wiki-edit-card">
          <!-- Editor -->
          <wiki-editor v-model="wikiData.content" />
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
import wikiService, { type WikiContent } from '@/services/wiki.service'
import {
  IconSave
} from '@arco-design/web-vue/es/icon'
import WikiEditor from '@/components/WikiEditor.vue'

const route = useRoute()
const router = useRouter()
const gamesStore = useGamesStore()
const uiStore = useUiStore()

const game = computed(() => gamesStore.currentGame)
const wiki = ref<WikiContent | null>(null)

const isSaving = ref(false)

const wikiData = ref({
  content: '',
})

const isExisting = computed(() => Boolean(wiki.value?.content))

const breadcrumbs = computed(() => {
  const items = [
    { title: 'Home', to: '/' },
    { title: 'Games', to: '/games' },
  ]

  if (game.value) {
    items.push({
      title: game.value.title,
      to: `/games/${game.value.id}`,
    })
  }

  items.push({ title: isExisting.value ? 'Edit Wiki' : 'Create Wiki', to: '' })

  return items
})

const handleSave = async () => {
  if (!game.value) return

  isSaving.value = true

  try {
    const wikiDataToSend = {
      content: wikiData.value.content,
      change_summary: undefined, // Can be added later if needed
    }

    await wikiService.updateWikiPage(String(game.value.id), wikiDataToSend)
    uiStore.addAlert(isExisting.value ? 'Wiki updated successfully' : 'Wiki created successfully', 'success')

    router.push({ name: 'game-detail', params: { id: String(game.value.id) } })
  } catch (error: any) {
    const errorMessage = error?.response?.data?.error || error?.message || 'Failed to save wiki'
    uiStore.addAlert(errorMessage, 'error')
    console.error('Failed to save wiki:', error)
  } finally {
    isSaving.value = false
  }
}

const loadWikiEditorData = async (gameId: string) => {
  try {
    await gamesStore.fetchGame(gameId)
    wiki.value = null
    wikiData.value = {
      content: '',
    }

    // Try to load existing wiki
    try {
      const wikiContent = await wikiService.getWikiPage(gameId)
      if (wikiContent && wikiContent.content) {
        wiki.value = wikiContent
        wikiData.value = {
          content: wikiContent.content,
        }
      }
    } catch {
      // Wiki doesn't exist yet
    }
  } catch (error) {
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

.wiki-edit-breadcrumb {
  margin-bottom: 8px;
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

.wiki-edit-title-input {
  margin-bottom: 16px;
}

.wiki-edit-info-card,
.wiki-edit-preview-card,
.wiki-edit-history-card {
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
  display: flex;
  flex-direction: column;
  gap: 2px;
  font-size: 12px;
}

.wiki-edit-history-label {
  color: var(--color-text-3);
}
</style>
