<template>
  <a-modal
    :visible="visible"
    title="从 Wiki 提取元数据"
    :width="760"
    :footer="false"
    @update:visible="emit('update:visible', $event)"
  >
    <div v-if="candidates.length > 0" class="wiki-metadata-picker">
      <div class="wiki-metadata-picker__hint">
        从当前 Wiki 中提取到了这些字段。你可以勾选要应用到表单的项目。
      </div>

      <div class="wiki-metadata-picker__list">
        <label
          v-for="item in candidates"
          :key="item.key"
          class="wiki-metadata-picker__item"
        >
          <a-checkbox
            :model-value="item.selected"
            @change="emit('selection-change', { key: item.key, selected: Boolean($event) })"
          />
          <div class="wiki-metadata-picker__meta">
            <div class="wiki-metadata-picker__label">{{ item.label }}</div>
            <div class="wiki-metadata-picker__value">{{ item.value }}</div>
          </div>
        </label>
      </div>

      <div class="cover-selector-actions">
        <a-button class="app-text-action-btn" type="text" html-type="button" @click="emit('update:visible', false)">取消</a-button>
        <a-button
          type="primary"
          html-type="button"
          :loading="isApplyingWikiMetadata"
          @click="emit('apply')"
        >
          应用到表单
        </a-button>
      </div>
    </div>
    <a-empty v-else description="没有识别到可提取的字段" />
  </a-modal>
</template>

<script setup lang="ts">
import type { WikiMetadataCandidateSelection } from '@/composables/useSteamImport'

defineProps<{
  visible: boolean
  candidates: WikiMetadataCandidateSelection[]
  isApplyingWikiMetadata: boolean
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'selection-change': [payload: { key: string; selected: boolean }]
  apply: []
}>()
</script>

<style scoped>
.wiki-metadata-picker {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.wiki-metadata-picker__hint {
  font-size: 13px;
  color: var(--color-text-2);
}

.wiki-metadata-picker__list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  max-height: 420px;
  overflow-y: auto;
}

.wiki-metadata-picker__item {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 12px;
  align-items: start;
  padding: 12px 14px;
  border: 1px solid var(--color-border-2);
  border-radius: 10px;
  background: var(--color-fill-1);
  cursor: pointer;
}

.wiki-metadata-picker__meta {
  min-width: 0;
}

.wiki-metadata-picker__label {
  font-size: 13px;
  font-weight: 700;
  color: var(--color-text-1);
}

.wiki-metadata-picker__value {
  margin-top: 4px;
  font-size: 12px;
  color: var(--color-text-3);
  white-space: pre-wrap;
  word-break: break-word;
}

.cover-selector-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 4px;
}
</style>
