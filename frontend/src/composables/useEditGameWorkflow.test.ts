import { ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { EditGameForm } from '@/utils/edit-game-form'
import type { AdminGameDetail } from '@/services/types'
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
    updateGameAggregateMock.mockResolvedValue({
      game: {
        id: 1,
        public_id: 'game-1',
      },
      warnings: [],
    })
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

  it('submits the already-resolved metadata ids without another metadata request', async () => {
    const { options } = buildOptions()
    options.form.value.series_id = 4
    options.form.value.developer_ids = [1, 3]
    options.form.value.publisher_ids = [2, 5]

    const workflow = useEditGameWorkflow(options)
    await workflow.handleSubmit()

    expect(updateGameAggregateMock).toHaveBeenCalledWith('game-1', expect.objectContaining({
      game: expect.objectContaining({
        series_id: 4,
        developer_ids: [1, 3],
        publisher_ids: [2, 5],
      }),
    }))
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

  it('keeps a successful aggregate save successful without a metadata refresh step', async () => {
    const { options, addAlert, emitSuccess, closeModal } = buildOptions()

    const workflow = useEditGameWorkflow(options)
    await workflow.handleSubmit()

    expect(updateGameAggregateMock).toHaveBeenCalledTimes(1)
    expect(addAlert).toHaveBeenCalledWith('保存成功', 'success')
    expect(emitSuccess).toHaveBeenCalledTimes(1)
    expect(closeModal).toHaveBeenCalledTimes(1)
  })

})
