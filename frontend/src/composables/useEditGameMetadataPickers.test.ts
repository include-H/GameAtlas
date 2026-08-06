import { ref } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import type { EditGameForm } from '@/utils/edit-game-form'
import { useEditGameMetadataPickers } from './useEditGameMetadataPickers'

const buildForm = () => ref<EditGameForm>({
  title: 'Example Game',
  title_alt: '',
  visibility: 'public',
  developer_ids: [],
  publisher_ids: [],
  release_date: undefined,
  series_id: null,
  summary: '',
  covers: [],
  banners: [],
  logos: [],
  logo_visible: true,
  preview_videos: [],
  screenshots: [],
  file_paths: [],
})

const createSeriesItem = (id: number, name: string) => ({
  id,
  name,
  slug: `slug-${id}`,
  sort_order: 0,
  created_at: '2026-01-01T00:00:00Z',
})

describe('useEditGameMetadataPickers', () => {
  it('exposes create option constants', () => {
    const pickers = useEditGameMetadataPickers({ form: buildForm(), addAlert: vi.fn() })
    expect(pickers.CREATE_SERIES_OPTION_VALUE).toBe('__create_series__')
    expect(pickers.CREATE_DEVELOPER_OPTION_VALUE).toBe('__create_developer__')
    expect(pickers.CREATE_PUBLISHER_OPTION_VALUE).toBe('__create_publisher__')
  })

  it('sorts filtered options alphabetically', () => {
    const pickers = useEditGameMetadataPickers({ form: buildForm(), addAlert: vi.fn() })
    pickers.seriesPicker.options.value = [
      createSeriesItem(2, 'Zeta'),
      createSeriesItem(1, 'Alpha'),
    ]
    expect(pickers.filteredSeriesOptions.value.map((item) => item.name)).toEqual(['Alpha', 'Zeta'])
    expect(pickers.seriesOptions.value).toHaveLength(2)
  })

  it('sets series_id on valid series selection', () => {
    const form = buildForm()
    const pickers = useEditGameMetadataPickers({ form, addAlert: vi.fn() })
    pickers.handleSeriesSelection(7)
    expect(form.value.series_id).toBe(7)
  })

  it('ignores invalid series selection values', () => {
    const form = buildForm()
    const pickers = useEditGameMetadataPickers({ form, addAlert: vi.fn() })
    pickers.handleSeriesSelection('bad-value')
    expect(form.value.series_id).toBeNull()
  })

  it('dedupes developer ids on selection', () => {
    const form = buildForm()
    const pickers = useEditGameMetadataPickers({ form, addAlert: vi.fn() })
    pickers.handleDeveloperSelection([1, 2, 1])
    expect(form.value.developer_ids).toEqual([1, 2])
  })

  it('does not store the create pseudo value in developer_ids', () => {
    const form = buildForm()
    const pickers = useEditGameMetadataPickers({ form, addAlert: vi.fn() })
    pickers.handleDeveloperSelection([1, '__create_developer__'])
    expect(form.value.developer_ids).toEqual([1])
  })

  it('normalizes publisher selection ids', () => {
    const form = buildForm()
    const pickers = useEditGameMetadataPickers({ form, addAlert: vi.fn() })
    pickers.handlePublisherSelection([5, 'junk'])
    expect(form.value.publisher_ids).toEqual([5])
  })
})
