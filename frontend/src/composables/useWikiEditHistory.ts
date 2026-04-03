import { ref, type Ref } from 'vue'
import wikiService, { type WikiHistoryEntry } from '@/services/wiki.service'

interface UseWikiEditHistoryOptions {
  wikiData: Ref<{
    content: string
    change_summary: string
  }>
  addAlert: (message: string, type: 'success' | 'warning' | 'error') => void
  formatDateTime: (value?: string) => string
}

export const useWikiEditHistory = ({
  wikiData,
  addAlert,
  formatDateTime,
}: UseWikiEditHistoryOptions) => {
  const historyEntries = ref<WikiHistoryEntry[]>([])
  const selectedHistory = ref<WikiHistoryEntry | null>(null)
  const isHistoryLoading = ref(false)
  const previewHistoryContent = ref(true)
  const historyPreviewVisible = ref(false)

  const resetHistoryState = () => {
    historyEntries.value = []
    selectedHistory.value = null
    previewHistoryContent.value = true
    historyPreviewVisible.value = false
  }

  const loadHistory = async (gameId: string) => {
    isHistoryLoading.value = true
    try {
      historyEntries.value = await wikiService.getWikiHistory(gameId)
      selectedHistory.value = historyEntries.value[0] || null
    } catch {
      historyEntries.value = []
      selectedHistory.value = null
    } finally {
      isHistoryLoading.value = false
    }
  }

  const restoreHistory = () => {
    if (!selectedHistory.value) return

    wikiData.value.content = selectedHistory.value.content
    wikiData.value.change_summary = `恢复历史版本：${selectedHistory.value.change_summary || formatDateTime(selectedHistory.value.created_at)}`
    historyPreviewVisible.value = false
    addAlert('已将历史版本内容恢复到编辑器', 'success')
  }

  const openHistoryDialog = () => {
    if (historyEntries.value.length === 0) return
    if (!selectedHistory.value) {
      selectedHistory.value = historyEntries.value[0] || null
    }
    previewHistoryContent.value = true
    historyPreviewVisible.value = true
  }

  const openHistoryPreview = (entry: WikiHistoryEntry) => {
    selectedHistory.value = entry
    previewHistoryContent.value = true
    historyPreviewVisible.value = true
  }

  return {
    historyEntries,
    selectedHistory,
    isHistoryLoading,
    previewHistoryContent,
    historyPreviewVisible,
    resetHistoryState,
    loadHistory,
    restoreHistory,
    openHistoryDialog,
    openHistoryPreview,
  }
}
