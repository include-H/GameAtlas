import { get } from './api'
import type { ApiEnvelope } from './types'

export interface DirectoryItem {
  name: string
  path: string
  type: 'file' | 'directory'
  size?: number | null
  extension?: string
}

interface DirectoryListResponse {
  currentPath: string
  parentPath: string | null
  items: DirectoryItem[]
}

interface DirectoryDefaultData {
  path: string
}

interface DirectoryListApiItem {
  name: string
  path: string
  is_directory: boolean
  size_bytes?: number | null
}

interface DirectoryListData {
  current_path: string
  parent_path: string | null
  items: DirectoryListApiItem[]
}

export const directoryService = {
  getDefaultDirectory(): Promise<string> {
    return get<ApiEnvelope<DirectoryDefaultData>>('/directory/default').then((res) => res.data.path)
  },

  listDirectory(path?: string): Promise<DirectoryListResponse> {
    return get<ApiEnvelope<DirectoryListData>>('/directory/list', { params: path ? { path } : undefined }).then((res) => ({
      currentPath: res.data.current_path,
      parentPath: res.data.parent_path,
      items: (res.data.items || []).map((item) => ({
        name: item.name,
        path: item.path,
        type: item.is_directory ? 'directory' : 'file',
        size: item.size_bytes,
      })),
    }))
  },
}
