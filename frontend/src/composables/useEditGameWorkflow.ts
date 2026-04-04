import { ref, type Ref } from 'vue'
import type { EditGameForm } from '@/composables/edit-game-form'
import gamesService from '@/services/games.service'
import { seriesService } from '@/services/series.service'
import platformService from '@/services/platforms.service'
import { developersService } from '@/services/developers.service'
import { publishersService } from '@/services/publishers.service'
import { resolveCreatableSelections } from '@/utils/creatable-select'
import { getHttpErrorMessage } from '@/utils/http-error'
import type {
  Developer,
  GameDetail,
  GameAggregateGameUpdateRequest,
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

interface UseEditGameWorkflowOptions {
  game: Ref<GameDetail | null>
  form: Ref<EditGameForm>
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

const slugifyMetadataName = (name: string) => {
  return name
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
}

const createWorkflowStepError = (message: string, cause: unknown) => {
  const error = new Error(message) as Error & { cause?: unknown }
  error.cause = cause
  return error
}

const toNullableFormText = (value: string | null | undefined) => {
  if (typeof value !== 'string') {
    return value ?? null
  }
  return value.trim() ? value : null
}

const resolveSeriesSelection = async (
  seriesValue: string | number | null,
) => {
  let seriesIds: number[] | undefined

  if (seriesValue === null || seriesValue === undefined || seriesValue === '') {
    seriesIds = []
  } else if (typeof seriesValue === 'number') {
    seriesIds = [seriesValue]
  } else if (typeof seriesValue === 'string' && seriesValue.trim()) {
    const normalizedValue = seriesValue.trim()
    const maybeId = Number(normalizedValue)
    if (!Number.isNaN(maybeId) && normalizedValue === String(maybeId)) {
      seriesIds = [maybeId]
    } else {
      try {
        // 2026-04-04: keep series creation inside the edit flow because game editing is a real
        // metadata authoring entry point in this project, not a deprecated compatibility path.
        const seriesName = normalizedValue
        const newSeries = await seriesService.createSeries({
          name: seriesName,
          slug: seriesName.toLowerCase().replace(/[^a-z0-9]+/g, '-'),
        })
        seriesIds = [newSeries.id]
      } catch (error) {
        console.error('Failed to process series:', seriesValue, error)
        throw createWorkflowStepError(`系列 "${seriesValue}" 处理失败`, error)
      }
    }
  }

  return seriesIds
}

const resolveDevelopers = async (
  values: Array<string | number>,
  options: Developer[],
) => {
  try {
    // 2026-04-04: developers can still be created from the edit form by product design.
    // Impact: form selections may contain names until submit resolves them into persistent ids.
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
    throw createWorkflowStepError('开发商处理失败', error)
  }
}

const resolvePublishers = async (
  values: Array<string | number>,
  options: Publisher[],
) => {
  try {
    // 2026-04-04: publishers share the same authoring flow as developers and platforms.
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
    throw createWorkflowStepError('发行商处理失败', error)
  }
}

const resolvePlatforms = async (
  values: Array<string | number>,
  options: Platform[],
) => {
  try {
    // 2026-04-04: platform creation stays here so game editing can commit a complete metadata set in one submit.
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
    throw createWorkflowStepError('平台处理失败', error)
  }
}

const createUpdatePayload = (params: {
  form: EditGameForm
  platformIds: number[]
  seriesId: number | null | undefined
  developerIds: number[]
  publisherIds: number[]
  tagIds: number[]
}): GameAggregateGameUpdateRequest => {
  return {
    title: params.form.title,
    title_alt: toNullableFormText(params.form.title_alt),
    visibility: params.form.visibility,
    release_date: params.form.release_date || undefined,
    engine: toNullableFormText(params.form.engine),
    platform_ids: params.platformIds,
    series_id: params.seriesId ?? null,
    developer_ids: params.developerIds,
    publisher_ids: params.publisherIds,
    tag_ids: params.tagIds,
    summary: toNullableFormText(params.form.summary),
    cover_image: toNullableFormText(params.form.cover_image),
    banner_image: toNullableFormText(params.form.banner_image),
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
      const seriesIds = await resolveSeriesSelection(options.form.value.series_id)
      const seriesId = seriesIds?.[0] ?? null

      const developerResult = await resolveDevelopers(
        options.form.value.developer_ids,
        options.developerOptions.value,
      )
      options.developerOptions.value = developerResult.options
      const developerIds = developerResult.ids
      options.form.value.developer_ids = [...developerIds]

      const publisherResult = await resolvePublishers(
        options.form.value.publisher_ids,
        options.publisherOptions.value,
      )
      options.publisherOptions.value = publisherResult.options
      const publisherIds = publisherResult.ids
      options.form.value.publisher_ids = [...publisherIds]

      const platformResult = await resolvePlatforms(
        options.form.value.platform_ids,
        options.platformOptions.value,
      )
      options.platformOptions.value = platformResult.options
      const platformIds = platformResult.ids
      options.form.value.platform_ids = [...platformIds]

      let tagIds: number[] = []
      try {
        tagIds = await options.resolveTagSelections()
        options.form.value.tag_ids = [...tagIds]
      } catch (error) {
        console.error('Failed to process tags:', options.form.value.tag_ids, error)
        throw createWorkflowStepError('标签处理失败', error)
      }

      const orderedScreenshotUids = options.form.value.screenshots
        .map((item) => item.asset_uid)
        .filter((assetUid): assetUid is string => Boolean(assetUid))
      const orderedVideoUids = options.form.value.preview_videos
        .map((item) => item.asset_uid)
        .filter((assetUid): assetUid is string => Boolean(assetUid))
      if (!game.public_id) {
        throw new Error('missing game public_id')
      }
      const aggregateResult = await gamesService.updateGameAggregate(game.public_id, {
        game: createUpdatePayload({
          form: options.form.value,
          platformIds,
          seriesId,
          developerIds,
          publisherIds,
          tagIds,
        }),
        assets: {
          files: options.form.value.file_paths
            .filter((item) => item.path.trim())
            .map((item) => ({
              id: item.id,
              file_path: item.path.trim(),
              label: item.label.trim() || null,
              // The edit modal does not expose file notes yet, so preserve existing values.
              notes: item.notes ?? null,
            })),
          delete_assets: pendingDeleteAssets.value.map((item) => ({
            asset_type: item.type,
            path: item.path,
            asset_id: item.assetId,
            asset_uid: item.assetUid,
          })),
          screenshot_order_asset_uids: orderedScreenshotUids,
          video_order_asset_uids: orderedVideoUids,
        },
      })
      pendingDeleteAssets.value = []
      if (aggregateResult.warnings.length > 0) {
        options.addAlert('部分素材文件未能物理删除，系统稍后可重试', 'warning')
      }
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
