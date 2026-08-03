import { get, put, post, del } from './api'
import type { ApiEnvelope } from './types'

export interface EnvEntry {
  key: string
  value: string
  label: string
  group: 'general' | 'smb' | 'runtime' | 'paths' | 'backup' | 'auth' | 'network'
}

export const settingsService = {
  getConfig(): Promise<EnvEntry[]> {
    return get<ApiEnvelope<EnvEntry[]>>('/settings/config').then((res) => res.data)
  },

  updateConfig(updates: Record<string, string>): Promise<{ message: string }> {
    return put<ApiEnvelope<{ message: string }>>('/settings/config', updates).then((res) => res.data)
  },

  uploadBackground(file: File): Promise<{ message: string }> {
    const formData = new FormData()
    formData.append('bg', file)
    return post<ApiEnvelope<{ message: string }>>('/settings/bg', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    }).then((res) => res.data)
  },

  removeBackground(): Promise<{ message: string }> {
    return del<ApiEnvelope<{ message: string }>>('/settings/bg').then((res) => res.data)
  },

  restart(): Promise<{ message: string }> {
    return post<ApiEnvelope<{ message: string }>>('/settings/restart', {}).then((res) => res.data)
  },
}
