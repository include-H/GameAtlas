import api, { del } from './api'
import type { ApiEnvelope } from './types'

export interface UploadedAssetResult {
  path: string
  asset_id?: number
  asset_uid?: string
}

export type AssetType = 'cover' | 'banner' | 'screenshot' | 'video' | 'logo'

export async function uploadAsset(
  assetType: AssetType,
  gameId: number,
  file: File,
  sortOrder = 0,
  onProgress?: (percent: number) => void,
) {
  const form = new FormData()
  form.append('game_id', String(gameId))
  form.append('sort_order', String(sortOrder))
  form.append('file', file)

  const { data } = await api.post<ApiEnvelope<UploadedAssetResult>>(`/assets/${assetType}`, form, {
    headers: { 'Content-Type': 'multipart/form-data' },
    onUploadProgress: (event) => {
      if (!onProgress || !event.total) return
      onProgress(Math.min(100, Math.round((event.loaded / event.total) * 100)))
    },
  })

  return data.data
}

export async function deleteAsset(
  assetType: AssetType,
  gameId: number,
  assetUid: string,
): Promise<void> {
  await del(`/assets/${assetType}?game_id=${gameId}&asset_uid=${encodeURIComponent(assetUid)}`)
}
