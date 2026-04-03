import { ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { EditGameForm } from '@/composables/edit-game-form'
import { useTagSelection } from './useTagSelection'

const { createTagMock } = vi.hoisted(() => ({
  createTagMock: vi.fn(),
}))

vi.mock('@/services/tags.service', () => ({
  default: {
    createTag: createTagMock,
  },
}))

const buildForm = (tagIds: Array<string | number>): EditGameForm => ({
  title: '',
  title_alt: '',
  visibility: 'public',
  developer_ids: [],
  publisher_ids: [],
  release_date: undefined,
  engine: '',
  platform_ids: [],
  series_id: null,
  tag_ids: tagIds,
  summary: '',
  cover_image: '',
  banner_image: '',
  preview_videos: [],
  screenshots: [],
  file_paths: [{ path: '', label: '' }],
})

describe('useTagSelection', () => {
  beforeEach(() => {
    createTagMock.mockReset()
  })

  it('keeps selected inactive tags visible and resolvable', async () => {
    const form = ref(buildForm([9]))
    const tagGroups = ref([
      {
        id: 1,
        key: 'genre',
        name: 'Genre',
        allow_multiple: true,
        is_filterable: true,
        sort_order: 1,
        created_at: '',
        updated_at: '',
      },
    ])
    const tagOptions = ref([
      {
        id: 9,
        group_id: 1,
        group_key: 'genre',
        group_name: 'Genre',
        name: 'Inactive Tag',
        slug: 'inactive-tag',
        sort_order: 9,
        is_active: false,
        created_at: '',
        updated_at: '',
      },
    ])

    const selection = useTagSelection({
      tagGroups,
      tagOptions,
      form,
      getWikiContent: () => '',
      addAlert: vi.fn(),
    })

    expect(selection.tagOptionsByGroup.value[1]?.map((item) => item.id)).toEqual([9])
    expect(selection.tagFieldValuesByGroup.value[1]).toEqual([9])
    await expect(selection.resolveTagSelections()).resolves.toEqual([9])
  })

  it('submits the same single-select value that the field displays', async () => {
    const form = ref(buildForm([1, 2]))
    const tagGroups = ref([
      {
        id: 1,
        key: 'perspective',
        name: 'Perspective',
        allow_multiple: false,
        is_filterable: true,
        sort_order: 1,
        created_at: '',
        updated_at: '',
      },
    ])
    const tagOptions = ref([
      {
        id: 1,
        group_id: 1,
        group_key: 'perspective',
        group_name: 'Perspective',
        name: 'First',
        slug: 'first',
        sort_order: 1,
        is_active: true,
        created_at: '',
        updated_at: '',
      },
      {
        id: 2,
        group_id: 1,
        group_key: 'perspective',
        group_name: 'Perspective',
        name: 'Second',
        slug: 'second',
        sort_order: 2,
        is_active: true,
        created_at: '',
        updated_at: '',
      },
    ])

    const selection = useTagSelection({
      tagGroups,
      tagOptions,
      form,
      getWikiContent: () => '',
      addAlert: vi.fn(),
    })

    expect(selection.tagFieldValuesByGroup.value[1]).toBe(1)
    await expect(selection.resolveTagSelections()).resolves.toEqual([1])
  })

  it('replaces existing single-select tags when applying wiki tags', async () => {
    const form = ref(buildForm([1]))
    const tagGroups = ref([
      {
        id: 1,
        key: 'perspective',
        name: 'Perspective',
        allow_multiple: false,
        is_filterable: true,
        sort_order: 1,
        created_at: '',
        updated_at: '',
      },
    ])
    const tagOptions = ref([
      {
        id: 1,
        group_id: 1,
        group_key: 'perspective',
        group_name: 'Perspective',
        name: 'First',
        slug: 'first',
        sort_order: 1,
        is_active: true,
        created_at: '',
        updated_at: '',
      },
      {
        id: 2,
        group_id: 1,
        group_key: 'perspective',
        group_name: 'Perspective',
        name: 'Second',
        slug: 'second',
        sort_order: 2,
        is_active: true,
        created_at: '',
        updated_at: '',
      },
    ])

    const selection = useTagSelection({
      tagGroups,
      tagOptions,
      form,
      getWikiContent: () => '',
      addAlert: vi.fn(),
    })

    selection.wikiTagCandidates.value = [
      {
        key: 'perspective:second',
        value: 'Second',
        sourceLabel: '视角',
        groupKey: 'perspective',
      },
    ]

    await selection.applySelectedWikiTags()

    expect(form.value.tag_ids).toEqual([2])
    expect(selection.tagFieldValuesByGroup.value[1]).toBe(2)
  })
})
