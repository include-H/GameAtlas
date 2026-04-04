import { ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { EditGameForm } from '@/composables/edit-game-form'
import type { GameDetail } from '@/services/types'
import { useEditGameWorkflow } from './useEditGameWorkflow'

const {
  updateGameAggregateMock,
} = vi.hoisted(() => ({
  updateGameAggregateMock: vi.fn(),
}))

vi.mock('@/services/games.service', () => ({
  default: {
    updateGameAggregate: updateGameAggregateMock,
  },
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

    updateGameAggregateMock.mockResolvedValue({
      game: {
        id: 1,
        public_id: 'game-1',
      },
      warnings: [],
    })
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

  it('normalizes blank optional fields before aggregate submit', async () => {
    const { options } = buildOptions()
    options.form.value.title_alt = '   '
    options.form.value.engine = ''
    options.form.value.summary = '  '
    options.form.value.cover_image = ''
    options.form.value.banner_image = '   '

    const workflow = useEditGameWorkflow(options)
    await workflow.handleSubmit()

    expect(updateGameAggregateMock).toHaveBeenCalledWith('game-1', expect.objectContaining({
      game: expect.objectContaining({
        title_alt: null,
        engine: null,
        summary: null,
        cover_image: null,
        banner_image: null,
      }),
    }))
  })

  it('submits existing metadata ids directly without front-end creation', async () => {
    const { options } = buildOptions()
    options.form.value.series_id = 9
    options.form.value.developer_ids = [7, 8]
    options.form.value.publisher_ids = [5]
    options.form.value.platform_ids = [3, 4]

    const workflow = useEditGameWorkflow(options)
    await workflow.handleSubmit()

    expect(updateGameAggregateMock).toHaveBeenCalledWith('game-1', expect.objectContaining({
      game: expect.objectContaining({
        series_id: 9,
        developer_ids: [7, 8],
        publisher_ids: [5],
        platform_ids: [3, 4],
      }),
    }))
  })
})
