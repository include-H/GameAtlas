import { type Ref } from 'vue'
import {
  createEmptyEditGameForm,
  type EditGameEditableScreenshot,
  type EditGameEditableVideo,
  type EditGameForm,
} from '@/composables/edit-game-form'
import { seriesService } from '@/services/series.service'
import tagsService from '@/services/tags.service'
import { developersService } from '@/services/developers.service'
import { publishersService } from '@/services/publishers.service'
import type {
  AdminGameDetail,
  Developer,
  Publisher,
  ScreenshotItem,
  Series,
  Tag,
  TagGroup,
  VideoAssetItem,
} from '@/services/types'

interface UseEditGameFormBootstrapOptions {
  form: Ref<EditGameForm>
  seriesOptions: Ref<Series[]>
  tagGroups: Ref<TagGroup[]>
  tagOptions: Ref<Tag[]>
  developerOptions: Ref<Developer[]>
  publisherOptions: Ref<Publisher[]>
  addAlert: (message: string, type: 'success' | 'warning' | 'error') => void
  resetTagSelectionState: () => void
  createEditableScreenshot: (asset: ScreenshotItem | string, index: number) => EditGameEditableScreenshot
  createEditableVideo: (asset: VideoAssetItem | string) => EditGameEditableVideo
}

export const useEditGameFormBootstrap = (options: UseEditGameFormBootstrapOptions) => {
  const handleInitializeOptionsError = (context: string, error: unknown) => {
    console.error(`Failed to load ${context}:`, error)
    options.addAlert(`加载编辑元数据失败：${context}`, 'error')
  }

  const hydrateFormFromGame = (game: AdminGameDetail | null) => {
    if (!game) {
      options.form.value = createEmptyEditGameForm()
      return
    }

    let filePaths = createEmptyEditGameForm().file_paths
    if (game.files.length > 0) {
      filePaths = game.files
        .map((item) => ({
          id: item.id,
          path: item.file_path,
          label: item.label || '',
          notes: item.notes || null,
        }))
    }

    options.form.value = {
      title: game.title || '',
      title_alt: game.title_alt || '',
      visibility: game.visibility || 'public',
      developer_ids: game.developers.map((item) => item.id),
      publisher_ids: game.publishers.map((item) => item.id),
      release_date: game.release_date || undefined,
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

  const resetSeriesOptionsForGame = (game?: AdminGameDetail | null) => {
    options.seriesOptions.value = game?.series ? [game.series] : []
  }

  const resetDeveloperOptionsForGame = (game?: AdminGameDetail | null) => {
    options.developerOptions.value = game?.developers ? [...game.developers] : []
  }

  const resetPublisherOptionsForGame = (game?: AdminGameDetail | null) => {
    options.publisherOptions.value = game?.publishers ? [...game.publishers] : []
  }

  const resetTagOptionsForGame = (game?: AdminGameDetail | null) => {
    options.tagGroups.value = game?.tag_groups
      ? game.tag_groups.map((group, index) => ({
        id: group.id,
        key: group.key,
        name: group.name,
        description: null,
        sort_order: index,
        allow_multiple: group.allow_multiple,
        is_filterable: group.is_filterable,
        created_at: '',
        updated_at: '',
      }))
      : []
    options.tagOptions.value = game?.tags ? [...game.tags] : []
  }

  const initializeOptions = async (currentGame?: AdminGameDetail | null) => {
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
      // 2026-04-08: failed edit metadata requests must not leave stale options from another game.
      // Impact: the modal falls back to the current game's native relations instead of old lookup data.
      resetSeriesOptionsForGame(currentGame)
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
      resetDeveloperOptionsForGame(currentGame)
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
      resetPublisherOptionsForGame(currentGame)
      handleInitializeOptionsError('发行商', error)
    }

    try {
      const [loadedGroups, loadedTags] = await Promise.all([
        tagsService.getTagGroups(),
        tagsService.getTags({ active: true }),
      ])
      options.tagGroups.value = loadedGroups
      const currentGameTags = currentGame?.tags || []
      options.tagOptions.value = [...loadedTags]
      for (const tag of currentGameTags) {
        if (!options.tagOptions.value.find((item) => item.id === tag.id)) {
          options.tagOptions.value.push(tag)
        }
      }
    } catch (error) {
      resetTagOptionsForGame(currentGame)
      handleInitializeOptionsError('标签', error)
    }
  }

  return {
    createEmptyForm: createEmptyEditGameForm,
    hydrateFormFromGame,
    initializeOptions,
  }
}
