import { ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { EditGameForm } from '@/utils/edit-game-form'
import type { AdminGameDetail, Developer, Publisher, Series } from '@/services/types'
import { useEditGameWorkflow } from './useEditGameWorkflow'

const {
  updateGameAggregateMock,
  createSeriesMock,
  getPopularSeriesMock,
  resolveCreatableSelectionsMock,
  createDeveloperMock,
  createPublisherMock,
} = vi.hoisted(() => ({
  updateGameAggregateMock: vi.fn(),
  createSeriesMock: vi.fn(),
  getPopularSeriesMock: vi.fn(),
  resolveCreatableSelectionsMock: vi.fn(),
  createDeveloperMock: vi.fn(),
  createPublisherMock: vi.fn(),
}))

vi.mock('@/services/games.service', () => ({
  default: {
    updateGameAggregate: updateGameAggregateMock,
  },
}))

vi.mock('@/services/series.service', () => ({
  seriesService: {
    createSeries: createSeriesMock,
    getPopularSeries: getPopularSeriesMock,
  },
}))

vi.mock('@/services/developers.service', () => ({
  developersService: {
    createDeveloper: createDeveloperMock,
  },
}))

vi.mock('@/services/publishers.service', () => ({
  publishersService: {
    createPublisher: createPublisherMock,
  },
}))

vi.mock('@/utils/creatable-select', () => ({
  normalizeCreatableOptionName: (value: string) => value.trim().replace(/\s+/g, ' ').toLowerCase(),
  resolveCreatableSelections: resolveCreatableSelectionsMock,
}))

const buildOptions = () => {
  const addAlert = vi.fn()
  const emitSuccess = vi.fn()
  const closeModal = vi.fn()

  return {
    addAlert,
    emitSuccess,
    closeModal,
    options: {
      game: ref({
        id: 1,
        public_id: 'game-1',
        title: 'Game One',
        title_alt: null,
        visibility: 'public',
        summary: null,
        release_date: null,
        cover_image: null,
        banner_image: null,
        covers: [],
        banners: [],
        logos: [],
        wiki_content: null,
        downloads: 0,
        preview_videos: [],
        screenshots: [],
        series: null,
        developers: [],
        publishers: [],
        files: [],
        primary_screenshot: null,
        screenshot_count: 0,
        logo_visible: true,
        file_count: 0,
        developer_count: 0,
        publisher_count: 0,
        is_favorite: false,
        created_at: '2026-03-25T00:00:00Z',
        updated_at: '2026-03-25T00:00:00Z',
        isFavorite: false,
      } as AdminGameDetail),
      form: ref<EditGameForm>({
        title: 'Game One',
        title_alt: '',
        visibility: 'public' as const,
        developer_ids: [1],
        publisher_ids: [2],
        release_date: undefined,
        series_id: null,
        summary: '',
        banners: [],
        covers: [],
        logo: null,
        logo_visible: true,
        preview_videos: [],
        screenshots: [],
        file_paths: [],
      }),
      isSubmitting: ref(false),
      seriesOptions: ref<Series[]>([]),
      developerOptions: ref<Developer[]>([]),
      publisherOptions: ref<Publisher[]>([]),
      validateForm: vi.fn().mockResolvedValue(true),
      addAlert,
      emitSuccess,
      closeModal,
    },
  }
}

describe('useEditGameWorkflow', () => {
  beforeEach(() => {
    updateGameAggregateMock.mockReset()
    createSeriesMock.mockReset()
    getPopularSeriesMock.mockReset()
    resolveCreatableSelectionsMock.mockReset()
    createDeveloperMock.mockReset()
    createPublisherMock.mockReset()

    updateGameAggregateMock.mockResolvedValue({
      game: {
        id: 1,
        public_id: 'game-1',
      },
      warnings: [],
    })
    getPopularSeriesMock.mockResolvedValue([])
    resolveCreatableSelectionsMock.mockImplementation(async ({ values, options }) => ({
      ids: values.map((value: string | number) => Number(value)),
      options,
    }))
  })

  it('aborts submit when series resolution fails', async () => {
    const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
    const { options, addAlert, emitSuccess, closeModal } = buildOptions()
    options.form.value.series_id = 'Broken Series'
    createSeriesMock.mockRejectedValue(new Error('boom'))

    const workflow = useEditGameWorkflow(options)
    await workflow.handleSubmit()

    expect(updateGameAggregateMock).not.toHaveBeenCalled()
    expect(addAlert).toHaveBeenCalledWith('系列 "Broken Series" 处理失败', 'error')
    expect(emitSuccess).not.toHaveBeenCalled()
    expect(closeModal).not.toHaveBeenCalled()
    expect(options.isSubmitting.value).toBe(false)
    consoleErrorSpy.mockRestore()
  })

  it('preserves existing file notes when aggregate save does not edit them', async () => {
    const { options } = buildOptions()
    options.form.value.file_paths = [
      {
        id: 11,
        path: '/roms/demo.vhdx',
        label: 'Demo',
        notes: 'keep me',
      },
    ]

    const workflow = useEditGameWorkflow(options)
    await workflow.handleSubmit()

    expect(updateGameAggregateMock).toHaveBeenCalledWith('game-1', expect.objectContaining({
      assets: expect.objectContaining({
        files: [
          expect.objectContaining({
            id: 11,
            file_path: '/roms/demo.vhdx',
            label: 'Demo',
            notes: 'keep me',
          }),
        ],
      }),
    }))
  })

  it('reuses matching series option instead of creating a duplicate', async () => {
    const { options } = buildOptions()
    options.form.value.series_id = ' 孤岛惊魂 '
    options.seriesOptions.value = [{
      id: 4,
      name: '孤岛惊魂',
      slug: 'far-cry',
      sort_order: 0,
      created_at: '2026-03-25T00:00:00Z',
    }]

    const workflow = useEditGameWorkflow(options)
    await workflow.handleSubmit()

    expect(createSeriesMock).not.toHaveBeenCalled()
    expect(updateGameAggregateMock).toHaveBeenCalledWith('game-1', expect.objectContaining({
      game: expect.objectContaining({
        series_id: 4,
      }),
    }))
    expect(options.form.value.series_id).toBe(4)
  })

  it('normalizes blank optional fields before aggregate submit', async () => {
    const { options } = buildOptions()
    options.form.value.title_alt = '   '
    options.form.value.summary = '  '

    const workflow = useEditGameWorkflow(options)
    await workflow.handleSubmit()

    expect(updateGameAggregateMock).toHaveBeenCalledWith('game-1', expect.objectContaining({
      game: expect.objectContaining({
        title_alt: null,
        summary: null,
      }),
    }))
  })

  it('warns when series picker refresh fails after a successful save', async () => {
    const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
    const { options, addAlert, emitSuccess, closeModal } = buildOptions()
    getPopularSeriesMock.mockRejectedValueOnce(new Error('series refresh failed'))

    const workflow = useEditGameWorkflow(options)
    await workflow.handleSubmit()

    expect(updateGameAggregateMock).toHaveBeenCalledTimes(1)
    expect(addAlert).toHaveBeenCalledWith('保存成功', 'success')
    expect(addAlert).toHaveBeenCalledWith('保存已生效，但系列选项刷新失败，请稍后重试', 'warning')
    expect(emitSuccess).toHaveBeenCalledTimes(1)
    expect(closeModal).toHaveBeenCalledTimes(1)
    consoleErrorSpy.mockRestore()
  })

})
