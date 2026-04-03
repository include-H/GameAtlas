import { ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { EditGameForm } from '@/composables/edit-game-form'
import type { GameDetail } from '@/services/types'
import { useEditGameWorkflow } from './useEditGameWorkflow'

const {
  updateGameAggregateMock,
  createSeriesMock,
  getPopularSeriesMock,
  createPlatformMock,
  resolveCreatableSelectionsMock,
  createDeveloperMock,
  createPublisherMock,
} = vi.hoisted(() => ({
  updateGameAggregateMock: vi.fn(),
  createSeriesMock: vi.fn(),
  getPopularSeriesMock: vi.fn(),
  createPlatformMock: vi.fn(),
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

vi.mock('@/services/platforms.service', () => ({
  default: {
    createPlatform: createPlatformMock,
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
      } as GameDetail),
      form: ref<EditGameForm>({
        title: 'Game One',
        title_alt: '',
        visibility: 'public' as const,
        developer_ids: [1],
        publisher_ids: [2],
        release_date: undefined,
        engine: '',
        platform_ids: [3],
        series_id: null,
        tag_ids: [4],
        summary: '',
        cover_image: '',
        banner_image: '',
        preview_videos: [],
        screenshots: [],
        file_paths: [],
      }),
      isSubmitting: ref(false),
      seriesOptions: ref([]),
      developerOptions: ref([]),
      publisherOptions: ref([]),
      platformOptions: ref([]),
      validateForm: vi.fn().mockResolvedValue(true),
      resolveTagSelections: vi.fn().mockResolvedValue([4]),
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
    createPlatformMock.mockReset()
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

  it('aborts submit when tag resolution fails', async () => {
    const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
    const { options, addAlert, emitSuccess, closeModal } = buildOptions()
    options.resolveTagSelections = vi.fn().mockRejectedValue(new Error('tag boom'))

    const workflow = useEditGameWorkflow(options)
    await workflow.handleSubmit()

    expect(updateGameAggregateMock).not.toHaveBeenCalled()
    expect(addAlert).toHaveBeenCalledWith('标签处理失败', 'error')
    expect(emitSuccess).not.toHaveBeenCalled()
    expect(closeModal).not.toHaveBeenCalled()
    expect(options.isSubmitting.value).toBe(false)
    consoleErrorSpy.mockRestore()
  })
})
