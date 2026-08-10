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
        save_path_template: '',
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
        save_path_template: '',
        banners: [],
        covers: [],
        logos: [],
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

  it('sends only newly uploaded assets in new_assets, keeping existing ones in order arrays', async () => {
    const { options } = buildOptions()
    options.form.value.screenshots = [
      { id: 101, asset_uid: 'shot-existing', path: '/assets/game-1/shot-existing.jpg', client_key: 'k1' },
      { asset_uid: 'shot-new', path: '/assets/game-1/shot-new.jpg', client_key: 'k2' },
    ]
    options.form.value.preview_videos = [
      { id: 401, asset_uid: 'video-existing', path: '/assets/game-1/video-existing.mp4', poster_path: null },
    ]
    options.form.value.covers = [
      { id: 201, asset_uid: 'cover-existing', path: '/assets/game-1/cover-existing.jpg' },
    ]
    options.form.value.banners = [
      { asset_uid: 'banner-new', path: '/assets/game-1/banner-new.jpg' },
    ]
    options.form.value.logos = [
      { id: 301, asset_uid: 'logo-existing', path: '/assets/game-1/logo-existing.png', position_x: null, position_y: null, width_pct: null },
    ]

    const workflow = useEditGameWorkflow(options)
    await workflow.handleSubmit()

    expect(updateGameAggregateMock).toHaveBeenCalledWith('game-1', expect.objectContaining({
      assets: expect.objectContaining({
        new_assets: [
          { asset_uid: 'shot-new', asset_type: 'screenshot', path: '/assets/game-1/shot-new.jpg' },
          { asset_uid: 'banner-new', asset_type: 'banner', path: '/assets/game-1/banner-new.jpg' },
        ],
        screenshot_order_asset_uids: ['shot-existing', 'shot-new'],
        video_order_asset_uids: ['video-existing'],
        cover_order_asset_uids: ['cover-existing'],
        logo_order_asset_uids: ['logo-existing'],
        banner_order_asset_uids: ['banner-new'],
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

  it('does not close the editor when the save is invalidated during a game switch', async () => {
    let resolveSave: (result: { game: { id: number; public_id: string }; warnings: string[] }) => void = () => {}
    updateGameAggregateMock.mockImplementationOnce(() => new Promise((resolve) => {
      resolveSave = resolve
    }))
    const { options, addAlert, emitSuccess, closeModal } = buildOptions()
    const workflow = useEditGameWorkflow(options)

    const submitPromise = workflow.handleSubmit()
    await vi.waitFor(() => expect(updateGameAggregateMock).toHaveBeenCalledTimes(1))

    workflow.invalidateSave()
    resolveSave({ game: { id: 1, public_id: 'game-1' }, warnings: [] })
    await submitPromise

    expect(addAlert).not.toHaveBeenCalledWith('保存成功', 'success')
    expect(emitSuccess).not.toHaveBeenCalled()
    expect(closeModal).not.toHaveBeenCalled()
    expect(options.isSubmitting.value).toBe(false)
  })

})
