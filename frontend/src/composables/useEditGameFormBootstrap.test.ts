import { ref } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import type { EditGameForm } from '@/composables/edit-game-form'
import type { AdminGameDetail, Series, Tag, TagGroup } from '@/services/types'
import { useEditGameFormBootstrap } from './useEditGameFormBootstrap'
import { seriesService } from '@/services/series.service'
import tagsService from '@/services/tags.service'
import { developersService } from '@/services/developers.service'
import { publishersService } from '@/services/publishers.service'

vi.mock('@/services/series.service', () => ({
  seriesService: {
    getPopularSeries: vi.fn(),
  },
}))

vi.mock('@/services/tags.service', () => ({
  default: {
    getTagGroups: vi.fn(),
    getTags: vi.fn(),
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
      tag_ids: [] as Array<string | number>,
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
      tagGroups: ref([]),
      tagOptions: ref([]),
      developerOptions: ref([]),
      publisherOptions: ref([]),
      addAlert: vi.fn(),
      resetTagSelectionState: vi.fn(),
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
      tags: [],
      tag_groups: [],
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
      tag_ids: [] as Array<string | number>,
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
      tagGroups: ref([]),
      tagOptions: ref([]),
      developerOptions: ref([]),
      publisherOptions: ref([]),
      addAlert: vi.fn(),
      resetTagSelectionState: vi.fn(),
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
      tags: [],
      tag_groups: [],
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
    vi.mocked(tagsService.getTagGroups).mockRejectedValueOnce(new Error('tags failed'))

    const { initializeOptions } = useEditGameFormBootstrap({
      form: ref({
        title: '',
        title_alt: '',
        visibility: 'public' as const,
        developer_ids: [],
        publisher_ids: [],
        release_date: undefined,
        series_id: null,
        tag_ids: [],
        summary: '',
        cover_image: '',
        banner_image: '',
        preview_videos: [],
        screenshots: [],
        file_paths: [{ path: '', label: '' }],
      }),
      seriesOptions: ref([]),
      tagGroups: ref([]),
      tagOptions: ref([]),
      developerOptions: ref([]),
      publisherOptions: ref([]),
      addAlert,
      resetTagSelectionState: vi.fn(),
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
    expect(addAlert).toHaveBeenCalledWith('加载编辑元数据失败：标签', 'error')
    consoleErrorSpy.mockRestore()
  })

  it('falls back to the current game metadata when initialization fails instead of keeping stale options', async () => {
    const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
    vi.mocked(seriesService.getPopularSeries).mockRejectedValueOnce(new Error('series failed'))
    vi.mocked(developersService.listDevelopers).mockRejectedValueOnce(new Error('developers failed'))
    vi.mocked(publishersService.listPublishers).mockRejectedValueOnce(new Error('publishers failed'))
    vi.mocked(tagsService.getTagGroups).mockRejectedValueOnce(new Error('tags failed'))

    const seriesOptions = ref([{ id: 99, name: 'Old Series', slug: 'old-series' }] as Series[])
    const developerOptions = ref([{
      id: 99,
      name: 'Old Dev',
      slug: 'old-dev',
      sort_order: 99,
      created_at: '2026-03-25T00:00:00Z',
    }])
    const publisherOptions = ref([{
      id: 99,
      name: 'Old Pub',
      slug: 'old-pub',
      sort_order: 99,
      created_at: '2026-03-25T00:00:00Z',
    }])
    const tagGroups = ref([{ id: 99, key: 'old', name: 'Old', sort_order: 99 }] as TagGroup[])
    const tagOptions = ref([{
      id: 99,
      group_id: 99,
      group_key: 'old',
      group_name: 'Old',
      name: 'Old Tag',
      slug: 'old-tag',
      sort_order: 99,
      is_active: true,
      created_at: '2026-03-25T00:00:00Z',
      updated_at: '2026-03-25T00:00:00Z',
    }] as Tag[])

    const { initializeOptions } = useEditGameFormBootstrap({
      form: ref({
        title: '',
        title_alt: '',
        visibility: 'public' as const,
        developer_ids: [],
        publisher_ids: [],
        release_date: undefined,
        series_id: null,
        tag_ids: [],
        summary: '',
        cover_image: '',
        banner_image: '',
        preview_videos: [],
        screenshots: [],
        file_paths: [{ path: '', label: '' }],
      }),
      seriesOptions,
      tagGroups,
      tagOptions,
      developerOptions,
      publisherOptions,
      addAlert: vi.fn(),
      resetTagSelectionState: vi.fn(),
      createEditableScreenshot: (asset, index) => ({
        path: typeof asset === 'string' ? asset : asset.path,
        client_key: `screenshot-${index}`,
      }),
      createEditableVideo: (asset) => ({
        asset_uid: typeof asset === 'string' ? undefined : asset.asset_uid,
        path: typeof asset === 'string' ? asset : asset.path,
      }),
    })

    await initializeOptions({
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
      series: {
        id: 1,
        name: 'Current Series',
        slug: 'current-series',
        created_at: '2026-03-25T00:00:00Z',
      },
      developers: [{
        id: 1,
        name: 'Current Dev',
        slug: 'current-dev',
        sort_order: 1,
        created_at: '2026-03-25T00:00:00Z',
      }],
      publishers: [{
        id: 2,
        name: 'Current Pub',
        slug: 'current-pub',
        sort_order: 1,
        created_at: '2026-03-25T00:00:00Z',
      }],
      tags: [{
        id: 3,
        group_id: 4,
        group_key: 'genre',
        group_name: 'Genre',
        name: 'Current Tag',
        slug: 'current-tag',
        sort_order: 1,
        is_active: true,
        created_at: '2026-03-25T00:00:00Z',
        updated_at: '2026-03-25T00:00:00Z',
      }],
      tag_groups: [{
        id: 4,
        key: 'genre',
        name: 'Genre',
        allow_multiple: false,
        is_filterable: true,
        tags: [],
      }],
      files: [],
      created_at: '2026-03-25T00:00:00Z',
      updated_at: '2026-03-25T00:00:00Z',
      isFavorite: false,
    } as AdminGameDetail)

    expect(seriesOptions.value.map((item) => item.id)).toEqual([1])
    expect(developerOptions.value.map((item) => item.id)).toEqual([1])
    expect(publisherOptions.value.map((item) => item.id)).toEqual([2])
    expect(tagGroups.value.map((item) => item.id)).toEqual([4])
    expect(tagOptions.value.map((item) => item.id)).toEqual([3])
    consoleErrorSpy.mockRestore()
  })

  it('keeps backend tag group order without re-sorting in the client', async () => {
    vi.mocked(seriesService.getPopularSeries).mockResolvedValueOnce([])
    vi.mocked(developersService.listDevelopers).mockResolvedValueOnce([])
    vi.mocked(publishersService.listPublishers).mockResolvedValueOnce([])
    vi.mocked(tagsService.getTagGroups).mockResolvedValueOnce([
      {
        id: 9,
        key: 'theme',
        name: 'Theme',
        sort_order: 9,
      },
      {
        id: 1,
        key: 'genre',
        name: 'Genre',
        sort_order: 1,
      },
    ] as never)
    vi.mocked(tagsService.getTags).mockResolvedValueOnce([])

    const tagGroups = ref([] as TagGroup[])

    const { initializeOptions } = useEditGameFormBootstrap({
      form: ref({
        title: '',
        title_alt: '',
        visibility: 'public' as const,
        developer_ids: [],
        publisher_ids: [],
        release_date: undefined,
        series_id: null,
        tag_ids: [],
        summary: '',
        cover_image: '',
        banner_image: '',
        preview_videos: [],
        screenshots: [],
        file_paths: [{ path: '', label: '' }],
      }),
      seriesOptions: ref([]),
      tagGroups,
      tagOptions: ref([]),
      developerOptions: ref([]),
      publisherOptions: ref([]),
      addAlert: vi.fn(),
      resetTagSelectionState: vi.fn(),
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

    expect(tagGroups.value.map((item) => item.id)).toEqual([9, 1])
  })

  it('preserves current inactive game tags in edit options', async () => {
    vi.mocked(seriesService.getPopularSeries).mockResolvedValueOnce([])
    vi.mocked(developersService.listDevelopers).mockResolvedValueOnce([])
    vi.mocked(publishersService.listPublishers).mockResolvedValueOnce([])
    vi.mocked(tagsService.getTagGroups).mockResolvedValueOnce([])
    vi.mocked(tagsService.getTags).mockResolvedValueOnce([
      {
        id: 2,
        group_id: 1,
        group_key: 'genre',
        group_name: 'Genre',
        name: 'Active Tag',
        slug: 'active-tag',
        sort_order: 1,
        is_active: true,
      },
    ] as never)

    const tagOptions = ref([] as Tag[])

    const { initializeOptions } = useEditGameFormBootstrap({
      form: ref({
        title: '',
        title_alt: '',
        visibility: 'public' as const,
        developer_ids: [],
        publisher_ids: [],
        release_date: undefined,
        series_id: null,
        tag_ids: [],
        summary: '',
        cover_image: '',
        banner_image: '',
        preview_videos: [],
        screenshots: [],
        file_paths: [{ path: '', label: '' }],
      }),
      seriesOptions: ref([]),
      tagGroups: ref([]),
      tagOptions,
      developerOptions: ref([]),
      publisherOptions: ref([]),
      addAlert: vi.fn(),
      resetTagSelectionState: vi.fn(),
      createEditableScreenshot: (asset, index) => ({
        path: typeof asset === 'string' ? asset : asset.path,
        client_key: `screenshot-${index}`,
      }),
      createEditableVideo: (asset) => ({
        asset_uid: typeof asset === 'string' ? undefined : asset.asset_uid,
        path: typeof asset === 'string' ? asset : asset.path,
      }),
    })

    await initializeOptions({
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
      tags: [
        {
          id: 9,
          group_id: 1,
          group_key: 'genre',
          group_name: 'Genre',
          name: 'Inactive Tag',
          slug: 'inactive-tag',
          sort_order: 9,
          is_active: false,
          created_at: '2026-03-25T00:00:00Z',
          updated_at: '2026-03-25T00:00:00Z',
        },
      ],
      tag_groups: [],
      files: [],
      created_at: '2026-03-25T00:00:00Z',
      updated_at: '2026-03-25T00:00:00Z',
      isFavorite: false,
    } as AdminGameDetail)

    expect(tagOptions.value.map((item) => item.id)).toEqual([2, 9])
  })
})
