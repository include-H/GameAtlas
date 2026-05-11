import { type Ref } from 'vue'
import type { EditGameForm } from '@/composables/edit-game-form'
import gamesService from '@/services/games.service'
import { seriesService } from '@/services/series.service'
import { developersService } from '@/services/developers.service'
import { publishersService } from '@/services/publishers.service'
import { resolveCreatableSelections } from '@/utils/creatable-select'
import { getHttpErrorMessage } from '@/utils/http-error'
import type {
  AdminGameDetail,
  Developer,
  GameAggregateGameUpdateRequest,
  GameAggregateNewAsset,
  Publisher,
  Series,
} from '@/services/types'

interface UseEditGameWorkflowOptions {
  game: Ref<AdminGameDetail | null>
  form: Ref<EditGameForm>
  isSubmitting: Ref<boolean>
  seriesOptions: Ref<Series[]>
  developerOptions: Ref<Developer[]>
  publisherOptions: Ref<Publisher[]>
  validateForm: () => Promise<boolean>
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
    // 2026-04-04: publishers share the same authoring flow as developers.
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

const createUpdatePayload = (params: {
  form: EditGameForm
  seriesId: number | null | undefined
  developerIds: number[]
  publisherIds: number[]
}): GameAggregateGameUpdateRequest => {
  return {
    title: params.form.title,
    title_alt: toNullableFormText(params.form.title_alt),
    visibility: params.form.visibility,
    release_date: toNullableFormText(params.form.release_date),
    series_id: params.seriesId ?? null,
    developer_ids: params.developerIds,
    publisher_ids: params.publisherIds,
    summary: toNullableFormText(params.form.summary),
    cover_image: null,
    banner_image: null,
    logo_visible: params.form.logo_visible,
  }
}

export const useEditGameWorkflow = (options: UseEditGameWorkflowOptions) => {
  const refreshSeriesOptions = async () => {
    try {
      const popularSeries = await seriesService.getPopularSeries(50)
      options.seriesOptions.value = popularSeries
      return true
    } catch (error) {
      console.error('Failed to refresh series:', error)
      return false
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

      const orderedScreenshotUids = options.form.value.screenshots
        .map((item) => item.asset_uid)
        .filter((assetUid): assetUid is string => Boolean(assetUid))
      const orderedVideoUids = options.form.value.preview_videos
        .map((item) => item.asset_uid)
        .filter((assetUid): assetUid is string => Boolean(assetUid))
      const orderedCoverUids = options.form.value.covers
        .map((item) => item.asset_uid)
        .filter((assetUid): assetUid is string => Boolean(assetUid))
      const orderedBannerUids = options.form.value.banners
        .map((item) => item.asset_uid)
        .filter((assetUid): assetUid is string => Boolean(assetUid))
      const logo = options.form.value.logo
      const orderedLogoUids = logo?.asset_uid ? [logo.asset_uid] : []
      const logoPositions = logo?.asset_uid ? [{
        asset_uid: logo.asset_uid,
        position_x: logo.position_x ?? null,
        position_y: logo.position_y ?? null,
        width_pct: logo.width_pct ?? null,
      }] : []

      const newAssets: GameAggregateNewAsset[] = []
      for (const s of options.form.value.screenshots) {
        if (s.asset_uid && s.path) {
          newAssets.push({ asset_uid: s.asset_uid, asset_type: 'screenshot', path: s.path })
        }
      }
      for (const v of options.form.value.preview_videos) {
        if (v.asset_uid && v.path) {
          newAssets.push({ asset_uid: v.asset_uid, asset_type: 'video', path: v.path })
        }
      }
      for (const c of options.form.value.covers) {
        if (c.asset_uid && c.path) {
          newAssets.push({ asset_uid: c.asset_uid, asset_type: 'cover', path: c.path })
        }
      }
      for (const b of options.form.value.banners) {
        if (b.asset_uid && b.path) {
          newAssets.push({ asset_uid: b.asset_uid, asset_type: 'banner', path: b.path })
        }
      }
      if (logo?.asset_uid && logo.path) {
        newAssets.push({ asset_uid: logo.asset_uid, asset_type: 'logo', path: logo.path })
      }

      if (!game.public_id) {
        throw new Error('missing game public_id')
      }
      const aggregateResult = await gamesService.updateGameAggregate(game.public_id, {
        game: createUpdatePayload({
          form: options.form.value,
          seriesId,
          developerIds,
          publisherIds,
        }),
        assets: {
          files: options.form.value.file_paths
            .filter((item) => item.path.trim())
            .map((item) => ({
              id: item.id,
              file_path: item.path.trim(),
              label: item.label.trim() || null,
              notes: item.notes ?? null,
            })),
          new_assets: newAssets,
          screenshot_order_asset_uids: orderedScreenshotUids,
          video_order_asset_uids: orderedVideoUids,
          cover_order_asset_uids: orderedCoverUids,
          logo_order_asset_uids: orderedLogoUids,
          banner_order_asset_uids: orderedBannerUids,
          logo_positions: logoPositions,
        },
      })
      if (aggregateResult.warnings.length > 0) {
        options.addAlert('部分素材文件未能物理删除，系统稍后可重试', 'warning')
      }
      const seriesRefreshSucceeded = await refreshSeriesOptions()

      options.addAlert('保存成功', 'success')
      if (!seriesRefreshSucceeded) {
        // 2026-04-08: aggregate save success does not guarantee local picker metadata refreshed.
        // Impact: keep the save successful, but surface that the post-save series lookup stayed stale.
        options.addAlert('保存已生效，但系列选项刷新失败，请稍后重试', 'warning')
      }
      options.emitSuccess()
      options.closeModal()
    } catch (error) {
      options.addAlert(getHttpErrorMessage(error, '保存失败'), 'error')
    } finally {
      options.isSubmitting.value = false
    }
  }

  return {
    handleSubmit,
  }
}
