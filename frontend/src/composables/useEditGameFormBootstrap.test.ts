import { ref } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import type { GameDetail } from '@/services/types'
import { useEditGameFormBootstrap } from './useEditGameFormBootstrap'

vi.mock('@/services/series.service', () => ({
  seriesService: {
    getPopularSeries: vi.fn(),
  },
}))

vi.mock('@/services/platforms.service', () => ({
  default: {
    getAllPlatforms: vi.fn(),
  },
}))

vi.mock('@/services/tags.service', () => ({
  default: {
    getTagGroups: vi.fn(),
    getTags: vi.fn(),
  },
}))

describe('useEditGameFormBootstrap', () => {
  it('hydrates preview videos without storing a separate primary uid', () => {
    const form = ref({
      title: '',
      title_alt: '',
      visibility: 'public' as const,
      developer_ids: [] as Array<string | number>,
      publisher_ids: [] as Array<string | number>,
      release_date: undefined as string | undefined,
      engine: '',
      platform_ids: [] as Array<string | number>,
      series_id: null as string | number | null,
      tag_ids: [] as Array<string | number>,
      summary: '',
      cover_image: '',
      banner_image: '',
      preview_videos: [] as Array<{ asset_uid?: string; path: string; sort_order?: number }>,
      screenshots: [] as Array<{ client_key: string; path: string; sort_order?: number }>,
      file_paths: [{ path: '', label: '' }],
    })

    const { hydrateFormFromGame } = useEditGameFormBootstrap({
      form,
      seriesOptions: ref([]),
      platformOptions: ref([]),
      tagGroups: ref([]),
      tagOptions: ref([]),
      developerOptions: ref([]),
      publisherOptions: ref([]),
      resetTagSelectionState: vi.fn(),
      createEditableScreenshot: (asset, index) => ({
        path: typeof asset === 'string' ? asset : asset.path,
        client_key: `screenshot-${index}`,
        sort_order: index,
      }),
      createEditableVideo: (asset) => ({
        asset_uid: typeof asset === 'string' ? undefined : asset.asset_uid,
        path: typeof asset === 'string' ? asset : asset.path,
        sort_order: typeof asset === 'string' ? undefined : asset.sort_order,
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
      engine: null,
      cover_image: null,
      banner_image: null,
      wiki_content: null,
      wiki_content_html: null,
      needs_review: false,
      downloads: 0,
      preview_video: {
        id: 2,
        asset_uid: 'video-primary',
        path: '/assets/video-primary.mp4',
        sort_order: 9,
      },
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
      platforms: [],
      developers: [],
      publishers: [],
      tags: [],
      tag_groups: [],
      files: [],
      created_at: '2026-03-25T00:00:00Z',
      updated_at: '2026-03-25T00:00:00Z',
      isFavorite: false,
    } as GameDetail)

    expect(form.value.preview_videos.map((item) => item.asset_uid)).toEqual(['video-first', 'video-primary'])
  })
})
