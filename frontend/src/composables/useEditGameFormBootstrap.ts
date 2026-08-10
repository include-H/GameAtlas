import { type Ref } from 'vue'
import {
  createEmptyEditGameForm,
  type EditGameEditableBanner,
  type EditGameEditableCover,
  type EditGameEditableLogo,
  type EditGameEditableScreenshot,
  type EditGameEditableVideo,
  type EditGameForm,
} from '@/utils/edit-game-form'
import { seriesService } from '@/services/series.service'
import { developersService } from '@/services/developers.service'
import { publishersService } from '@/services/publishers.service'
import type {
  AdminGameDetail,
  BannerItem,
  CoverItem,
  Developer,
  LogoItem,
  Publisher,
  ScreenshotItem,
  Series,
  VideoAssetItem,
} from '@/services/types'
import { createRequestGeneration } from '@/utils/request-generation'

interface UseEditGameFormBootstrapOptions {
  form: Ref<EditGameForm>
  seriesOptions: Ref<Series[]>
  developerOptions: Ref<Developer[]>
  publisherOptions: Ref<Publisher[]>
  addAlert: (message: string, type: 'success' | 'warning' | 'error') => void
  createEditableCover: (asset: CoverItem | string) => EditGameEditableCover
  createEditableBanner: (asset: BannerItem | string) => EditGameEditableBanner
  createEditableLogo: (asset: LogoItem | string) => EditGameEditableLogo
  createEditableScreenshot: (asset: ScreenshotItem | string, index: number) => EditGameEditableScreenshot
  createEditableVideo: (asset: VideoAssetItem | string) => EditGameEditableVideo
}

export const useEditGameFormBootstrap = (options: UseEditGameFormBootstrapOptions) => {
  const optionRequests = createRequestGeneration()

  const handleInitializeOptionsError = (context: string, _error: unknown) => {
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

    const banners = game.banners.map((asset) =>
      options.createEditableBanner(asset),
    )

    options.form.value = {
      title: game.title || '',
      title_alt: game.title_alt || '',
      visibility: game.visibility || 'public',
      developer_ids: game.developers.map((item) => item.id),
      publisher_ids: game.publishers.map((item) => item.id),
      release_date: game.release_date || undefined,
      series_id: game.series?.id ?? null,
      summary: game.summary || '',
      save_path_template: game.save_path_template || '',
      covers: game.covers.map((asset) =>
        options.createEditableCover(asset),
      ),
      banners,
      logos: game.logos.map((asset) =>
        options.createEditableLogo(asset),
      ),
      logo_visible: game.logo_visible ?? true,
      preview_videos: game.preview_videos.map((asset) =>
        options.createEditableVideo(asset),
      ),
      screenshots: game.screenshots.map((asset, index) =>
        options.createEditableScreenshot(asset, index),
      ),
      file_paths: filePaths,
    }
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

  const initializeOptions = async (currentGame?: AdminGameDetail | null): Promise<boolean> => {
    const request = optionRequests.begin()
    resetSeriesOptionsForGame(currentGame)
    resetDeveloperOptionsForGame(currentGame)
    resetPublisherOptionsForGame(currentGame)

    try {
      const popularSeries = await seriesService.getPopularSeries(50)
      if (!request.isCurrent()) return false
      options.seriesOptions.value = popularSeries
      const currentSeries = currentGame?.series
      if (currentSeries) {
        const existing = popularSeries.find((item) => item.id === currentSeries.id)
        if (!existing) {
          options.seriesOptions.value.push(currentSeries)
        }
      }
    } catch (error) {
      if (!request.isCurrent()) return false
      // 2026-04-08: failed edit metadata requests must not leave stale options from another game.
      // Impact: the modal falls back to the current game's native relations instead of old lookup data.
      resetSeriesOptionsForGame(currentGame)
      handleInitializeOptionsError('系列', error)
    }

    try {
      const initialDevelopers = await developersService.listDevelopers({ limit: 50 })
      if (!request.isCurrent()) return false
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
      if (!request.isCurrent()) return false
      resetDeveloperOptionsForGame(currentGame)
      handleInitializeOptionsError('开发商', error)
    }

    try {
      const initialPublishers = await publishersService.listPublishers({ limit: 50 })
      if (!request.isCurrent()) return false
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
      if (!request.isCurrent()) return false
      resetPublisherOptionsForGame(currentGame)
      handleInitializeOptionsError('发行商', error)
    }

    return request.isCurrent()
  }

  return {
    createEmptyForm: createEmptyEditGameForm,
    hydrateFormFromGame,
    initializeOptions,
  }
}
