import api from './api'

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

export const directoryService = {
  getDefaultDirectory(): Promise<string> {
    return api.get('/directory/default').then((res) => res.data.data.path)
  },

  listDirectory(path?: string): Promise<DirectoryListResponse> {
    return api.get('/directory/list', { params: path ? { path } : undefined }).then((res) => ({
      currentPath: res.data.data.current_path,
      parentPath: res.data.data.parent_path,
      items: (res.data.data.items || []).map((item: any) => ({
        name: item.name,
        path: item.path,
        type: item.is_directory ? 'directory' : 'file',
        size: item.size_bytes,
      })),
    }))
  },
}
