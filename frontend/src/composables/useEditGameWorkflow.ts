import { ref, type Ref } from 'vue'
import type { EditGameForm } from '@/composables/edit-game-form'
import gamesService from '@/services/games.service'
import { getHttpErrorMessage } from '@/utils/http-error'
import type { GameDetail, GameAggregateGameUpdateRequest } from '@/services/types'

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
  validateForm: () => Promise<boolean>
  resolveTagSelections: () => Promise<number[]>
  addAlert: (message: string, type: 'success' | 'warning' | 'error') => void
  emitSuccess: () => void
  closeModal: () => void
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

  const handleSubmit = async () => {
    const game = options.game.value
    if (!game) return
    if (options.isSubmitting.value) return

    const isValid = await options.validateForm()
    if (!isValid) return

    options.isSubmitting.value = true

    try {
      const seriesId = options.form.value.series_id
      const developerIds = [...options.form.value.developer_ids]
      const publisherIds = [...options.form.value.publisher_ids]
      const platformIds = [...options.form.value.platform_ids]

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
