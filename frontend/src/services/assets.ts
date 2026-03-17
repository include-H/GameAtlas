import api from './api'
import type { ApiResponse } from './types'

export async function uploadAsset(assetType: 'cover' | 'banner' | 'screenshot', gameId: number, file: File, sortOrder = 0) {
  const form = new FormData()
  form.append('game_id', String(gameId))
  form.append('sort_order', String(sortOrder))
  form.append('file', file)

  const { data } = await api.post<ApiResponse<{ path: string }>>(`/assets/${assetType}`, form, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })

  return data.data.path
}

export async function deleteAsset(gameId: number, assetType: 'cover' | 'banner' | 'screenshot', path: string) {
  const { data } = await api.delete<ApiResponse<{ deleted: boolean }>>('/assets', {
    data: {
      game_id: gameId,
      asset_type: assetType,
      path,
    },
  })

  return data.data
}
