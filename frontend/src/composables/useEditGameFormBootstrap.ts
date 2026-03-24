import { type Ref } from 'vue'
import { seriesService } from '@/services/series.service'
import platformService from '@/services/platforms.service'
import tagsService from '@/services/tags.service'
import type {
  Developer,
  Game,
  Platform,
  Publisher,
  ScreenshotItem,
  Series,
  Tag,
  TagGroup,
  VideoAssetItem,
} from '@/services/types'

interface BootstrapFilePathItem {
  id?: number
  path: string
  label: string
}

interface BootstrapEditableScreenshot {
  id?: number
  asset_uid?: string
  path: string
  sort_order?: number
  client_key: string
}

interface BootstrapEditableVideo {
  id?: number
  asset_uid?: string
  path: string
  sort_order?: number
}

interface BootstrapGameForm {
  title: string
  title_alt: string
  visibility: 'public' | 'private'
  developers: Array<string | number>
  publishers: Array<string | number>
  release_date: string | undefined
  engine: string
  platform: Array<string | number>
  series: string | number | null
  tag_ids: Array<string | number>
  summary: string
  cover_image: string
  banner_image: string
  preview_videos: BootstrapEditableVideo[]
  primary_preview_video_uid: string
  screenshots: BootstrapEditableScreenshot[]
  file_paths: BootstrapFilePathItem[]
}

interface UseEditGameFormBootstrapOptions {
  form: Ref<BootstrapGameForm>
  releaseDate: Ref<Date | null>
  seriesOptions: Ref<Series[]>
  platformOptions: Ref<Platform[]>
  tagGroups: Ref<TagGroup[]>
  tagOptions: Ref<Tag[]>
  developerOptions: Ref<Developer[]>
  publisherOptions: Ref<Publisher[]>
  resetTagSelectionState: () => void
  createEditableScreenshot: (asset: ScreenshotItem | string, index: number) => BootstrapEditableScreenshot
  createEditableVideo: (asset: VideoAssetItem | string) => BootstrapEditableVideo
}

type GameFilePathEntry = NonNullable<Game['file_paths']>[number]

const isGameFilePathObject = (
  item: GameFilePathEntry,
): item is Exclude<GameFilePathEntry, string> => {
  return typeof item === 'object' && item !== null
}

const createEmptyForm = (): BootstrapGameForm => ({
  title: '',
  title_alt: '',
  visibility: 'public',
  developers: [],
  publishers: [],
  release_date: undefined,
  engine: '',
  platform: [],
  series: null,
  tag_ids: [],
  summary: '',
  cover_image: '',
  banner_image: '',
  preview_videos: [],
  primary_preview_video_uid: '',
  screenshots: [],
  file_paths: [{ path: '', label: '' }],
})

