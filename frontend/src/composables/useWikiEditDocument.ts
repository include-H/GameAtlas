import { computed, ref } from 'vue'
import wikiService, { type WikiDocumentResponse } from '@/services/wiki.service'
import { getHttpErrorMessage } from '@/utils/http-error'
import { useGamesStore } from '@/stores/games'
import { useUiStore } from '@/stores/ui'

interface UseWikiEditDocumentOptions {
  gamesStore: ReturnType<typeof useGamesStore>
  uiStore: ReturnType<typeof useUiStore>
  onLoadGameFailed: () => void | Promise<void>
  onSaveSuccess?: (gameId: string) => void | Promise<void>
}

export const useWikiEditDocument = ({
  gamesStore,
  uiStore,
  onLoadGameFailed,
  onSaveSuccess,
}: UseWikiEditDocumentOptions) => {
  const game = computed(() => gamesStore.currentGame)
  const wiki = ref<WikiDocumentResponse | null>(null)
  const isSaving = ref(false)
  const wikiData = ref({
    content: '',
    change_summary: '',
  })

  const isExisting = computed(() => Boolean(wiki.value?.content))

  const resetWikiEditorState = () => {
    wiki.value = null
    wikiData.value = {
      content: '',
      change_summary: '',
    }
  }

  const loadWikiEditorData = async (gameId: string) => {
    try {
      await gamesStore.fetchGame(gameId)
      resetWikiEditorState()

      const wikiContent = await wikiService.getWikiPage(gameId)
      if (wikiContent?.content) {
        wiki.value = wikiContent
        wikiData.value = {
          content: wikiContent.content,
          change_summary: '',
        }
      }
    } catch {
      uiStore.addAlert('Failed to load game', 'error')
      await onLoadGameFailed()
    }
  }

  const handleSave = async () => {
    if (!game.value?.public_id) return

    isSaving.value = true

    try {
      const wasExisting = isExisting.value
      await wikiService.updateWikiPage(game.value.public_id, {
        content: wikiData.value.content,
        change_summary: wikiData.value.change_summary.trim() || undefined,
      })

      uiStore.addAlert(wasExisting ? 'Wiki 已更新' : 'Wiki 已创建', 'success')
      wikiData.value.change_summary = ''
      await onSaveSuccess?.(game.value.public_id)
    } catch (error) {
      const errorMessage = getHttpErrorMessage(error, '保存 Wiki 失败')
      uiStore.addAlert(errorMessage, 'error')
      console.error('Failed to save wiki:', error)
    } finally {
      isSaving.value = false
    }
  }

  return {
    game,
    wiki,
    wikiData,
    isSaving,
    isExisting,
    loadWikiEditorData,
    handleSave,
  }
}
