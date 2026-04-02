import { type Ref } from 'vue'
import {
  createEmptyEditGameForm,
  type EditGameEditableScreenshot,
  type EditGameEditableVideo,
  type EditGameForm,
} from '@/composables/edit-game-form'
import { seriesService } from '@/services/series.service'
import platformService from '@/services/platforms.service'
import tagsService from '@/services/tags.service'
import type {
  Developer,
  GameDetail,
  Platform,
  Publisher,
  ScreenshotItem,
  Series,
  Tag,
  TagGroup,
  VideoAssetItem,
  GameFileEntry,
} from '@/services/types'

interface UseEditGameFormBootstrapOptions {
  form: Ref<EditGameForm>
  seriesOptions: Ref<Series[]>
  platformOptions: Ref<Platform[]>
  tagGroups: Ref<TagGroup[]>
  tagOptions: Ref<Tag[]>
  developerOptions: Ref<Developer[]>
  publisherOptions: Ref<Publisher[]>
  resetTagSelectionState: () => void
  createEditableScreenshot: (asset: ScreenshotItem | string, index: number) => EditGameEditableScreenshot
  createEditableVideo: (asset: VideoAssetItem | string) => EditGameEditableVideo
}

const hasResolvedFilePath = (item: GameFileEntry) => {
  return typeof item.file_path === 'string' && item.file_path.trim().length > 0
}

export const useEditGameFormBootstrap = (options: UseEditGameFormBootstrapOptions) => {
  const hydrateFormFromGame = (game: GameDetail | null) => {
    if (!game) {
      options.form.value = createEmptyEditGameForm()
      return
    }

    let filePaths = createEmptyEditGameForm().file_paths
    if (game.files.length > 0) {
      filePaths = game.files
        .filter(hasResolvedFilePath)
        .sort((a, b) => a.sort_order - b.sort_order)
        .map((item) => ({ id: item.id, path: item.file_path || '', label: item.label || '' }))
    }

    options.form.value = {
      title: game.title || '',
      title_alt: game.title_alt || '',
      visibility: game.visibility || 'public',
      developer_ids: game.developers.map((item) => item.id),
      publisher_ids: game.publishers.map((item) => item.id),
      release_date: game.release_date || undefined,
      engine: game.engine || '',
      platform_ids: game.platforms.map((item) => item.id),
      series_id: game.series?.id ?? null,
      tag_ids: game.tags.map((item) => item.id),
      summary: game.summary || '',
      cover_image: game.cover_image || '',
      banner_image: game.banner_image || '',
      preview_videos: game.preview_videos.map((asset) =>
        options.createEditableVideo(asset),
      ),
      screenshots: game.screenshots.map((asset, index) =>
        options.createEditableScreenshot(asset, index),
      ),
      file_paths: filePaths,
    }
    options.resetTagSelectionState()
  }

  const initializeOptions = async (currentGame?: GameDetail | null) => {
    try {
      const popularSeries = await seriesService.getPopularSeries(50)
      options.seriesOptions.value = popularSeries
      const currentSeries = currentGame?.series
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
      if (currentGame?.developers.length) {
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
      if (currentGame?.publishers.length) {
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
    createEmptyForm: createEmptyEditGameForm,
    hydrateFormFromGame,
    initializeOptions,
  }
}