export const useEditGameFormBootstrap = (options: UseEditGameFormBootstrapOptions) => {
  const hydrateFormFromGame = (game: Game | null) => {
    if (!game) {
      options.form.value = createEmptyForm()
      options.releaseDate.value = null
      return
    }

    let platformList: Array<string | number> = []
    if (game.platforms && game.platforms.length > 0) {
      platformList = game.platforms.map((name) => {
        const match = options.platformOptions.value.find((item) => item.name === name)
        return match ? match.id : name
      })
    } else if (game.platform) {
      const match = options.platformOptions.value.find((item) => item.name === game.platform)
      platformList = [match ? match.id : game.platform]
    }

    let filePaths: BootstrapFilePathItem[] = [{ path: '', label: '' }]
    if (game.file_paths && game.file_paths.length > 0) {
      filePaths = game.file_paths.map((item) => {
        if (typeof item === 'string') {
          return { path: item, label: '' }
        }

        if (isGameFilePathObject(item)) {
          return { id: item.id, path: item.path || '', label: item.label || '' }
        }

        return { path: '', label: '' }
      })
    } else if (game.file_path) {
      filePaths = [{ path: game.file_path, label: '' }]
    }

    const seriesId: string | number | null = game.series && game.series.length > 0
      ? game.series[0].id
      : null
    const developerIds: Array<string | number> = game.developers
      ? game.developers.map((item) => item.id)
      : []
    const publisherIds: Array<string | number> = game.publishers
      ? game.publishers.map((item) => item.id)
      : []

    options.form.value = {
      title: game.title || '',
      title_alt: game.title_alt || '',
      visibility: game.visibility || 'public',
      developers: developerIds,
      publishers: publisherIds,
      release_date: game.release_date || undefined,
      engine: game.engine || '',
      platform: platformList,
      series: seriesId,
      tag_ids: (game.tags || []).map((item) => item.id),
      summary: game.summary || '',
      cover_image: game.cover_image || '',
      banner_image: game.banner_image || '',
      preview_videos: (game.preview_videos || (game.preview_video ? [game.preview_video] : [])).map((asset) =>
        options.createEditableVideo(asset),
      ),
      primary_preview_video_uid: game.preview_video?.asset_uid || game.preview_videos?.[0]?.asset_uid || '',
      screenshots: (game.screenshot_items || game.screenshots || []).map((asset, index) =>
        options.createEditableScreenshot(asset, index),
      ),
      file_paths: filePaths,
    }
    options.resetTagSelectionState()

    if (game.release_date) {
      const parts = game.release_date.split('-')
      if (parts.length === 3) {
        options.releaseDate.value = new Date(
          Number.parseInt(parts[0], 10),
          Number.parseInt(parts[1], 10) - 1,
          Number.parseInt(parts[2], 10),
        )
      } else {
        options.releaseDate.value = new Date(game.release_date)
      }
    } else {
      options.releaseDate.value = null
    }
  }

  const initializeOptions = async (currentGame?: Game | null) => {
    try {
      const popularSeries = await seriesService.getPopularSeries(50)
      options.seriesOptions.value = popularSeries
      const currentSeries = currentGame?.series?.[0]
      if (currentSeries) {
        const existing = popularSeries.find((item) => item.id === currentSeries.id)
        if (!existing) {
          options.seriesOptions.value.push(currentSeries)
        }
      }
    } catch (error) {
      console.error('Failed to load series:', error)
    }

    try {
      const { developersService } = await import('@/services/developers.service')
      const popularDevelopers = await developersService.getPopularDevelopers(50)
      options.developerOptions.value = popularDevelopers
      if (currentGame?.developers && currentGame.developers.length > 0) {
        for (const developer of currentGame.developers) {
          const existing = options.developerOptions.value.find((item) => item.id === developer.id)
          if (!existing) {
            options.developerOptions.value.push(developer)
          }
        }
      }
    } catch (error) {
      console.error('Failed to load developers:', error)
    }

    try {
      const { publishersService } = await import('@/services/publishers.service')
      const popularPublishers = await publishersService.getPopularPublishers(50)
      options.publisherOptions.value = popularPublishers
      if (currentGame?.publishers && currentGame.publishers.length > 0) {
        for (const publisher of currentGame.publishers) {
          const existing = options.publisherOptions.value.find((item) => item.id === publisher.id)
          if (!existing) {
            options.publisherOptions.value.push(publisher)
          }
        }
      }
    } catch (error) {
      console.error('Failed to load publishers:', error)
    }

    try {
      const allPlatforms = await platformService.getAllPlatforms()
      options.platformOptions.value = allPlatforms
    } catch (error) {
      console.error('Failed to load platforms:', error)
    }

    try {
      const [loadedGroups, loadedTags] = await Promise.all([
        tagsService.getTagGroups(),
        tagsService.getTags({ active: true }),
      ])
      options.tagGroups.value = loadedGroups.sort((a, b) => a.sort_order - b.sort_order || a.id - b.id)
      options.tagOptions.value = loadedTags
    } catch (error) {
      console.error('Failed to load tags:', error)
    }
  }

  return {
    createEmptyForm,
    hydrateFormFromGame,
    initializeOptions,
  }
}
