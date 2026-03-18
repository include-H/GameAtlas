import api from './api'
import type { ApiResponse } from './types'

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

  const { data } = await api.post<ApiResponse<UploadedAssetResult>>(`/assets/${assetType}`, form, {
    headers: { 'Content-Type': 'multipart/form-data' },
    onUploadProgress: (event) => {
      if (!onProgress || !event.total) return
      onProgress(Math.min(100, Math.round((event.loaded / event.total) * 100)))
    },
  })

  return data.data
}

export async function deleteAsset(
  gameId: number,
  assetType: 'cover' | 'banner' | 'screenshot' | 'video',
  path: string,
  assetId?: number,
  assetUid?: string,
) {
  const { data } = await api.delete<ApiResponse<{ deleted: boolean }>>('/assets', {
    data: {
      game_id: gameId,
      asset_id: assetId,
      asset_uid: assetUid,
      asset_type: assetType,
      path,
    },
  })

  return data.data
}

export async function reorderScreenshots(gameId: number, assetUids: string[]) {
  const { data } = await api.put<ApiResponse<{ updated: boolean }>>('/assets/screenshot/order', {
    game_id: gameId,
    asset_uids: assetUids,
  })

  return data.data
}
