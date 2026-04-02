import { ref, type Ref } from 'vue'
import type { EditGameForm } from '@/composables/edit-game-form'
import { getHttpErrorMessage } from '@/utils/http-error'

export interface FilePathItem {
  id?: number
  path: string
  label: string
}

export interface FilePathItemUpdatePayload {
  index: number
  field: 'path' | 'label'
  value: string
}

interface UseGameFilePathsOptions {
  form: Ref<Pick<EditGameForm, 'file_paths'>>
  getDefaultDirectory: () => Promise<string>
  onResolveInitialPathError?: (message: string) => void
}

export const useGameFilePaths = (options: UseGameFilePathsOptions) => {
  const showFileBrowser = ref(false)
  const initialPath = ref('')
  const currentFileIndex = ref(-1)

  const addFilePath = () => {
    options.form.value.file_paths.push({ path: '', label: '' })
  }

  const removeFilePath = (index: number) => {
    options.form.value.file_paths.splice(index, 1)
  }

  const openFileBrowser = async (index: number) => {
    currentFileIndex.value = index
    try {
      const defaultPath = await options.getDefaultDirectory()
      const existingPath = (options.form.value.file_paths[index]?.path || '').trim()
      if (!existingPath) {
        initialPath.value = defaultPath
      } else if (!existingPath.includes('/') && !existingPath.includes('\\')) {
        initialPath.value = defaultPath
      } else {
        initialPath.value = existingPath.replace(/[\\/][^\\/]*$/, '') || defaultPath
      }
      showFileBrowser.value = true
    } catch (error) {
      options.onResolveInitialPathError?.(getHttpErrorMessage(error, '获取默认目录失败'))
    }
  }

  const handleFileSelect = (path: string) => {
    if (currentFileIndex.value >= 0) {
      options.form.value.file_paths[currentFileIndex.value].path = path
    }
  }

  const handleFilePathItemUpdate = (payload: FilePathItemUpdatePayload) => {
    const target = options.form.value.file_paths[payload.index]
    if (!target) return
    target[payload.field] = payload.value
  }

  const resetFileBrowserState = () => {
    showFileBrowser.value = false
    initialPath.value = ''
    currentFileIndex.value = -1
  }

  return {
    showFileBrowser,
    initialPath,
    currentFileIndex,
    addFilePath,
    removeFilePath,
    openFileBrowser,
    handleFileSelect,
    handleFilePathItemUpdate,
    resetFileBrowserState,
  }
}
