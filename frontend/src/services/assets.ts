import api from './api'
import type { ApiEnvelope } from './types'

export interface UploadedAssetResult {
  path: string
  asset_id?: number
  asset_uid?: string
}

export async function uploadAsset(
  assetType: 'cover' | 'banner' | 'screenshot' | 'video',
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
