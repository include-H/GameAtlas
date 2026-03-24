import { ref, type Ref } from 'vue'

export interface FilePathItem {
  id?: number
  path: string
  label: string
}

interface UseGameFilePathsOptions {
  filePaths: Ref<FilePathItem[]>
  getDefaultDirectory: () => Promise<string>
  onResolveInitialPathError?: (error: unknown) => void
}

export const useGameFilePaths = (options: UseGameFilePathsOptions) => {
  const showFileBrowser = ref(false)
  const initialPath = ref('')
  const currentFileIndex = ref(-1)

  const addFilePath = () => {
    options.filePaths.value.push({ path: '', label: '' })
  }

  const removeFilePath = (index: number) => {
    options.filePaths.value.splice(index, 1)
  }

  const openFileBrowser = async (index: number) => {
    currentFileIndex.value = index
    try {
      const defaultPath = await options.getDefaultDirectory()
      const existingPath = (options.filePaths.value[index]?.path || '').trim()
      if (!existingPath) {
        initialPath.value = defaultPath
      } else if (!existingPath.includes('/') && !existingPath.includes('\\')) {
        initialPath.value = defaultPath
      } else {
        initialPath.value = existingPath.replace(/[\\/][^\\/]*$/, '') || defaultPath
      }
      showFileBrowser.value = true
    } catch (error) {
      options.onResolveInitialPathError?.(error)
    }
  }

  const handleFileSelect = (path: string) => {
    if (currentFileIndex.value >= 0) {
      options.filePaths.value[currentFileIndex.value].path = path
    }
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
    resetFileBrowserState,
  }
}
