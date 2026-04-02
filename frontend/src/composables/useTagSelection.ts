import { computed, ref, type Ref } from 'vue'
import type { EditGameForm } from '@/composables/edit-game-form'
import tagsService from '@/services/tags.service'
import type { Tag, TagGroup } from '@/services/types'
import { normalizeOptionId, resolveCreatableSelections } from '@/utils/creatable-select'
import { extractWikiTagCandidates, type WikiTagGroupKey } from '@/utils/wiki-tag-parser'

type TagSelectionValue = number | number[] | string | string[] | undefined

export interface TagSectionSelectionChangePayload {
  groupId: number
  value: number | number[] | string | string[] | Array<string | number> | null | undefined
}

export interface WikiTagCandidateSelection {
  key: string
  value: string
  sourceLabel: string
  groupKey: WikiTagGroupKey | 'ignore'
}

interface UseTagSelectionOptions {
  tagGroups: Ref<TagGroup[]>
  tagOptions: Ref<Tag[]>
  form: Ref<Pick<EditGameForm, 'tag_ids'>>
  getWikiContent: () => string
  addAlert: (message: string, type: 'success' | 'warning' | 'error') => void
}

const slugifyMetadataName = (name: string) => {
  return name
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
}

export const useTagSelection = (options: UseTagSelectionOptions) => {
  const pendingTagDraftsByGroup = ref<Record<number, string[]>>({})
  const isPreparingWikiTagCandidates = ref(false)
  const isApplyingWikiTags = ref(false)
  const wikiTagPickerVisible = ref(false)
  const wikiTagCandidates = ref<WikiTagCandidateSelection[]>([])

  const tagGroupIdByTagId = computed(() => {
    return new Map(options.tagOptions.value.map((tag) => [tag.id, tag.group_id]))
  })

  const tagOptionsByGroup = computed<Record<number, Tag[]>>(() => {
    const grouped: Record<number, Tag[]> = {}

    for (const tag of options.tagOptions.value) {
      if (!tag.is_active) continue
      if (!grouped[tag.group_id]) {
        grouped[tag.group_id] = []
      }
      grouped[tag.group_id].push(tag)
    }

    for (const groupId of Object.keys(grouped)) {
      grouped[Number(groupId)].sort((a, b) => a.sort_order - b.sort_order || a.id - b.id)
    }

    return grouped
  })

  const tagSelectionsByGroup = computed<Record<number, string | number | Array<string | number> | undefined>>(() => {
    const grouped: Record<number, Array<string | number>> = {}

    for (const tagId of options.form.value.tag_ids) {
      if (typeof tagId !== 'number') continue
      const groupId = tagGroupIdByTagId.value.get(tagId)
      if (!groupId) continue
      if (!grouped[groupId]) {
        grouped[groupId] = []
      }
      grouped[groupId].push(tagId)
    }

    for (const [groupId, drafts] of Object.entries(pendingTagDraftsByGroup.value)) {
      const normalizedGroupId = Number(groupId)
      if (!grouped[normalizedGroupId]) {
        grouped[normalizedGroupId] = []
      }
      grouped[normalizedGroupId].push(...drafts)
    }

    const selections: Record<number, string | number | Array<string | number> | undefined> = {}

    for (const group of options.tagGroups.value) {
      const values = grouped[group.id] || []
      selections[group.id] = group.allow_multiple ? values : values[0]
    }

    return selections
  })

  const pendingTagOptionsByGroup = computed<Record<number, Array<{ value: string; label: string }>>>(() => {
    const grouped: Record<number, Array<{ value: string; label: string }>> = {}

    for (const [groupId, names] of Object.entries(pendingTagDraftsByGroup.value)) {
      const normalizedGroupId = Number(groupId)
      grouped[normalizedGroupId] = names.map((name) => ({
        value: name,
        label: name,
      }))
    }

    return grouped
  })

  const handleFormTagChange = (groupId: number, value: TagSelectionValue) => {
    const rawValues = Array.isArray(value) ? value : value === undefined || value === null || value === '' ? [] : [value]
    const nextIds: number[] = []
    const nextDrafts: string[] = []

    for (const item of rawValues) {
      const normalizedId = normalizeOptionId(item)
      if (normalizedId !== null) {
        nextIds.push(normalizedId)
        continue
      }

      if (typeof item !== 'string') continue

      const name = item.trim()
      if (!name) continue

      const existing = options.tagOptions.value.find(
        (tag) => tag.group_id === groupId && tag.name.trim().toLowerCase() === name.toLowerCase(),
      )
      if (existing) {
        nextIds.push(existing.id)
        continue
      }

      if (!nextDrafts.some((draft) => draft.toLowerCase() === name.toLowerCase())) {
        nextDrafts.push(name)
      }
    }

    const preserved = options.form.value.tag_ids.filter((tagId) => {
      const normalizedId = normalizeOptionId(tagId)
      if (normalizedId === null) return false
      return tagGroupIdByTagId.value.get(normalizedId) !== groupId
    })

    options.form.value.tag_ids = [...preserved, ...nextIds]
    pendingTagDraftsByGroup.value = {
      ...pendingTagDraftsByGroup.value,
      [groupId]: nextDrafts,
    }
  }

  const handleTagSelectionChange = (groupId: number, value: TagSelectionValue) => {
    handleFormTagChange(groupId, value)
  }

  const handleTagSectionSelectionChange = (payload: TagSectionSelectionChangePayload) => {
    const { groupId, value } = payload
    if (value === null || value === undefined) {
      handleTagSelectionChange(groupId, undefined)
      return
    }

    if (typeof value === 'string' || typeof value === 'number') {
      handleTagSelectionChange(groupId, value)
      return
    }

    if (value.every((item) => typeof item === 'string')) {
      handleTagSelectionChange(groupId, value as string[])
      return
    }

    if (value.every((item) => typeof item === 'number')) {
      handleTagSelectionChange(groupId, value as number[])
      return
    }

    handleTagSelectionChange(groupId, value.map((item) => String(item)))
  }

  const resolveTagSelections = async () => {
    const idsByGroup = new Map<number, number[]>()

    for (const tagId of options.form.value.tag_ids) {
      const normalizedId = normalizeOptionId(tagId)
      if (normalizedId === null) continue
      const groupId = tagGroupIdByTagId.value.get(normalizedId)
      if (!groupId) continue
      const current = idsByGroup.get(groupId) || []
      current.push(normalizedId)
      idsByGroup.set(groupId, current)
    }

    for (const group of options.tagGroups.value) {
      const values: Array<string | number> = [
        ...(idsByGroup.get(group.id) || []),
        ...(pendingTagDraftsByGroup.value[group.id] || []),
      ]
      if (values.length === 0) continue

      const result = await resolveCreatableSelections({
        values,
        options: options.tagOptions.value,
        findExisting: (name, currentOptions) =>
          currentOptions.find(
            (item) => item.group_id === group.id && item.name.trim().toLowerCase() === name.toLowerCase(),
          ),
        createItem: (name) =>
          tagsService.createTag({
            group_id: group.id,
            name,
            slug: slugifyMetadataName(name),
          }),
      })

      options.tagOptions.value = result.options
      idsByGroup.set(group.id, result.ids)
    }

    pendingTagDraftsByGroup.value = {}
    return Array.from(idsByGroup.values()).flat()
  }

  const handleParseWikiTags = async () => {
    const content = options.getWikiContent()
    if (!content.trim()) {
      options.addAlert('当前游戏没有可解析的 Wiki 内容', 'warning')
      return
    }

    if (options.tagGroups.value.length === 0) {
      options.addAlert('当前没有可用标签组', 'warning')
      return
    }

    isPreparingWikiTagCandidates.value = true

    try {
      const extracted = extractWikiTagCandidates(content)
      if (extracted.length === 0) {
        options.addAlert('没有识别到可提取的标签字段', 'warning')
        return
      }

      wikiTagCandidates.value = extracted.map((item) => ({
        key: `${item.sourceLabel}:${item.value.toLowerCase()}`,
        value: item.value,
        sourceLabel: item.sourceLabel,
        groupKey: item.groupKey ?? 'ignore',
      }))
      wikiTagPickerVisible.value = true
    } catch (error) {
      console.error('Failed to extract wiki tags:', error)
      options.addAlert('从 Wiki 提取字段失败', 'warning')
    } finally {
      isPreparingWikiTagCandidates.value = false
    }
  }

  const handleWikiTagCandidateGroupChange = (
    key: string,
    value: WikiTagGroupKey | 'ignore' | number | string | undefined,
  ) => {
    const nextValue: WikiTagGroupKey | 'ignore' =
      value === 'genre' || value === 'subgenre' || value === 'perspective' || value === 'theme'
        ? value
        : 'ignore'

    wikiTagCandidates.value = wikiTagCandidates.value.map((item) =>
      item.key === key
        ? {
            ...item,
            groupKey: nextValue,
          }
        : item,
    )
  }

  const applySelectedWikiTags = async () => {
    const selected = wikiTagCandidates.value.filter((item) => item.groupKey !== 'ignore')
    if (selected.length === 0) {
        options.addAlert('还没有选择要应用的字段', 'warning')
        return
      }

    isApplyingWikiTags.value = true

    try {
      const mergedIds = new Set<number>(
        options.form.value.tag_ids
          .map((item) => normalizeOptionId(item))
          .filter((item): item is number => item !== null),
      )

      const grouped = new Map<WikiTagGroupKey, string[]>()
      for (const item of selected) {
        const groupKey = item.groupKey as WikiTagGroupKey
        const values = grouped.get(groupKey) || []
        if (!values.some((value) => value.toLowerCase() === item.value.toLowerCase())) {
          values.push(item.value)
        }
        grouped.set(groupKey, values)
      }

      const appliedLabels: string[] = []

      for (const [groupKey, names] of grouped.entries()) {
        const group = options.tagGroups.value.find((item) => item.key === groupKey)
        if (!group) continue

        const result = await resolveCreatableSelections({
          values: names,
          options: options.tagOptions.value,
          findExisting: (name, currentOptions) =>
            currentOptions.find(
              (item) => item.group_id === group.id && item.name.trim().toLowerCase() === name.toLowerCase(),
            ),
          createItem: (name) =>
            tagsService.createTag({
              group_id: group.id,
              name,
            }),
        })

        options.tagOptions.value = result.options
        for (const id of result.ids) {
          mergedIds.add(id)
        }

        if (result.ids.length > 0) {
          appliedLabels.push(`${group.name}：${names.join('、')}`)
        }
      }

      options.form.value.tag_ids = Array.from(mergedIds)
      pendingTagDraftsByGroup.value = {}
      wikiTagPickerVisible.value = false

      if (appliedLabels.length === 0) {
        options.addAlert('已选择字段，但没有成功应用到标签组', 'warning')
        return
      }

      options.addAlert(`已应用 Wiki 字段：${appliedLabels.join('；')}`, 'success')
    } catch (error) {
      console.error('Failed to apply wiki tags:', error)
      options.addAlert('应用 Wiki 字段失败', 'warning')
    } finally {
      isApplyingWikiTags.value = false
    }
  }

  const resetTagSelectionState = () => {
    pendingTagDraftsByGroup.value = {}
  }

  return {
    pendingTagDraftsByGroup,
    isPreparingWikiTagCandidates,
    isApplyingWikiTags,
    wikiTagPickerVisible,
    wikiTagCandidates,
    tagOptionsByGroup,
    tagSelectionsByGroup,
    pendingTagOptionsByGroup,
    handleTagSelectionChange,
    handleTagSectionSelectionChange,
    handleParseWikiTags,
    handleWikiTagCandidateGroupChange,
    applySelectedWikiTags,
    resolveTagSelections,
    resetTagSelectionState,
  }
}
