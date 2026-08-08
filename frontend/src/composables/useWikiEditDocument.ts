import { computed, ref, type Ref } from 'vue'
import wikiService, { type WikiDocumentResponse } from '@/services/wiki.service'
import { getHttpErrorMessage } from '@/utils/http-error'
import { useGamesStore } from '@/stores/games'
import { useUiStore } from '@/stores/ui'

interface UseWikiEditDocumentOptions {
  gamesStore: ReturnType<typeof useGamesStore>
  uiStore: ReturnType<typeof useUiStore>
  requestedGameId: Ref<string>
  onLoadGameFailed: () => void | Promise<void>
  onSaveSuccess?: (gameId: string) => void | Promise<void>
}

export const useWikiEditDocument = ({
  gamesStore,
  uiStore,
  requestedGameId,
  onLoadGameFailed,
  onSaveSuccess,
}: UseWikiEditDocumentOptions) => {
  const game = computed(() => {
    // 2026-04-08: wiki edit reads the route-targeted game only.
    // Impact: route changes or failed loads must not leave the editor rendering a stale
    // currentGame from a previous page.
    return gamesStore.currentGame?.public_id === requestedGameId.value
      ? gamesStore.currentGame
      : null
  })
  const wiki = ref<WikiDocumentResponse | null>(null)
  const isSaving = ref(false)
  const wikiData = ref({
    content: '',
    change_summary: '',
  })

  const isExisting = computed(() => Boolean(wiki.value && wiki.value.content !== null))

  const resetWikiEditorState = () => {
    wiki.value = null
    wikiData.value = {
      content: '',
      change_summary: '',
    }
  }

  const loadWikiEditorData = async (gameId: string): Promise<boolean> => {
    try {
      await gamesStore.fetchGame(gameId)
    } catch {
      uiStore.addAlert('加载游戏失败', 'error')
      await onLoadGameFailed()
      return false
    }

    resetWikiEditorState()

    try {
      const wikiContent = await wikiService.getWikiPage(gameId)
      // 2026-04-06: the wiki endpoint returns a document envelope for existing games
      // even when content is empty, so the editor must not treat empty text as "no wiki".
      // A 404 here is a broken cross-endpoint contract, not a valid "missing document" state.
      wiki.value = wikiContent
      wikiData.value = {
        content: wikiContent.content ?? '',
        change_summary: wikiContent.content === null ? '首次添加' : '',
      }
    } catch (error) {
      uiStore.addAlert(getHttpErrorMessage(error, '加载 Wiki 失败'), 'error')
      await onLoadGameFailed()
      return false
    }

    return true
  }

  const handleSave = async () => {
    if (!requestedGameId.value || !game.value?.public_id) return

    isSaving.value = true

    try {
      const wasExisting = isExisting.value
      const document = await wikiService.updateWikiPage(requestedGameId.value, {
        content: wikiData.value.content,
        change_summary: wikiData.value.change_summary.trim() || undefined,
      })
      wiki.value = document

      uiStore.addAlert(wasExisting ? 'Wiki 已更新' : 'Wiki 已创建', 'success')
      wikiData.value.change_summary = ''
      await onSaveSuccess?.(requestedGameId.value)
    } catch (error) {
      const errorMessage = getHttpErrorMessage(error, '保存 Wiki 失败')
      uiStore.addAlert(errorMessage, 'error')
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
