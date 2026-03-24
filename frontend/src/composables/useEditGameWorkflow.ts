import { ref, type Ref } from 'vue'
import { deleteAsset, reorderScreenshots, reorderVideos } from '@/services/assets'
import gamesService from '@/services/games.service'
import { seriesService } from '@/services/series.service'
import platformService from '@/services/platforms.service'
import { resolveCreatableSelections } from '@/utils/creatable-select'
import { getHttpErrorMessage } from '@/utils/http-error'
import type {
  Developer,
  Game,
  GameInput,
  Platform,
  Publisher,
  Series,
} from '@/services/types'

type AssetType = 'cover' | 'banner' | 'screenshot' | 'video'

interface PendingDeleteAsset {
  type: AssetType
  path: string
  assetId?: number
  assetUid?: string
}

interface WorkflowFilePathItem {
  id?: number
  path: string
  label: string
}

interface WorkflowEditableScreenshot {
  id?: number
  asset_uid?: string
  path: string
  sort_order?: number
  client_key: string
}

interface WorkflowEditableVideo {
  id?: number
  asset_uid?: string
  path: string
  sort_order?: number
}

interface EditGameFormBridge {
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
  preview_videos: WorkflowEditableVideo[]
  primary_preview_video_uid: string
  screenshots: WorkflowEditableScreenshot[]
  file_paths: WorkflowFilePathItem[]
}

interface UseEditGameWorkflowOptions {
  game: Ref<Game | null>
  form: Ref<EditGameFormBridge>
  isSubmitting: Ref<boolean>
  seriesOptions: Ref<Series[]>
  developerOptions: Ref<Developer[]>
  publisherOptions: Ref<Publisher[]>
  platformOptions: Ref<Platform[]>
  validateForm: () => Promise<boolean>
  resolveTagSelections: () => Promise<number[]>
  addAlert: (message: string, type: 'success' | 'warning' | 'error') => void
  emitSuccess: () => void
  closeModal: () => void
}

const hasResolvedFilePath = (item: NonNullable<Game['files']>[number]) => {
  return typeof item.file_path === 'string' && item.file_path.trim().length > 0
}

const slugifyMetadataName = (name: string) => {
  return name
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
}

const persistGameFilePaths = async (
  gameId: number,
  game: Game,
  filePaths: WorkflowFilePathItem[],
) => {
  const originalFileIds = new Set(
    (game.files || [])
      .filter(hasResolvedFilePath)
      .map((item) => item.id)
      .filter((id): id is number => typeof id === 'number'),
  )

  const keptFileIds = new Set<number>()
  const validFilePaths = filePaths.filter((item) => item.path.trim())

  for (let index = 0; index < validFilePaths.length; index += 1) {
    const item = validFilePaths[index]
    const payload = {
      file_path: item.path.trim(),
      label: item.label.trim() || null,
      notes: null,
      sort_order: index,
    }

    if (item.id) {
      keptFileIds.add(item.id)
      await gamesService.updateGameFile(String(gameId), String(item.id), payload)
    } else {
      const created = await gamesService.createGameFile(String(gameId), payload)
      if (created.id) keptFileIds.add(created.id)
    }
  }

  for (const fileId of originalFileIds) {
    if (!keptFileIds.has(fileId)) {
      await gamesService.deleteGameFile(String(gameId), String(fileId))
    }
  }
}

const submitAssetDeletionQueue = async (
  gameId: number,
  pendingDeleteAssets: PendingDeleteAsset[],
) => {
  for (const item of pendingDeleteAssets) {
    try {
      await deleteAsset(gameId, item.type, item.path, item.assetId, item.assetUid)
    } catch (error) {
      console.error('Failed to delete asset after save:', item.path, error)
    }
  }
}

const submitAssetOrder = async (
  gameId: number,
  screenshots: WorkflowEditableScreenshot[],
  previewVideos: WorkflowEditableVideo[],
) => {
  const orderedScreenshotUids = screenshots
    .map((item, index) => {
      item.sort_order = index
      return item.asset_uid
    })
    .filter((assetUid): assetUid is string => Boolean(assetUid))
  if (orderedScreenshotUids.length > 0) {
    await reorderScreenshots(gameId, orderedScreenshotUids)
  }

  const orderedVideoUids = previewVideos
    .map((item, index) => {
      item.sort_order = index
      return item.asset_uid
    })
    .filter((assetUid): assetUid is string => Boolean(assetUid))
  if (orderedVideoUids.length > 0) {
    await reorderVideos(gameId, orderedVideoUids)
  }
}

const resolveSeriesSelection = async (
  seriesValue: string | number | null,
  addAlert: UseEditGameWorkflowOptions['addAlert'],
) => {
  let seriesIds: number[] | undefined

  if (seriesValue === null || seriesValue === undefined || seriesValue === '') {
    seriesIds = []
  } else if (typeof seriesValue === 'number') {
    seriesIds = [seriesValue]
  } else if (typeof seriesValue === 'string' && seriesValue.trim()) {
    try {
      const seriesName = seriesValue.trim()
      const newSeries = await seriesService.createSeries({
        name: seriesName,
        slug: seriesName.toLowerCase().replace(/[^a-z0-9]+/g, '-'),
      })
      seriesIds = [newSeries.id]
    } catch (error) {
      console.error('Failed to process series:', seriesValue, error)
      addAlert(`系列 "${seriesValue}" 处理失败`, 'warning')
    }
  }

  return seriesIds
}

