<template>
  <a-form-item>
    <template #label>
      <div class="summary-label">
        <span>标签</span>
        <a-button
          type="text"
          size="mini"
          html-type="button"
          :disabled="!wikiContentExists"
          :loading="isPreparingWikiTagCandidates"
          @click="emit('parse-wiki-tags')"
        >
          从 Wiki 提取字段
        </a-button>
      </div>
    </template>
    <div v-if="tagGroups.length > 0" class="tag-group-grid">
      <div
        v-for="group in tagGroups"
        :key="group.id"
        class="tag-group-field"
      >
        <div class="tag-group-field__label">
          <span>{{ group.name }}</span>
        </div>
        <a-select
          class="tag-group-select"
          :model-value="tagSelectionsByGroup[group.id]"
          :multiple="group.allow_multiple"
          allow-clear
          allow-search
          allow-create
          :max-tag-count="group.allow_multiple ? 2 : 1"
          :placeholder="`选择${group.name}`"
          @change="handleRawTagSelectionChange(group.id, $event)"
        >
          <a-option
            v-for="pendingTag in pendingTagOptionsByGroup[group.id] || []"
            :key="pendingTag.value"
            :value="pendingTag.value"
            :label="pendingTag.label"
          >
            {{ pendingTag.label }}
          </a-option>
          <a-option
            v-for="tag in tagOptionsByGroup[group.id] || []"
            :key="tag.id"
            :value="tag.id"
            :label="tag.name"
          >
            {{ tag.name }}
          </a-option>
        </a-select>
      </div>
    </div>
    <div v-else class="tag-group-empty">
      暂无可用标签。重启后端完成 migration 后，这里会显示可选标签组。
    </div>
  </a-form-item>
</template>

<script setup lang="ts">
import type { Tag, TagGroup } from '@/services/types'
import type { SelectOptionValue } from '@arco-design/web-vue/es/select/interface'

export type GameTagSelectionValue =
  | string
  | number
  | string[]
  | number[]
  | Array<string | number>
  | null
  | undefined

interface PendingTagOption {
  value: string
  label: string
}

defineProps<{
  tagGroups: TagGroup[]
  tagSelectionsByGroup: Record<number, GameTagSelectionValue>
  pendingTagOptionsByGroup: Record<number, PendingTagOption[]>
  tagOptionsByGroup: Record<number, Tag[]>
  wikiContentExists: boolean
  isPreparingWikiTagCandidates: boolean
}>()

const emit = defineEmits<{
  'parse-wiki-tags': []
  'tag-selection-change': [payload: { groupId: number; value: GameTagSelectionValue }]
}>()

type RawTagSelectionValue = SelectOptionValue | SelectOptionValue[] | null | undefined

const normalizeTagSelectionValue = (value: RawTagSelectionValue): GameTagSelectionValue => {
  if (
    value === null ||
    value === undefined ||
    typeof value === 'string' ||
    typeof value === 'number'
  ) {
    return value
  }

  if (Array.isArray(value)) {
    if (value.every((item) => typeof item === 'string' || typeof item === 'number')) {
      return value
    }
  }

  return undefined
}

const handleRawTagSelectionChange = (groupId: number, value: RawTagSelectionValue) => {
  handleTagSelectionChange(groupId, normalizeTagSelectionValue(value))
}

const handleTagSelectionChange = (groupId: number, value: GameTagSelectionValue) => {
  emit('tag-selection-change', { groupId, value })
}
</script>

<style scoped>
.summary-label {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  gap: 12px;
  font-weight: 700;
}

.tag-group-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
  width: 100%;
}

.tag-group-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.tag-group-select {
  width: 100%;
}

.tag-group-select :deep(.arco-select-view) {
  min-height: 36px;
  align-items: flex-start;
}

.tag-group-select :deep(.arco-select-view-value) {
  flex-wrap: wrap;
  gap: 4px;
}

.tag-group-select :deep(.arco-select-view-tag) {
  max-width: 100%;
}

.tag-group-field__label {
  display: flex;
  font-size: 13px;
  font-weight: 700;
  color: var(--color-text-2);
}

.tag-group-empty {
  font-size: 12px;
  color: var(--color-text-3);
}

@media (max-width: 1200px) {
  .tag-group-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 768px) {
  .tag-group-grid {
    grid-template-columns: 1fr;
  }
}
</style>
