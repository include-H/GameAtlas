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
import { developersService } from '@/services/developers.service'
import { publishersService } from '@/services/publishers.service'
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
  addAlert: (message: string, type: 'success' | 'warning' | 'error') => void
  resetTagSelectionState: () => void
  createEditableScreenshot: (asset: ScreenshotItem | string, index: number) => EditGameEditableScreenshot
  createEditableVideo: (asset: VideoAssetItem | string) => EditGameEditableVideo
}

const hasResolvedFilePath = (item: GameFileEntry) => {
  return typeof item.file_path === 'string' && item.file_path.trim().length > 0
}

export const useEditGameFormBootstrap = (options: UseEditGameFormBootstrapOptions) => {
  const handleInitializeOptionsError = (context: string, error: unknown) => {
    console.error(`Failed to load ${context}:`, error)
    options.addAlert(`加载编辑元数据失败：${context}`, 'error')
  }

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
      handleInitializeOptionsError('系列', error)
    }

    try {
      const initialDevelopers = await developersService.listDevelopers({ limit: 50 })
      options.developerOptions.value = initialDevelopers
      if (currentGame?.developers.length) {
        for (const developer of currentGame.developers) {
          const existing = options.developerOptions.value.find((item) => item.id === developer.id)
          if (!existing) {
            options.developerOptions.value.push(developer)
          }
        }
      }
    } catch (error) {
      handleInitializeOptionsError('开发商', error)
    }

    try {
      const initialPublishers = await publishersService.listPublishers({ limit: 50 })
      options.publisherOptions.value = initialPublishers
      if (currentGame?.publishers.length) {
        for (const publisher of currentGame.publishers) {
          const existing = options.publisherOptions.value.find((item) => item.id === publisher.id)
          if (!existing) {
            options.publisherOptions.value.push(publisher)
          }
        }
      }
    } catch (error) {
      handleInitializeOptionsError('发行商', error)
    }

    try {
      const allPlatforms = await platformService.listPlatforms()
      options.platformOptions.value = allPlatforms
    } catch (error) {
      handleInitializeOptionsError('平台', error)
    }

    try {
      const [loadedGroups, loadedTags] = await Promise.all([
        tagsService.getTagGroups(),
        tagsService.getTags({ active: true }),
      ])
      options.tagGroups.value = loadedGroups.sort((a, b) => a.sort_order - b.sort_order || a.id - b.id)
      options.tagOptions.value = loadedTags
    } catch (error) {
      handleInitializeOptionsError('标签', error)
    }
  }

  return {
    createEmptyForm: createEmptyEditGameForm,
    hydrateFormFromGame,
    initializeOptions,
  }
}
