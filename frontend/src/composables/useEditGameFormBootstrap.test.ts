import { ref } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import type { EditGameForm } from '@/composables/edit-game-form'
import type { AdminGameDetail, Series } from '@/services/types'
import { useEditGameFormBootstrap } from './useEditGameFormBootstrap'
import { seriesService } from '@/services/series.service'
import { developersService } from '@/services/developers.service'
import { publishersService } from '@/services/publishers.service'

vi.mock('@/services/series.service', () => ({
  seriesService: {
    getPopularSeries: vi.fn(),
  },
}))

vi.mock('@/services/developers.service', () => ({
  developersService: {
    listDevelopers: vi.fn(),
  },
}))

vi.mock('@/services/publishers.service', () => ({
  publishersService: {
    listPublishers: vi.fn(),
  },
}))

describe('useEditGameFormBootstrap', () => {
  it('hydrates preview videos without storing a separate primary uid', () => {
    const form = ref<EditGameForm>({
      title: '',
      title_alt: '',
      visibility: 'public' as const,
      developer_ids: [] as Array<string | number>,
      publisher_ids: [] as Array<string | number>,
      release_date: undefined as string | undefined,
      series_id: null as string | number | null,
      summary: '',
      cover_image: '',
      banner_image: '',
      preview_videos: [] as Array<{ asset_uid?: string; path: string }>,
      screenshots: [] as Array<{ client_key: string; path: string }>,
      file_paths: [{ path: '', label: '' }],
    })

    const { hydrateFormFromGame } = useEditGameFormBootstrap({
      form,
      seriesOptions: ref([]),
      developerOptions: ref([]),
      publisherOptions: ref([]),
      addAlert: vi.fn(),
      createEditableScreenshot: (asset, index) => ({
        path: typeof asset === 'string' ? asset : asset.path,
        client_key: `screenshot-${index}`,
      }),
      createEditableVideo: (asset) => ({
        asset_uid: typeof asset === 'string' ? undefined : asset.asset_uid,
        path: typeof asset === 'string' ? asset : asset.path,
      }),
    })

    hydrateFormFromGame({
      id: 1,
      public_id: 'game-1',
      title: 'Game One',
      title_alt: null,
      visibility: 'public',
      summary: null,
      release_date: null,
      cover_image: null,
      banner_image: null,
      wiki_content: null,
      downloads: 0,
      preview_videos: [
        {
          id: 3,
          asset_uid: 'video-first',
          path: '/assets/video-first.mp4',
          sort_order: 0,
        },
        {
          id: 2,
          asset_uid: 'video-primary',
          path: '/assets/video-primary.mp4',
          sort_order: 9,
        },
      ],
      screenshots: [],
      series: null,
      developers: [],
      publishers: [],
      files: [],
      created_at: '2026-03-25T00:00:00Z',
      updated_at: '2026-03-25T00:00:00Z',
      isFavorite: false,
    } as AdminGameDetail)

    expect(form.value.preview_videos.map((item) => item.asset_uid)).toEqual(['video-first', 'video-primary'])
  })

  it('keeps backend file order when hydrating edit form', () => {
    const form = ref<EditGameForm>({
      title: '',
      title_alt: '',
      visibility: 'public' as const,
      developer_ids: [] as Array<string | number>,
      publisher_ids: [] as Array<string | number>,
      release_date: undefined as string | undefined,
      series_id: null as string | number | null,
      summary: '',
      cover_image: '',
      banner_image: '',
      preview_videos: [] as Array<{ asset_uid?: string; path: string }>,
      screenshots: [] as Array<{ client_key: string; path: string }>,
      file_paths: [{ path: '', label: '' }],
    })

    const { hydrateFormFromGame } = useEditGameFormBootstrap({
      form,
      seriesOptions: ref([]),
      developerOptions: ref([]),
      publisherOptions: ref([]),
      addAlert: vi.fn(),
      createEditableScreenshot: (asset, index) => ({
        path: typeof asset === 'string' ? asset : asset.path,
        client_key: `screenshot-${index}`,
      }),
      createEditableVideo: (asset) => ({
        asset_uid: typeof asset === 'string' ? undefined : asset.asset_uid,
        path: typeof asset === 'string' ? asset : asset.path,
      }),
    })

    hydrateFormFromGame({
      id: 1,
      public_id: 'game-1',
      title: 'Game One',
      title_alt: null,
      visibility: 'public',
      summary: null,
      release_date: null,
      cover_image: null,
      banner_image: null,
      wiki_content: null,
      downloads: 0,
      preview_videos: [],
      screenshots: [],
      series: null,
      developers: [],
      publishers: [],
      files: [
        {
          id: 22,
          game_id: 1,
          file_name: 'Second.vhdx',
          file_path: '/roms/second.vhdx',
          label: 'Second',
          notes: null,
          size_bytes: null,
          sort_order: 9,
          source_created_at: null,
          created_at: '2026-03-25T00:00:00Z',
          updated_at: '2026-03-25T00:00:00Z',
        },
        {
          id: 21,
          game_id: 1,
          file_name: 'First.vhdx',
          file_path: '/roms/first.vhdx',
          label: 'First',
          notes: null,
          size_bytes: null,
          sort_order: 1,
          source_created_at: null,
          created_at: '2026-03-25T00:00:00Z',
          updated_at: '2026-03-25T00:00:00Z',
        },
      ],
      created_at: '2026-03-25T00:00:00Z',
      updated_at: '2026-03-25T00:00:00Z',
      isFavorite: false,
    } as AdminGameDetail)

    expect(form.value.file_paths.map((item) => item.id)).toEqual([22, 21])
  })

  it('shows alerts when edit metadata initialization fails', async () => {
    const addAlert = vi.fn()
    const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
    vi.mocked(seriesService.getPopularSeries).mockRejectedValueOnce(new Error('series failed'))
    vi.mocked(developersService.listDevelopers).mockRejectedValueOnce(new Error('developers failed'))
    vi.mocked(publishersService.listPublishers).mockRejectedValueOnce(new Error('publishers failed'))

    const { initializeOptions } = useEditGameFormBootstrap({
      form: ref({
        title: '',
        title_alt: '',
        visibility: 'public' as const,
        developer_ids: [],
        publisher_ids: [],
        release_date: undefined,
        series_id: null,
        summary: '',
        cover_image: '',
        banner_image: '',
        preview_videos: [],
        screenshots: [],
        file_paths: [{ path: '', label: '' }],
      }),
      seriesOptions: ref([]),
      developerOptions: ref([]),
      publisherOptions: ref([]),
      addAlert,
      createEditableScreenshot: (asset, index) => ({
        path: typeof asset === 'string' ? asset : asset.path,
        client_key: `screenshot-${index}`,
      }),
      createEditableVideo: (asset) => ({
        asset_uid: typeof asset === 'string' ? undefined : asset.asset_uid,
        path: typeof asset === 'string' ? asset : asset.path,
      }),
    })

    await initializeOptions()

    expect(addAlert).toHaveBeenCalledWith('加载编辑元数据失败：系列', 'error')
    expect(addAlert).toHaveBeenCalledWith('加载编辑元数据失败：开发商', 'error')
    expect(addAlert).toHaveBeenCalledWith('加载编辑元数据失败：发行商', 'error')
    consoleErrorSpy.mockRestore()
  })

})