const resolveDevelopers = async (
  values: Array<string | number>,
  options: Developer[],
  addAlert: UseEditGameWorkflowOptions['addAlert'],
) => {
  try {
    const { developersService } = await import('@/services/developers.service')
    const result = await resolveCreatableSelections({
      values,
      options,
      createItem: (name) =>
        developersService.createDeveloper({
          name,
          slug: slugifyMetadataName(name),
        }),
    })
    return result
  } catch (error) {
    console.error('Failed to process developers:', values, error)
    addAlert('开发商处理失败', 'warning')
    return {
      ids: undefined,
      options,
    }
  }
}

const resolvePublishers = async (
  values: Array<string | number>,
  options: Publisher[],
  addAlert: UseEditGameWorkflowOptions['addAlert'],
) => {
  try {
    const { publishersService } = await import('@/services/publishers.service')
    const result = await resolveCreatableSelections({
      values,
      options,
      createItem: (name) =>
        publishersService.createPublisher({
          name,
          slug: slugifyMetadataName(name),
        }),
    })
    return result
  } catch (error) {
    console.error('Failed to process publishers:', values, error)
    addAlert('发行商处理失败', 'warning')
    return {
      ids: undefined,
      options,
    }
  }
}

const resolvePlatforms = async (
  values: Array<string | number>,
  options: Platform[],
  addAlert: UseEditGameWorkflowOptions['addAlert'],
) => {
  try {
    const result = await resolveCreatableSelections({
      values,
      options,
      createItem: (name) =>
        platformService.createPlatform({
          name,
          slug: slugifyMetadataName(name),
        }),
    })
    return result
  } catch (error) {
    console.error('Failed to process platform:', values, error)
    addAlert('平台处理失败', 'warning')
    return {
      ids: undefined,
      options,
    }
  }
}

const createUpdatePayload = (params: {
  form: EditGameFormBridge
  platformIds: number[]
  seriesIds: number[] | undefined
  developerIds: number[] | undefined
  publisherIds: number[] | undefined
  tagIds: number[]
}): Partial<GameInput> => {
  return {
    title: params.form.title,
    title_alt: params.form.title_alt,
    visibility: params.form.visibility,
    release_date: params.form.release_date || undefined,
    engine: params.form.engine,
    platforms: params.platformIds,
    series: params.seriesIds,
    developers: params.developerIds,
    publishers: params.publisherIds,
    tag_ids: params.tagIds,
    summary: params.form.summary,
    cover_image: params.form.cover_image,
    banner_image: params.form.banner_image,
    preview_video_asset_uid: params.form.primary_preview_video_uid || null,
  }
}

export const useEditGameWorkflow = (options: UseEditGameWorkflowOptions) => {
  const pendingDeleteAssets = ref<PendingDeleteAsset[]>([])

  const queueAssetDeletion = (
    type: AssetType,
    path: string,
    assetId?: number,
    assetUid?: string,
  ) => {
    if (!path) return
    pendingDeleteAssets.value.push({ type, path, assetId, assetUid })
  }

  const resetPendingDeleteAssets = () => {
    pendingDeleteAssets.value = []
  }

  const refreshSeriesOptions = async () => {
    try {
      const popularSeries = await seriesService.getPopularSeries(50)
      options.seriesOptions.value = popularSeries
    } catch (error) {
      console.error('Failed to refresh series:', error)
    }
  }

  const handleSubmit = async () => {
    const game = options.game.value
    if (!game) return
    if (options.isSubmitting.value) return

    const isValid = await options.validateForm()
    if (!isValid) return

    options.isSubmitting.value = true

    try {
      const seriesIds = await resolveSeriesSelection(options.form.value.series, options.addAlert)

      const developerResult = await resolveDevelopers(
        options.form.value.developers,
        options.developerOptions.value,
        options.addAlert,
      )
      options.developerOptions.value = developerResult.options
      const developerIds = developerResult.ids
      if (developerIds) {
        options.form.value.developers = [...developerIds]
      }

      const publisherResult = await resolvePublishers(
        options.form.value.publishers,
        options.publisherOptions.value,
        options.addAlert,
      )
      options.publisherOptions.value = publisherResult.options
      const publisherIds = publisherResult.ids
      if (publisherIds) {
        options.form.value.publishers = [...publisherIds]
      }

      const platformResult = await resolvePlatforms(
        options.form.value.platform,
        options.platformOptions.value,
        options.addAlert,
      )
      options.platformOptions.value = platformResult.options
      const platformIds = platformResult.ids || []
      if (platformResult.ids) {
        options.form.value.platform = [...platformIds]
      }

      let tagIds: number[] = []
      try {
        tagIds = await options.resolveTagSelections()
        options.form.value.tag_ids = [...tagIds]
      } catch (error) {
        console.error('Failed to process tags:', options.form.value.tag_ids, error)
        options.addAlert('标签处理失败', 'warning')
      }

      await gamesService.updateGame(
        String(game.id),
        createUpdatePayload({
          form: options.form.value,
          platformIds,
          seriesIds,
          developerIds,
          publisherIds,
          tagIds,
        }),
      )

      await persistGameFilePaths(game.id, game, options.form.value.file_paths)
      await submitAssetDeletionQueue(game.id, pendingDeleteAssets.value)
      pendingDeleteAssets.value = []
      await submitAssetOrder(game.id, options.form.value.screenshots, options.form.value.preview_videos)
      await refreshSeriesOptions()

      options.addAlert('保存成功', 'success')
      options.emitSuccess()
      options.closeModal()
    } catch (error) {
      options.addAlert(getHttpErrorMessage(error, '保存失败'), 'error')
    } finally {
      options.isSubmitting.value = false
    }
  }

  return {
    pendingDeleteAssets,
    queueAssetDeletion,
    resetPendingDeleteAssets,
    handleSubmit,
  }
}
