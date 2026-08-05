import { type Ref } from 'vue'
import type { EditGameForm } from '@/utils/edit-game-form'
import gamesService from '@/services/games.service'
import { getHttpErrorMessage } from '@/utils/http-error'
import type {
  AdminGameDetail,
  GameAggregateGameUpdateRequest,
  GameAggregateNewAsset,
} from '@/services/types'

interface UseEditGameWorkflowOptions {
  game: Ref<AdminGameDetail | null>
  form: Ref<EditGameForm>
  isSubmitting: Ref<boolean>
  validateForm: () => Promise<boolean>
  addAlert: (message: string, type: 'success' | 'warning' | 'error') => void
  emitSuccess: () => void
  closeModal: () => void
}

const toNullableFormText = (value: string | null | undefined) => {
  if (typeof value !== 'string') {
    return value ?? null
  }
  return value.trim() ? value : null
}

const createUpdatePayload = (params: {
  form: EditGameForm
  seriesId: number | null
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
    logo_visible: params.form.logo_visible,
  }
}

export const useEditGameWorkflow = (options: UseEditGameWorkflowOptions) => {
  const handleSubmit = async () => {
    const game = options.game.value
    if (!game) return
    if (options.isSubmitting.value) return

    const isValid = await options.validateForm()
    if (!isValid) return

    options.isSubmitting.value = true

    try {
      const seriesId = options.form.value.series_id
      const developerIds = options.form.value.developer_ids
      const publisherIds = options.form.value.publisher_ids

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
      const orderedLogoUids = options.form.value.logos
        .map((item) => item.asset_uid)
        .filter((assetUid): assetUid is string => Boolean(assetUid))
      const logoPositions = options.form.value.logos
        .filter((item) => item.asset_uid)
        .map((item) => ({
          asset_uid: item.asset_uid as string,
          position_x: item.position_x ?? null,
          position_y: item.position_y ?? null,
          width_pct: item.width_pct ?? null,
        }))

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
      for (const l of options.form.value.logos) {
        if (l.asset_uid && l.path) {
          newAssets.push({ asset_uid: l.asset_uid, asset_type: 'logo', path: l.path })
        }
      }

      if (!game.public_id) {
        throw new Error('缺少游戏 public_id')
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
    handleSubmit,
  }
}
