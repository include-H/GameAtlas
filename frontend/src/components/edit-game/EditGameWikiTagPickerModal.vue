<template>
  <a-modal
    :visible="visible"
    title="从 Wiki 提取字段"
    :width="760"
    :footer="false"
    @update:visible="emit('update:visible', $event)"
  >
    <div v-if="candidates.length > 0" class="wiki-tag-picker">
      <div class="wiki-tag-picker__hint">
        从当前 Wiki 中提取到了这些词条。你可以给每一项选择归类，也可以忽略。
      </div>

      <div class="wiki-tag-picker__list">
        <div
          v-for="item in candidates"
          :key="item.key"
          class="wiki-tag-picker__item"
        >
          <div class="wiki-tag-picker__meta">
            <div class="wiki-tag-picker__value">{{ item.value }}</div>
            <div class="wiki-tag-picker__source">来源：{{ item.sourceLabel }}</div>
          </div>
          <a-select
            class="wiki-tag-picker__select"
            :model-value="item.groupKey"
            @change="emit('group-change', { key: item.key, value: $event })"
          >
            <a-option value="ignore">忽略</a-option>
            <a-option value="genre">题材</a-option>
            <a-option value="subgenre">子类型</a-option>
            <a-option value="perspective">视角</a-option>
            <a-option value="theme">内容属性</a-option>
          </a-select>
        </div>
      </div>

      <div class="cover-selector-actions">
        <a-button type="text" html-type="button" @click="emit('update:visible', false)">取消</a-button>
        <a-button
          type="primary"
          html-type="button"
          :loading="isApplyingWikiTags"
          @click="emit('apply')"
        >
          应用到标签
        </a-button>
      </div>
    </div>
    <a-empty v-else description="没有识别到可提取的字段" />
  </a-modal>
</template>

<script setup lang="ts">
import type { WikiTagCandidateSelection } from '@/composables/useTagSelection'

defineProps<{
  visible: boolean
  candidates: WikiTagCandidateSelection[]
  isApplyingWikiTags: boolean
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'group-change': [payload: { key: string; value: string | number | undefined }]
  apply: []
}>()
</script>

<style scoped>
.wiki-tag-picker {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.wiki-tag-picker__hint {
  font-size: 13px;
  color: var(--color-text-2);
}

.wiki-tag-picker__list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  max-height: 420px;
  overflow-y: auto;
}

.wiki-tag-picker__item {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 180px;
  gap: 12px;
  align-items: center;
  padding: 12px 14px;
  border: 1px solid var(--color-border-2);
  border-radius: 10px;
  background: var(--color-fill-1);
}

.wiki-tag-picker__meta {
  min-width: 0;
}

.wiki-tag-picker__value {
  font-size: 14px;
  font-weight: 700;
  color: var(--color-text-1);
  word-break: break-word;
}

.wiki-tag-picker__source {
  margin-top: 4px;
  font-size: 12px;
  color: var(--color-text-3);
}

.wiki-tag-picker__select {
  width: 100%;
}

.cover-selector-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 4px;
}

@media (max-width: 768px) {
  .wiki-tag-picker__item {
    grid-template-columns: 1fr;
  }
}
</style>
